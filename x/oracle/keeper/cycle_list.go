package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) SetCycleList(ctx sdk.Context, queries []types.CycleListQuery) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CycleListKey())
	var list = make([]*types.CycleListQuery, len(queries))
	for i := range queries {
		list[i] = &queries[i]
	}

	bz := k.cdc.MustMarshal(&types.CycleList{QueryData: list})
	store.Set(types.CycleListKey(), bz)
}

func (k Keeper) GetCyclList(ctx sdk.Context) types.CycleList {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CycleListKey())
	bz := store.Get(types.CycleListKey())
	var d types.CycleList
	k.cdc.MustUnmarshal(bz, &d)
	return d
}

// rotation what query is next
func (k Keeper) RotateQueries(ctx sdk.Context) string {
	queries := k.GetCyclList(ctx)

	currentIndex := k.GetCurrentIndex(ctx)
	if currentIndex >= int64(len(queries.QueryData)) {
		currentIndex = 0
	}

	k.SetCurrentIndex(ctx, (currentIndex+1)%int64(len(queries.QueryData)))
	return queries.QueryData[currentIndex].QueryData
}

func (k Keeper) SetCurrentIndex(ctx sdk.Context, index int64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CycleListKey())
	store.Set(types.CurrentIndexKey(), types.NumKey(index))
}

func (k Keeper) GetCurrentIndex(ctx sdk.Context) int64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CycleListKey())
	bz := store.Get(types.CurrentIndexKey())
	return int64(sdk.BigEndianToUint64(bz))
}
