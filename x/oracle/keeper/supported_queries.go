package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) SetSupportedQueries(ctx sdk.Context, queries []types.QueryChange) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.SupportedQueriesKey())
	var list = make([]*types.QueryChange, len(queries))
	for i := range queries {
		list[i] = &queries[i]
	}

	bz := k.cdc.MustMarshal(&types.SupportedQueries{QueryData: list})
	store.Set(types.SupportedQueriesKey(), bz)
}

func (k Keeper) GetSupportedQueries(ctx sdk.Context) types.SupportedQueries {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.SupportedQueriesKey())
	bz := store.Get(types.SupportedQueriesKey())
	var d types.SupportedQueries
	k.cdc.MustUnmarshal(bz, &d)
	return d
}

// rotation what query is next
func (k Keeper) RotateQueries(ctx sdk.Context) string {
	queries := k.GetSupportedQueries(ctx)

	currentIndex := k.GetCurrentIndex(ctx)
	if currentIndex >= int64(len(queries.QueryData)) {
		currentIndex = 0
	}

	k.SetCurrentIndex(ctx, (currentIndex+1)%int64(len(queries.QueryData)))
	return queries.QueryData[currentIndex].QueryData
}

func (k Keeper) SetCurrentIndex(ctx sdk.Context, index int64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.SupportedQueriesKey())
	store.Set(types.CurrentIndexKey(), types.NumKey(index))
}

func (k Keeper) GetCurrentIndex(ctx sdk.Context) int64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.SupportedQueriesKey())
	bz := store.Get(types.CurrentIndexKey())
	return int64(sdk.BigEndianToUint64(bz))
}
