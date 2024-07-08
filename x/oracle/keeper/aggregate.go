package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetAggregatedReport calculates and allocates rewards to reporters based on aggregated reports.
// at a specific blockchain height (to be ran in end-blocker)
// It retrieves the revealed reports from the reports store, by query.
// calculates the aggregate report for each query using either the weighted-median or weighted-mode method.
// Rewards based on the source are then allocated to the reporters.
func (k Keeper) SetAggregatedReport(ctx context.Context) (err error) {
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

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkCtx.HeaderInfo().Time

	var aggrFunc func(ctx context.Context, reports []types.MicroReport) (*types.Aggregate, error)
	reportersToPay := make([]*types.AggregateReporter, 0)
	for _, query := range queries {
		if query.Expiration.Add(offset).Before(blockTime) {
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
				err = k.AllocateRewards(ctx, report.Reporters, query.Amount, types.ModuleName)
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
	if len(reportersToPay) == 0 {
		return nil
	}
	// Allocate time-based rewards to all eligible reporters.
	return k.AllocateRewards(ctx, reportersToPay, tbr, minttypes.TimeBasedRewards)
}

func (k Keeper) SetAggregate(ctx context.Context, report *types.Aggregate) error {
	nonce, err := k.Nonces.Get(ctx, report.QueryId)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	nonce++
	err = k.Nonces.Set(ctx, report.QueryId, nonce)
	if err != nil {
		return err
	}
	report.Nonce = nonce

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentTimestamp := sdkCtx.HeaderInfo().Time.UnixMilli()
	report.Height = sdkCtx.BlockHeight()

	return k.Aggregates.Set(ctx, collections.Join(report.QueryId, currentTimestamp), *report)
}

// getDataBefore returns the last aggregate before or at the given timestamp for the given query id.
func (k Keeper) GetDataBefore(ctx context.Context, queryId []byte, timestamp time.Time) (*types.Aggregate, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).EndInclusive(timestamp.UnixMilli()).Descending()
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
		return nil, err
	}

	if mostRecent == nil {
		return nil, types.ErrNoAvailableReports.Wrapf("no data before timestamp %v available for query id %s", timestamp, hex.EncodeToString(queryId))
	}

	return mostRecent, nil
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

func (k Keeper) GetTimestampBefore(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).EndExclusive(timestamp.UnixMilli()).Descending()
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

	return time.UnixMilli(mostRecent), nil
}

func (k Keeper) GetTimestampAfter(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).StartExclusive(timestamp.UnixMilli())
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

	return time.UnixMilli(mostRecent), nil
}

func (k Keeper) GetAggregatedReportsByHeight(ctx context.Context, height int64) []types.Aggregate {
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

func (k Keeper) GetCurrentAggregateReport(ctx context.Context, queryId []byte) (aggregate *types.Aggregate, timestamp time.Time) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).Descending()
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		aggregate = &value
		timestamp = time.UnixMilli(key.K2())
		return true, nil
	})
	if err != nil {
		panic(err) // Handle the error appropriately
	}
	return aggregate, timestamp
}

func (k Keeper) GetAggregateBefore(ctx context.Context, queryId []byte, timestampBefore time.Time) (aggregate *types.Aggregate, timestamp time.Time, err error) {
	// Convert the timestampBefore to Unix time and create a range that ends just before this timestamp
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).EndExclusive(timestampBefore.UnixMilli()).Descending()

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
	timestamp = time.UnixMilli(mostRecentTimestamp)
	return mostRecent, timestamp, nil
}

func (k Keeper) GetAggregateByTimestamp(ctx context.Context, queryId []byte, timestamp time.Time) (aggregate *types.Aggregate, err error) {
	timestampUnix := timestamp.UnixMilli()

	// Create a range that specifically targets the exact timestamp
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).StartInclusive(timestampUnix).EndInclusive(timestampUnix)

	// Walk through the aggregates to find the one that exactly matches the timestamp
	err = k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		if key.K2() == timestampUnix {
			aggregate = &value
			return true, nil // Stop when the exact match is found
		}
		return false, nil // Continue if this is not the exact match
	})
	if err != nil {
		return nil, err
	}

	if aggregate == nil {
		return nil, fmt.Errorf("no aggregate report found at timestamp %v for query id %s", timestamp, hex.EncodeToString(queryId))
	}

	return aggregate, nil
}

func (k Keeper) GetAggregateByIndex(ctx context.Context, queryId []byte, index uint64) (aggregate *types.Aggregate, timestamp time.Time, err error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId)

	// var aggregate *types.Aggregate
	var currentIndex uint64

	err = k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		if currentIndex == index {
			aggregate = &value
			timestamp = time.UnixMilli(key.K2())
			return true, nil // Stop when the desired index is reached
		}
		currentIndex++
		return false, nil // Continue to the next aggregate
	})
	if err != nil {
		return nil, time.Time{}, err
	}

	if aggregate == nil {
		return nil, time.Time{}, fmt.Errorf("no aggregate found at index %d for query id %s", index, hex.EncodeToString(queryId))
	}

	return aggregate, timestamp, nil
}
