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
	var itererror error
	err := oldquery.Walk(ctx, nil, func(key []byte, value types.QueryMeta) (bool, error) {
		if value.Expiration.Add(time.Second * 3).Before(blockTime) {
			itererror = oldquery.Remove(ctx, key)
			if itererror != nil {
				return true, itererror
			}
		} else {
			itererror = newquery.Set(ctx, collections.Join(key, value.Id), value)
			if itererror != nil {
				return true, itererror
			}
		}
		return false, nil
	})
	if itererror != nil {
		return itererror
	}
	return err
}
