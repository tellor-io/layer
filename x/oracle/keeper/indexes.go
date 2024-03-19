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
	HasReveals *indexes.Multi[bool, []byte, types.QueryMeta]
	QueryType  *indexes.Multi[string, []byte, types.QueryMeta]
}

func (a queryMetaIndex) IndexesList() []collections.Index[[]byte, types.QueryMeta] {
	return []collections.Index[[]byte, types.QueryMeta]{a.HasReveals, a.QueryType}
}

func NewQueryIndex(sb *collections.SchemaBuilder) queryMetaIndex {
	return queryMetaIndex{
		HasReveals: indexes.NewMulti(
			sb, types.QueryRevealedIdsIndexPrefix, "query_by_revealed",
			collections.BoolKey, collections.BytesKey,
			func(_ []byte, v types.QueryMeta) (bool, error) {
				return v.HasRevealedReports, nil
			},
		),
		QueryType: indexes.NewMulti(
			sb, types.QueryTypeIndexPrefix, "query_by_type",
			collections.StringKey, collections.BytesKey,
			func(_ []byte, v types.QueryMeta) (string, error) {
				return v.QueryType, nil
			},
		),
	}
}
