package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"

	gomath "math"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	math "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"sort"

	"github.com/tellor-io/layer/x/bridge/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService storetypes.KVStoreService

		Schema                    collections.Schema
		Params                    collections.Item[types.Params]
		BridgeValset              collections.Item[types.BridgeValidatorSet]
		ValidatorCheckpoint       collections.Item[types.ValidatorCheckpoint]
		OperatorToEVMAddressMap   collections.Map[string, types.EVMAddress]
		BridgeValsetSignaturesMap collections.Map[uint64, types.BridgeValsetSignatures]

		stakingKeeper  types.StakingKeeper
		slashingKeeper types.SlashingKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	stakingKeeper types.StakingKeeper,
	slashingKeeper types.SlashingKeeper,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:                       cdc,
		storeService:              storeService,
		Params:                    collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		BridgeValset:              collections.NewItem(sb, types.BridgeValsetKey, "bridge_valset", codec.CollValue[types.BridgeValidatorSet](cdc)),
		ValidatorCheckpoint:       collections.NewItem(sb, types.ValidatorCheckpointKey, "validator_checkpoint", codec.CollValue[types.ValidatorCheckpoint](cdc)),
		OperatorToEVMAddressMap:   collections.NewMap(sb, types.OperatorToEVMAddressMapKey, "operator_to_evm_address_map", collections.StringKey, codec.CollValue[types.EVMAddress](cdc)),
		BridgeValsetSignaturesMap: collections.NewMap(sb, types.BridgeValsetSignaturesMapKey, "bridge_valset_signatures_map", collections.Uint64Key, codec.CollValue[types.BridgeValsetSignatures](cdc)),
		stakingKeeper:             stakingKeeper,
		slashingKeeper:            slashingKeeper,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetCurrentValidatorsEVMCompatible(ctx sdk.Context) ([]*types.BridgeValidator, error) {
	validators, err := k.stakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return nil, err
	}

	bridgeValset := make([]*types.BridgeValidator, len(validators))

	for i, validator := range validators {
		evmAddress, err := k.OperatorToEVMAddressMap.Get(ctx, validator.GetOperator())
		evmAddressHex := hex.EncodeToString(evmAddress.EVMAddress)
		if err != nil {
			k.Logger(ctx).Info("Error getting EVM address from operator address", "error", err)
			return nil, err
		}
		bridgeValset[i] = &types.BridgeValidator{
			EthereumAddress: evmAddressHex,
			Power:           uint64(validator.GetConsensusPower(math.NewInt(10))),
		}
		k.Logger(ctx).Info("@GetBridgeValidators - bridge validator DDDD", "test", bridgeValset[i].EthereumAddress)
	}

	// Sort the validators
	sort.Slice(bridgeValset, func(i, j int) bool {
		if bridgeValset[i].Power == bridgeValset[j].Power {
			// If power is equal, sort alphabetically
			return bridgeValset[i].EthereumAddress < bridgeValset[j].EthereumAddress
		}
		// Otherwise, sort by power in descending order
		return bridgeValset[i].Power > bridgeValset[j].Power
	})

	return bridgeValset, nil
}

func (k Keeper) GetCurrentValidatorSetEVMCompatible(ctx sdk.Context) (*types.BridgeValidatorSet, error) {
	// use GetBridgeValidators to get the current bridge validator set
	bridgeValset, err := k.GetCurrentValidatorsEVMCompatible(ctx)
	if err != nil {
		return nil, err
	}

	return &types.BridgeValidatorSet{BridgeValidatorSet: bridgeValset}, nil
}

// function for loading last saved bridge validator set and comparing it to current set
func (k Keeper) CompareBridgeValidators(ctx sdk.Context) (bool, error) {
	k.Logger(ctx).Info("@CompareBridgeValidators", "msg", "comparing bridge validators")
	// load current validator set in EVM compatible format
	currentValidatorSetEVMCompatible, err := k.GetCurrentValidatorSetEVMCompatible(ctx)
	if err != nil {
		k.Logger(ctx).Info("No current validator set found")
		return false, err
	}
	k.Logger(ctx).Info("@COMPARE", "msg", "setting bridge validator params")

	lastSavedBridgeValidators, err := k.BridgeValset.Get(ctx)
	if err != nil {
		k.Logger(ctx).Info("No saved bridge validator set found")
		err := k.BridgeValset.Set(ctx, *currentValidatorSetEVMCompatible)
		if err != nil {
			k.Logger(ctx).Info("Error setting bridge validator set: ", "error", err)
			return false, err
		}
		error := k.SetBridgeValidatorParams(ctx, currentValidatorSetEVMCompatible)
		if error != nil {
			k.Logger(ctx).Info("Error setting bridge validator params: ", "error", error)
			return false, error
		}
		return false, err
	}
	if bytes.Equal(k.cdc.MustMarshal(&lastSavedBridgeValidators), k.cdc.MustMarshal(currentValidatorSetEVMCompatible)) {
		k.Logger(ctx).Info("Bridge validator set has not changed")
		return true, nil
	} else if k.PowerDiff(ctx, lastSavedBridgeValidators, *currentValidatorSetEVMCompatible) < 0.05 {
		k.Logger(ctx).Info("Power diff is less than 5%")
		return false, nil
	} else {
		err := k.BridgeValset.Set(ctx, *currentValidatorSetEVMCompatible)
		if err != nil {
			k.Logger(ctx).Info("Error setting bridge validator set: ", "error", err)
			return false, err
		}
		error := k.SetBridgeValidatorParams(ctx, currentValidatorSetEVMCompatible)
		if error != nil {
			k.Logger(ctx).Info("Error setting bridge validator params: ", "error", error)
			return false, error
		}
		k.Logger(ctx).Info("Bridge validator set updated")
		for i, validator := range lastSavedBridgeValidators.BridgeValidatorSet {
			k.Logger(ctx).Info("Last saved bridge validator ", "savedVal", validator.EthereumAddress)
			k.Logger(ctx).Info("i ", "i", i)
		}
		for i, validator := range currentValidatorSetEVMCompatible.BridgeValidatorSet {
			k.Logger(ctx).Info("Current bridge validator ", i, ": ", validator.EthereumAddress+" "+fmt.Sprint(validator.Power))
		}
		return true, nil
	}
}

func (k Keeper) SetBridgeValidatorParams(ctx sdk.Context, bridgeValidatorSet *types.BridgeValidatorSet) error {
	k.Logger(ctx).Info("@SetBridgeValidatorParams", "msg", "setting bridge validator params")
	var totalPower uint64
	for _, validator := range bridgeValidatorSet.BridgeValidatorSet {
		totalPower += validator.GetPower()
	}
	powerThreshold := totalPower * 2 / 3

	validatorTimestamp := uint64(ctx.BlockTime().Unix())

	// calculate validator set hash
	_, validatorSetHash, err := k.EncodeAndHashValidatorSet(ctx, bridgeValidatorSet)
	if err != nil {
		k.Logger(ctx).Info("Error encoding and hashing validator set: ", "error", err)
		return err
	}

	// calculate validator set checkpoint
	checkpoint, err := k.CalculateValidatorSetCheckpoint(ctx, powerThreshold, validatorTimestamp, validatorSetHash)
	if err != nil {
		k.Logger(ctx).Info("Error calculating validator set checkpoint: ", "error", err)
		return err
	}

	k.Logger(ctx).Info("@SetBridgeValidatorParams", "powerThreshold", fmt.Sprint(powerThreshold))
	k.Logger(ctx).Info("SetBridgeValidatorParams", "validatorTimestamp", fmt.Sprint(validatorTimestamp))
	k.Logger(ctx).Info("SetBridgeValidatorParams", "validatorSetHash", fmt.Sprintf("%x", validatorSetHash))
	k.Logger(ctx).Info("SetBridgeValidatorParams", "checkpoint", fmt.Sprintf("%x", checkpoint))

	// Set the validator checkpoint
	checkpointType := types.ValidatorCheckpoint{
		Checkpoint: checkpoint,
	}

	error := k.ValidatorCheckpoint.Set(ctx, checkpointType)
	if error != nil {
		k.Logger(ctx).Info("Error setting validator checkpoint: ", "error", error)
		return error
	}

	valsetSigs := types.NewBridgeValsetSignatures(len(bridgeValidatorSet.BridgeValidatorSet))
	err = k.BridgeValsetSignaturesMap.Set(ctx, validatorTimestamp, *valsetSigs)
	if err != nil {
		k.Logger(ctx).Info("Error setting bridge valset signatures: ", "error", err)
		return err
	}

	// Emit EventTypeBridgeValidatorSetUpdated event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBridgeValidatorSetUpdated, // Assuming types.EventTypeBridgeValidatorSetUpdated is the constant for the event type
			sdk.NewAttribute(types.AttributeKeyValidatorSetCheckpoint, fmt.Sprintf("%x", checkpoint)),
		),
	)

	return nil
}

func (k Keeper) CalculateValidatorSetCheckpoint(
	ctx sdk.Context,
	powerThreshold uint64,
	validatorTimestamp uint64,
	validatorSetHash []byte,
) ([]byte, error) {
	k.Logger(ctx).Info("@CalculateValidatorSetCheckpoint", "msg", "calculating validator set checkpoint")

	// Define the domain separator for the validator set hash, fixed size 32 bytes
	VALIDATOR_SET_HASH_DOMAIN_SEPARATOR := []byte("checkpoint")
	var domainSeparatorFixSize [32]byte
	copy(domainSeparatorFixSize[:], VALIDATOR_SET_HASH_DOMAIN_SEPARATOR)

	// Convert validatorSetHash to a fixed size 32 bytes
	var validatorSetHashFixSize [32]byte
	copy(validatorSetHashFixSize[:], validatorSetHash)

	// Convert powerThreshold and validatorTimestamp to *big.Int for ABI encoding
	powerThresholdBigInt := new(big.Int).SetUint64(powerThreshold)
	validatorTimestampBigInt := new(big.Int).SetUint64(validatorTimestamp)

	Bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		k.Logger(ctx).Warn("Error creating new bytes32 ABI type", "error", err)
		return nil, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		k.Logger(ctx).Warn("Error creating new uint256 ABI type", "error", err)
		return nil, err
	}

	// Prepare the types for encoding
	arguments := abi.Arguments{
		{Type: Bytes32Type},
		{Type: Uint256Type},
		{Type: Uint256Type},
		{Type: Bytes32Type},
	}

	// // ********** DELETE THIS, JUST FOR TESTING - START **********
	// arg1 := abi.Arguments{{Type: Bytes32Type}}
	// arg2 := abi.Arguments{{Type: Uint256Type}}
	// arg3 := abi.Arguments{{Type: Uint256Type}}
	// arg4 := abi.Arguments{{Type: Bytes32Type}}

	// encodedData1, err := arg1.Pack(domainSeparatorFixSize)
	// if err != nil {
	// 	k.Logger(ctx).Warn("Error encoding arguments arg1", "error", err)
	// 	return nil, err
	// }
	// encodedData2, err := arg2.Pack(powerThresholdBigInt)
	// if err != nil {
	// 	k.Logger(ctx).Warn("Error encoding arguments arg2", "error", err)
	// 	return nil, err
	// }
	// encodedData3, err := arg3.Pack(validatorTimestampBigInt)
	// if err != nil {
	// 	k.Logger(ctx).Warn("Error encoding arguments arg3", "error", err)
	// 	return nil, err
	// }
	// encodedData4, err := arg4.Pack(validatorSetHashFixSize)
	// if err != nil {
	// 	k.Logger(ctx).Warn("Error encoding arguments arg4", "error", err)
	// 	return nil, err
	// }

	// encodedDataTest := append(encodedData1, encodedData2...)
	// encodedDataTest = append(encodedDataTest, encodedData3...)
	// encodedDataTest = append(encodedDataTest, encodedData4...)

	// // ********** DELETE THIS, JUST FOR TESTING - END ************

	// Encode the arguments
	encodedCheckpointData, err := arguments.Pack(
		domainSeparatorFixSize,
		powerThresholdBigInt,
		validatorTimestampBigInt,
		validatorSetHashFixSize,
	)
	if err != nil {
		k.Logger(ctx).Warn("Error encoding arguments", "error", err)
		return nil, err
	}

	checkpoint := crypto.Keccak256(encodedCheckpointData)

	k.Logger(ctx).Info("DOMAIN_SEPARATOR", "DOMAIN_SEPARATOR", fmt.Sprintf("%x", domainSeparatorFixSize))
	k.Logger(ctx).Info("POWER_THRESHOLD", "POWER_THRESHOLD", fmt.Sprint(powerThreshold))
	k.Logger(ctx).Info("VALIDATOR_TIMESTAMP", "VALIDATOR_TIMESTAMP", fmt.Sprint(validatorTimestamp))
	k.Logger(ctx).Info("encodedData", "encodedData", fmt.Sprintf("%x", encodedCheckpointData))
	k.Logger(ctx).Info("checkpoint", "checkpoint", fmt.Sprintf("%x", checkpoint))

	// Hash the encoded data
	return checkpoint, nil
}

func (k Keeper) GetValidatorCheckpointFromStorage(ctx sdk.Context) (*types.ValidatorCheckpoint, error) {
	checkpoint, err := k.ValidatorCheckpoint.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("Failed to get validator checkpoint", "error", err)
		return nil, err
	}
	return &checkpoint, nil
}

func (k Keeper) EncodeAndHashValidatorSet(ctx sdk.Context, validatorSet *types.BridgeValidatorSet) (encodedBridgeValidatorSet []byte, bridgeValidatorSetHash []byte, err error) {
	k.Logger(ctx).Info("@EncodeAndHashValidatorSet", "msg", "encoding and hashing validator set")
	// Define Go equivalent of the Solidity Validator struct
	type Validator struct {
		Addr  common.Address
		Power *big.Int
	}

	// Convert validatorSet to a slice of the Validator struct defined above
	var validators []Validator
	for _, v := range validatorSet.BridgeValidatorSet {
		k.Logger(ctx).Info("EthereumAddress", "EthereumAddress", v.EthereumAddress)
		k.Logger(ctx).Info("Power", "Power", fmt.Sprint(v.Power))
		addr := common.HexToAddress(v.EthereumAddress)
		power := big.NewInt(0).SetUint64(v.Power)
		validators = append(validators, Validator{Addr: addr, Power: power})
	}

	// Solidity dynamic array encoding starts with the offset to the data
	// followed by the length of the array itself. Since we're directly encoding the array next,
	// the data starts immediately after these two fields, which is at 64 bytes offset.
	offsetToData := make([]byte, 32)
	binary.BigEndian.PutUint64(offsetToData[24:], uint64(32)) // 64 bytes offset to the start of the data

	// Encode the length of the array
	arrayLength := len(validators)
	lengthEncoded := make([]byte, 32)
	binary.BigEndian.PutUint64(lengthEncoded[24:], uint64(arrayLength))

	AddressType, err := abi.NewType("address", "", nil)
	if err != nil {
		k.Logger(ctx).Warn("Error creating new address ABI type", "error", err)
		return nil, nil, err
	}
	UintType, err := abi.NewType("uint256", "", nil)
	if err != nil {
		k.Logger(ctx).Warn("Error creating new uint256 ABI type", "error", err)
		return nil, nil, err
	}

	// Encode each Validator struct
	var encodedVals []byte
	for _, val := range validators {
		encodedVal, err := abi.Arguments{
			{Type: AddressType},
			{Type: UintType},
		}.Pack(val.Addr, val.Power)
		if err != nil {
			return nil, nil, err
		}
		encodedVals = append(encodedVals, encodedVal...)
	}

	// Concatenate the offset, length, and encoded validators
	finalEncoded := append(offsetToData, lengthEncoded...)
	finalEncoded = append(finalEncoded, encodedVals...)

	// Hash the encoded bytes
	valSetHash := crypto.Keccak256(finalEncoded)

	// print finalEncoded string
	k.Logger(ctx).Info("finalEncoded valset", "finalEncoded", fmt.Sprintf("%x", finalEncoded))
	// valsethash string
	k.Logger(ctx).Info("valsethash", "valsethash", fmt.Sprintf("%x", valSetHash))
	return finalEncoded, valSetHash, nil
}

func (k Keeper) PowerDiff(ctx sdk.Context, b types.BridgeValidatorSet, c types.BridgeValidatorSet) float64 {
	powers := map[string]int64{}
	for _, bv := range b.BridgeValidatorSet {
		powers[bv.EthereumAddress] = int64(bv.GetPower())
	}

	for _, bv := range c.BridgeValidatorSet {
		if val, ok := powers[bv.EthereumAddress]; ok {
			powers[bv.EthereumAddress] = val - int64(bv.GetPower())
		} else {
			powers[bv.EthereumAddress] = -int64(bv.GetPower())
		}
	}

	var delta float64
	for _, v := range powers {
		delta += gomath.Abs(float64(v))
	}

	return gomath.Abs(delta / float64(gomath.MaxUint32))
}

func (k Keeper) EVMAddressFromSignature(ctx sdk.Context, sigHexString string) (string, error) {
	message := "TellorLayer: Initial bridge daemon signature"
	// convert message to bytes
	msgBytes := []byte(message)
	// hash message
	msgHashBytes32 := sha256.Sum256(msgBytes)
	// convert [32]byte to []byte
	msgHashBytes := msgHashBytes32[:]

	// hash the hash, since the keyring signer automatically hashes the message
	msgDoubleHashBytes32 := sha256.Sum256(msgHashBytes)
	msgDoubleHashBytes := msgDoubleHashBytes32[:]

	// Convert the hex signature to bytes
	signatureBytes, err := hex.DecodeString(sigHexString)
	if err != nil {
		k.Logger(ctx).Warn("Error decoding signature hex", "error", err)
		return "", err
	}
	// append 01

	// Recover the public key
	sigPublicKey, err := crypto.SigToPub(msgDoubleHashBytes, signatureBytes)
	if err != nil {
		k.Logger(ctx).Warn("Error recovering public key from signature", "error", err)
		return "", err
	}

	// Get the address
	recoveredAddr := crypto.PubkeyToAddress(*sigPublicKey)

	k.Logger(ctx).Info("Recovered Address:", recoveredAddr.Hex())
	return recoveredAddr.Hex(), nil
}

func (k Keeper) SetEVMAddressByOperator(ctx sdk.Context, operatorAddr string, evmAddr string) error {
	k.Logger(ctx).Info("@SetEVMAddressByOperator", "msg", "setting EVM address by operator")
	evmAddrBytes := common.HexToAddress(evmAddr).Bytes()
	evmAddrType := types.EVMAddress{
		EVMAddress: evmAddrBytes,
	}

	err := k.OperatorToEVMAddressMap.Set(ctx, operatorAddr, types.EVMAddress(evmAddrType))
	if err != nil {
		k.Logger(ctx).Info("Error setting EVM address by operator", "error", err)
		return err
	}
	return nil
}

func (k Keeper) SetBridgeValsetSignature(ctx sdk.Context, operatorAddress string, timestamp uint64, signature string) error {
	k.Logger(ctx).Info("@SetBridgeValsetSignature", "msg", "setting bridge valset signature")
	// get the bridge valset signatures array by timestamp
	valsetSigs, err := k.BridgeValsetSignaturesMap.Get(ctx, timestamp)
	if err != nil {
		k.Logger(ctx).Info("Error getting bridge valset signatures", "error", err)
		return err
	}
	// get the evm address associated with the operator address
	ethAddress, err := k.OperatorToEVMAddressMap.Get(ctx, operatorAddress)
	if err != nil {
		k.Logger(ctx).Info("Error getting EVM address from operator address", "error", err)
		return err
	}
	// get the last saved bridge validator set
	lastSavedBridgeValidators, err := k.BridgeValset.Get(ctx)
	if err != nil {
		k.Logger(ctx).Info("Error getting last saved bridge validators", "error", err)
		return err
	}
	// decode the signature hex
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		k.Logger(ctx).Info("Error decoding signature hex", "error", err)
		return err
	}
	// set the signature in the valset signatures array by finding the index of the operator address
	ethAddressHex := hex.EncodeToString(ethAddress.EVMAddress)
	for i, val := range lastSavedBridgeValidators.BridgeValidatorSet {
		if val.EthereumAddress == ethAddressHex {
			valsetSigs.SetSignature(i, signatureBytes)
		}
	}
	// set the valset signatures array by timestamp
	err = k.BridgeValsetSignaturesMap.Set(ctx, timestamp, valsetSigs)
	if err != nil {
		k.Logger(ctx).Info("Error setting bridge valset signatures", "error", err)
		return err
	}
	return nil
}
