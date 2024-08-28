package migrations

import (
	"context"
	"time"

	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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

func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec, newquery *collections.IndexedMap[collections.Pair[[]byte, uint64], types.QueryMeta, types.QueryMetaIndex]) error {
	sb := collections.NewSchemaBuilder(storeService)
	oldquery := collections.NewIndexedMap(sb,
		types.QueryTipPrefix,
		"query",
		collections.BytesKey,
		codec.CollValue[types.QueryMeta](cdc),
		NewQueryIndex(sb),
	)

	blockTime := sdk.UnwrapSDKContext(ctx).BlockTime()

	type query struct {
		key   collections.Pair[[]byte, uint64]
		value types.QueryMeta
	}

	newqueries := make([]query, 0)
	oldqueries := make([]query, 0)

	err := oldquery.Walk(ctx, nil, func(key []byte, value types.QueryMeta) (bool, error) {
		oldqueries = append(oldqueries, query{key: collections.Join(key, value.Id), value: value})
		if !value.Expiration.Add(time.Second * 3).Before(blockTime) {
			newqueries = append(newqueries, query{key: collections.Join(key, value.Id), value: value})
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	for _, q := range oldqueries {
		if err := oldquery.Remove(ctx, q.key.K1()); err != nil {
			return err
		}
	}
	for _, q := range newqueries {
		if err := newquery.Set(ctx, q.key, q.value); err != nil {
			return err
		}
	}
	return nil
}
