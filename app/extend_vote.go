package app

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	signerv1 "github.com/tellor-io/bridge-remote-signer/api/gen/signer/v1"
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
	GetAggregateByTimestamp(ctx context.Context, queryId []byte, timestamp uint64) (oracletypes.Aggregate, error)
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
	GetValsetCheckpointDomainSeparator(ctx context.Context) ([]byte, error)
	GetAttestationSnapshotDataBySnapshot(ctx context.Context, snapshot []byte) (bridgetypes.AttestationSnapshotData, error)
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

	// signer may be nil at startup when the keyring path is configured
	// (or when nothing is configured at all). It is populated on
	// the first ExtendVote invocation via ensureSigner.
	signerInitOnce sync.Once
	signerInitErr  error
	signer         VoteExtensionSigner
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

// ensureSigner is called at the top of ExtendVote. If a signer was
// supplied at startup (remote signer path), it is a noop. Otherwise it
// attempts to build a KeyringSigner from viper once, caches the result,
// and returns any error to the caller. Subsequent calls return the
// cached state.
func (h *VoteExtHandler) ensureSigner() error {
	if h.signer != nil {
		return nil
	}
	h.signerInitOnce.Do(func() {
		h.logger.Info("init of bridge vote extension signer (keyring path)")
		signer, err := NewKeyringSignerFromViper(h.codec)
		if err != nil {
			h.signerInitErr = err
			return
		}
		h.signer = signer
	})
	return h.signerInitErr
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
	if err := h.ensureSigner(); err != nil {
		h.ForceProcessTermination("CRITICAL: CometBFT invoked ExtendVote but the bridge signer is unavailable: %v", err)
		return nil, err
	}

	voteExt := BridgeVoteExtension{}

	operatorAddress, errOp := h.signer.GetOperatorAddress(ctx)
	if errOp != nil {
		h.logger.Error("ExtendVoteHandler: failed to get operator address", "error", errOp)
		h.ForceProcessTermination("CRITICAL: failed to get operator address: %v", errOp)
	}
	_, err := h.bridgeKeeper.GetEVMAddressByOperator(ctx, operatorAddress)
	if err != nil {
		h.logger.Info("ExtendVoteHandler: EVM address not found for operator address, registering evm address", "operatorAddress", operatorAddress)
		initialSigA, initialSigB, err := h.SignInitialMessage(ctx, operatorAddress)
		if err != nil {
			h.logger.Info("ExtendVoteHandler: failed to sign initial message", "error", err)
			return h.marshalVoteExt(voteExt)
		}
		// include the initial sig in the vote extension
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
	} else {
		// iterate through snapshots and generate sigs
		for _, snapshot := range attestationRequests.Requests {
			sig, err := h.SignSnapshot(ctx, snapshot.Snapshot)
			if err != nil {
				h.logger.Error("ExtendVoteHandler: failed to sign snapshot", "error", err)
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

// SignSnapshot builds the structured oracle-attestation request from state
// and signs it via the configured signer.
func (h *VoteExtHandler) SignSnapshot(ctx context.Context, snapshot []byte) ([]byte, error) {
	sd, err := h.bridgeKeeper.GetAttestationSnapshotDataBySnapshot(ctx, snapshot)
	if err != nil {
		h.logger.Error("SignSnapshot: failed to get attestation snapshot data", "error", err)
		return nil, err
	}
	agg, err := h.oracleKeeper.GetAggregateByTimestamp(ctx, sd.QueryId, sd.Timestamp)
	if err != nil {
		h.logger.Error("SignSnapshot: failed to get aggregate by timestamp", "error", err)
		return nil, err
	}
	// The node stores AggregateValue as a hex string; the signer expects the
	// already-decoded raw value bytes (same as EncodeOracleAttestationData).
	valueBytes, err := hex.DecodeString(registrytypes.Remove0xPrefix(agg.AggregateValue))
	if err != nil {
		h.logger.Error("SignSnapshot: failed to hex-decode aggregate value", "error", err)
		return nil, err
	}
	req := &signerv1.SignOracleAttestationRequest{
		QueryId:                sd.QueryId,
		Value:                  valueBytes,
		Timestamp:              sd.Timestamp,
		AggregatePower:         agg.AggregatePower,
		PreviousTimestamp:      sd.PrevReportTimestamp,
		NextTimestamp:          sd.NextReportTimestamp,
		ValsetCheckpoint:       sd.ValidatorCheckpoint,
		AttestationTimestamp:   sd.AttestationTimestamp,
		LastConsensusTimestamp: sd.LastConsensusTimestamp,
		ExpectedSnapshot:       snapshot,
		RequestId:              voteExtRequestID,
	}
	sig, err := h.signer.SignOracleAttestation(ctx, req)
	if err != nil {
		h.logger.Error("SignSnapshot: failed to sign oracle attestation", "error", err)
		return nil, err
	}
	return sig, nil
}

func (h *VoteExtHandler) SignInitialMessage(ctx context.Context, operatorAddress string) ([]byte, []byte, error) {
	return h.signer.SignInitial(ctx, operatorAddress)
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
	if didSign {
		return nil, 0, nil
	} else if valIndex < 0 {
		return nil, 0, nil
	} else {
		// sign the latest checkpoint
		checkpointParams, err := h.bridgeKeeper.GetValidatorCheckpointParamsFromStorage(ctx, latestCheckpointTimestamp.Timestamp)
		if err != nil {
			h.logger.Error("failed to get checkpoint params", "error", err)
			return nil, 0, err
		}

		// Build the structured signing request from state.
		domainSeparator, err := h.bridgeKeeper.GetValsetCheckpointDomainSeparator(ctx)
		if err != nil {
			h.logger.Error("failed to get valset checkpoint domain separator", "error", err)
			return nil, 0, err
		}
		valset, err := h.bridgeKeeper.GetBridgeValsetByTimestamp(ctx, latestCheckpointTimestamp.Timestamp)
		if err != nil {
			h.logger.Error("failed to get bridge valset by timestamp", "error", err)
			return nil, 0, err
		}
		validatorSet := make([]*signerv1.BridgeValidator, 0, len(valset.BridgeValidatorSet))
		for _, val := range valset.BridgeValidatorSet {
			validatorSet = append(validatorSet, &signerv1.BridgeValidator{
				EthereumAddress: val.EthereumAddress,
				Power:           val.Power,
			})
		}
		req := &signerv1.SignBridgeCheckpointRequest{
			DomainSeparator:    domainSeparator,
			PowerThreshold:     checkpointParams.PowerThreshold,
			ValidatorTimestamp: checkpointParams.Timestamp,
			ValidatorSetHash:   checkpointParams.ValsetHash,
			ValidatorSet:       validatorSet,
			BlockHeight:        checkpointParams.BlockHeight,
			CheckpointIndex:    latestCheckpointIdx,
			ChainId:            sdk.UnwrapSDKContext(ctx).ChainID(),
			ExpectedCheckpoint: checkpointParams.Checkpoint,
			RequestId:          voteExtRequestID,
		}
		signature, err := h.signer.SignCheckpoint(ctx, req)
		if err != nil {
			h.logger.Error("failed to sign bridge checkpoint", "error", err)
			return nil, 0, err
		}
		return signature, latestCheckpointTimestamp.Timestamp, nil
	}
}

func (h *VoteExtHandler) GetValidatorIndexInValset(ctx context.Context, evmAddress []byte, valset *bridgetypes.BridgeValidatorSet) (int, error) {
	for i, val := range valset.BridgeValidatorSet {
		if bytes.Equal(val.EthereumAddress, evmAddress) {
			return i, nil
		}
	}
	return -1, fmt.Errorf("validator not found in valset")
}
