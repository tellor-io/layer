package keeper

import (
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetCycleList(ctx sdk.Context) []string {
	return k.GetParams(ctx).CycleList
}

// rotation what query is next
func (k Keeper) RotateQueries(ctx sdk.Context) error {
	queries := k.GetCycleList(ctx)
	// cycle list collections is empty at block 0
	if ctx.BlockHeight() == 0 {
		return nil
	}
	currentIndex, err := k.CycleIndex.Get(ctx)
	if err != nil {
		return err
	}
	if currentIndex >= int64(len(queries)) {
		currentIndex = 0
	}

	err = k.CycleIndex.Set(ctx, (currentIndex+1)%int64(len(queries)))
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) GetCurrentQueryInCycleList(ctx sdk.Context) string {
	queries := k.GetCycleList(ctx)
	currentIndex, err := k.CycleIndex.Get(ctx)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		panic(err)
	}
	return queries[currentIndex]
}
