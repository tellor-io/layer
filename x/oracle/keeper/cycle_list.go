package keeper

import (
	"context"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) GetCyclelist(ctx context.Context) ([][]byte, error) {

	iter, err := k.Cyclelist.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	q, err := iter.Values()
	if err != nil {
		return nil, err
	}
	return q, nil
}

// rotation of the cycle list
func (k Keeper) RotateQueries(ctx context.Context) error {
	// todo: better to set length of cycle list as an item and read that
	// so we don't do this read operation every time

	q, err := k.GetCyclelist(ctx)
	if err != nil {
		return err
	}
	n, err := k.CyclelistSequencer.Next(ctx)
	if err != nil {
		return err
	}

	max := len(q)
	switch {
	case n == uint64(max-1):
		return k.CyclelistSequencer.Set(ctx, 0)
	default:
		return nil
	}

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

// should be called only once when updating the cycle list
func (k Keeper) InitCycleListQuery(ctx context.Context, queries [][]byte) error {

	for _, querydata := range queries {

		query, err := k.initializeQuery(ctx, querydata)
		if err != nil {
			return err
		}
		queryId := utils.QueryIDFromData(querydata)
		err = k.Query.Set(ctx, queryId, query)
		if err != nil {
			return err
		}
		err = k.Cyclelist.Set(ctx, queryId, querydata)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) GenesisCycleList(ctx context.Context, cyclelist [][]byte) error {

	for _, queryData := range cyclelist {
		queryId := utils.QueryIDFromData(queryData)

		nextId, err := k.QuerySequnecer.Next(ctx)
		if err != nil {
			return err
		}
		meta := types.QueryMeta{
			Id:                    nextId,
			RegistrySpecTimeframe: 0,
			QueryId:               queryId,
		}
		err = k.Query.Set(ctx, queryId, meta)
		if err != nil {
			return err
		}
		err = k.Cyclelist.Set(ctx, queryId, queryData)
		if err != nil {
			return err
		}
	}
	return nil
}
