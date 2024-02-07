package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"strconv"

	gomath "math"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	math "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

		Schema       collections.Schema
		Params       collections.Item[types.Params]
		BridgeValset collections.Item[types.BridgeValidatorSet]

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
		cdc:            cdc,
		storeService:   storeService,
		Params:         collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		BridgeValset:   collections.NewItem(sb, types.BridgeValsetKey, "bridge_valset", codec.CollValue[types.BridgeValidatorSet](cdc)),
		stakingKeeper:  stakingKeeper,
		slashingKeeper: slashingKeeper,
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

func (k Keeper) GetBridgeValidators(ctx sdk.Context) ([]*types.BridgeValidator, error) {
	validators, err := k.stakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return nil, err
	}

	bridgeValset := make([]*types.BridgeValidator, len(validators))

	for i, validator := range validators {
		valAddress, err := sdk.ValAddressFromBech32(validator.GetOperator())
		if err != nil {
			return nil, err
		}
		bridgeValset[i] = &types.BridgeValidator{
			EthereumAddress: DefaultEVMAddress(valAddress).String(),
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

func (k Keeper) GetBridgeValidatorSet(ctx sdk.Context) (*types.BridgeValidatorSet, error) {
	// use GetBridgeValidators to get the current bridge validator set
	bridgeValset, err := k.GetBridgeValidators(ctx)
	if err != nil {
		return nil, err
	}

	return &types.BridgeValidatorSet{BridgeValidatorSet: bridgeValset}, nil
}

// function for loading last saved bridge validator set and comparing it to current set
func (k Keeper) CompareBridgeValidators(ctx sdk.Context) (bool, error) {
	currentBridgeValidators, err := k.GetBridgeValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Info("No current bridge validator set found")
		return false, err
	}
	k.EncodeAndHashValidatorSet(ctx, currentBridgeValidators)
	lastSavedBridgeValidators, err := k.BridgeValset.Get(ctx)
	if err != nil {

		k.Logger(ctx).Info("No saved bridge validator set found")
		k.BridgeValset.Set(ctx, *currentBridgeValidators)
		return false, err
	}
	if bytes.Equal(k.cdc.MustMarshal(&lastSavedBridgeValidators), k.cdc.MustMarshal(currentBridgeValidators)) {
		return true, nil
	} else if k.PowerDiff(ctx, lastSavedBridgeValidators, *currentBridgeValidators) < 0.05 {
		k.Logger(ctx).Info("Power diff is less than 5%")
		return false, nil
	} else {
		err := k.BridgeValset.Set(ctx, *currentBridgeValidators)
		if err != nil {
			return false, err
		}
		k.Logger(ctx).Info("Bridge validator set updated")
		for i, validator := range lastSavedBridgeValidators.BridgeValidatorSet {
			k.Logger(ctx).Info("Last saved bridge validator ", "savedVal", validator.EthereumAddress)
			k.Logger(ctx).Info("i ", "i", i)
		}
		for i, validator := range currentBridgeValidators.BridgeValidatorSet {
			k.Logger(ctx).Info("Current bridge validator ", i, ": ", validator.EthereumAddress+" "+fmt.Sprint(validator.Power))
		}
		return true, nil
	}
}

// func (k Keeper) SetBridgeValidatorParams(ctx sdk.Context, validatorSet *types.BridgeValidatorSet) error {
// 	var totalPower uint64
// 	for _, validator := range currentBridgeValidators.BridgeValidatorSet {
// 		totalPower += validator.GetPower()
// 	}
// 	powerThreshold := totalPower * 2 / 3

// 	// calculate validator set hash
// 	validatorSetHash := k.GetBridgeValidatorSetHash(validatorSet)
// }

// func (k Keeper) GetBridgeValidatorSetHash(validatorSet *types.BridgeValidatorSet) []byte {
// 	// get keccak256 hash of the validator set
// 	validatorSetBytes := k.cdc.MustMarshal(validatorSet)
// 	return crypto.Keccak256(validatorSetBytes)
// }

// func (k Keeper) CalculateValidatorSetCheckpoint(
//     powerThreshold uint64,
//     validatorTimestamp uint64,
//     validatorSetHash []byte,
// ) ([]byte, error) {
//     // Convert powerThreshold and validatorTimestamp to *big.Int for ABI encoding
//     powerThresholdBigInt := new(big.Int).SetUint64(powerThreshold)
//     validatorTimestampBigInt := new(big.Int).SetUint64(validatorTimestamp)

//     // Prepare the types for encoding
//     arguments := abi.Arguments{
//         {Type: abi.Bytes},
//         {Type: abi.Uint256},
//         {Type: abi.Uint256},
//         {Type: abi.Bytes32},
//     }

//     // Encode the arguments
//     // Note: Ensure VALIDATOR_SET_HASH_DOMAIN_SEPARATOR is correctly defined as per your contract
//     encodedData, err := arguments.Pack(
//         VALIDATOR_SET_HASH_DOMAIN_SEPARATOR,
//         powerThresholdBigInt,
//         validatorTimestampBigInt,
//         validatorSetHash,
//     )
//     if err != nil {
//         return nil, err
//     }

//     // Hash the encoded data
//     return crypto.Keccak256(encodedData), nil
// }

func (k Keeper) EncodeAndHashValidatorSet(ctx sdk.Context, validatorSet *types.BridgeValidatorSet) ([]byte, []byte, error) {
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
	k.Logger(ctx).Info("finalEncoded", "finalEncoded", fmt.Sprintf("%x", finalEncoded))
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

// https://github.com/ethereum/go-ethereum/blob/master/accounts/abi/argument.go
func EncodeArguments(dataTypes []string, dataFields []string) ([]byte, error) {
	var arguments abi.Arguments

	interfaceFields := make([]interface{}, len(dataFields))
	for i, dataType := range dataTypes {
		argType, err := abi.NewType(dataType, dataType, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create new ABI type: %v", err)
		}

		interfaceFields[i], err = ConvertStringToType(dataType, dataFields[i])
		if err != nil {
			return nil, err
		}

		arguments = append(arguments, abi.Argument{
			Name:    "",
			Type:    argType,
			Indexed: false,
		})
	}

	return arguments.Pack(interfaceFields...)
}

func ConvertStringToType(dataType, dataField string) (interface{}, error) {
	switch dataType {
	case "string":
		return dataField, nil
	case "bool":
		return strconv.ParseBool(dataField)
	case "address":
		// TODO: Validate address, maybe?
		return dataField, nil
	case "bytes":
		return []byte(dataField), nil
	case "int8", "int16", "int32", "int64", "int128", "int256", "uint8", "uint16", "uint32", "uint64", "uint128", "uint256":
		// https://docs.soliditylang.org/en/latest/types.html#integers
		value := new(big.Int)
		value, success := value.SetString(dataField, 10)
		if !success {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("could not set string to big.Int for value %s", dataField))
		}
		return value, nil
	default:
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("unsupported data type: %s", dataType))
	}
}
