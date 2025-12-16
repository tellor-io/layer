package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"strconv"

	"github.com/tellor-io/layer/utils"

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

	// rotation is happening - increment total queries in period for liveness tracking
	if err := k.IncrementTotalQueriesInPeriod(ctx); err != nil {
		return err
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

		// cycle complete - check if it's time to distribute liveness rewards
		if err := k.CheckAndDistributeLivenessRewards(ctx); err != nil {
			return err
		}
	default:
		n += 1
	}
	// next query
	queryId = utils.QueryIDFromData(q[n])

	// increment query opportunities for liveness tracking
	if err := k.IncrementQueryOpportunities(ctx, queryId); err != nil {
		return err
	}

	// get query if it exists, should exist if it has a tip that wasn't cleared by aggregation (ie no one reported for it)
	querymeta, err := k.CurrentQuery(ctx, queryId)
	// cycle list queries that are without a tip could linger in the store if
	// there are no reports to be aggregated (where queries are removed from the store)
	// and since each rotation we generate a new query, here we clear the old queries that are expired, have no tip and have no reports
	if err == nil && querymeta.Expiration < (uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())) && !querymeta.HasRevealedReports && querymeta.Amount.IsZero() {
		// remove query
		err = k.Query.Remove(ctx, collections.Join(queryId, querymeta.Id))
		if err != nil {
			return err
		}
		id, err := k.QuerySequencer.Next(ctx)
		if err != nil {
			return err
		}
		querymeta.Id = id
		querymeta.CycleList = true
		querymeta.Expiration = uint64(blockHeight) + querymeta.RegistrySpecBlockWindow
		emitRotateQueriesEvent(sdkCtx, hex.EncodeToString(queryId), strconv.Itoa(int(querymeta.Id)))
		return k.Query.Set(ctx, collections.Join(queryId, querymeta.Id), querymeta)
	}
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
		emitRotateQueriesEvent(sdkCtx, hex.EncodeToString(queryId), strconv.Itoa(int(querymeta.Id)))
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
		emitRotateQueriesEvent(sdkCtx, hex.EncodeToString(queryId), strconv.Itoa(int(querymeta.Id)))

		return k.Query.Set(ctx, collections.Join(queryId, querymeta.Id), querymeta)
	}
	return nil
}

func emitRotateQueriesEvent(sdkCtx sdk.Context, queryId, nextId string) {
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"rotating-cyclelist-with-next-query",
			sdk.NewAttribute("query_id", queryId),
			sdk.NewAttribute("New QueryMeta Id", nextId),
		),
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
