package v4

import (
	"context"

	"github.com/gogo/protobuf/proto"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/core/store"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
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

// MigrateStore migrates the ValidatorCheckpointParams from the legacy format to the new format
// by adding the BlockHeight field with a default value of 0
func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))

	// Migrate ValidatorCheckpointParams
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
