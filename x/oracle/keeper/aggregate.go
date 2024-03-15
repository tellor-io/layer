package keeper

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"
)

// SetAggregatedReport calculates and allocates rewards to reporters based on aggregated reports.
// at a specific blockchain height (to be ran in begin-blocker)
// It retrieves the revealed reports from the reports store, sorts them by query ID, and then
// calculates the aggregate report for each query using either the weighted-median or weighted-mode method.
// TODO: Add support for other aggregation methods.
// Rewards are allocated to the reporters based on the query tip amount, and time-based rewards are also
// allocated to the reporters.
func (k Keeper) SetAggregatedReport(ctx sdk.Context) error {
	// Get the current block height of the blockchain.
	currentHeight := ctx.BlockHeight()

	// Retrieve the stored reports for the current block height.
	iter, err := k.Reports.Indexes.BlockHeight.MatchExact(ctx, currentHeight)
	if err != nil {
		return err
	}

	kvs, err := indexes.CollectKeyValues(ctx, k.Reports, iter)
	if err != nil {
		return err
	}

	revealedReportsByQueryID := map[string][]types.MicroReport{}
	for _, kv := range kvs {
		// key is queryId, reporter
		qIdStr := hex.EncodeToString(kv.Key.K1())
		if _, ok := revealedReportsByQueryID[qIdStr]; !ok {
			revealedReportsByQueryID[qIdStr] = []types.MicroReport{kv.Value}
		} else {
			revealedReportsByQueryID[qIdStr] = append(revealedReportsByQueryID[qIdStr], kv.Value)
		}
	}

	// Prepare a list to keep track of reporters who are eligible for tbr.
	reportersToPay := make([]*types.AggregateReporter, 0)

	// Process each set of reports based on their aggregation method.
	for queryIdStr, reports := range revealedReportsByQueryID {
		// Handle weighted-median aggregation method.
		if reports[0].AggregateMethod == "weighted-median" {
			// Calculate the aggregated report.
			report, err := k.WeightedMedian(ctx, reports)
			if err != nil {
				return err
			}
			// Get the tip for this query.
			tip := k.GetQueryTip(ctx, []byte(queryIdStr))
			// Allocate rewards if there is a tip.
			if !tip.Amount.IsZero() {
				k.AllocateRewards(ctx, report.Reporters, tip)
			}
			// Add reporters to the tbr payment list.
			reportersToPay = append(reportersToPay, report.Reporters...)
		}

		// Handle weighted-mode aggregation method.
		if reports[0].AggregateMethod == "weighted-mode" {
			// Calculate the aggregated report.
			report := k.WeightedMode(ctx, reports)
			// Get the tip for this query.
			tip := k.GetQueryTip(ctx, []byte(queryIdStr))
			// Allocate rewards if there is a tip.
			if !tip.Amount.IsZero() {
				k.AllocateRewards(ctx, report.Reporters, tip)
			}
			// Add reporters to the tbr payment list.
			reportersToPay = append(reportersToPay, report.Reporters...)
		}
	}

	// Process time-based rewards for reporters.
	tbr := k.getTimeBasedRewards(ctx)
	// Allocate time-based rewards to all eligible reporters.
	k.AllocateRewards(ctx, reportersToPay, tbr)
	return nil
}

func (k Keeper) SetAggregate(ctx sdk.Context, report *types.Aggregate) {
	k.Logger(ctx).Info("@SetAggregate", "report", report)
	report.QueryId = regtypes.Remove0xPrefix(report.QueryId)
	queryId, err := utils.QueryIDFromString(report.QueryId)
	if err != nil {
		panic(err)
	}
	nonce, _ := k.Nonces.Get(ctx, queryId)
	nonce++
	err = k.Nonces.Set(ctx, queryId, nonce)
	if err != nil {
		panic(err)
	}
	report.Nonce = nonce // TODO: do we want to use int64 for nonce?

	currentTimestamp := ctx.BlockTime().Unix()
	report.Height = ctx.BlockHeight()

	// TODO: handle error
	err = k.Aggregates.Set(ctx, collections.Join(queryId, currentTimestamp), *report)
	if err != nil {
		panic(err)
	}

}

// getDataBefore returns the last aggregate before or at the given timestamp for the given query id.
// TODO: add a test for this function.
func (k Keeper) getDataBefore(ctx sdk.Context, queryId []byte, timestamp time.Time) (*types.Aggregate, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).EndInclusive(timestamp.Unix()).Descending()
	var mostRecent *types.Aggregate
	// This should get us the most recent aggregate, as they are walked in descending order
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		mostRecent = &value
		return true, nil
	})

	if err != nil {
		panic(err)
	}

	if mostRecent == nil {
		return nil, fmt.Errorf("no data before timestamp %v available for query id %s", timestamp, hex.EncodeToString(queryId))
	}

	return mostRecent, nil
}

func (k Keeper) GetCurrentValueForQueryId(ctx context.Context, queryId []byte) *types.Aggregate {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).Descending()
	var mostRecent *types.Aggregate
	// This should get us the most recent aggregate, as they are walked in descending order
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		mostRecent = &value
		return true, nil
	})

	if err != nil {
		panic(err)
	}

	return mostRecent
}

func (k Keeper) GetTimestampBefore(ctx sdk.Context, queryId []byte, timestamp time.Time) (time.Time, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).EndInclusive(timestamp.Unix()).Descending()
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
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).StartInclusive(timestamp.Unix())
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
