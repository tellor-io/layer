package app

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
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
	Attestations      []string `json:"attestations"`
	QueryIds          []string `json:"query_ids"`
	Timestamps        []int64  `json:"timestamps"`
}

type VoteExtTx struct {
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
		h.logger.Info("failed to validate vote extensions", "error", err)
		return nil, err
	}
	proposalTxs := req.Txs
	injectedVoteExtTx := OperatorAndEVM{}

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

		oracleSigs, oracleQueryIds, oracleOperatorAddresses, oracleTimestamps, err := h.CheckOracleAttestationsFromLastCommit(ctx, req.LocalLastCommit)
		if err != nil {
			h.logger.Info("failed to check oracle attestations from last commit", "error", err)
		}

		oracleAttestations := OracleAttestations{
			OperatorAddresses: oracleOperatorAddresses,
			Attestations:      oracleSigs,
			QueryIds:          oracleQueryIds,
			Timestamps:        oracleTimestamps,
		}

		injectedVoteExtTx := VoteExtTx{
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
				queryId := injectedVoteExtTx.OracleAttestations.QueryIds[i]
				attestation := injectedVoteExtTx.OracleAttestations.Attestations[i]
				timestamp := injectedVoteExtTx.OracleAttestations.Timestamps[i]
				err := h.bridgeKeeper.SetOracleAttestation(ctx, operatorAddress, queryId, uint64(timestamp), attestation)
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
			// sigHexString := hex.EncodeToString(voteExt.InitialSignature.SignatureA)
			// evmAddress, err := h.bridgeKeeper.EVMAddressFromSignature(ctx, sigHexString)
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

func (h *ProposalHandler) SetEVMAddresses(ctx sdk.Context, operatorAddresses []string, evmAddresses []string) error {
	for i, operatorAddress := range operatorAddresses {
		h.logger.Info("SetEVMAddressByOperator", "operatorAddress", operatorAddress, "evmAddress", evmAddresses[i])
		err := h.bridgeKeeper.SetEVMAddressByOperator(ctx, operatorAddress, evmAddresses[i])
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

func (h *ProposalHandler) CheckOracleAttestationsFromLastCommit(ctx sdk.Context, commit abci.ExtendedCommitInfo) ([]string, []string, []string, []int64, error) {
	var attestations []string
	var timestamps []int64
	var operatorAddresses []string
	var queryIds []string

	for _, vote := range commit.Votes {
		extension := vote.GetVoteExtension()
		// unmarshal vote extension
		voteExt := BridgeVoteExtension{}
		err := json.Unmarshal(extension, &voteExt)
		if err != nil {
			h.logger.Error("failed to unmarshal vote extension", "error", err)
			return nil, nil, nil, nil, errors.New("failed to unmarshal vote extension")
		}

		// check for oracle attestation
		if len(voteExt.OracleAttestations) > 0 {
			// verify oracle attestation
			for _, attestation := range voteExt.OracleAttestations {
				operatorAddress, err := h.ValidatorOperatorAddressFromVote(ctx, vote)
				if err != nil {
					h.logger.Error("failed to get operator address from vote", "error", err)
					return nil, nil, nil, nil, err
				}
				h.logger.Info("Operator address from oracle attestation", "operatorAddress", operatorAddress)

				operatorAddresses = append(operatorAddresses, operatorAddress)

				queryId := attestation.QueryId
				queryIds = append(queryIds, queryId)

				attestations = append(attestations, hex.EncodeToString(attestation.Attestation))
				timestamps = append(timestamps, int64(attestation.Timestamp))
				h.logger.Info("Oracle attestation added to proposal", "queryId", queryId, "attestation", hex.EncodeToString(attestation.Attestation), "timestamp", attestation.Timestamp)
			}
		}
	}
	return attestations, queryIds, operatorAddresses, timestamps, nil
}
