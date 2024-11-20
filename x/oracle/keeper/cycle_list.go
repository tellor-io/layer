package keeper

import (
	"context"
	"errors"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetCyclelist returns the entire cycle list
func (k Keeper) GetCyclelist(ctx context.Context) ([][]byte, error) {
	iter, err := k.Cyclelist.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	return iter.Values()
}

// rotation of the cycle list (called in EndBlocker)
func (k Keeper) RotateQueries(ctx context.Context) error {
	// only rotate if current query is expired
	// get current query
	// if current query is not expired, return
	// if current query is expired, rotate the cycle list
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	querydata, err := k.GetCurrentQueryInCycleList(ctx)
	if err != nil {
		return err
	}
	queryId := utils.QueryIDFromData(querydata)

	queryMeta, err := k.CurrentQuery(ctx, queryId)
	// if current query has not expired, return and don't create a new query/rotate
	if err == nil && queryMeta.Expiration > uint64(blockHeight) {
		return nil
	}
	// rotate
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
	case n >= uint64(max-1): // n could be gt if the cycle list is updated, otherwise n == max-1
		err := k.CyclelistSequencer.Set(ctx, 0)
		if err != nil {
			return err
		}
		n = 0
	default:
		n += 1
	}
	// next query
	queryId = utils.QueryIDFromData(q[n])
	// cycle list queries that are without a tip could linger in the store if
	// there are no reports to be aggregated (where queries are removed from the store)
	// and since each rotation we generate a new query, here we clear the old queries that are expired, have no tip and have no reports
	err = k.ClearOldqueries(ctx, queryId)
	if err != nil {
		return err
	}
	// get query if it exists, should exist if it has a tip that wasn't cleared by aggregation (ie no one reported for it)
	querymeta, err := k.CurrentQuery(ctx, queryId)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		// initialize a query since it was cleared either by aggregation or expiration and not tipped
		querymeta, err = k.InitializeQuery(ctx, q[n])
		if err != nil {
			return err
		}
		querymeta.CycleList = true
		querymeta.Expiration = uint64(blockHeight) + querymeta.RegistrySpecBlockWindow
		return k.Query.Set(ctx, collections.Join(queryId, querymeta.Id), querymeta)

	}
	// if query has a tip don't generate a new query
	// shouldn't enter here if query has no tip and is expired since it would have been cleared
	// if query has a tip and is not expired and has reports/or no reports then set cycle list true and nothing else
	// if query has a tip and is expired, then this by default means it has no reports because
	// it would have been cleared by aggregation which is called before RotateQueries; so
	// set cycle list to true and extend the expiration time.
	// sidenote: similar to tipping, tipping only extends the expiration if a query is expired or
	// only increments the tip amount for a query that is ongoing.
	if !querymeta.Amount.IsZero() {
		querymeta.CycleList = true
		expired := querymeta.Expiration <= uint64(blockHeight)
		// this should not be required since SetAggregate happens before rotation which should clear any query that is expired and has revealed reports
		// noRevealedReports := !querymeta.HasRevealedReports
		if expired {
			// extend time as if tbr is a tip that would extend the time (tipping)
			querymeta.Expiration = uint64(blockHeight) + querymeta.RegistrySpecBlockWindow
		}

		return k.Query.Set(ctx, collections.Join(queryId, querymeta.Id), querymeta)
	}

	// todo: should not reach here
	// if query has no tip generate a new query window
	// nextId, err := k.QuerySequencer.Next(ctx)
	// if err != nil {
	// 	return err
	// }
	// sdkCtx.EventManager().EmitEvents(sdk.Events{
	// 	sdk.NewEvent(
	// 		"rotating-cyclelist-with-existing-nontipped-query",
	// 		sdk.NewAttribute("query_id", string(queryId)),
	// 		sdk.NewAttribute("Old QueryMeta Id", strconv.Itoa(int(querymeta.Id))),
	// 		sdk.NewAttribute("New QueryMeta Id", strconv.Itoa(int(nextId))),
	// 	),
	// })
	// querymeta.Id = nextId
	// querymeta.Expiration = uint64(blockHeight) + querymeta.RegistrySpecBlockWindow
	// querymeta.HasRevealedReports = false
	// querymeta.CycleList = true

	// return k.Query.Set(ctx, collections.Join(queryId, querymeta.Id), querymeta)
	return nil
}

// removes query that are expired, no tip, and no revealed reports
// used in RotateQueries to clear cycle list queries that weren't cleared by aggregation
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
