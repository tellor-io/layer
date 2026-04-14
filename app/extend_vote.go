package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type OracleKeeper interface {
	GetTimestampBefore(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error)
	GetTimestampAfter(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error)
	GetAggregatedReportsByHeight(ctx context.Context, height uint64) ([]oracletypes.Aggregate, error)
}

type BridgeKeeper interface {
	GetValidatorCheckpointFromStorage(ctx context.Context) (*bridgetypes.ValidatorCheckpoint, error)
	Logger(ctx context.Context) log.Logger
	GetEVMAddressByOperator(ctx context.Context, operatorAddress string) ([]byte, error)
	EVMAddressFromSignatures(ctx context.Context, sigA, sigB []byte, operatorAddress string) (common.Address, error)
	SetEVMAddressByOperator(ctx context.Context, operatorAddr string, evmAddr []byte) error
	GetValidatorSetSignaturesFromStorage(ctx context.Context, timestamp uint64) (*bridgetypes.BridgeValsetSignatures, error)
	SetBridgeValsetSignature(ctx context.Context, operatorAddress string, timestamp uint64, signature string) error
	GetLatestCheckpointIndex(ctx context.Context) (uint64, error)
	GetBridgeValsetByTimestamp(ctx context.Context, timestamp uint64) (*bridgetypes.BridgeValidatorSet, error)
	GetValidatorTimestampByIdxFromStorage(ctx context.Context, checkpointIdx uint64) (bridgetypes.CheckpointTimestamp, error)
	GetValidatorCheckpointParamsFromStorage(ctx context.Context, timestamp uint64) (bridgetypes.ValidatorCheckpointParams, error)
	GetValidatorDidSignCheckpoint(ctx context.Context, operatorAddr string, checkpointTimestamp uint64) (didSign bool, prevValsetIndex int64, err error)
	GetAttestationRequestsByHeight(ctx context.Context, height uint64) (*bridgetypes.AttestationRequests, error)
	SetOracleAttestation(ctx context.Context, operatorAddress string, snapshot, sig []byte) error
}

type StakingKeeper interface {
	GetValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (validator stakingtypes.Validator, err error)
	GetParams(ctx context.Context) (stakingtypes.Params, error)
	Jail(ctx context.Context, consAddr sdk.ConsAddress) error
}

type VoteExtHandler struct {
	logger       log.Logger
	oracleKeeper OracleKeeper
	bridgeKeeper BridgeKeeper
	codec        codec.Codec
	signer       VoteExtensionSigner
}

type OracleAttestation struct {
	Snapshot    []byte
	Attestation []byte
}

type InitialSignature struct {
	SignatureA []byte
	SignatureB []byte
}

type BridgeValsetSignature struct {
	Signature []byte
	Timestamp uint64
}

type BridgeVoteExtension struct {
	OracleAttestations []OracleAttestation
	InitialSignature   InitialSignature
	ValsetSignature    BridgeValsetSignature
}

func NewVoteExtHandler(logger log.Logger, appCodec codec.Codec, oracleKeeper OracleKeeper, bridgeKeeper BridgeKeeper, signer VoteExtensionSigner) *VoteExtHandler {
	return &VoteExtHandler{
		oracleKeeper: oracleKeeper,
		bridgeKeeper: bridgeKeeper,
		logger:       logger,
		codec:        appCodec,
		signer:       signer,
	}
}

func (h *VoteExtHandler) ForceProcessTermination(format string, args ...interface{}) {
	h.logger.Error(format, args...)
	// Send SIGABRT to the current process
	process, _ := os.FindProcess(os.Getpid())
	err := process.Signal(syscall.SIGABRT)
	if err != nil {
		h.logger.Error("failed to send SIGABRT to process", "error", err)
	}
	// In case SIGABRT doesn't work, fall back to Exit
	os.Exit(1)
}

func (h *VoteExtHandler) ExtendVoteHandler(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	voteExt := BridgeVoteExtension{}

	operatorAddress, errOp := h.signer.GetOperatorAddress(ctx)
	if errOp != nil {
		h.logger.Error("ExtendVoteHandler: failed to get operator address", "error", errOp)
		h.ForceProcessTermination("CRITICAL: failed to get operator address: %v", errOp)
	}
	_, err := h.bridgeKeeper.GetEVMAddressByOperator(ctx, operatorAddress)
	if err != nil {
		h.logger.Info("ExtendVoteHandler: EVM address not found for operator address, registering evm address", "operatorAddress", operatorAddress)
		initialSigA, initialSigB, err := h.SignInitialMessage(operatorAddress)
		if err != nil {
			h.logger.Info("ExtendVoteHandler: failed to sign initial message", "error", err)
			return h.marshalVoteExt(voteExt)
		}
		voteExt.InitialSignature = InitialSignature{
			SignatureA: initialSigA,
			SignatureB: initialSigB,
		}
	}
	// generate oracle attestations and include them via vote extensions
	blockHeight := ctx.BlockHeight() - 1
	attestationRequests, err := h.bridgeKeeper.GetAttestationRequestsByHeight(ctx, uint64(blockHeight))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			h.logger.Error("ExtendVoteHandler: failed to get attestation requests", "error", err)
			return h.marshalVoteExt(voteExt)
		}
	} else if len(attestationRequests.Requests) > 0 {
		for _, snapshot := range attestationRequests.Requests {
			sig, err := h.signer.Sign(ctx, snapshot.Snapshot)
			if err != nil {
				h.logger.Error("ExtendVoteHandler: failed to sign attestation", "error", err)
				return h.marshalVoteExt(voteExt)
			}
			voteExt.OracleAttestations = append(voteExt.OracleAttestations, OracleAttestation{
				Snapshot:    snapshot.Snapshot,
				Attestation: sig,
			})
		}
	}
	// include the valset sig in the vote extension
	sig, timestamp, err := h.CheckAndSignValidatorCheckpoint(ctx)
	if err != nil {
		h.logger.Error("ExtendVoteHandler: failed to sign validator checkpoint", "error", err)
		return h.marshalVoteExt(voteExt)
	}
	voteExt.ValsetSignature = BridgeValsetSignature{
		Signature: sig,
		Timestamp: timestamp,
	}

	return h.marshalVoteExt(voteExt)
}

// marshalVoteExt marshals the vote extension, returning an error only if marshaling itself fails.
func (h *VoteExtHandler) marshalVoteExt(voteExt BridgeVoteExtension) (*abci.ResponseExtendVote, error) {
	bz, err := json.Marshal(voteExt)
	if err != nil {
		h.logger.Error("ExtendVoteHandler: failed to marshal vote extension", "error", err)
		return &abci.ResponseExtendVote{}, fmt.Errorf("failed to marshal vote extension: %w", err)
	}
	return &abci.ResponseExtendVote{VoteExtension: bz}, nil
}

const maxVoteExtensionSize = 512 * 1024 // 512KB upper bound; legitimate VEs are ~171KB max with snapshotLimit=1000

func (h *VoteExtHandler) VerifyVoteExtensionHandler(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	if len(req.VoteExtension) > maxVoteExtensionSize {
		h.logger.Error("VerifyVoteExtensionHandler: vote extension exceeds max size", "size", len(req.VoteExtension), "max", maxVoteExtensionSize)
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
	}

	var voteExt BridgeVoteExtension
	decoder := json.NewDecoder(bytes.NewReader(req.VoteExtension))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&voteExt); err != nil {
		validatorAddress := sdk.ConsAddress(req.ValidatorAddress)
		h.logger.Error("VerifyVoteExtensionHandler: failed to unmarshal vote extension", "error", err, "validator", validatorAddress)
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
	}
	// Enforce a single top-level JSON value to reject trailing data.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		validatorAddress := sdk.ConsAddress(req.ValidatorAddress)
		h.logger.Error("VerifyVoteExtensionHandler: vote extension contains trailing data", "error", err, "validator", validatorAddress)
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
	}

	// ensure oracle attestations length is less than or equal to the number of attestation requests
	attestationRequests, err := h.bridgeKeeper.GetAttestationRequestsByHeight(ctx, uint64(ctx.BlockHeight()-1))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			h.logger.Error("VerifyVoteExtensionHandler: failed to get attestation requests", "error", err)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		} else if len(voteExt.OracleAttestations) > 0 {
			h.logger.Error("VerifyVoteExtensionHandler: oracle attestations length is greater than 0, should be 0", "voteExt", voteExt)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}
	} else if len(voteExt.OracleAttestations) > len(attestationRequests.Requests) {
		h.logger.Error("VerifyVoteExtensionHandler: oracle attestations length is greater than attestation requests length", "voteExt", voteExt)
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
	}

	// verify per-attestation field sizes (snapshots are Keccak256 hashes, attestations are ECDSA sigs)
	for _, att := range voteExt.OracleAttestations {
		if len(att.Snapshot) > 32 {
			h.logger.Error("VerifyVoteExtensionHandler: attestation snapshot size exceeds 32 bytes", "size", len(att.Snapshot))
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}
		if len(att.Attestation) > 65 {
			h.logger.Error("VerifyVoteExtensionHandler: attestation signature size exceeds 65 bytes", "size", len(att.Attestation))
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}
	}

	// verify the initial signature size
	if len(voteExt.InitialSignature.SignatureA) > 65 || len(voteExt.InitialSignature.SignatureB) > 65 {
		h.logger.Error("VerifyVoteExtensionHandler: initial signature size is greater than 65", "voteExt", voteExt)
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
	}
	// verify the valset signature size
	if len(voteExt.ValsetSignature.Signature) > 65 {
		h.logger.Error("VerifyVoteExtensionHandler: valset signature size is greater than 65", "voteExt", voteExt)
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
	}

	return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
}

func (h *VoteExtHandler) SignInitialMessage(operatorAddress string) ([]byte, []byte, error) {
	messageA := fmt.Sprintf("TellorLayer: Initial bridge signature A for operator %s", operatorAddress)
	messageB := fmt.Sprintf("TellorLayer: Initial bridge signature B for operator %s", operatorAddress)

	// convert message to bytes
	msgBytesA := []byte(messageA)
	msgBytesB := []byte(messageB)

	// hash message
	msgHashABytes32 := sha256.Sum256(msgBytesA)
	msgHashBBytes32 := sha256.Sum256(msgBytesB)

	// convert [32]byte to []byte
	msgHashABytes := msgHashABytes32[:]
	msgHashBBytes := msgHashBBytes32[:]

	// sign message
	sigA, err := h.signer.Sign(context.Background(), msgHashABytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign message A: %w", err)
	}

	sigB, err := h.signer.Sign(context.Background(), msgHashBBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign message B: %w", err)
	}
	return sigA, sigB, nil
}

func (h *VoteExtHandler) CheckAndSignValidatorCheckpoint(ctx context.Context) (signature []byte, timestamp uint64, err error) {
	// get latest checkpoint index
	latestCheckpointIdx, err := h.bridgeKeeper.GetLatestCheckpointIndex(ctx)
	if err != nil {
		h.logger.Error("failed to get latest checkpoint index", "error", err)
		return nil, 0, err
	}
	// get the latest checkpoint timestamp
	latestCheckpointTimestamp, err := h.bridgeKeeper.GetValidatorTimestampByIdxFromStorage(ctx, latestCheckpointIdx)
	if err != nil {
		h.logger.Error("failed to get latest checkpoint timestamp", "error", err)
		return nil, 0, err
	}

	operatorAddress, err := h.signer.GetOperatorAddress(ctx)
	if err != nil {
		h.logger.Error("failed to get operator address", "error", err)
		return nil, 0, err
	}
	didSign, valIndex, err := h.bridgeKeeper.GetValidatorDidSignCheckpoint(ctx, operatorAddress, latestCheckpointTimestamp.Timestamp)
	if err != nil {
		h.logger.Error("failed to get validator did sign checkpoint", "error", err)
		return nil, 0, err
	}
	if didSign || valIndex < 0 {
		return nil, 0, nil
	}

	// sign the latest checkpoint
	checkpointParams, err := h.bridgeKeeper.GetValidatorCheckpointParamsFromStorage(ctx, latestCheckpointTimestamp.Timestamp)
	if err != nil {
		h.logger.Error("failed to get checkpoint params", "error", err)
		return nil, 0, err
	}
	checkpoint := checkpointParams.Checkpoint
	checkpointString := hex.EncodeToString(checkpoint)
	signature, err = h.EncodeAndSignMessage(checkpointString)
	if err != nil {
		h.logger.Error("failed to encode and sign message", "error", err)
		return nil, 0, err
	}
	return signature, latestCheckpointTimestamp.Timestamp, nil
}

func (h *VoteExtHandler) GetValidatorIndexInValset(ctx context.Context, evmAddress []byte, valset *bridgetypes.BridgeValidatorSet) (int, error) {
	for i, val := range valset.BridgeValidatorSet {
		if bytes.Equal(val.EthereumAddress, evmAddress) {
			return i, nil
		}
	}
	return -1, fmt.Errorf("validator not found in valset")
}

func (h *VoteExtHandler) EncodeAndSignMessage(checkpointString string) ([]byte, error) {
	// Encode the checkpoint string to bytes
	checkpoint, err := hex.DecodeString(registrytypes.Remove0xPrefix(checkpointString))
	if err != nil {
		h.logger.Error("Failed to decode checkpoint", "error", err)
		return nil, err
	}
	signature, err := h.signer.Sign(context.Background(), checkpoint)
	if err != nil {
		h.logger.Error("Failed to sign message", "error", err)
		return nil, err
	}
	return signature, nil
}
