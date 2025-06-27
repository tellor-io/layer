package v4

import (
	"context"
	"encoding/hex"

	"github.com/gogo/protobuf/proto"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/core/store"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidatorCheckpointParamsLegacy represents the old ValidatorCheckpointParams struct without BlockHeight
type ValidatorCheckpointParamsLegacy struct {
	Checkpoint     []byte `protobuf:"bytes,1,opt,name=checkpoint,proto3"`
	ValsetHash     []byte `protobuf:"bytes,2,opt,name=valset_hash,json=valsetHash,proto3"`
	Timestamp      uint64 `protobuf:"varint,3,opt,name=timestamp,proto3"`
	PowerThreshold uint64 `protobuf:"varint,4,opt,name=power_threshold,json=powerThreshold,proto3"`
}

var _ proto.Message = &ValidatorCheckpointParamsLegacy{}

func (*ValidatorCheckpointParamsLegacy) Reset() {}
func (m *ValidatorCheckpointParamsLegacy) String() string {
	return proto.CompactTextString(m)
}
func (*ValidatorCheckpointParamsLegacy) ProtoMessage() {}

// MigrateStore migrates the bridge module from v3 to v4:
// 1. Migrates ValidatorCheckpointParams from the legacy format to the new format by adding the BlockHeight field
// 2. Populates EVMToOperatorAddressMap from existing OperatorToEVMAddressMap entries for slashing functionality
func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))

	// migrate ValidatorCheckpointParams (existing migration)
	err := migrateValidatorCheckpointParams(store, cdc)
	if err != nil {
		return err
	}

	// migrate EVMToOperatorAddressMap (new migration)
	err = migrateEVMToOperatorAddressMap(store, cdc)
	if err != nil {
		return err
	}

	return nil
}

// migrateValidatorCheckpointParams migrates ValidatorCheckpointParams from legacy format to new format
func migrateValidatorCheckpointParams(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	checkpointStore := prefix.NewStore(store, bridgetypes.ValidatorCheckpointParamsMapKey)
	iter := checkpointStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var legacyParams ValidatorCheckpointParamsLegacy
		if err := cdc.Unmarshal(iter.Value(), &legacyParams); err != nil {
			panic("failed to unmarshal legacy ValidatorCheckpointParams")
		}

		// create new ValidatorCheckpointParams with BlockHeight set to 0
		newParams := bridgetypes.ValidatorCheckpointParams{
			Checkpoint:     legacyParams.Checkpoint,
			ValsetHash:     legacyParams.ValsetHash,
			Timestamp:      legacyParams.Timestamp,
			PowerThreshold: legacyParams.PowerThreshold,
			BlockHeight:    0, // default value for existing entries
		}

		newData, err := cdc.Marshal(&newParams)
		if err != nil {
			panic("unable to marshal new ValidatorCheckpointParams")
		}

		checkpointStore.Set(iter.Key(), newData)
	}

	return nil
}

// migrateEVMToOperatorAddressMap populates EVMToOperatorAddressMap from existing OperatorToEVMAddressMap entries
func migrateEVMToOperatorAddressMap(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// get the operator to EVM address map store
	operatorToEVMStore := prefix.NewStore(store, bridgetypes.OperatorToEVMAddressMapKey)
	operatorIter := operatorToEVMStore.Iterator(nil, nil)
	defer operatorIter.Close()

	// get the EVM to operator address map store for writing
	evmToOperatorStore := prefix.NewStore(store, bridgetypes.EVMToOperatorAddressMapKey)

	for ; operatorIter.Valid(); operatorIter.Next() {
		// decode the operator address from the key
		operatorAddr := string(operatorIter.Key())

		// decode the EVM address from the value
		var evmAddr bridgetypes.EVMAddress
		if err := cdc.Unmarshal(operatorIter.Value(), &evmAddr); err != nil {
			// log error but continue migration for other entries
			continue
		}

		// convert EVM address bytes to hex string (this is how it's stored as key in EVMToOperatorAddressMap)
		evmAddressString := hex.EncodeToString(evmAddr.EVMAddress)

		// check if reverse mapping already exists
		evmKey := []byte(evmAddressString)
		if evmToOperatorStore.Has(evmKey) {
			// mapping already exists, skip
			continue
		}

		// create the operator address type
		sdkValAddr, err := sdk.ValAddressFromBech32(operatorAddr)
		if err != nil {
			continue
		}

		operatorAddrType := bridgetypes.OperatorAddress{
			OperatorAddress: sdkValAddr,
		}

		operatorData, err := cdc.Marshal(&operatorAddrType)
		if err != nil {
			continue
		}

		// set the reverse mapping
		evmToOperatorStore.Set(evmKey, operatorData)
	}

	return nil
}
