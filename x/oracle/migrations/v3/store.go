package v3

import (
	"context"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
)

type AggregateLegacy struct {
	// query_id is the id of the query
	QueryId []byte `protobuf:"bytes,1,opt,name=query_id,json=queryId,proto3" json:"query_id,omitempty"`
	// aggregate_value is the value of the aggregate
	AggregateValue string `protobuf:"bytes,2,opt,name=aggregate_value,json=aggregateValue,proto3" json:"aggregate_value,omitempty"`
	// aggregate_reporter is the address of the reporter
	AggregateReporter string `protobuf:"bytes,3,opt,name=aggregate_reporter,json=aggregateReporter,proto3" json:"aggregate_reporter,omitempty"`
	// reporter_power is the power of the reporter
	ReporterPower uint64 `protobuf:"varint,4,opt,name=reporter_power,json=reporterPower,proto3" json:"reporter_power,omitempty"`
	// list of reporters that were included in the aggregate
	Reporters []*types.AggregateReporter `protobuf:"bytes,6,rep,name=reporters,proto3" json:"reporters,omitempty"`
	// flagged is true if the aggregate was flagged by a dispute
	Flagged bool `protobuf:"varint,7,opt,name=flagged,proto3" json:"flagged,omitempty"`
	// index is the index of the aggregate
	Index uint64 `protobuf:"varint,6,opt,name=index,proto3" json:"index,omitempty"`
	// aggregate_report_index is the index of the aggregate report in the micro reports
	AggregateReportIndex uint64 `protobuf:"varint,9,opt,name=aggregate_report_index,json=aggregateReportIndex,proto3" json:"aggregate_report_index,omitempty"`
	// height of the aggregate report
	Height uint64 `protobuf:"varint,10,opt,name=height,proto3" json:"height,omitempty"`
	// height of the micro report
	MicroHeight uint64 `protobuf:"varint,11,opt,name=micro_height,json=microHeight,proto3" json:"micro_height,omitempty"`
	// meta_id is the id of the querymeta iterator
	MetaId uint64 `protobuf:"varint,12,opt,name=meta_id,json=metaId,proto3" json:"meta_id,omitempty"`
}

var _ proto.Message = &AggregateLegacy{}

func (*AggregateLegacy) Reset() {}
func (m *AggregateLegacy) String() string {
	return proto.CompactTextString(m)
}
func (*AggregateLegacy) ProtoMessage() {}

type AggregatesIndex struct {
	BlockHeight *indexes.Multi[uint64, collections.Pair[[]byte, uint64], AggregateLegacy]
	MicroHeight *indexes.Multi[uint64, collections.Pair[[]byte, uint64], AggregateLegacy]
}

func (a AggregatesIndex) IndexesList() []collections.Index[collections.Pair[[]byte, uint64], AggregateLegacy] {
	return []collections.Index[collections.Pair[[]byte, uint64], AggregateLegacy]{
		a.BlockHeight, a.MicroHeight,
	}
}

func NewAggregatesIndex(sb *collections.SchemaBuilder) AggregatesIndex {
	return AggregatesIndex{
		BlockHeight: indexes.NewMulti(
			sb, types.AggregatesHeightIndexPrefix, "aggregates_by_height",
			collections.Uint64Key, collections.PairKeyCodec[[]byte, uint64](collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v AggregateLegacy) (uint64, error) {
				return v.Height, nil
			},
		),
		MicroHeight: indexes.NewMulti(
			sb, types.AggregatesMicroHeightIndexPrefix, "aggregates_by_micro_height",
			collections.Uint64Key, collections.PairKeyCodec[[]byte, uint64](collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v AggregateLegacy) (uint64, error) {
				return v.MicroHeight, nil
			},
		),
	}
}

func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec,
	aggregateMap *collections.IndexedMap[collections.Pair[[]byte, uint64], types.Aggregate, types.AggregatesIndex],
	queryMap *collections.IndexedMap[collections.Pair[[]byte, uint64], types.QueryMeta, types.QueryMetaIndex],
	reportsMap *collections.IndexedMap[collections.Triple[[]byte, []byte, uint64], types.MicroReport, types.ReportsIndex],
	medianReport, modeReport func(ctx context.Context, id uint64, report types.MicroReport) error,
) error {
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	aggStore := prefix.NewStore(store, types.AggregatesPrefix)
	iter := aggStore.Iterator(nil, nil)
	sb := collections.NewSchemaBuilder(storeService)
	aggIndexMap := collections.NewIndexedMap(sb,
		types.AggregatesPrefix,
		"aggregates",
		collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
		codec.CollValue[AggregateLegacy](cdc),
		NewAggregatesIndex(sb),
	)

	allkeys := make([][]byte, 0)
	newValues := make([]types.Aggregate, 0)

	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var agg AggregateLegacy
		if err := cdc.Unmarshal(iter.Value(), &agg); err != nil {
			panic("failed to unmarshall value")
		}

		newValue := types.Aggregate{
			QueryId:           agg.QueryId,
			AggregateValue:    agg.AggregateValue,
			AggregateReporter: agg.AggregateReporter,
			AggregatePower:    agg.ReporterPower,
			Flagged:           agg.Flagged,
			Index:             agg.Index,
			Height:            agg.Height,
			MicroHeight:       agg.MicroHeight,
			MetaId:            agg.MetaId,
		}
		allkeys = append(allkeys, iter.Key())
		newValues = append(newValues, newValue)
	}

	for i, k := range allkeys {
		kcdc := collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)
		_, key, err := kcdc.Decode(k)
		if err != nil {
			panic("failed to decode key")
		}
		err = aggIndexMap.Remove(ctx, key)
		if err != nil {
			panic(fmt.Sprintf("failed to remove aggregate value %v", err))
		}
		err = aggregateMap.Set(ctx, key, newValues[i])
		if err != nil {
			panic(fmt.Sprintf("failed to set aggregate value %v", err))
		}
	}

	err := reportsMap.Walk(ctx, nil, func(key collections.Triple[[]byte, []byte, uint64], value types.MicroReport) (stop bool, err error) {
		err = reportsMap.Set(ctx, key, value)
		if err != nil {
			panic(fmt.Sprintf("failed to set reports %v", err))
		}
		// get query
		_, err = queryMap.Get(ctx, collections.Join(key.K1(), key.K3()))
		if err != nil && errors.Is(err, collections.ErrNotFound) {
			return false, nil
		}
		if err != nil {
			panic(fmt.Sprintf("failed to get query meta %v", err))
		}
		if value.AggregateMethod == "weighted-median" {
			err = medianReport(ctx, key.K3(), value)
		} else {
			err = modeReport(ctx, key.K3(), value)
		}
		if err != nil {
			panic("failed to set micro report")
		}
		return false, nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to walk through reports successfully %v", err))
	}
	err = queryMap.Walk(ctx, nil, func(key collections.Pair[[]byte, uint64], value types.QueryMeta) (stop bool, err error) {
		err = queryMap.Set(ctx, key, value)
		if err != nil {
			panic(fmt.Sprintf("failed to set query %v", err))
		}
		return false, nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to walk through queries successfully %v", err))
	}

	return nil
}
