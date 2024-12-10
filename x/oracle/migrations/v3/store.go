package v3

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
)

func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec,
	queryMap *collections.IndexedMap[collections.Pair[[]byte, uint64], types.QueryMeta, types.QueryMetaIndex],
	reportsMap *collections.IndexedMap[collections.Triple[[]byte, []byte, uint64], types.MicroReport, types.ReportsIndex],
	medianReport, modeReport func(ctx context.Context, id uint64, report types.MicroReport) error,
) error {
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

	return nil
}
