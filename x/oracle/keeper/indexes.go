package keeper

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/math"
	"github.com/tellor-io/layer/x/oracle/types"
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
			collections.BytesKey, collections.PairKeyCodec[[]byte, []byte](collections.BytesKey, collections.BytesKey),
			func(k collections.Pair[[]byte, []byte], _ math.Int) ([]byte, error) {
				return k.K2(), nil
			},
		),
	}
}

type aggregatesIndex struct {
	BlockHeight *indexes.Multi[int64, collections.Pair[[]byte, int64], types.Aggregate]
}

func (a aggregatesIndex) IndexesList() []collections.Index[collections.Pair[[]byte, int64], types.Aggregate] {
	return []collections.Index[collections.Pair[[]byte, int64], types.Aggregate]{
		a.BlockHeight,
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
	}
}

type reportsIndex struct {
	BlockHeight *indexes.Multi[int64, collections.Triple[[]byte, []byte, int64], types.MicroReport]
	Reporter    *indexes.Multi[[]byte, collections.Triple[[]byte, []byte, int64], types.MicroReport]
}

func (a reportsIndex) IndexesList() []collections.Index[collections.Triple[[]byte, []byte, int64], types.MicroReport] {
	return []collections.Index[collections.Triple[[]byte, []byte, int64], types.MicroReport]{
		a.BlockHeight,
		a.Reporter,
	}
}

func NewReportsIndex(sb *collections.SchemaBuilder) reportsIndex {
	return reportsIndex{
		BlockHeight: indexes.NewMulti(
			sb, types.ReportsHeightIndexPrefix, "reports_by_height",
			collections.Int64Key, collections.TripleKeyCodec[[]byte, []byte, int64](collections.BytesKey, collections.BytesKey, collections.Int64Key),
			func(k collections.Triple[[]byte, []byte, int64], _ types.MicroReport) (int64, error) {
				return k.K3(), nil
			},
		),
		Reporter: indexes.NewMulti(
			sb, types.ReportsReporterIndexPrefix, "reports_by_reporter",
			collections.BytesKey, collections.TripleKeyCodec[[]byte, []byte, int64](collections.BytesKey, collections.BytesKey, collections.Int64Key),
			func(k collections.Triple[[]byte, []byte, int64], _ types.MicroReport) ([]byte, error) {
				return k.K2(), nil
			},
		),
	}
}
