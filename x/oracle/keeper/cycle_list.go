package keeper

import (
	"bytes"
	"context"
	"errors"

	"github.com/tellor-io/layer/utils"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetCyclelist(ctx context.Context) ([][]byte, error) {
	iter, err := k.Cyclelist.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	return iter.Values()
}

// rotation of the cycle list
func (k Keeper) RotateQueries(ctx context.Context) error {
	q, err := k.GetCyclelist(ctx)
	if err != nil {
		return err
	}
	n, err := k.CyclelistSequencer.Next(ctx)
	if err != nil {
		return err
	}

	max := len(q)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkCtx.BlockTime()
	switch {
	case n >= uint64(max-1):
		err := k.CyclelistSequencer.Set(ctx, 0)
		if err != nil {
			return err
		}
		n = 0
	default:
		n += 1
	}
	queryId := utils.QueryIDFromData(q[n])
	querymeta, err := k.CurrentQuery(ctx, queryId)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		querymeta, err = k.InitializeQuery(ctx, q[n])
		if err != nil {
			return err
		}
		querymeta.CycleList = true
		querymeta.Expiration = blockTime.Add(querymeta.RegistrySpecTimeframe)
		return k.Query.Set(ctx, collections.Join(queryId, querymeta.Id), querymeta)

	}

	nextId, err := k.QuerySequencer.Next(ctx)
	if err != nil {
		return err
	}
	querymeta.Id = nextId
	querymeta.Expiration = blockTime.Add(querymeta.RegistrySpecTimeframe)
	querymeta.HasRevealedReports = false
	querymeta.CycleList = true
	return k.Query.Set(ctx, collections.Join(queryId, querymeta.Id), querymeta)
}

func (k Keeper) GetCurrentQueryInCycleList(ctx context.Context) ([]byte, error) {
	idx, err := k.CyclelistSequencer.Peek(ctx)
	if err != nil {
		return nil, err
	}

	q, err := k.GetCyclelist(ctx)
	if err != nil {
		return nil, err
	}

	return q[idx], nil
}

func (k Keeper) GetNextCurrentQueryInCycleList(ctx context.Context) ([]byte, error) {
	idx, err := k.CyclelistSequencer.Peek(ctx)
	if err != nil {
		return nil, err
	}

	q, err := k.GetCyclelist(ctx)
	if err != nil {
		return nil, err
	}
	next := idx + 1
	if next >= uint64(len(q)) {
		next = 0
	}
	return q[next], nil
}

// should be called only once when updating the cycle list
func (k Keeper) InitCycleListQuery(ctx context.Context, queries [][]byte) error {
	for _, querydata := range queries {
		queryId := utils.QueryIDFromData(querydata)
		err := k.Cyclelist.Set(ctx, queryId, querydata)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) GenesisCycleList(ctx context.Context, cyclelist [][]byte) error {
	for _, queryData := range cyclelist {
		queryId := utils.QueryIDFromData(queryData)
		err := k.Cyclelist.Set(ctx, queryId, queryData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) IncycleCheck(ctx context.Context, queryId []byte) (bool, error) {
	peek, err := k.CyclelistSequencer.Peek(ctx)
	if err != nil {
		return false, err
	}
	cyclelist, err := k.GetCyclelist(ctx)
	if err != nil {
		return false, err
	}
	return bytes.Equal(queryId, cyclelist[peek]), nil
}
