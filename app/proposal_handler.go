package app

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	abci "github.com/cometbft/cometbft/abci/types"
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
	BlockHeight        int64              `json:"block_height"`
	OpAndEVMAddrs      OperatorAndEVM     `json:"op_and_evm_addrs"`
	ValsetSigs         ValsetSignatures   `json:"valset_sigs"`
	OracleAttestations OracleAttestations `json:"oracle_attestations"`
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
	err := baseapp.ValidateVoteExtensions(ctx, h.valStore, req.Height, ctx.ChainID(), req.LocalLastCommit)
	if err != nil {
		h.logger.Info("failed to validate vote extensions", "error", err, "votes", len(req.LocalLastCommit.Votes))
		// return nil, err
	}
	proposalTxs := req.Txs
	injectedVoteExtTx := VoteExtTx{}

	if req.Height > ctx.ConsensusParams().Abci.VoteExtensionsEnableHeight {
		operatorAddresses, evmAddresses, err := h.CheckInitialSignaturesFromLastCommit(ctx, req.LocalLastCommit)
		if err != nil {
			h.logger.Info("failed to check initial signatures from last commit", "error", err)
			bz, err := json.Marshal(injectedVoteExtTx)
			if err != nil {
				h.logger.Error("failed to encode injected vote extension tx", "err", err)
				return nil, errors.New("failed to encode injected vote extension tx")
			}
			proposalTxs = append([][]byte{bz}, proposalTxs...)
			return &abci.ResponsePrepareProposal{
				Txs: proposalTxs,
			}, nil
		}
		operatorAndEvm := OperatorAndEVM{
			OperatorAddresses: operatorAddresses,
			EVMAddresses:      evmAddresses,
		}

		valsetOperatorAddresses, valsetTimestamps, valsetSignatures, err := h.CheckValsetSignaturesFromLastCommit(ctx, req.LocalLastCommit)
		if err != nil {
			h.logger.Info("failed to check valset signatures from last commit", "error", err)
			bz, err := json.Marshal(injectedVoteExtTx)
			if err != nil {
				h.logger.Error("failed to encode injected vote extension tx", "err", err)
				return nil, errors.New("failed to encode injected vote extension tx")
			}
			proposalTxs = append([][]byte{bz}, proposalTxs...)
			return &abci.ResponsePrepareProposal{
				Txs: proposalTxs,
			}, nil
		}
		valsetSigs := ValsetSignatures{
			OperatorAddresses: valsetOperatorAddresses,
			Timestamps:        valsetTimestamps,
			Signatures:        valsetSignatures,
		}

		oracleSigs, oracleSnapshots, oracleOperatorAddresses, err := h.CheckOracleAttestationsFromLastCommit(ctx, req.LocalLastCommit)
		if err != nil {
			h.logger.Info("failed to check oracle attestations from last commit", "error", err)
		}

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
		}

		bz, err := json.Marshal(injectedVoteExtTx)
		if err != nil {
			h.logger.Error("failed to encode injected vote extension tx", "err", err)
			return nil, errors.New("failed to encode injected vote extension tx")
		}

		proposalTxs = append([][]byte{bz}, proposalTxs...)
	}

	return &abci.ResponsePrepareProposal{
		Txs: proposalTxs,
	}, nil
}

func (h *ProposalHandler) ProcessProposalHandler(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
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
			h.logger.Error("failed to decode injected vote extension tx", "err", err)
			return nil, errors.New("failed to decode injected vote extension tx")
		}

		if len(injectedVoteExtTx.OpAndEVMAddrs.OperatorAddresses) > 0 {
			if err := h.SetEVMAddresses(ctx, injectedVoteExtTx.OpAndEVMAddrs.OperatorAddresses, injectedVoteExtTx.OpAndEVMAddrs.EVMAddresses); err != nil {
				return nil, err
			}
		}

		if len(injectedVoteExtTx.ValsetSigs.OperatorAddresses) > 0 {
			for i, operatorAddress := range injectedVoteExtTx.ValsetSigs.OperatorAddresses {
				timestamp := injectedVoteExtTx.ValsetSigs.Timestamps[i]
				sigHexString := injectedVoteExtTx.ValsetSigs.Signatures[i]
				err := h.bridgeKeeper.SetBridgeValsetSignature(ctx, operatorAddress, uint64(timestamp), sigHexString)
				if err != nil {
					return nil, err
				}
			}
		}

		if len(injectedVoteExtTx.OracleAttestations.OperatorAddresses) > 0 {
			for i, operatorAddress := range injectedVoteExtTx.OracleAttestations.OperatorAddresses {
				snapshot := injectedVoteExtTx.OracleAttestations.Snapshots[i]
				attestation := injectedVoteExtTx.OracleAttestations.Attestations[i]
				err := h.bridgeKeeper.SetOracleAttestation(ctx, operatorAddress, snapshot, attestation)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return res, nil
}

func (h *ProposalHandler) CheckInitialSignaturesFromLastCommit(ctx sdk.Context, commit abci.ExtendedCommitInfo) ([]string, []string, error) {
	var operatorAddresses []string
	var evmAddresses []string

	for _, vote := range commit.Votes {
		extension := vote.GetVoteExtension()
		// unmarshal vote extension
		voteExt := BridgeVoteExtension{}
		err := json.Unmarshal(extension, &voteExt)
		if err != nil {
			h.logger.Error("failed to unmarshal vote extension", "error", err)
			return nil, nil, errors.New("failed to unmarshal vote extension")
		}

		// check for initial sig
		if len(voteExt.InitialSignature.SignatureA) > 0 {
			// verify initial sig
			evmAddress, err := h.bridgeKeeper.EVMAddressFromSignatures(ctx, voteExt.InitialSignature.SignatureA, voteExt.InitialSignature.SignatureB)
			if err != nil {
				h.logger.Error("failed to get evm address from initial sig", "error", err)
				return nil, nil, err
			}

			operatorAddress, err := h.ValidatorOperatorAddressFromVote(ctx, vote)
			if err != nil {
				h.logger.Error("failed to get operator address from vote", "error", err)
				return nil, nil, err
			}

			operatorAddresses = append(operatorAddresses, operatorAddress)
			evmAddresses = append(evmAddresses, evmAddress.Hex())
		}
	}

	if len(operatorAddresses) == 0 {
		emptyStringArray := make([]string, 0)
		return emptyStringArray, emptyStringArray, nil
	}

	return operatorAddresses, evmAddresses, nil
}

func (h *ProposalHandler) CheckValsetSignaturesFromLastCommit(ctx sdk.Context, commit abci.ExtendedCommitInfo) ([]string, []int64, []string, error) {
	var operatorAddresses []string
	var timestamps []int64
	var signatures []string

	for _, vote := range commit.Votes {
		extension := vote.GetVoteExtension()
		// unmarshal vote extension
		voteExt := BridgeVoteExtension{}
		err := json.Unmarshal(extension, &voteExt)
		if err != nil {
			h.logger.Error("failed to unmarshal vote extension", "error", err)
			return nil, nil, nil, errors.New("failed to unmarshal vote extension")
		}

		// check for valset sig
		if len(voteExt.ValsetSignature.Signature) > 0 {
			// verify valset sig
			sigHexString := hex.EncodeToString(voteExt.ValsetSignature.Signature)
			operatorAddress, err := h.ValidatorOperatorAddressFromVote(ctx, vote)
			if err != nil {
				h.logger.Error("failed to get operator address from vote", "error", err)
				return nil, nil, nil, err
			}

			timestamp := voteExt.ValsetSignature.Timestamp
			operatorAddresses = append(operatorAddresses, operatorAddress)
			timestamps = append(timestamps, int64(timestamp))
			signatures = append(signatures, sigHexString)
		}
	}
	return operatorAddresses, timestamps, signatures, nil
}

func (h *ProposalHandler) SetEVMAddresses(ctx sdk.Context, operatorAddresses, evmAddresses []string) error {
	for i, operatorAddress := range operatorAddresses {
		bzAddress := common.HexToAddress(evmAddresses[i])
		err := h.bridgeKeeper.SetEVMAddressByOperator(ctx, operatorAddress, bzAddress.Bytes())
		if err != nil {
			h.logger.Error("failed to set evm address by operator", "error", err)
			return err
		}
	}
	return nil
}

func (h *ProposalHandler) ValidatorOperatorAddressFromVote(ctx sdk.Context, vote abci.ExtendedVoteInfo) (string, error) {
	consAddress := vote.Validator.Address
	validator, err := h.stakingKeeper.GetValidatorByConsAddr(ctx, consAddress)
	if err != nil {
		h.logger.Error("failed to get validator by consensus address", "error", err)
		return "", err
	}
	operatorAddress := validator.OperatorAddress
	return operatorAddress, nil
}

func (h *ProposalHandler) CheckOracleAttestationsFromLastCommit(ctx sdk.Context, commit abci.ExtendedCommitInfo) ([][]byte, [][]byte, []string, error) {
	var attestations [][]byte
	var operatorAddresses []string
	var snapshots [][]byte

	for _, vote := range commit.Votes {
		extension := vote.GetVoteExtension()
		// unmarshal vote extension
		voteExt := BridgeVoteExtension{}
		err := json.Unmarshal(extension, &voteExt)
		if err != nil {
			h.logger.Error("failed to unmarshal vote extension", "error", err)
			return nil, nil, nil, errors.New("failed to unmarshal vote extension")
		}

		// check for oracle attestation
		if len(voteExt.OracleAttestations) > 0 {
			// verify oracle attestation
			for _, attestation := range voteExt.OracleAttestations {
				operatorAddress, err := h.ValidatorOperatorAddressFromVote(ctx, vote)
				if err != nil {
					h.logger.Error("failed to get operator address from vote", "error", err)
					return nil, nil, nil, err
				}

				operatorAddresses = append(operatorAddresses, operatorAddress)

				snapshot := attestation.Snapshot
				snapshots = append(snapshots, snapshot)

				attestations = append(attestations, attestation.Attestation)
			}
		}
	}
	return attestations, snapshots, operatorAddresses, nil
}
