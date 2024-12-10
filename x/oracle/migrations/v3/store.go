package v3

import (
	"context"

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
	Index uint64 `protobuf:"varint,8,opt,name=index,proto3" json:"index,omitempty"`
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

func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec,
	aggregateMap *collections.IndexedMap[collections.Pair[[]byte, uint64], types.Aggregate, types.AggregatesIndex],
	queryMap *collections.IndexedMap[collections.Pair[[]byte, uint64], types.QueryMeta, types.QueryMetaIndex],
	reportsMap *collections.IndexedMap[collections.Triple[[]byte, []byte, uint64], types.MicroReport, types.ReportsIndex],
	medianReport, modeReport func(ctx context.Context, id uint64, report types.MicroReport) error,
) error {
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	aggStore := prefix.NewStore(store, types.AggregatesPrefix)
	iter := aggStore.Iterator(nil, nil)

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
		kcdc := collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)
		_, key, err := kcdc.Decode(iter.Key())
		if err != nil {
			panic("failed to decode key")
		}
		if err := aggregateMap.Set(ctx, key, newValue); err != nil {
			panic("failed to set aggregate value")
		}
	}
	sb := collections.NewSchemaBuilder(storeService)
	reportsIndex := indexes.NewMulti(
		sb, types.ReportsHeightIndexPrefix, "reports_by_id",
		collections.Uint64Key, collections.TripleKeyCodec[[]byte, []byte, uint64](collections.BytesKey, collections.BytesKey, collections.Uint64Key),
		func(k collections.Triple[[]byte, []byte, uint64], _ types.MicroReport) (uint64, error) {
			return k.K3(), nil
		},
	)

	err := queryMap.Walk(ctx, nil, func(key collections.Pair[[]byte, uint64], value types.QueryMeta) (stop bool, err error) {
		repIter, err := reportsIndex.MatchExact(ctx, key.K2())
		if err != nil {
			panic("failed to fetch reports")
		}
		defer repIter.Close()
		for ; repIter.Valid(); repIter.Next() {
			repkey, err := repIter.PrimaryKey()
			if err != nil {
				panic("failed to get primary reports key")
			}
			microReport, err := reportsMap.Get(ctx, repkey)
			if err != nil {
				panic("failed to get primary reports key")
			}
			if microReport.AggregateMethod == "weighted-median" {
				err = medianReport(ctx, key.K2(), microReport)
			} else {
				err = modeReport(ctx, key.K2(), microReport)
			}
			if err != nil {
				panic("failed to set micro report")
			}
		}
		return false, nil
	})
	if err != nil {
		panic("failed to walk through queries successfully")
	}
	// handle reports that haven't aggregated yet
	// see the queries that exist and then addReport or modeReport
	return nil
}
