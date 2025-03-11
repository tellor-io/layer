package v3_0_4

import (
	"context"

	"github.com/gogo/protobuf/proto"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/core/store"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
)

type AttestationSnapshotDataLegacy struct {
	ValidatorCheckpoint  []byte `protobuf:"bytes,1,rep,name=validator_checkpoint,proto3"`
	AttestationTimestamp uint64 `protobuf:"varint,2,rep,name=attestation_timestamp,proto3"`
	PrevReportTimestamp  uint64 `protobuf:"varint,3,rep,name=prev_report_timestamp,proto3"`
	NextReportTimestamp  uint64 `protobuf:"varint,4,rep,name=next_report_timestamp,proto3"`
	QueryId              []byte `protobuf:"bytes,5,rep,name=query_id,proto3"`
	Timestamp            uint64 `protobuf:"varint,6,rep,name=timestamp,proto3"`
}

var _ proto.Message = &AttestationSnapshotDataLegacy{}

func (*AttestationSnapshotDataLegacy) Reset() {}
func (m *AttestationSnapshotDataLegacy) String() string {
	return proto.CompactTextString(m)
}
func (*AttestationSnapshotDataLegacy) ProtoMessage() {}

func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	attestStore := prefix.NewStore(store, bridgetypes.AttestSnapshotDataMapKey)
	iter := attestStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var att AttestationSnapshotDataLegacy
		if err := cdc.Unmarshal(iter.Value(), &att); err != nil {
			panic("failed to unmarshal value")
		}

		newValue := bridgetypes.AttestationSnapshotData{
			ValidatorCheckpoint:    att.ValidatorCheckpoint,
			AttestationTimestamp:   att.AttestationTimestamp,
			PrevReportTimestamp:    att.PrevReportTimestamp,
			NextReportTimestamp:    att.NextReportTimestamp,
			QueryId:                att.QueryId,
			Timestamp:              att.Timestamp,
			LastConsensusTimestamp: 0,
		}
		newData, err := cdc.Marshal(&newValue)
		if err != nil {
			panic("unable to marshal new value")
		}

		attestStore.Set(iter.Key(), newData)
	}

	return nil
}
