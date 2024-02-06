package keeper

import (
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) GetCycleList(ctx sdk.Context) []string {
	return k.GetParams(ctx).CycleList
}

// rotation what query is next
func (k Keeper) RotateQueries(ctx sdk.Context) string {
	queries := k.GetCycleList(ctx)
	currentIndex := k.GetCurrentIndex(ctx)

	// Increment currentIndex first, then adjust if it exceeds bounds
	currentIndex = (currentIndex + 1) % int64(len(queries))
	// After incrementing, update the stored currentIndex for the next rotation
	k.SetCurrentIndex(ctx, currentIndex)
	// Return the query at the updated currentIndex
	return queries[currentIndex]
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

func (k Keeper) GetCurrentQueryInCycleList(ctx sdk.Context) string {
	queries := k.GetCycleList(ctx)
	currentIndex := k.GetCurrentIndex(ctx)
	return queries[currentIndex]
}
