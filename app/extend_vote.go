package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type OracleKeeper interface {
	GetTimestampBefore(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error)
	GetTimestampAfter(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error)
	GetAggregatedReportsByHeight(ctx context.Context, height int64) []oracletypes.Aggregate
}

type BridgeKeeper interface {
	GetValidatorCheckpointFromStorage(ctx context.Context) (*bridgetypes.ValidatorCheckpoint, error)
	Logger(ctx context.Context) log.Logger
	GetEVMAddressByOperator(ctx context.Context, operatorAddress string) ([]byte, error)
	EVMAddressFromSignatures(ctx context.Context, sigA, sigB []byte) (common.Address, error)
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
}

type VoteExtHandler struct {
	logger       log.Logger
	oracleKeeper OracleKeeper
	bridgeKeeper BridgeKeeper
	codec        codec.Codec
	kr           keyring.Keyring
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

func NewVoteExtHandler(logger log.Logger, appCodec codec.Codec, oracleKeeper OracleKeeper, bridgeKeeper BridgeKeeper) *VoteExtHandler {
	return &VoteExtHandler{
		oracleKeeper: oracleKeeper,
		bridgeKeeper: bridgeKeeper,
		logger:       logger,
		codec:        appCodec,
	}
}

func (h *VoteExtHandler) ExtendVoteHandler(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	// check if evm address by operator exists
	voteExt := BridgeVoteExtension{}
	operatorAddress, err := h.GetOperatorAddress()
	if err != nil {
		h.logger.Error("ExtendVoteHandler: failed to get operator address", "error", err)
		bz, err := json.Marshal(voteExt)
		if err != nil {
			h.logger.Error("ExtendVoteHandler: failed to marshal vote extension", "error", err)
			return &abci.ResponseExtendVote{}, err
		}
		return &abci.ResponseExtendVote{VoteExtension: bz}, nil
	}
	_, err = h.bridgeKeeper.GetEVMAddressByOperator(ctx, operatorAddress)
	if err != nil {
		h.logger.Info("ExtendVoteHandler: EVM address not found for operator address, registering evm address", "operatorAddress", operatorAddress)
		initialSigA, initialSigB, err := h.SignInitialMessage()
		if err != nil {
			h.logger.Info("ExtendVoteHandler: failed to sign initial message", "error", err)
			bz, err := json.Marshal(voteExt)
			if err != nil {
				h.logger.Error("ExtendVoteHandler: failed to marshal vote extension", "error", err)
				return &abci.ResponseExtendVote{}, err
			}
			return &abci.ResponseExtendVote{VoteExtension: bz}, nil
		}
		// include the initial sig in the vote extension
		initialSignature := InitialSignature{
			SignatureA: initialSigA,
			SignatureB: initialSigB,
		}
		voteExt.InitialSignature = initialSignature
	}
	// generate oracle attestations and include them via vote extensions
	blockHeight := ctx.BlockHeight() - 1
	attestationRequests, err := h.bridgeKeeper.GetAttestationRequestsByHeight(ctx, uint64(blockHeight))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			h.logger.Error("ExtendVoteHandler: failed to get attestation requests", "error", err)
			bz, err := json.Marshal(voteExt)
			if err != nil {
				h.logger.Error("ExtendVoteHandler: failed to marshal vote extension", "error", err)
				return &abci.ResponseExtendVote{}, err
			}
			return &abci.ResponseExtendVote{VoteExtension: bz}, nil
		}
	} else {
		snapshots := attestationRequests.Requests
		// iterate through snapshots and generate sigs
		if len(snapshots) > 0 {
			for _, snapshot := range snapshots {
				sig, err := h.SignMessage(snapshot.Snapshot)
				if err != nil {
					h.logger.Error("ExtendVoteHandler: failed to sign message", "error", err)
					bz, err := json.Marshal(voteExt)
					if err != nil {
						h.logger.Error("ExtendVoteHandler: failed to marshal vote extension", "error", err)
						return &abci.ResponseExtendVote{}, err
					}
					return &abci.ResponseExtendVote{VoteExtension: bz}, nil
				}
				oracleAttestation := OracleAttestation{
					Snapshot:    snapshot.Snapshot,
					Attestation: sig,
				}
				voteExt.OracleAttestations = append(voteExt.OracleAttestations, oracleAttestation)
			}
		}
	}
	// include the valset sig in the vote extension
	sig, timestamp, err := h.CheckAndSignValidatorCheckpoint(ctx)
	if err != nil {
		h.logger.Error("ExtendVoteHandler: failed to sign validator checkpoint", "error", err)
		bz, err := json.Marshal(voteExt)
		if err != nil {
			h.logger.Error("ExtendVoteHandler: failed to marshal vote extension", "error", err)
			return &abci.ResponseExtendVote{}, fmt.Errorf("failed to marshal vote extension: %w", err)
		}
		return &abci.ResponseExtendVote{VoteExtension: bz}, nil
	}
	valsetSignature := BridgeValsetSignature{
		Signature: sig,
		Timestamp: timestamp,
	}
	voteExt.ValsetSignature = valsetSignature
	bz, err := json.Marshal(voteExt)
	if err != nil {
		h.logger.Error("ExtendVoteHandler: failed to marshal vote extension", "error", err)
		return &abci.ResponseExtendVote{}, fmt.Errorf("failed to marshal vote extension: %w", err)
	}
	return &abci.ResponseExtendVote{VoteExtension: bz}, nil
}

func (h *VoteExtHandler) VerifyVoteExtensionHandler(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	var voteExt BridgeVoteExtension
	err := json.Unmarshal(req.VoteExtension, &voteExt)
	if err != nil {
		h.logger.Error("VerifyVoteExtensionHandler: failed to unmarshal vote extension", "error", err)
		// lookup whether validator has registered evm address
		validatorAddress, err := sdk.Bech32ifyAddressBytes(sdk.GetConfig().GetBech32ValidatorAddrPrefix(), req.ValidatorAddress)
		if err != nil {
			h.logger.Error("VerifyVoteExtensionHandler: failed to convert validator address to Bech32", "error", err)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_REJECT}, nil
		}
		_, err = h.bridgeKeeper.GetEVMAddressByOperator(ctx, validatorAddress)
		if err != nil {
			h.logger.Info("VerifyVoteExtensionHandler: validator does not have evm address, accepting vote", "validatorAddress", validatorAddress)
			return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
		}
		h.logger.Info("VerifyVoteExtensionHandler: validator has evm address, rejecting vote", "validatorAddress", validatorAddress)
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

	h.logger.Info("VerifyVoteExtensionHandler: vote extension verified", "voteExt", voteExt)
	return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
}

func (h *VoteExtHandler) SignMessage(msg []byte) ([]byte, error) {
	kr, err := h.GetKeyring()
	if err != nil {
		return nil, fmt.Errorf("failed to get keyring: %w", err)
	}
	keyName := viper.GetString("key-name")
	if keyName == "" {
		return nil, fmt.Errorf("key name not found, please set --key-name flag")
	}
	sig, _, err := kr.Sign(keyName, msg, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}
	return sig, nil
}

func (h *VoteExtHandler) SignInitialMessage() ([]byte, []byte, error) {
	messageA := "TellorLayer: Initial bridge signature A"
	messageB := "TellorLayer: Initial bridge signature B"

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
	sigA, err := h.SignMessage(msgHashABytes)
	if err != nil {
		return nil, nil, err
	}

	sigB, err := h.SignMessage(msgHashBBytes)
	if err != nil {
		return nil, nil, err
	}
	return sigA, sigB, nil
}

func (h *VoteExtHandler) GetOperatorAddress() (string, error) {
	kr, err := h.GetKeyring()
	if err != nil {
		return "", fmt.Errorf("failed to get keyring: %w", err)
	}
	keyName := viper.GetString("key-name")
	if keyName == "" {
		return "", fmt.Errorf("key name not found, please set --key-name flag")
	}
	// list all keys
	krlist, err := kr.List()
	if err != nil {
		return "", fmt.Errorf("failed to list keys: %w", err)
	}
	if len(krlist) == 0 {
		return "", fmt.Errorf("no keys found in keyring")
	}

	// Fetch the operator key from the keyring.
	info, err := kr.Key(keyName)
	if err != nil {
		return "", fmt.Errorf("failed to get operator key: %w", err)
	}
	// Output the public key associated with the operator key.
	key, _ := info.GetPubKey()

	// Convert the operator's public key to a Bech32 validator address
	config := sdk.GetConfig()
	bech32PrefixValAddr := config.GetBech32ValidatorAddrPrefix()
	bech32ValAddr, err := sdk.Bech32ifyAddressBytes(bech32PrefixValAddr, key.Address().Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to convert operator public key to Bech32 validator address: %w", err)
	}
	return bech32ValAddr, nil
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

	operatorAddress, err := h.GetOperatorAddress()
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
		checkpoint := checkpointParams.Checkpoint
		checkpointString := hex.EncodeToString(checkpoint)
		signature, err := h.EncodeAndSignMessage(checkpointString)
		if err != nil {
			h.logger.Error("failed to encode and sign message", "error", err)
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

func (h *VoteExtHandler) EncodeAndSignMessage(checkpointString string) ([]byte, error) {
	// Encode the checkpoint string to bytes
	checkpoint, err := hex.DecodeString(checkpointString)
	if err != nil {
		h.logger.Error("Failed to decode checkpoint", "error", err)
		return nil, err
	}
	signature, err := h.SignMessage(checkpoint)
	if err != nil {
		h.logger.Error("Failed to sign message", "error", err)
		return nil, err
	}
	return signature, nil
}

func (h *VoteExtHandler) InitKeyring() (keyring.Keyring, error) {
	krBackend := viper.GetString("keyring-backend")
	if krBackend == "" {
		return nil, fmt.Errorf("keyring-backend not set, please use --keyring-backend flag")
	}
	krDir := viper.GetString("keyring-dir")
	if krDir == "" {
		krDir = viper.GetString("home")
	}
	if krDir == "" {
		return nil, fmt.Errorf("keyring directory not set, please use --home or --keyring-dir flag")
	}
	kr, err := keyring.New(sdk.KeyringServiceName(), krBackend, krDir, os.Stdin, h.codec)
	if err != nil {
		return nil, err
	}
	return kr, nil
}

func (h *VoteExtHandler) GetKeyring() (keyring.Keyring, error) {
	if h.kr == nil {
		kr, err := h.InitKeyring()
		if err != nil {
			return nil, err
		}
		h.kr = kr
	}
	return h.kr, nil
}
