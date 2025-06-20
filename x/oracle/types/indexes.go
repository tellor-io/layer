package types

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AggregatesIndex struct {
	BlockHeight *indexes.Multi[uint64, collections.Pair[[]byte, uint64], Aggregate]
	Reporter    *indexes.Multi[collections.Triple[[]byte, []byte, uint64], collections.Pair[[]byte, uint64], Aggregate]
}

func (a AggregatesIndex) IndexesList() []collections.Index[collections.Pair[[]byte, uint64], Aggregate] {
	return []collections.Index[collections.Pair[[]byte, uint64], Aggregate]{
		a.BlockHeight,
		a.Reporter,
	}
}

func NewAggregatesIndex(sb *collections.SchemaBuilder) AggregatesIndex {
	return AggregatesIndex{
		BlockHeight: indexes.NewMulti(
			sb, AggregatesHeightIndexPrefix, "aggregates_by_height",
			collections.Uint64Key, collections.PairKeyCodec[[]byte, uint64](collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v Aggregate) (uint64, error) {
				return v.Height, nil
			},
		),
		Reporter: indexes.NewMulti(
			sb, AggregatesMicroHeightIndexPrefix, "aggregates_by_micro_height",
			collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.Uint64Key), collections.PairKeyCodec[[]byte, uint64](collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v Aggregate) (collections.Triple[[]byte, []byte, uint64], error) {
				reporter := sdk.MustAccAddressFromBech32(v.AggregateReporter)
				return collections.Join3(reporter.Bytes(), v.QueryId, v.MicroHeight), nil
			},
		),
	}
}

type ReportsIndex struct {
	Reporter  *indexes.Multi[[]byte, collections.Triple[[]byte, []byte, uint64], MicroReport]
	IdQueryId *indexes.Multi[collections.Pair[uint64, []byte], collections.Triple[[]byte, []byte, uint64], MicroReport]
}

func (a ReportsIndex) IndexesList() []collections.Index[collections.Triple[[]byte, []byte, uint64], MicroReport] {
	return []collections.Index[collections.Triple[[]byte, []byte, uint64], MicroReport]{
		a.Reporter,
		a.IdQueryId,
	}
}

func NewReportsIndex(sb *collections.SchemaBuilder) ReportsIndex {
	return ReportsIndex{
		Reporter: indexes.NewMulti(
			sb, ReportsReporterIndexPrefix, "reports_by_reporter",
			collections.BytesKey, collections.TripleKeyCodec[[]byte, []byte, uint64](collections.BytesKey, collections.BytesKey, collections.Uint64Key),
			func(k collections.Triple[[]byte, []byte, uint64], _ MicroReport) ([]byte, error) {
				size := collections.Uint64Key.Size(k.K3())
				buffer := make([]byte, size)
				_, err := collections.Uint64Key.Encode(buffer, k.K3())
				if err != nil {
					return nil, err
				}
				buffer = append(k.K2(), buffer...)
				return buffer, nil
			},
		),
		IdQueryId: indexes.NewMulti(
			sb, collections.NewPrefix("reporter"), "reporters",
			collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey), collections.TripleKeyCodec[[]byte, []byte, uint64](collections.BytesKey, collections.BytesKey, collections.Uint64Key),
			func(k collections.Triple[[]byte, []byte, uint64], _ MicroReport) (collections.Pair[uint64, []byte], error) {
				return collections.Join(k.K3(), k.K1()), nil
			},
		),
	}
}

type QueryMetaIndex struct {
	Expiration *indexes.Multi[collections.Pair[bool, uint64], collections.Pair[[]byte, uint64], QueryMeta]
	QueryType  *indexes.Multi[string, collections.Pair[[]byte, uint64], QueryMeta]
}

func (a QueryMetaIndex) IndexesList() []collections.Index[collections.Pair[[]byte, uint64], QueryMeta] {
	return []collections.Index[collections.Pair[[]byte, uint64], QueryMeta]{a.Expiration, a.QueryType}
}

func NewQueryIndex(sb *collections.SchemaBuilder) QueryMetaIndex {
	return QueryMetaIndex{
		Expiration: indexes.NewMulti(
			sb, QueryByExpirationPrefix, "query_by_expiration",
			collections.PairKeyCodec(collections.BoolKey, collections.Uint64Key), collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v QueryMeta) (collections.Pair[bool, uint64], error) {
				return collections.Join(v.HasRevealedReports, v.Expiration), nil
			},
		),
		QueryType: indexes.NewMulti(
			sb, QueryTypeIndexPrefix, "query_by_type",
			collections.StringKey, collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v QueryMeta) (string, error) {
				return v.QueryType, nil
			},
		),
	}
}

type ValuesIndex struct {
	// todo: what do you do when two powers have the same power?
	Power *indexes.Multi[collections.Pair[uint64, uint64], collections.Pair[uint64, string], Value]
}

func (a ValuesIndex) IndexesList() []collections.Index[collections.Pair[uint64, string], Value] {
	return []collections.Index[collections.Pair[uint64, string], Value]{
		a.Power,
	}
}

func NewValuesIndex(sb *collections.SchemaBuilder) ValuesIndex {
	return ValuesIndex{
		Power: indexes.NewMulti(
			sb, ValuesPowerPrefix, "values_by_power",
			collections.PairKeyCodec(collections.Uint64Key, collections.Uint64Key), collections.PairKeyCodec(collections.Uint64Key, collections.StringKey),
			func(k collections.Pair[uint64, string], v Value) (collections.Pair[uint64, uint64], error) {
				return collections.Join(k.K1(), v.CrossoverWeight), nil
			},
		),
	}
}

type ReporterIndex struct {
	Reporter *indexes.Multi[[]byte, collections.Pair[[]byte, uint64], NoStakeMicroReport]
}

func (a ReporterIndex) IndexesList() []collections.Index[collections.Pair[[]byte, uint64], NoStakeMicroReport] {
	return []collections.Index[collections.Pair[[]byte, uint64], NoStakeMicroReport]{a.Reporter}
}

// maps the reporter address and timestamp to the no stake report
func NewReporterIndex(sb *collections.SchemaBuilder) ReporterIndex {
	return ReporterIndex{
		Reporter: indexes.NewMulti(
			sb, ReporterIndexPrefix, "reporter_index",
			collections.BytesKey,
			collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			func(k collections.Pair[[]byte, uint64], report NoStakeMicroReport) ([]byte, error) {
				size := collections.Uint64Key.Size(k.K2())
				buffer := make([]byte, size)
				_, err := collections.Uint64Key.Encode(buffer, k.K2())
				if err != nil {
					return nil, err
				}
				buffer = append(report.Reporter, buffer...)
				return buffer, nil
			},
		),
	}
}

// ReporterRange implements Ranger[collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]] for efficient reporter queries
type ReporterRange struct {
	reporterAddr []byte
	order        collections.Order
	startKey     *collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]
}

// NewReporterRange creates a new ReporterRange for the given reporter address
func NewReporterRange(reporterAddr []byte) *ReporterRange {
	return &ReporterRange{
		reporterAddr: reporterAddr,
		order:        collections.OrderAscending,
	}
}

// Descending sets the range to iterate in descending order
func (r *ReporterRange) Descending() *ReporterRange {
	r.order = collections.OrderDescending
	return r
}

// StartInclusive sets the starting key for pagination
func (r *ReporterRange) StartInclusive(key collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]) *ReporterRange {
	r.startKey = &key
	return r
}

// RangeValues implements the Ranger interface
func (r *ReporterRange) RangeValues() (start, end *collections.RangeKey[collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]], order collections.Order, err error) {
	// The Reporter index key is: reporter_address + encoded_metaId
	// We want to create a range that covers all entries for this reporter

	// Create start bound: reporter_address + 0 (minimum metaId)
	startBuffer := make([]byte, 8)
	_, err = collections.Uint64Key.Encode(startBuffer, 0)
	if err != nil {
		return nil, nil, 0, err
	}
	startIndexKey := append(r.reporterAddr, startBuffer...)

	// Create end bound: reporter_address + maxUint64 (maximum metaId)
	endBuffer := make([]byte, 8)
	_, err = collections.Uint64Key.Encode(endBuffer, ^uint64(0))
	if err != nil {
		return nil, nil, 0, err
	}
	endIndexKey := append(r.reporterAddr, endBuffer...)

	// For the Pair type, we need to construct pairs with the index key and a dummy primary key
	// The actual primary key will be determined by the iteration
	dummyPrimaryKey := collections.Join3([]byte{}, []byte{}, uint64(0))

	var startPair collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]
	var endPair collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]

	if r.startKey != nil {
		// Use the provided start key for pagination
		startPair = *r.startKey
	} else {
		// Start from the beginning of this reporter's data
		startPair = collections.Join(startIndexKey, dummyPrimaryKey)
	}

	endPair = collections.Join(endIndexKey, dummyPrimaryKey)

	start = collections.RangeKeyExact(startPair)
	end = collections.RangeKeyNext(endPair) // Next to make it inclusive

	return start, end, r.order, nil
}
