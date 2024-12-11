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
				return k.K2(), nil
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
