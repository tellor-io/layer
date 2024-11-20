package types

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
)

// type TipsIndex struct {
// 	Tipper *indexes.Multi[[]byte, collections.Pair[[]byte, []byte], math.Int]
// }

// func (a TipsIndex) IndexesList() []collections.Index[collections.Pair[[]byte, []byte], math.Int] {
// 	return []collections.Index[collections.Pair[[]byte, []byte], math.Int]{a.Tipper}
// }

// func NewTipsIndex(sb *collections.SchemaBuilder) TipsIndex {
// 	return TipsIndex{
// 		Tipper: indexes.NewMulti(
// 			sb, TipsIndexPrefix, "tips_by_tipper",
// 			collections.BytesKey, collections.PairKeyCodec(collections.BytesKey, collections.BytesKey),
// 			func(k collections.Pair[[]byte, []byte], _ math.Int) ([]byte, error) {
// 				return k.K2(), nil
// 			},
// 		),
// 	}
// }

type AggregatesIndex struct {
	BlockHeight *indexes.Multi[uint64, collections.Pair[[]byte, uint64], Aggregate]
	MicroHeight *indexes.Multi[uint64, collections.Pair[[]byte, uint64], Aggregate]
}

func (a AggregatesIndex) IndexesList() []collections.Index[collections.Pair[[]byte, uint64], Aggregate] {
	return []collections.Index[collections.Pair[[]byte, uint64], Aggregate]{
		a.BlockHeight, a.MicroHeight,
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
		MicroHeight: indexes.NewMulti(
			sb, AggregatesMicroHeightIndexPrefix, "aggregates_by_micro_height",
			collections.Uint64Key, collections.PairKeyCodec[[]byte, uint64](collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v Aggregate) (uint64, error) {
				return v.MicroHeight, nil
			},
		),
	}
}

type ReportsIndex struct {
	Id       *indexes.Multi[uint64, collections.Triple[[]byte, []byte, uint64], MicroReport]
	Reporter *indexes.Multi[[]byte, collections.Triple[[]byte, []byte, uint64], MicroReport]
}

func (a ReportsIndex) IndexesList() []collections.Index[collections.Triple[[]byte, []byte, uint64], MicroReport] {
	return []collections.Index[collections.Triple[[]byte, []byte, uint64], MicroReport]{
		a.Id,
		a.Reporter,
	}
}

func NewReportsIndex(sb *collections.SchemaBuilder) ReportsIndex {
	return ReportsIndex{
		Id: indexes.NewMulti(
			sb, ReportsHeightIndexPrefix, "reports_by_id",
			collections.Uint64Key, collections.TripleKeyCodec[[]byte, []byte, uint64](collections.BytesKey, collections.BytesKey, collections.Uint64Key),
			func(k collections.Triple[[]byte, []byte, uint64], _ MicroReport) (uint64, error) {
				return k.K3(), nil
			},
		),
		Reporter: indexes.NewMulti(
			sb, ReportsReporterIndexPrefix, "reports_by_reporter",
			collections.BytesKey, collections.TripleKeyCodec[[]byte, []byte, uint64](collections.BytesKey, collections.BytesKey, collections.Uint64Key),
			func(k collections.Triple[[]byte, []byte, uint64], _ MicroReport) ([]byte, error) {
				return k.K2(), nil
			},
		),
	}
}

type QueryMetaIndex struct {
	HasReveals *indexes.Multi[bool, collections.Pair[[]byte, uint64], QueryMeta]
	QueryType  *indexes.Multi[string, collections.Pair[[]byte, uint64], QueryMeta]
}

func (a QueryMetaIndex) IndexesList() []collections.Index[collections.Pair[[]byte, uint64], QueryMeta] {
	return []collections.Index[collections.Pair[[]byte, uint64], QueryMeta]{a.HasReveals, a.QueryType}
}

func NewQueryIndex(sb *collections.SchemaBuilder) QueryMetaIndex {
	return QueryMetaIndex{
		HasReveals: indexes.NewMulti(
			sb, QueryRevealedIdsIndexPrefix, "query_by_revealed",
			collections.BoolKey, collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v QueryMeta) (bool, error) {
				return v.HasRevealedReports, nil
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

// type TipperTotalIndex struct {
// 	BlockNumber *indexes.Unique[uint64, collections.Pair[[]byte, uint64], math.Int]
// }

// func (a TipperTotalIndex) IndexesList() []collections.Index[collections.Pair[[]byte, uint64], math.Int] {
// 	return []collections.Index[collections.Pair[[]byte, uint64], math.Int]{a.BlockNumber}
// }

// func NewTippersIndex(sb *collections.SchemaBuilder) TipperTotalIndex {
// 	return TipperTotalIndex{
// 		BlockNumber: indexes.NewUnique(
// 			sb, TipsBlockIndexPrefix, "tips_by_block",
// 			collections.Uint64Key, collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
// 			func(k collections.Pair[[]byte, uint64], v math.Int) (uint64, error) {
// 				return k.K2(), nil
// 			},
// 		),
// 	}
// }
