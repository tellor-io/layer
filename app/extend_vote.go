package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/viper"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

type OracleKeeper interface {
	// GetQueryIdAndTimestampPairsByBlockHeight(ctx sdk.Context, height uint64) oracletypes.QueryIdTimestampPairsArray
	// GetAggregateReport(ctx sdk.Context, queryId []byte, timestamp time.Time) (*oracletypes.Aggregate, error)
	GetTimestampBefore(ctx sdk.Context, queryId []byte, timestamp time.Time) (time.Time, error)
	GetTimestampAfter(ctx sdk.Context, queryId []byte, timestamp time.Time) (time.Time, error)
	GetAggregatedReportsByHeight(ctx sdk.Context, height int64) []oracletypes.Aggregate
	GetDataBeforePublic(ctx sdk.Context, queryId []byte, timestamp time.Time) (*oracletypes.Aggregate, error)
}

type BridgeKeeper interface {
	GetValidatorCheckpointFromStorage(ctx sdk.Context) (*bridgetypes.ValidatorCheckpoint, error)
	Logger(ctx context.Context) log.Logger
	GetEVMAddressByOperator(ctx sdk.Context, operatorAddress string) (string, error)
	EVMAddressFromSignature(ctx sdk.Context, sigHexString string) (string, error)
	EVMAddressFromSignatures(ctx sdk.Context, sigA []byte, sigB []byte) (common.Address, error)
	SetEVMAddressByOperator(ctx sdk.Context, operatorAddr string, evmAddr string) error
	GetValidatorSetSignaturesFromStorage(ctx sdk.Context, timestamp uint64) (*bridgetypes.BridgeValsetSignatures, error)
	SetBridgeValsetSignature(ctx sdk.Context, operatorAddress string, timestamp uint64, signature string) error
	GetLatestCheckpointIndex(ctx sdk.Context) (uint64, error)
	GetBridgeValsetByTimestamp(ctx sdk.Context, timestamp uint64) (*bridgetypes.BridgeValidatorSet, error)
	GetValidatorTimestampByIdxFromStorage(ctx sdk.Context, checkpointIdx uint64) (*bridgetypes.CheckpointTimestamp, error)
	GetValidatorCheckpointParamsFromStorage(ctx sdk.Context, timestamp uint64) (*bridgetypes.ValidatorCheckpointParams, error)
	SetOracleAttestation(ctx sdk.Context, operatorAddress string, queryId string, timestamp uint64, signature string) error
	GetValidatorDidSignCheckpoint(ctx sdk.Context, operatorAddr string, checkpointTimestamp uint64) (didSign bool, prevValsetIndex int64, err error)
}

type StakingKeeper interface {
	GetValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (validator stakingtypes.Validator, err error)
}

type VoteExtHandler struct {
	logger       log.Logger
	oracleKeeper OracleKeeper
	bridgeKeeper BridgeKeeper
	codec        codec.Codec
	// cosmosCtx    sdk.Context
}

type OracleAttestation struct {
	QueryId     string
	Timestamp   uint64
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
		return nil, err
	}
	_, err = h.bridgeKeeper.GetEVMAddressByOperator(ctx, operatorAddress)
	if err != nil {
		h.logger.Info("EVM address not found for operator address", "operatorAddress", operatorAddress)
		h.logger.Info("Error message", "error", err)
		initialSigA, initialSigB, err := h.SignInitialMessage()
		if err != nil {
			h.logger.Info("Failed to sign initial message", "error", err)
			return nil, err
		}
		// include the initial sig in the vote extension
		initialSignature := InitialSignature{
			SignatureA: initialSigA,
			SignatureB: initialSigB,
		}
		voteExt.InitialSignature = initialSignature
	}
	// logic for generating oracle sigs and including them via vote extensions
	blockHeight := ctx.BlockHeight() - 1
	reports := h.oracleKeeper.GetAggregatedReportsByHeight(ctx, int64(blockHeight))
	// iterate through reports and generate sigs
	if len(reports) > 0 {
		valsetCheckpoint, err := h.bridgeKeeper.GetValidatorCheckpointFromStorage(ctx)
		if err != nil {
			return nil, err
		}
		for _, aggReport := range reports {
			currentTime := time.Now()
			ts := currentTime.Unix() + 100
			queryId, err := hex.DecodeString(aggReport.QueryId)
			if err != nil {
				panic(err)
			}
			reportTime, err := h.oracleKeeper.GetTimestampBefore(ctx, queryId, time.Unix(ts, 0))
			if err != nil {
				return nil, err
			}
			tsBefore, err := h.oracleKeeper.GetTimestampBefore(ctx, queryId, reportTime)
			if err != nil {
				// set to 0
				tsBefore = time.Unix(0, 0)
			}
			tsAfter, err := h.oracleKeeper.GetTimestampAfter(ctx, queryId, reportTime)
			if err != nil {
				// set to 0
				tsAfter = time.Unix(0, 0)
			}
			oracleAttestationHash, err := h.EncodeOracleAttestationData(
				aggReport.QueryId,
				aggReport.AggregateValue,
				reportTime.Unix(),
				aggReport.ReporterPower,
				tsBefore.Unix(),
				tsAfter.Unix(),
				hex.EncodeToString(valsetCheckpoint.Checkpoint),
				reportTime.Unix(),
			)
			if err != nil {
				return nil, err
			}
			// sign the oracleAttestationHash
			sig, err := h.SignMessage(oracleAttestationHash)
			if err != nil {
				return nil, err
			}
			oracleAttestation := OracleAttestation{
				Attestation: sig,
				QueryId:     aggReport.QueryId,
				Timestamp:   uint64(reportTime.Unix()),
			}
			voteExt.OracleAttestations = append(voteExt.OracleAttestations, oracleAttestation)
		}
	}
	// include the valset sig in the vote extension
	sig, timestamp, err := h.CheckAndSignValidatorCheckpoint(ctx)
	if err != nil {
		h.logger.Error("Failed to sign validator checkpoint", "error", err)
		bz, err := json.Marshal(voteExt)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal vote extension: %w", err)
		}
		return &abci.ResponseExtendVote{VoteExtension: bz}, nil
	}
	valsetSignature := BridgeValsetSignature{
		Signature: sig,
		Timestamp: timestamp,
	}
	voteExt.ValsetSignature = valsetSignature
	h.logger.Info("Vote extension data", "voteExt", voteExt)

	bz, err := json.Marshal(voteExt)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vote extension: %w", err)
	}
	return &abci.ResponseExtendVote{VoteExtension: bz}, nil
}

func (h *VoteExtHandler) VerifyVoteExtensionHandler(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	// TODO: implement the logic to verify the vote extension
	return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
}

func (h *VoteExtHandler) EncodeOracleAttestationData(
	queryId string,
	value string,
	timestamp int64,
	aggregatePower int64,
	previousTimestamp int64,
	nextTimestamp int64,
	valsetCheckpoint string,
	attestationTimestamp int64,
) ([]byte, error) {
	// domainSeparator is bytes "tellorNewReport"
	domainSep := "74656c6c6f7243757272656e744174746573746174696f6e0000000000000000"
	NEW_REPORT_ATTESTATION_DOMAIN_SEPERATOR, err := hex.DecodeString(domainSep)
	if err != nil {
		return nil, err
	}
	// Convert domain separator to bytes32
	var domainSepBytes32 [32]byte
	copy(domainSepBytes32[:], NEW_REPORT_ATTESTATION_DOMAIN_SEPERATOR)

	// Convert queryId to bytes32
	queryIdBytes, err := hex.DecodeString(queryId)
	if err != nil {
		return nil, err
	}
	var queryIdBytes32 [32]byte
	copy(queryIdBytes32[:], queryIdBytes)

	// Convert value to bytes
	valueBytes, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}

	// Convert timestamp to uint64
	timestampUint64 := new(big.Int)
	timestampUint64.SetInt64(timestamp)

	// Convert aggregatePower to uint64
	aggregatePowerUint64 := new(big.Int)
	aggregatePowerUint64.SetInt64(aggregatePower)

	// Convert previousTimestamp to uint64
	previousTimestampUint64 := new(big.Int)
	previousTimestampUint64.SetInt64(previousTimestamp)

	// Convert nextTimestamp to uint64
	nextTimestampUint64 := new(big.Int)
	nextTimestampUint64.SetInt64(nextTimestamp)

	// Convert valsetCheckpoint to bytes32
	valsetCheckpointBytes, err := hex.DecodeString(valsetCheckpoint)
	if err != nil {
		return nil, err
	}
	var valsetCheckpointBytes32 [32]byte
	copy(valsetCheckpointBytes32[:], valsetCheckpointBytes)

	// Convert attestationTimestamp to uint64
	attestationTimestampUint64 := new(big.Int)
	attestationTimestampUint64.SetInt64(attestationTimestamp)

	// Prepare Encoding
	Bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	arguments := abi.Arguments{
		{Type: Bytes32Type},
		{Type: Bytes32Type},
		{Type: BytesType},
		{Type: Uint256Type},
		{Type: Uint256Type},
		{Type: Uint256Type},
		{Type: Uint256Type},
		{Type: Bytes32Type},
		{Type: Uint256Type},
	}

	// Encode the data
	encodedData, err := arguments.Pack(
		domainSepBytes32,
		queryIdBytes32,
		valueBytes,
		timestampUint64,
		aggregatePowerUint64,
		previousTimestampUint64,
		nextTimestampUint64,
		valsetCheckpointBytes32,
		attestationTimestampUint64,
	)
	if err != nil {
		return nil, err
	}

	oracleAttestationHash := crypto.Keccak256(encodedData)
	return oracleAttestationHash, nil
}

func (h *VoteExtHandler) SignMessage(msg []byte) ([]byte, error) {
	// define keyring backend and the path to the keystore dir
	krBackend := keyring.BackendTest
	keyName := h.GetKeyName()
	if keyName == "" {
		return nil, fmt.Errorf("key name not found")
	}
	krDir := os.ExpandEnv("$HOME/.layer/" + keyName)

	kr, err := keyring.New("layer", krBackend, krDir, os.Stdin, h.codec)
	if err != nil {
		fmt.Printf("Failed to create keyring: %v\n", err)
		return nil, err
	}

	krlist, err := kr.List()
	if err != nil {
		fmt.Printf("Failed to list keys: %v\n", err)
		return nil, err
	}

	for _, k := range krlist {
		fmt.Println("name: ", k.Name)
	}

	// sign message
	// tempmsg := []byte("hello")
	sig, _, err := kr.Sign(keyName, msg, 1)
	if err != nil {
		fmt.Printf("Failed to sign message: %v\n", err)
		return nil, err
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
	// sigA = append(sigA, 0)

	sigB, err := h.SignMessage(msgHashBBytes)
	if err != nil {
		return nil, nil, err
	}
	return sigA, sigB, nil
}

func (h *VoteExtHandler) GetOperatorAddress() (string, error) {
	// define keyring backend and the path to the keystore dir
	keyName := h.GetKeyName()
	if keyName == "" {
		return "", fmt.Errorf("key name not found")
	}
	krBackend := keyring.BackendTest
	krDir := os.ExpandEnv("$HOME/.layer/" + keyName)

	userInput := os.Stdin

	kr, err := keyring.New("layer", krBackend, krDir, userInput, h.codec)
	if err != nil {
		fmt.Printf("Failed to create keyring: %v\n", err)
		return "", err
	}

	// list all keys
	krlist, err := kr.List()
	if err != nil {
		fmt.Printf("Failed to list keys: %v\n", err)
		return "", err
	}
	if len(krlist) == 0 {
		h.logger.Info("No keys found in keyring")
	}

	// Fetch the operator key from the keyring.
	info, err := kr.Key(keyName)
	if err != nil {
		fmt.Printf("Failed to get operator key: %v\n", err)
		return "", err
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

func (h *VoteExtHandler) GetKeyName() string {
	globalHome := os.ExpandEnv("$HOME/.layer")
	homeDir := viper.GetString("home")
	// if home is global/alice, then the key name is alice
	if homeDir == globalHome+"/alice" {
		return "alice"
	} else if homeDir == globalHome+"/bill" {
		return "bill"
	} else {
		return ""
	}
}

func (h *VoteExtHandler) CheckAndSignValidatorCheckpoint(ctx sdk.Context) (signature []byte, timestamp uint64, err error) {
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

func (h *VoteExtHandler) GetValidatorIndexInValset(ctx sdk.Context, evmAddress string, valset *bridgetypes.BridgeValidatorSet) (int, error) {
	for i, val := range valset.BridgeValidatorSet {
		if val.EthereumAddress == evmAddress {
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
