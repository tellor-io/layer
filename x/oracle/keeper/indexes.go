package keeper

import (
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/math"
)

type tipsIndex struct {
	Tipper *indexes.Multi[[]byte, collections.Pair[[]byte, []byte], math.Int]
}

func (a tipsIndex) IndexesList() []collections.Index[collections.Pair[[]byte, []byte], math.Int] {
	return []collections.Index[collections.Pair[[]byte, []byte], math.Int]{a.Tipper}
}

func NewTipsIndex(sb *collections.SchemaBuilder) tipsIndex {
	return tipsIndex{
		Tipper: indexes.NewMulti(
			sb, types.TipsIndexPrefix, "tips_by_tipper",
			collections.BytesKey, collections.PairKeyCodec(collections.BytesKey, collections.BytesKey),
			func(k collections.Pair[[]byte, []byte], _ math.Int) ([]byte, error) {
				return k.K2(), nil
			},
		),
	}
}

type aggregatesIndex struct {
	BlockHeight *indexes.Multi[int64, collections.Pair[[]byte, int64], types.Aggregate]
	MicroHeight *indexes.Multi[int64, collections.Pair[[]byte, int64], types.Aggregate]
}

func (a aggregatesIndex) IndexesList() []collections.Index[collections.Pair[[]byte, int64], types.Aggregate] {
	return []collections.Index[collections.Pair[[]byte, int64], types.Aggregate]{
		a.BlockHeight, a.MicroHeight,
	}
}

func NewAggregatesIndex(sb *collections.SchemaBuilder) aggregatesIndex {
	return aggregatesIndex{
		BlockHeight: indexes.NewMulti(
			sb, types.AggregatesHeightIndexPrefix, "aggregates_by_height",
			collections.Int64Key, collections.PairKeyCodec[[]byte, int64](collections.BytesKey, collections.Int64Key),
			func(_ collections.Pair[[]byte, int64], v types.Aggregate) (int64, error) {
				return v.Height, nil
			},
		),
		MicroHeight: indexes.NewMulti(
			sb, types.AggregatesMicroHeightIndexPrefix, "aggregates_by_micro_height",
			collections.Int64Key, collections.PairKeyCodec[[]byte, int64](collections.BytesKey, collections.Int64Key),
			func(_ collections.Pair[[]byte, int64], v types.Aggregate) (int64, error) {
				return v.MicroHeight, nil
			},
		),
	}
}

type reportsIndex struct {
	Id       *indexes.Multi[uint64, collections.Triple[[]byte, []byte, uint64], types.MicroReport]
	Reporter *indexes.Multi[[]byte, collections.Triple[[]byte, []byte, uint64], types.MicroReport]
}

func (a reportsIndex) IndexesList() []collections.Index[collections.Triple[[]byte, []byte, uint64], types.MicroReport] {
	return []collections.Index[collections.Triple[[]byte, []byte, uint64], types.MicroReport]{
		a.Id,
		a.Reporter,
	}
}

func NewReportsIndex(sb *collections.SchemaBuilder) reportsIndex {
	return reportsIndex{
		Id: indexes.NewMulti(
			sb, types.ReportsHeightIndexPrefix, "reports_by_id",
			collections.Uint64Key, collections.TripleKeyCodec[[]byte, []byte, uint64](collections.BytesKey, collections.BytesKey, collections.Uint64Key),
			func(k collections.Triple[[]byte, []byte, uint64], _ types.MicroReport) (uint64, error) {
				return k.K3(), nil
			},
		),
		Reporter: indexes.NewMulti(
			sb, types.ReportsReporterIndexPrefix, "reports_by_reporter",
			collections.BytesKey, collections.TripleKeyCodec[[]byte, []byte, uint64](collections.BytesKey, collections.BytesKey, collections.Uint64Key),
			func(k collections.Triple[[]byte, []byte, uint64], _ types.MicroReport) ([]byte, error) {
				return k.K2(), nil
			},
		),
	}
}

type queryMetaIndex struct {
	HasReveals *indexes.Multi[bool, collections.Pair[[]byte, uint64], types.QueryMeta]
	QueryType  *indexes.Multi[string, collections.Pair[[]byte, uint64], types.QueryMeta]
}

func (a queryMetaIndex) IndexesList() []collections.Index[collections.Pair[[]byte, uint64], types.QueryMeta] {
	return []collections.Index[collections.Pair[[]byte, uint64], types.QueryMeta]{a.HasReveals, a.QueryType}
}

func NewQueryIndex(sb *collections.SchemaBuilder) queryMetaIndex {
	return queryMetaIndex{
		HasReveals: indexes.NewMulti(
			sb, types.QueryRevealedIdsIndexPrefix, "query_by_revealed",
			collections.BoolKey, collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v types.QueryMeta) (bool, error) {
				return v.HasRevealedReports, nil
			},
		),
		QueryType: indexes.NewMulti(
			sb, types.QueryTypeIndexPrefix, "query_by_type",
			collections.StringKey, collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v types.QueryMeta) (string, error) {
				return v.QueryType, nil
			},
		),
	}
}

type tipperTotalIndex struct {
	BlockNumber *indexes.Unique[int64, collections.Pair[[]byte, int64], math.Int]
}

func (a tipperTotalIndex) IndexesList() []collections.Index[collections.Pair[[]byte, int64], math.Int] {
	return []collections.Index[collections.Pair[[]byte, int64], math.Int]{a.BlockNumber}
}

func NewTippersIndex(sb *collections.SchemaBuilder) tipperTotalIndex {
	return tipperTotalIndex{
		BlockNumber: indexes.NewUnique(
			sb, types.TipsBlockIndexPrefix, "tips_by_block",
			collections.Int64Key, collections.PairKeyCodec(collections.BytesKey, collections.Int64Key),
			func(k collections.Pair[[]byte, int64], v math.Int) (int64, error) {
				return k.K2(), nil
			},
		),
	}
}
