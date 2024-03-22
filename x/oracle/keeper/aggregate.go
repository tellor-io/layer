package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
)

// SetAggregatedReport calculates and allocates rewards to reporters based on aggregated reports.
// at a specific blockchain height (to be ran in end-blocker)
// It retrieves the revealed reports from the reports store, by query.
// calculates the aggregate report for each query using either the weighted-median or weighted-mode method.
// Rewards based on the source are then allocated to the reporters.
func (k Keeper) SetAggregatedReport(ctx sdk.Context) (err error) {

	// aggregate
	idsIterator, err := k.Query.Indexes.HasReveals.MatchExact(ctx, true)
	if err != nil {
		return err
	}

	defer idsIterator.Close()
	queries, err := indexes.CollectValues(ctx, k.Query, idsIterator)
	if err != nil {
		return err
	}

	var aggrFunc func(ctx sdk.Context, reports []types.MicroReport) (*types.Aggregate, error)
	reportersToPay := make([]*types.AggregateReporter, 0)
	for _, query := range queries {
		if query.Expiration.Add(offset).Before(ctx.BlockTime()) {
			reportsIterator, err := k.Reports.Indexes.Id.MatchExact(ctx, query.Id)
			if err != nil {
				return err
			}
			defer reportsIterator.Close()
			reports, err := indexes.CollectValues(ctx, k.Reports, reportsIterator)
			if err != nil {
				return err
			}
			// there should always be at least one report otherwise how did the query set hasrevealedreports to true
			if reports[0].AggregateMethod == "weighted-median" {
				// Calculate the aggregated report.
				aggrFunc = k.WeightedMedian
			} else {
				// default to weighted-mode aggregation method.
				// Calculate the aggregated report.
				aggrFunc = k.WeightedMode
			}

			report, err := aggrFunc(ctx, reports)
			if err != nil {
				return err
			}

			if !query.Amount.IsZero() {
				err = k.AllocateRewards(ctx, report.Reporters, query.Amount, true)
				if err != nil {
					return err
				}
				// zero out the amount in the query
				query.Amount = math.ZeroInt()
			}
			// Add reporters to the tbr payment list.
			if reports[0].Cyclelist {
				reportersToPay = append(reportersToPay, report.Reporters...)
			}

			query.HasRevealedReports = false
			err = k.Query.Set(ctx, query.QueryId, query)
			if err != nil {
				return err
			}
		}
	}

	// Process time-based rewards for reporters.
	tbr := k.getTimeBasedRewards(ctx)
	// Allocate time-based rewards to all eligible reporters.
	return k.AllocateRewards(ctx, reportersToPay, tbr, false)
}

func (k Keeper) SetAggregate(ctx sdk.Context, report *types.Aggregate) error {
	queryId, err := utils.QueryBytesFromString(report.QueryId)
	if err != nil {
		return err
	}
	nonce, err := k.Nonces.Get(ctx, queryId)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	nonce++
	err = k.Nonces.Set(ctx, queryId, nonce)
	if err != nil {
		return err
	}
	report.Nonce = nonce

	currentTimestamp := ctx.BlockTime().Unix()
	report.Height = ctx.BlockHeight()

	return k.Aggregates.Set(ctx, collections.Join(queryId, currentTimestamp), *report)
}

// getDataBefore returns the last aggregate before or at the given timestamp for the given query id.
// TODO: add a test for this function.
func (k Keeper) getDataBefore(ctx sdk.Context, queryId []byte, timestamp time.Time) (*types.Aggregate, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).EndInclusive(timestamp.Unix()).Descending()
	var mostRecent *types.Aggregate
	// This should get us the most recent aggregate, as they are walked in descending order
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		if !value.Flagged {
			mostRecent = &value
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		// why panic here? should we return an error instead?
		panic(err)
	}

	if mostRecent == nil {
		return nil, types.ErrNoAvailableReports.Wrapf("no data before timestamp %v available for query id %s", timestamp, hex.EncodeToString(queryId))
	}

	return mostRecent, nil
}

func (k Keeper) GetDataBeforePublic(ctx sdk.Context, queryId []byte, timestamp time.Time) (*types.Aggregate, error) {
	return k.getDataBefore(ctx, queryId, timestamp)
}

func (k Keeper) GetCurrentValueForQueryId(ctx context.Context, queryId []byte) (*types.Aggregate, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).Descending()
	var mostRecent *types.Aggregate
	// This should get us the most recent aggregate, as they are walked in descending order
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		mostRecent = &value
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return mostRecent, nil
}

func (k Keeper) GetTimestampBefore(ctx sdk.Context, queryId []byte, timestamp time.Time) (time.Time, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).EndExclusive(timestamp.Unix()).Descending()
	var mostRecent int64
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		mostRecent = key.K2()
		return true, nil
	})

	if err != nil {
		panic(err)
	}

	if mostRecent == 0 {
		return time.Time{}, fmt.Errorf("no data before timestamp %v available for query id %s", timestamp, hex.EncodeToString(queryId))
	}

	return time.Unix(mostRecent, 0), nil
}

func (k Keeper) GetTimestampAfter(ctx sdk.Context, queryId []byte, timestamp time.Time) (time.Time, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).StartExclusive(timestamp.Unix())
	var mostRecent int64
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		mostRecent = key.K2()
		return true, nil
	})

	if err != nil {
		panic(err)
	}

	if mostRecent == 0 {
		return time.Time{}, fmt.Errorf("no data before timestamp %v available for query id %s", timestamp, hex.EncodeToString(queryId))
	}

	return time.Unix(mostRecent, 0), nil
}

func (k Keeper) GetAggregatedReportsByHeight(ctx sdk.Context, height int64) []types.Aggregate {
	iter, err := k.Aggregates.Indexes.BlockHeight.MatchExact(ctx, height)
	if err != nil {
		panic(err)
	}

	kvs, err := indexes.CollectKeyValues(ctx, k.Aggregates, iter)
	if err != nil {
		panic(err)
	}

	reports := make([]types.Aggregate, len(kvs))
	for i, kv := range kvs {
		reports[i] = kv.Value
	}

	return reports
}

func (k Keeper) GetCurrentAggregateReport(ctx sdk.Context, queryId []byte) (aggregate *types.Aggregate, timestamp time.Time) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).Descending()
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		aggregate = &value
		timestamp = time.Unix(key.K2(), 0)
		return true, nil
	})
	if err != nil {
		panic(err) // Handle the error appropriately
	}
	return aggregate, timestamp
}

func (k Keeper) GetAggregateBefore(ctx sdk.Context, queryId []byte, timestampBefore time.Time) (aggregate *types.Aggregate, timestamp time.Time, err error) {
	// Convert the timestampBefore to Unix time and create a range that ends just before this timestamp
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).EndExclusive(timestampBefore.Unix()).Descending()

	var mostRecent *types.Aggregate
	var mostRecentTimestamp int64

	// Walk through the aggregates in descending order to find the most recent one before timestampBefore
	err = k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		mostRecent = &value
		mostRecentTimestamp = key.K2()
		return true, nil // Stop after the first (most recent) match
	})

	if err != nil {
		return nil, time.Time{}, err
	}

	if mostRecent == nil {
		return nil, time.Time{}, fmt.Errorf("no aggregate report found before timestamp %v for query id %s", timestampBefore, hex.EncodeToString(queryId))
	}

	// Convert the Unix timestamp back to time.Time
	timestamp = time.Unix(mostRecentTimestamp, 0)
	return mostRecent, timestamp, nil
}
