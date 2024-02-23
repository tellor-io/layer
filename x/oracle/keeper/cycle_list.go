package keeper

import (
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// rotation what query is next
func (k Keeper) RotateQueries(ctx sdk.Context) error {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}
	currentIndex, err := k.CycleIndex.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			err = k.CycleIndex.Set(ctx, 0)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if currentIndex >= int64(len(params.CycleList)) {
		currentIndex = 0
	}
	i := (currentIndex + 1) % int64(len(params.CycleList))
	err = k.CycleIndex.Set(ctx, i)
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) GetCurrentQueryInCycleList(ctx sdk.Context) (string, error) {
	currentIndex, err := k.CycleIndex.Get(ctx)
	if err != nil {
		return "", err
	}
	params, err := k.Params.Get(ctx)
	if err != nil {
		return "", err
	}
	return params.CycleList[currentIndex], nil
}
