package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

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
	fmt.Println("RotateQueries")
	// only rotate if current query is expired
	// get current query
	// if current query is not expired, return
	// if current query is expired, rotate the cycle list
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()
	fmt.Println("blockHeight", blockHeight)

	querydata, err := k.GetCurrentQueryInCycleList(ctx)
	if err != nil {
		return err
	}
	queryId := utils.QueryIDFromData(querydata)
	fmt.Println("current queryId", hex.EncodeToString(queryId))
	nPeek, err := k.CyclelistSequencer.Peek(ctx)
	if err != nil {
		return err
	}
	fmt.Println("nPeek", nPeek)
	queryMeta, err := k.CurrentQuery(ctx, queryId)
	if err == nil && queryMeta.Expiration > uint64(blockHeight) {
		return nil
	}

	q, err := k.GetCyclelist(ctx)
	if err != nil {
		return err
	}
	n, err := k.CyclelistSequencer.Next(ctx)
	if err != nil {
		return err
	}
	fmt.Println("n", n)
	max := len(q)

	switch {
	case n >= uint64(max-1): // n could be gt if the cycle list is updated, otherwise n == max-1
		err := k.CyclelistSequencer.Set(ctx, 0)
		if err != nil {
			return err
		}
		n = 0
	default:
		n += 1
	}
	fmt.Println("n after", n)
	queryId = utils.QueryIDFromData(q[n])
	fmt.Println("queryId", hex.EncodeToString(queryId))
	// queries that are without tip (ie cycle list queries) could linger in the store
	// if there are no reports to be aggregated (where queries removed) since you each query cycle we generate a new query
	err = k.ClearOldqueries(ctx, queryId)
	if err != nil {
		return err
	}
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
		querymeta.Expiration = uint64(blockHeight) + querymeta.RegistrySpecBlockWindow
		return k.Query.Set(ctx, collections.Join(queryId, querymeta.Id), querymeta)

	}
	// if query has a tip don't generate a new query but extend if revealing time is expired
	if !querymeta.Amount.IsZero() {
		querymeta.CycleList = true

		if querymeta.Expiration >= uint64(blockHeight) { // wrong, shouldn't use same query if expired
			querymeta.Expiration = uint64(blockHeight) + querymeta.RegistrySpecBlockWindow
		}
		return k.Query.Set(ctx, collections.Join(queryId, querymeta.Id), querymeta)
	}
	// if query has no tip generate a new query window
	nextId, err := k.QuerySequencer.Next(ctx)
	if err != nil {
		return err
	}
	querymeta.Id = nextId
	querymeta.Expiration = uint64(blockHeight) + querymeta.RegistrySpecBlockWindow
	querymeta.HasRevealedReports = false
	querymeta.CycleList = true
	return k.Query.Set(ctx, collections.Join(queryId, querymeta.Id), querymeta)
}

func (k Keeper) ClearOldqueries(ctx context.Context, queryId []byte) error {
	rng := collections.NewPrefixedPairRange[[]byte, uint64](queryId)

	return k.Query.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.QueryMeta) (stop bool, err error) {
		if value.Expiration < (uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())) && !value.HasRevealedReports && value.Amount.IsZero() {
			err := k.Query.Remove(ctx, key)
			if err != nil {
				return false, err
			}
		}
		return false, nil
	})
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
