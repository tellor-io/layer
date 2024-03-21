package app

import (
	"bytes"
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
	cosbytes "github.com/cometbft/cometbft/libs/bytes"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
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
	SetEVMAddressByOperator(ctx sdk.Context, operatorAddr string, evmAddr string) error
	GetValidatorSetSignaturesFromStorage(ctx sdk.Context, timestamp uint64) (*bridgetypes.BridgeValsetSignatures, error)
	SetBridgeValsetSignature(ctx sdk.Context, operatorAddress string, timestamp uint64, signature string) error
	GetLatestCheckpointIndex(ctx sdk.Context) (uint64, error)
	GetBridgeValsetByTimestamp(ctx sdk.Context, timestamp uint64) (*bridgetypes.BridgeValidatorSet, error)
	GetValidatorTimestampByIdxFromStorage(ctx sdk.Context, checkpointIdx uint64) (*bridgetypes.CheckpointTimestamp, error)
	GetValidatorCheckpointParamsFromStorage(ctx sdk.Context, timestamp uint64) (*bridgetypes.ValidatorCheckpointParams, error)
	SetOracleAttestation(ctx sdk.Context, operatorAddress string, queryId string, timestamp uint64, signature string) error
	GetValidatorDidSignCheckpoint(ctx sdk.Context, operatorAddr string, checkpointTimestamp uint64) (didSign bool, prevValsetIndex int64, err error)
	RecoverETHAddress(ctx sdk.Context, msg []byte, sig []byte, signer []byte) ([]byte, uint8, error)
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
	Signature []byte
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

// type Aggregate struct {
//     QueryId              string               `protobuf:"bytes,1,opt,name=queryId,proto3" json:"queryId,omitempty"`
//     AggregateValue       string               `protobuf:"bytes,2,opt,name=aggregateValue,proto3" json:"aggregateValue,omitempty"`
//     AggregateReporter    string               `protobuf:"bytes,3,opt,name=aggregateReporter,proto3" json:"aggregateReporter,omitempty"`
//     ReporterPower        int64                `protobuf:"varint,4,opt,name=reporterPower,proto3" json:"reporterPower,omitempty"`
//     StandardDeviation    float64              `protobuf:"fixed64,5,opt,name=standardDeviation,proto3" json:"standardDeviation,omitempty"`
//     Reporters            []*AggregateReporter `protobuf:"bytes,6,rep,name=reporters,proto3" json:"reporters,omitempty"`
//     Flagged              bool                 `protobuf:"varint,7,opt,name=flagged,proto3" json:"flagged,omitempty"`
//     Nonce                int64                `protobuf:"varint,8,opt,name=nonce,proto3" json:"nonce,omitempty"`
//     AggregateReportIndex int64                `protobuf:"varint,9,opt,name=aggregateReportIndex,proto3" json:"aggregateReportIndex,omitempty"`
// }

func (h *VoteExtHandler) ExtendVoteHandler(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	h.logger.Info("@BridgeExtendVoteHandler called", "req", req)
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
		initialSig, err := h.SignInitialMessage()
		if err != nil {
			h.logger.Info("Failed to sign initial message", "error", err)
			return nil, err
		}
		// include the initial sig in the vote extension
		initialSignature := InitialSignature{
			Signature: initialSig,
		}
		// voteExt := BridgeVoteExtension{
		// 	InitialSignature: initialSignature,
		// }
		voteExt.InitialSignature = initialSignature
		bz, err := json.Marshal(voteExt)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal vote extension: %w", err)
		}
		h.logger.Info("Vote extension data", "voteExt", string(bz))
		// return &abci.ResponseExtendVote{VoteExtension: bz}, nil
	}
	// logic for generating oracle sigs and including them via vote extensions
	blockHeight := ctx.BlockHeight() - 1
	reports := h.oracleKeeper.GetAggregatedReportsByHeight(ctx, int64(blockHeight))
	// voteExt := BridgeVoteExtension{}
	// iterate through reports and generate sigs
	if len(reports) == 0 {
		h.logger.Info("No reports found for block height", "blockHeight", blockHeight)
		// voteExt := BridgeVoteExtension{}
		// bz, err := json.Marshal(voteExt)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to marshal empty vote extension: %w", err)
		// }
		// return &abci.ResponseExtendVote{VoteExtension: bz}, nil
	} else {
		h.logger.Info("Reports were found for block height", "blockHeight", blockHeight)
		valsetCheckpoint, err := h.bridgeKeeper.GetValidatorCheckpointFromStorage(ctx)
		if err != nil {
			return nil, err
		}
		for _, aggReport := range reports {
			currentTime := time.Now()
			ts := currentTime.Unix() + 100
			// get dataBefore
			// queryId, err := utils.QueryIDFromString(aggReport.QueryId)
			queryId, err := hex.DecodeString(aggReport.QueryId)
			if err != nil {
				h.logger.Error("Failed to get query id from string", "error", err)
				panic(err)
			} else {
				h.logger.Info("Query ID from string", "queryId", queryId)
			}
			h.logger.Info("getting data before")
			dataBefore, err := h.oracleKeeper.GetDataBeforePublic(ctx, queryId, time.Unix(ts, 0))
			if err != nil {
				h.logger.Error("Failed to get data before", "error", err)
				h.logger.Info("dataBefore", "dataBefore", dataBefore)
				return nil, err
			}
			h.logger.Info("getting timestamp before for report time")
			reportTime, err := h.oracleKeeper.GetTimestampBefore(ctx, queryId, time.Unix(ts, 0))
			if err != nil {
				h.logger.Error("Failed to get timestamp before", "error", err)
				h.logger.Info("reportTime", "reportTime", reportTime)
				return nil, err
			} else {
				h.logger.Info("Report time", "reportTime", reportTime)
			}
			// report, err := h.oracleKeeper.GetAggregateReport(ctx, []byte(aggReport.QueryId), ts)
			// if err != nil {
			// 	return nil, err
			// }
			h.logger.Info("getting timestamp before current report")
			tsBefore, err := h.oracleKeeper.GetTimestampBefore(ctx, queryId, reportTime)
			if err != nil {
				h.logger.Info("Failed to get timestamp before", "error", err)
				// set to 0
				tsBefore = time.Unix(0, 0)
			}
			h.logger.Info("getting timestamp after current report")
			tsAfter, err := h.oracleKeeper.GetTimestampAfter(ctx, queryId, reportTime)
			if err != nil {
				h.logger.Info("Failed to get timestamp after", "error", err)
				// set to 0
				tsAfter = time.Unix(0, 0)
			}
			h.logger.Info("encoding oracle attestation data")
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
			h.logger.Info("signing oracle attestation hash")
			sig, err := h.SignMessage(oracleAttestationHash)
			if err != nil {
				return nil, err
			}
			h.logger.Info("Signature generated", "sig", hex.EncodeToString(sig))
			oracleAttestation := OracleAttestation{
				Attestation: sig,
				QueryId:     aggReport.QueryId,
				Timestamp:   uint64(reportTime.Unix()),
			}
			h.logger.Info("appending oracle attestation to vote extension")
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
	h.logger.Info("@VerifyVoteExtensionHandler", "req", req)
	// logic for verifying oracle sigs
	extension := req.GetVoteExtension()
	// unmarshal vote extension
	voteExt := BridgeVoteExtension{}
	err := json.Unmarshal(extension, &voteExt)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal vote extension: %w", err)
	}
	// check for initial sig
	if len(voteExt.InitialSignature.Signature) > 0 {
		// verify initial sig
		sigHexString := hex.EncodeToString(voteExt.InitialSignature.Signature)
		evmAddress, err := h.bridgeKeeper.EVMAddressFromSignature(ctx, sigHexString)
		if err != nil {
			return nil, err
		}
		h.logger.Info("EVM address from initial sig", "evmAddress", evmAddress)
	}

	if bytes.Equal(extension, []byte("vote extension data")) {
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	} else {
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
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
	h.logger.Info("@EncodeOracleAttestationData - extend_vote.go")
	// NEW_REPORT_ATTESTATION_DOMAIN_SEPERATOR := []byte("tellorNewReport")
	domainSep := "74656c6c6f7243757272656e744174746573746174696f6e0000000000000000"
	NEW_REPORT_ATTESTATION_DOMAIN_SEPERATOR, err := hex.DecodeString(domainSep)
	if err != nil {
		return nil, err
	}
	// Convert domain separator to bytes32
	var domainSepBytes32 [32]byte
	copy(domainSepBytes32[:], NEW_REPORT_ATTESTATION_DOMAIN_SEPERATOR)

	// print everything
	h.logger.Info("domainSepBytes32", "domainSepBytes32", hex.EncodeToString(domainSepBytes32[:]))
	h.logger.Info("queryId", "queryId", queryId)
	h.logger.Info("value", "value", value)
	h.logger.Info("timestamp", "timestamp", fmt.Sprintf("%d", timestamp))
	h.logger.Info("aggregatePower", "aggregatePower", fmt.Sprintf("%d", aggregatePower))
	h.logger.Info("previousTimestamp", "previousTimestamp", fmt.Sprintf("%d", previousTimestamp))
	h.logger.Info("nextTimestamp", "nextTimestamp", fmt.Sprintf("%d", nextTimestamp))
	h.logger.Info("valsetCheckpoint", "valsetCheckpoint", valsetCheckpoint)
	h.logger.Info("attestationTimestamp", "attestationTimestamp", fmt.Sprintf("%d", attestationTimestamp))

	// Convert queryId to bytes32
	h.logger.Info("encoding queryId to bytes32")
	queryIdBytes, err := hex.DecodeString(queryId)
	if err != nil {
		return nil, err
	}
	var queryIdBytes32 [32]byte
	copy(queryIdBytes32[:], queryIdBytes)

	// Convert value to bytes
	h.logger.Info("encoding value to bytes")
	valueBytes, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}

	// Convert timestamp to uint64
	h.logger.Info("encoding timestamp to uint64")
	// timestampUint64 := uint64(timestamp)
	// h.logger.Info("timestamps", "timestampUint64", timestampUint64, "timestamp", timestamp)
	//convert timestamp to bigInt
	timestampUint64 := new(big.Int)
	timestampUint64.SetInt64(timestamp)
	h.logger.Info("timestampUint64", "timestampUint64", timestampUint64)

	// Convert aggregatePower to uint64
	h.logger.Info("encoding aggregatePower to uint64")
	aggregatePowerUint64 := new(big.Int)
	aggregatePowerUint64.SetInt64(aggregatePower)

	// Convert previousTimestamp to uint64
	h.logger.Info("encoding previousTimestamp to uint64")
	previousTimestampUint64 := new(big.Int)
	previousTimestampUint64.SetInt64(previousTimestamp)

	// Convert nextTimestamp to uint64
	h.logger.Info("encoding nextTimestamp to uint64")
	nextTimestampUint64 := new(big.Int)
	nextTimestampUint64.SetInt64(nextTimestamp)

	// Convert valsetCheckpoint to bytes32
	h.logger.Info("encoding valsetCheckpoint to bytes32")
	valsetCheckpointBytes, err := hex.DecodeString(valsetCheckpoint)
	if err != nil {
		return nil, err
	}
	var valsetCheckpointBytes32 [32]byte
	copy(valsetCheckpointBytes32[:], valsetCheckpointBytes)

	// Convert attestationTimestamp to uint64
	h.logger.Info("encoding attestationTimestamp to uint64")
	attestationTimestampUint64 := new(big.Int)
	attestationTimestampUint64.SetInt64(attestationTimestamp)

	// Prepare Encoding
	h.logger.Info("preparing encoding")
	Bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	h.logger.Info("encoding oracle attestation data")
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	h.logger.Info("encoding oracle attestation data")
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	h.logger.Info("preparing arguments")
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

	// remove all of this
	argsDomainSep := abi.Arguments{{Type: Bytes32Type}}
	argsQueryId := abi.Arguments{{Type: Bytes32Type}}
	argsValue := abi.Arguments{{Type: BytesType}}
	argsTimestamp := abi.Arguments{{Type: Uint256Type}}
	argsAggregatePower := abi.Arguments{{Type: Uint256Type}}
	argsPreviousTimestamp := abi.Arguments{{Type: Uint256Type}}
	argsNextTimestamp := abi.Arguments{{Type: Uint256Type}}
	argsValsetCheckpoint := abi.Arguments{{Type: Bytes32Type}}
	argsAttestationTimestamp := abi.Arguments{{Type: Uint256Type}}

	h.logger.Info("encoding domain separator")
	domainSepEncoded, err := argsDomainSep.Pack(domainSepBytes32)
	if err != nil {
		h.logger.Error("failed to encode domain separator", "domainSepBytes", domainSepEncoded)
		return nil, err
	}
	h.logger.Info("encoding queryId")
	queryIdEncoded, err := argsQueryId.Pack(queryIdBytes32)
	if err != nil {
		h.logger.Error("failed to encode queryId", "queryIdBytes", queryIdEncoded)
	}
	h.logger.Info("encoding value")
	valueEncoded, err := argsValue.Pack(valueBytes)
	if err != nil {
		h.logger.Error("failed to encode value", "valueBytes", valueEncoded)
	}
	h.logger.Info("encoding timestamp")
	timestampEncoded, err := argsTimestamp.Pack(timestampUint64)
	if err != nil {
		h.logger.Error("failed to encode timestamp", "timestampEncoded", timestampEncoded)
		h.logger.Error("error", "error", err)
	}
	h.logger.Info("encoding aggregatePower")
	aggregatePowerEncoded, err := argsAggregatePower.Pack(aggregatePowerUint64)
	if err != nil {
		h.logger.Error("failed to encode aggregatePower", "aggregatePowerUint64", aggregatePowerEncoded)
	}
	h.logger.Info("encoding previousTimestamp")
	previousTimestampEncoded, err := argsPreviousTimestamp.Pack(previousTimestampUint64)
	if err != nil {
		h.logger.Error("failed to encode previousTimestamp", "previousTimestampUint64", previousTimestampEncoded)
	}
	h.logger.Info("encoding nextTimestamp")
	nextTimestampEncoded, err := argsNextTimestamp.Pack(nextTimestampUint64)
	if err != nil {
		h.logger.Error("failed to encode nextTimestamp", "nextTimestampUint64", nextTimestampEncoded)
	}
	h.logger.Info("encoding valsetCheckpoint")
	valsetCheckpointEncoded, err := argsValsetCheckpoint.Pack(valsetCheckpointBytes32)
	if err != nil {
		h.logger.Error("failed to encode valsetCheckpoint", "valsetCheckpointBytes32", valsetCheckpointEncoded)
	}
	h.logger.Info("encoding attestationTimestamp")
	attestationTimestampEncoded, err := argsAttestationTimestamp.Pack(attestationTimestampUint64)
	if err != nil {
		h.logger.Error("failed to encode attestationTimestamp", "attestationTimestampUint64", attestationTimestampEncoded)
	}

	// Encode the data
	h.logger.Info("encoding data")
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

	h.logger.Info("hashing encoded data")
	oracleAttestationHash := crypto.Keccak256(encodedData)
	h.logger.Info("oracleAttestationHash", "oracleAttestationHash", hex.EncodeToString(oracleAttestationHash))
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
	h.logger.Info("Keyring dir:", "dir", krDir)

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

	// Fetch the operator key from the keyring.
	info, err := kr.Key(keyName)
	if err != nil {
		fmt.Printf("Failed to get operator key: %v\n", err)
		return nil, err
	}
	// Output the public key associated with the operator key.
	key, _ := info.GetPubKey()
	keyAddrStr := key.Address().String()
	fmt.Println("Operator Public Key:", keyAddrStr)

	// sign message
	// tempmsg := []byte("hello")
	sig, pubKeyReturned, err := kr.Sign(keyName, msg, 1)
	if err != nil {
		fmt.Printf("Failed to sign message: %v\n", err)
		return nil, err
	}
	h.logger.Info("Signature:", "sig", cosbytes.HexBytes(sig).String())
	h.logger.Info("Public Key:", pubKeyReturned.Address().String())
	return sig, nil
}

func (h *VoteExtHandler) SignInitialMessage() ([]byte, error) {
	message := "TellorLayer: Initial bridge daemon signature"
	// convert message to bytes
	msgBytes := []byte(message)
	// hash message
	msgHashBytes32 := sha256.Sum256(msgBytes)
	// convert [32]byte to []byte
	msgHashBytes := msgHashBytes32[:]
	// sign message
	sig, err := h.SignMessage(msgHashBytes)
	if err != nil {
		return nil, err
	}
	sig = append(sig, 0)
	return sig, nil
}

func (h *VoteExtHandler) GetOperatorAddress() (string, error) {
	h.logger.Info("@GetOperatorAddress - extend_vote.go")
	// define keyring backend and the path to the keystore dir
	keyName := h.GetKeyName()
	h.logger.Info("keyName:", "keyName", keyName)
	if keyName == "" {
		return "", fmt.Errorf("key name not found")
	}
	krBackend := keyring.BackendTest
	h.logger.Info("keyring backend:", "krBackend", krBackend)
	krDir := os.ExpandEnv("$HOME/.layer/" + keyName)

	h.logger.Info("Keyring dir:", "dir", krDir)

	userInput := os.Stdin
	// userInput := os.Stdin
	h.logger.Info("userInput:", "userInput", userInput)

	kr, err := keyring.New("layer", krBackend, krDir, userInput, h.codec)
	if err != nil {
		fmt.Printf("Failed to create keyring: %v\n", err)
		return "", err
	}

	// print kr info
	h.logger.Info("Keyring info:", "kr", kr)
	h.logger.Info("Keyring backend:", "kr.Backend()", kr.Backend())

	// list all keys
	krlist, err := kr.List()
	if err != nil {
		fmt.Printf("Failed to list keys: %v\n", err)
		return "", err
	}
	if len(krlist) == 0 {
		h.logger.Info("No keys found in keyring")
	}
	// log all keys
	for _, k := range krlist {
		h.logger.Info("name: ", "name", k.Name)
		h.logger.Info("type: ", "type", k.GetType())
		// h.logger.Info("item", "item", k.Item)
		pubkey, _ := k.GetPubKey()
		h.logger.Info("pubkey", "pubkey", pubkey.String())
		address, _ := k.GetAddress()
		h.logger.Info("address", "address", address.String())
	}

	// Fetch the operator key from the keyring.
	info, err := kr.Key(keyName)
	if err != nil {
		fmt.Printf("Failed to get operator key: %v\n", err)
		return "", err
	}
	// Output the public key associated with the operator key.
	key, _ := info.GetPubKey()
	keyAddrStr := key.Address().String()
	pubkeystr := key.String()
	h.logger.Info("@pubkeystr:", "pubkeystr", pubkeystr)
	h.logger.Info("Operator Public Key:", "keyAddrStr", keyAddrStr)

	// Convert the operator's public key to a Bech32 validator address
	config := sdk.GetConfig()
	bech32PrefixValAddr := config.GetBech32ValidatorAddrPrefix()
	bech32ValAddr, err := sdk.Bech32ifyAddressBytes(bech32PrefixValAddr, key.Address().Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to convert operator public key to Bech32 validator address: %w", err)
	}
	h.logger.Info("Operator Validator Address:", "bech32ValAddr", bech32ValAddr)
	return bech32ValAddr, nil
}

func (h *VoteExtHandler) GetKeyName() string {
	globalHome := os.ExpandEnv("$HOME/.layer")
	homeDir := viper.GetString("home")
	// if home is global/alice, then the key name is alice
	if homeDir == globalHome+"/alice" {
		h.logger.Info("@keyname - alice")
		return "alice"
	} else if homeDir == globalHome+"/bill" {
		h.logger.Info("@keyname - bill")
		return "bill"
	} else {
		h.logger.Info("@keyname - empty")
		return ""
	}
}

func (h *VoteExtHandler) CheckAndSignValidatorCheckpoint(ctx sdk.Context) (signature []byte, timestamp uint64, err error) {
	h.logger.Info("@CheckAndSignValidatorCheckpoint - extend_vote.go")
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
	// // get the latest validator set signatures
	// latestValsetSignatures, err := h.bridgeKeeper.GetValidatorSetSignaturesFromStorage(ctx, latestCheckpointTimestamp.Timestamp)
	// if err != nil {
	// 	h.logger.Error("failed to get latest validator set signatures", "error", err)
	// 	return nil, 0, err
	// }
	// // get the latest validator set
	// latestValset, err := h.bridgeKeeper.GetBridgeValsetByTimestamp(ctx, latestCheckpointTimestamp.Timestamp)
	// if err != nil {
	// 	h.logger.Error("failed to get latest validator set", "error", err)
	// 	return nil, 0, err
	// }
	// // get operator address
	// operatorAddress, err := h.GetOperatorAddress()
	// if err != nil {
	// 	h.logger.Error("failed to get operator address", "error", err)
	// 	return nil, 0, err
	// }

	// // get evm address by operator
	// evmAddress, err := h.bridgeKeeper.GetEVMAddressByOperator(ctx, operatorAddress)
	// if err != nil {
	// 	h.logger.Error("failed to get evm address by operator", "error", err)
	// 	return nil, 0, err
	// }

	// // get validator's index in the latest validator set
	// valIndex, err := h.GetValidatorIndexInValset(ctx, evmAddress, latestValset)
	// if valIndex < 0 {
	// 	h.logger.Error("validator not found in latest validator set", "error", err)
	// 	return nil, 0, err
	// }

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
		h.logger.Info("Validator already signed checkpoint", "operatorAddress", operatorAddress, "checkpointTimestamp", latestCheckpointTimestamp.Timestamp)
		return nil, 0, nil
	} else if valIndex < 0 {
		h.logger.Error("Validator not found in previous validator set", "operatorAddress", operatorAddress, "checkpointTimestamp", latestCheckpointTimestamp.Timestamp)
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
		h.logger.Info("Signature generated", "signature", hex.EncodeToString(signature))
		return signature, latestCheckpointTimestamp.Timestamp, nil
	}

	// // check for signature at index
	// if len(latestValsetSignatures.Signatures) > valIndex {
	// 	sig := latestValsetSignatures.Signatures[valIndex]
	// 	if len(sig) > 0 {
	// 		h.logger.Info("Signature found at index", "index", valIndex)
	// 		return nil, 0, nil
	// 	} else {
	// 		// check previous valset for inclusion
	// 		if latestCheckpointIdx > 0 {
	// 			previousCheckpointTimestamp, err := h.bridgeKeeper.GetValidatorTimestampByIdxFromStorage(ctx, latestCheckpointIdx-1)
	// 			if err != nil {
	// 				h.logger.Error("failed to get previous checkpoint timestamp", "error", err)
	// 				return nil, 0, err
	// 			}
	// 			previousValset, err := h.bridgeKeeper.GetBridgeValsetByTimestamp(ctx, previousCheckpointTimestamp.Timestamp)
	// 			if err != nil {
	// 				h.logger.Error("failed to get previous validator set", "error", err)
	// 				return nil, 0, err
	// 			}
	// 			// check if validator is included in previous valset
	// 			valIndex, err := h.GetValidatorIndexInValset(ctx, evmAddress, previousValset)
	// 			if valIndex < 0 {
	// 				h.logger.Error("validator not found in previous validator set", "error", err)
	// 				return nil, 0, nil
	// 			}
	// 			// sign the latest checkpoint
	// 			checkpointParams, err := h.bridgeKeeper.GetValidatorCheckpointParamsFromStorage(ctx, latestCheckpointTimestamp.Timestamp)
	// 			if err != nil {
	// 				h.logger.Error("failed to get checkpoint params", "error", err)
	// 				return nil, 0, err
	// 			}
	// 			checkpoint := checkpointParams.Checkpoint
	// 			checkpointString := hex.EncodeToString(checkpoint)
	// 			signature, err := h.EncodeAndSignMessage(checkpointString)
	// 			if err != nil {
	// 				h.logger.Error("failed to encode and sign message", "error", err)
	// 				return nil, 0, err
	// 			}
	// 			h.logger.Info("Signature generated", "signature", hex.EncodeToString(signature))
	// 			return signature, latestCheckpointTimestamp.Timestamp, nil
	// 		} else {
	// 			h.logger.Info("No previous valset found")
	// 			return nil, 0, nil
	// 		}
	// 	}
	// } else {
	// 	h.logger.Info("No signature found at index", "index", valIndex)
	// 	return nil, 0, nil
	// }
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
