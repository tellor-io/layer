package keeper

import (
	"encoding/hex"
	"errors"
	"strings"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/utils"
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

func (k Keeper) CycleListAsQueryIds(ctx sdk.Context) (map[string]bool, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	queryIds := make(map[string]bool, len(params.CycleList))
	for _, q := range params.CycleList {
		queryId, err := utils.QueryIDFromDataString(q)
		if err != nil {
			return nil, err
		}
		queryIds[strings.ToLower(hex.EncodeToString(queryId))] = true
	}
	return queryIds, nil
}
