package app

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/ethereum/go-ethereum/common"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ProposalHandler struct {
	logger        log.Logger
	valStore      baseapp.ValidatorStore // to get the current validators' pubkeys
	oracleKeeper  OracleKeeper
	bridgeKeeper  BridgeKeeper
	stakingKeeper StakingKeeper
	codec         codec.Codec
}

type OperatorAndEVM struct {
	OperatorAddresses []string `json:"operator_addresses"`
	EVMAddresses      []string `json:"evm_addresses"`
}

type ValsetSignatures struct {
	OperatorAddresses []string `json:"operator_addresses"`
	Timestamps        []int64  `json:"timestamps"`
	Signatures        []string `json:"signatures"`
}

type OracleAttestations struct {
	OperatorAddresses []string `json:"operator_addresses"`
	Attestations      [][]byte `json:"attestations"`
	Snapshots         [][]byte `json:"snapshots"`
}

type VoteExtTx struct {
	BlockHeight        int64                   `json:"block_height"`
	OpAndEVMAddrs      OperatorAndEVM          `json:"op_and_evm_addrs"`
	ValsetSigs         ValsetSignatures        `json:"valset_sigs"`
	OracleAttestations OracleAttestations      `json:"oracle_attestations"`
	ExtendedCommitInfo abci.ExtendedCommitInfo `json:"extended_commit_info"`
}

func NewProposalHandler(logger log.Logger, valStore baseapp.ValidatorStore, appCodec codec.Codec, oracleKeeper OracleKeeper, bridgeKeeper BridgeKeeper, stakingKeeper StakingKeeper) *ProposalHandler {
	return &ProposalHandler{
		oracleKeeper:  oracleKeeper,
		bridgeKeeper:  bridgeKeeper,
		stakingKeeper: stakingKeeper,
		logger:        logger,
		codec:         appCodec,
		valStore:      valStore,
	}
}

func (h *ProposalHandler) PrepareProposalHandler(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	maxValidators, err := h.GetMaxValidators(ctx)
	if err != nil {
		h.logger.Error("PrepareProposalHandler: failed to get max validators", "error", err)
		return nil, err
	}
	// Verify number of votes doesn't exceed max validators
	if len(req.LocalLastCommit.Votes) > int(maxValidators) {
		h.logger.Error("PrepareProposalHandler: number of votes exceeds max validators",
			"votes", len(req.LocalLastCommit.Votes),
			"max_validators", maxValidators)
		return nil, errors.New("number of votes exceeds max validators")
	}
	proposalTxs := req.Txs

	if req.Height > ctx.ConsensusParams().Abci.VoteExtensionsEnableHeight {
		err = baseapp.ValidateVoteExtensions(ctx, h.valStore, req.Height, ctx.ChainID(), req.LocalLastCommit)
		if err != nil {
			h.logger.Info("PrepareProposalHandler: failed to validate vote extensions", "error", err, "votes", req.LocalLastCommit.Votes)
			return nil, err
		}
		operatorAddresses, evmAddresses := h.CheckInitialSignaturesFromLastCommit(ctx, req.LocalLastCommit)
		operatorAndEvm := OperatorAndEVM{
			OperatorAddresses: operatorAddresses,
			EVMAddresses:      evmAddresses,
		}

		valsetOperatorAddresses, valsetTimestamps, valsetSignatures := h.CheckValsetSignaturesFromLastCommit(ctx, req.LocalLastCommit)
		valsetSigs := ValsetSignatures{
			OperatorAddresses: valsetOperatorAddresses,
			Timestamps:        valsetTimestamps,
			Signatures:        valsetSignatures,
		}

		oracleSigs, oracleSnapshots, oracleOperatorAddresses := h.CheckOracleAttestationsFromLastCommit(ctx, req.LocalLastCommit)
		oracleAttestations := OracleAttestations{
			OperatorAddresses: oracleOperatorAddresses,
			Attestations:      oracleSigs,
			Snapshots:         oracleSnapshots,
		}

		injectedVoteExtTx := VoteExtTx{
			BlockHeight:        req.Height,
			OpAndEVMAddrs:      operatorAndEvm,
			ValsetSigs:         valsetSigs,
			OracleAttestations: oracleAttestations,
			ExtendedCommitInfo: req.LocalLastCommit,
		}

		bz, err := json.Marshal(injectedVoteExtTx)
		if err != nil {
			h.logger.Error("PrepareProposalHandler: failed to encode injected vote extension tx", "err", err)
			return nil, errors.New("failed to encode injected vote extension tx")
		}

		proposalTxs = append([][]byte{bz}, proposalTxs...)
	}

	return &abci.ResponsePrepareProposal{
		Txs: proposalTxs,
	}, nil
}

func (h *ProposalHandler) ProcessProposalHandler(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	if req.Height > ctx.ConsensusParams().Abci.VoteExtensionsEnableHeight {
		var injectedVoteExtTx VoteExtTx
		if err := json.Unmarshal(req.Txs[0], &injectedVoteExtTx); err != nil {
			h.logger.Error("ProcessProposalHandler: failed to decode injected vote extension tx", "err", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		err := baseapp.ValidateVoteExtensions(ctx, h.valStore, req.Height, ctx.ChainID(), injectedVoteExtTx.ExtendedCommitInfo)
		if err != nil {
			h.logger.Error("ProcessProposalHandler: rejecting proposal, failed to validate vote extension", "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}
		maxValidators, err := h.GetMaxValidators(ctx)
		if err != nil {
			h.logger.Error("ProcessProposalHandler: failed to get max validators", "error", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, err
		}
		// Verify number of votes doesn't exceed max validators
		if len(injectedVoteExtTx.ExtendedCommitInfo.Votes) > int(maxValidators) {
			h.logger.Error("ProcessProposalHandler: number of votes exceeds max validators", "votes", len(injectedVoteExtTx.ExtendedCommitInfo.Votes), "max_validators", maxValidators)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}
		operatorAddresses, evmAddresses := h.CheckInitialSignaturesFromLastCommit(ctx, injectedVoteExtTx.ExtendedCommitInfo)

		if !reflect.DeepEqual(operatorAddresses, injectedVoteExtTx.OpAndEVMAddrs.OperatorAddresses) {
			h.logger.Error("ProcessProposalHandler: rejecting proposal, operator addresses do not match", "operatorAddresses", operatorAddresses, "injectedVoteExtTx", injectedVoteExtTx.OpAndEVMAddrs.OperatorAddresses)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		if !reflect.DeepEqual(evmAddresses, injectedVoteExtTx.OpAndEVMAddrs.EVMAddresses) {
			h.logger.Error("ProcessProposalHandler: rejecting proposal, evm addresses do not match", "evmAddresses", evmAddresses, "injectedVoteExtTx", injectedVoteExtTx.OpAndEVMAddrs.EVMAddresses)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		valsetOperatorAddresses, valsetTimestamps, valsetSignatures := h.CheckValsetSignaturesFromLastCommit(ctx, injectedVoteExtTx.ExtendedCommitInfo)

		if !reflect.DeepEqual(valsetOperatorAddresses, injectedVoteExtTx.ValsetSigs.OperatorAddresses) {
			h.logger.Error("ProcessProposalHandler: rejecting proposal, valset operator addresses do not match", "valsetOperatorAddresses", valsetOperatorAddresses, "injectedVoteExtTx", injectedVoteExtTx.ValsetSigs.OperatorAddresses)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		if !reflect.DeepEqual(valsetTimestamps, injectedVoteExtTx.ValsetSigs.Timestamps) {
			h.logger.Error("ProcessProposalHandler: rejecting proposal, valset timestamps do not match", "valsetTimestamps", valsetTimestamps, "injectedVoteExtTx", injectedVoteExtTx.ValsetSigs.Timestamps)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		if !reflect.DeepEqual(valsetSignatures, injectedVoteExtTx.ValsetSigs.Signatures) {
			h.logger.Error("ProcessProposalHandler: rejecting proposal, valset signatures do not match", "valsetSignatures", valsetSignatures, "injectedVoteExtTx", injectedVoteExtTx.ValsetSigs.Signatures)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		oracleSigs, oracleSnapshots, oracleOperatorAddresses := h.CheckOracleAttestationsFromLastCommit(ctx, injectedVoteExtTx.ExtendedCommitInfo)

		if !reflect.DeepEqual(oracleSigs, injectedVoteExtTx.OracleAttestations.Attestations) {
			h.logger.Error("ProcessProposalHandler: rejecting proposal, oracle signatures do not match", "oracleSigs", oracleSigs, "injectedVoteExtTx", injectedVoteExtTx.OracleAttestations.Attestations)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		if !reflect.DeepEqual(oracleSnapshots, injectedVoteExtTx.OracleAttestations.Snapshots) {
			h.logger.Error("ProcessProposalHandler: rejecting proposal, oracle snapshots do not match", "oracleSnapshots", oracleSnapshots, "injectedVoteExtTx", injectedVoteExtTx.OracleAttestations.Snapshots)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}

		if !reflect.DeepEqual(oracleOperatorAddresses, injectedVoteExtTx.OracleAttestations.OperatorAddresses) {
			h.logger.Error("ProcessProposalHandler: rejecting proposal, oracle operator addresses do not match", "oracleOperatorAddresses", oracleOperatorAddresses, "injectedVoteExtTx", injectedVoteExtTx.OracleAttestations.OperatorAddresses)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, nil
		}
	}

	return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
}

func (h *ProposalHandler) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	res := &sdk.ResponsePreBlock{}
	if len(req.Txs) == 0 {
		return res, nil
	}

	if req.Height > ctx.ConsensusParams().Abci.VoteExtensionsEnableHeight {
		var injectedVoteExtTx VoteExtTx
		if err := json.Unmarshal(req.Txs[0], &injectedVoteExtTx); err != nil {
			h.logger.Error("PreBlocker: failed to decode injected vote extension tx", "err", err)
			return nil, errors.New("failed to decode injected vote extension tx")
		}

		if len(injectedVoteExtTx.OpAndEVMAddrs.OperatorAddresses) > 0 {
			if err := h.SetEVMAddresses(ctx, injectedVoteExtTx.OpAndEVMAddrs.OperatorAddresses, injectedVoteExtTx.OpAndEVMAddrs.EVMAddresses); err != nil {
				h.logger.Error("PreBlocker: failed to set EVM addresses", "error", err)
				return nil, err
			}
		}

		if len(injectedVoteExtTx.ValsetSigs.OperatorAddresses) > 0 {
			for i, operatorAddress := range injectedVoteExtTx.ValsetSigs.OperatorAddresses {
				timestamp := injectedVoteExtTx.ValsetSigs.Timestamps[i]
				sigHexString := injectedVoteExtTx.ValsetSigs.Signatures[i]
				err := h.bridgeKeeper.SetBridgeValsetSignature(ctx, operatorAddress, uint64(timestamp), sigHexString)
				if err != nil {
					h.logger.Error("PreBlocker: failed to set valset signature", "error", err)
				}
			}
		}

		if len(injectedVoteExtTx.OracleAttestations.OperatorAddresses) > 0 {
			for i, operatorAddress := range injectedVoteExtTx.OracleAttestations.OperatorAddresses {
				snapshot := injectedVoteExtTx.OracleAttestations.Snapshots[i]
				attestation := injectedVoteExtTx.OracleAttestations.Attestations[i]
				err := h.bridgeKeeper.SetOracleAttestation(ctx, operatorAddress, snapshot, attestation)
				if err != nil {
					h.logger.Error("PreBlocker: failed to set oracle attestation", "error", err)
				}
			}
		}
	}
	return res, nil
}

func (h *ProposalHandler) CheckInitialSignaturesFromLastCommit(ctx sdk.Context, commit abci.ExtendedCommitInfo) ([]string, []string) {
	var operatorAddresses []string
	var evmAddresses []string

	for _, vote := range commit.Votes {
		// Only check if the vote is a commit vote
		if vote.BlockIdFlag != cmtproto.BlockIDFlagCommit {
			continue
		}
		extension := vote.GetVoteExtension()
		// unmarshal vote extension
		voteExt := BridgeVoteExtension{}
		err := json.Unmarshal(extension, &voteExt)
		if err != nil {
			h.logger.Error("CheckInitialSignaturesFromLastCommit: failed to unmarshal vote extension", "error", err)
			// check for initial sig
		} else if len(voteExt.InitialSignature.SignatureA) > 0 {
			// get operator address from vote
			operatorAddress, err := h.ValidatorOperatorAddressFromVote(ctx, vote)
			if err != nil {
				h.logger.Error("CheckInitialSignaturesFromLastCommit: failed to get operator address from vote", "error", err)
			} else {
				// verify initial sig
				evmAddress, err := h.bridgeKeeper.EVMAddressFromSignatures(ctx, voteExt.InitialSignature.SignatureA, voteExt.InitialSignature.SignatureB, operatorAddress)
				if err != nil {
					h.logger.Error("CheckInitialSignaturesFromLastCommit: failed to get evm address from initial sig", "error", err)
				} else {
					// check for existing EVM address for operator
					_, err := h.bridgeKeeper.GetEVMAddressByOperator(ctx, operatorAddress)
					if err != nil {
						// no existing EVM address for operator
						operatorAddresses = append(operatorAddresses, operatorAddress)
						evmAddresses = append(evmAddresses, evmAddress.Hex())
					}
				}
			}
		}
	}
	if len(operatorAddresses) == 0 {
		emptyStringArray := make([]string, 0)
		return emptyStringArray, emptyStringArray
	}
	return operatorAddresses, evmAddresses
}

func (h *ProposalHandler) GetMaxValidators(ctx sdk.Context) (uint32, error) {
	params, err := h.stakingKeeper.GetParams(ctx)
	if err != nil {
		return 0, errors.New("failed to get max validators from params")
	}
	return params.MaxValidators, nil
}

func (h *ProposalHandler) CheckValsetSignaturesFromLastCommit(ctx sdk.Context, commit abci.ExtendedCommitInfo) ([]string, []int64, []string) {
	var operatorAddresses []string
	var timestamps []int64
	var signatures []string

	for _, vote := range commit.Votes {
		// Only check if the vote is a commit vote
		if vote.BlockIdFlag != cmtproto.BlockIDFlagCommit {
			continue
		}
		extension := vote.GetVoteExtension()
		// unmarshal vote extension
		voteExt := BridgeVoteExtension{}
		err := json.Unmarshal(extension, &voteExt)
		if err != nil {
			h.logger.Error("failed to unmarshal vote extension", "error", err)
			// check for valset sig
		} else if len(voteExt.ValsetSignature.Signature) > 0 {
			// verify valset sig
			sigHexString := hex.EncodeToString(voteExt.ValsetSignature.Signature)
			operatorAddress, err := h.ValidatorOperatorAddressFromVote(ctx, vote)
			if err != nil {
				h.logger.Error("failed to get operator address from vote", "error", err)
			} else {
				timestamp := voteExt.ValsetSignature.Timestamp
				operatorAddresses = append(operatorAddresses, operatorAddress)
				timestamps = append(timestamps, int64(timestamp))
				signatures = append(signatures, sigHexString)
			}
		}
	}
	return operatorAddresses, timestamps, signatures
}

func (h *ProposalHandler) SetEVMAddresses(ctx sdk.Context, operatorAddresses, evmAddresses []string) error {
	for i, operatorAddress := range operatorAddresses {
		bzAddress := common.HexToAddress(evmAddresses[i])
		err := h.bridgeKeeper.SetEVMAddressByOperator(ctx, operatorAddress, bzAddress.Bytes())
		if err != nil {
			h.logger.Error("SetEVMAddresses: failed to set evm address by operator", "error", err)
		}
	}
	return nil
}

func (h *ProposalHandler) ValidatorOperatorAddressFromVote(ctx sdk.Context, vote abci.ExtendedVoteInfo) (string, error) {
	consAddress := vote.Validator.Address
	validator, err := h.stakingKeeper.GetValidatorByConsAddr(ctx, consAddress)
	if err != nil {
		h.logger.Error("ValidatorOperatorAddressFromVote: failed to get validator by consensus address", "error", err)
		return "", err
	}
	operatorAddress := validator.OperatorAddress
	return operatorAddress, nil
}

func (h *ProposalHandler) CheckOracleAttestationsFromLastCommit(ctx sdk.Context, commit abci.ExtendedCommitInfo) ([][]byte, [][]byte, []string) {
	var attestations [][]byte
	var operatorAddresses []string
	var snapshots [][]byte

	for _, vote := range commit.Votes {
		// Only check if the vote is a commit vote
		if vote.BlockIdFlag != cmtproto.BlockIDFlagCommit {
			continue
		}
		extension := vote.GetVoteExtension()
		// unmarshal vote extension
		voteExt := BridgeVoteExtension{}
		err := json.Unmarshal(extension, &voteExt)
		if err != nil {
			h.logger.Error("CheckOracleAttestationsFromLastCommit: failed to unmarshal vote extension", "error", err)
			// check for oracle attestation
		} else if len(voteExt.OracleAttestations) > 0 {
			// verify oracle attestation
			for _, attestation := range voteExt.OracleAttestations {
				operatorAddress, err := h.ValidatorOperatorAddressFromVote(ctx, vote)
				if err != nil {
					h.logger.Error("CheckOracleAttestationsFromLastCommit: failed to get operator address from vote", "error", err)
				} else {
					operatorAddresses = append(operatorAddresses, operatorAddress)
					snapshot := attestation.Snapshot
					snapshots = append(snapshots, snapshot)
					attestations = append(attestations, attestation.Attestation)
				}
			}
		}
	}
	return attestations, snapshots, operatorAddresses
}
