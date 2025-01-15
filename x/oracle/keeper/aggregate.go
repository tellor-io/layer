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

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetAggregatedReport (called in EndBlocker) fetches the Query iterator for queries
// that have revealed reports, then iterates over the queries and checks whether the query has expired.
// If the query has expired, it fetches all the microReports for a query.Id and aggregates them based
// on the query spec's aggregate method.
// If the query has a tip then that tip is distributed to the micro-reports' reporters,
// proportional to their reporting power.
// In addition, all the micro-reports that are part of a cyclelist are gathered and their reporters are
// rewarded with the time-based rewards.
func (k Keeper) SetAggregatedReport(ctx context.Context) (err error) {
	// aggregate
	idsIterator, err := k.Query.Indexes.HasReveals.MatchExact(ctx, true)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())

	var aggrFunc func(ctx context.Context, reports []types.MicroReport, metaId uint64) (*types.Aggregate, error)
	reportersToPay := make([]*types.Aggregate, 0)
	i := 0
	defer idsIterator.Close()
	for ; idsIterator.Valid(); idsIterator.Next() {
		key, err := idsIterator.PrimaryKey()
		if err != nil {
			return err
		}
		query, err := k.Query.Get(ctx, key)
		if err != nil {
			return err
		}
		// enter if query has expired
		if query.Expiration <= blockHeight {
			i++
			reportsIterator, err := k.Reports.Indexes.Id.MatchExact(ctx, query.Id)
			if err != nil {
				return err
			}
			microReports, err := indexes.CollectValues(ctx, k.Reports, reportsIterator)
			if err != nil {
				return err
			}
			// there should always be at least one report otherwise how did the query set hasrevealedreports to true
			if microReports[0].AggregateMethod == "weighted-median" {
				// Calculate the aggregated report.
				aggrFunc = k.WeightedMedian
			} else {
				// default to weighted-mode aggregation method.
				// Calculate the aggregated report.
				aggrFunc = k.WeightedMode
			}

			aggregateReport, err := aggrFunc(ctx, microReports, query.Id)
			if err != nil {
				return err
			}
			err = k.SetAggregate(ctx, aggregateReport)
			if err != nil {
				return err
			}
			if !query.Amount.IsZero() {
				err = k.AllocateRewards(ctx, []*types.Aggregate{aggregateReport}, query.Amount, types.ModuleName)
				if err != nil {
					return err
				}
			}
			// Add reporters to the tbr payment list.
			if microReports[0].Cyclelist {
				reportersToPay = append(reportersToPay, aggregateReport)
			}
			err = k.Query.Remove(ctx, key)
			if err != nil {
				return err
			}
		}
	}
	fmt.Println("itertated through: ", i, " queries")
	if len(reportersToPay) == 0 {
		return nil
	}
	fmt.Println("paying this many reporters: ", len(reportersToPay))
	// Process time-based rewards for reporters.
	// tbr is in loya
	tbr := k.GetTimeBasedRewards(ctx)
	// Allocate time-based rewards to all eligible reporters.
	return k.AllocateRewards(ctx, reportersToPay, tbr, minttypes.TimeBasedRewards)
}

// SetAggregate increments the queryId's report index plus sets the timestamp and blockHeight and stores the aggregate report
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
	report.Index = nonce

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentTimestamp := uint64(sdkCtx.BlockTime().UnixMilli())
	report.Height = uint64(sdkCtx.BlockHeight())

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"aggregate_report",
			sdk.NewAttribute("query_id", hex.EncodeToString(report.QueryId)),
			sdk.NewAttribute("value", report.AggregateValue),
			sdk.NewAttribute("number_of_reporters", fmt.Sprintf("%d", len(report.Reporters))),
			sdk.NewAttribute("micro_report_height", fmt.Sprintf("%d", report.MicroHeight)),
		),
	})
	return k.Aggregates.Set(ctx, collections.Join(report.QueryId, currentTimestamp), *report)
}

func (k Keeper) GetTimestampBefore(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error) {
	rng := collections.NewPrefixedPairRange[[]byte, uint64](queryId).EndExclusive(uint64(timestamp.UnixMilli())).Descending()
	var mostRecent uint64
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.Aggregate) (stop bool, err error) {
		mostRecent = key.K2()
		return true, nil
	})
	if err != nil {
		panic(err)
	}

	if mostRecent == 0 {
		return time.Time{}, fmt.Errorf("no data before timestamp %v available for query id %s", timestamp, hex.EncodeToString(queryId))
	}

	return time.UnixMilli(int64(mostRecent)), nil
}

func (k Keeper) GetTimestampAfter(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error) {
	rng := collections.NewPrefixedPairRange[[]byte, uint64](queryId).StartExclusive(uint64(timestamp.UnixMilli()))
	var mostRecent uint64
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.Aggregate) (stop bool, err error) {
		mostRecent = key.K2()
		return true, nil
	})
	if err != nil {
		panic(err)
	}

	if mostRecent == 0 {
		return time.Time{}, fmt.Errorf("no data before timestamp %v available for query id %s", timestamp, hex.EncodeToString(queryId))
	}

	return time.UnixMilli(int64(mostRecent)), nil
}

func (k Keeper) GetAggregatedReportsByHeight(ctx context.Context, height uint64) []types.Aggregate {
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

func (k Keeper) GetCurrentAggregateReport(ctx context.Context, queryId []byte) (aggregate *types.Aggregate, timestamp time.Time, err error) {
	rng := collections.NewPrefixedPairRange[[]byte, uint64](queryId).Descending()
	err = k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.Aggregate) (stop bool, err error) {
		aggregate = &value
		timestamp = time.UnixMilli(int64(key.K2()))
		return true, nil
	})
	if err != nil {
		return nil, time.Time{}, err
	}
	if aggregate == nil {
		return nil, time.Time{}, fmt.Errorf("aggregate not found")
	}
	return aggregate, timestamp, nil
}

func (k Keeper) GetAggregateBefore(ctx context.Context, queryId []byte, timestampBefore time.Time) (aggregate *types.Aggregate, timestamp time.Time, err error) {
	// Convert the timestampBefore to Unix time and create a range that ends just before this timestamp
	rng := collections.NewPrefixedPairRange[[]byte, uint64](queryId).EndExclusive(uint64(timestampBefore.UnixMilli())).Descending()

	var mostRecent *types.Aggregate
	var mostRecentTimestamp uint64

	// Walk through the aggregates in descending order to find the most recent one before timestampBefore
	err = k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.Aggregate) (stop bool, err error) {
		if !value.Flagged {
			mostRecent = &value
			mostRecentTimestamp = key.K2()
			return true, nil // Stop after the first (most recent) match
		}
		return false, nil
	})
	if err != nil {
		return nil, time.Time{}, err
	}

	if mostRecent == nil {
		return nil, time.Time{}, fmt.Errorf("no aggregate report found before timestamp %v for query id %s", timestampBefore, hex.EncodeToString(queryId))
	}

	// Convert the Unix timestamp back to time.Time
	timestamp = time.UnixMilli(int64(mostRecentTimestamp))
	return mostRecent, timestamp, nil
}

func (k Keeper) GetAggregateByTimestamp(ctx context.Context, queryId []byte, timestamp time.Time) (aggregate types.Aggregate, err error) {
	agg, err := k.Aggregates.Get(ctx, collections.Join(queryId, uint64(timestamp.UnixMilli())))
	if err != nil {
		return types.Aggregate{}, err
	}
	return agg, nil
}

func (k Keeper) GetAggregateByIndex(ctx context.Context, queryId []byte, index uint64) (aggregate *types.Aggregate, timestamp time.Time, err error) {
	rng := collections.NewPrefixedPairRange[[]byte, uint64](queryId)

	var currentIndex uint64
	err = k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.Aggregate) (stop bool, err error) {
		if currentIndex == index {
			aggregate = &value
			timestamp = time.UnixMilli(int64(key.K2()))
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

func (k Keeper) GetAggregateBeforeByReporter(ctx context.Context, queryId []byte, timestamp time.Time, reporter sdk.AccAddress) (aggregate *types.Aggregate, err error) {
	rng := collections.NewPrefixedPairRange[[]byte, uint64](queryId).EndExclusive(uint64(timestamp.UnixMilli())).Descending()
	err = k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.Aggregate) (stop bool, err error) {
		if !value.Flagged && sdk.MustAccAddressFromBech32(value.AggregateReporter).Equals(reporter) {
			aggregate = &value
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return aggregate, nil
}
