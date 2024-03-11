package keeper

import (
	"encoding/hex"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	rk "github.com/tellor-io/layer/x/registry/keeper"
)

// SetAggregatedReport calculates and allocates rewards to reporters based on aggregated reports.
// at a specific blockchain height (to be ran in begin-blocker)
// It retrieves the revealed reports from the reports store, sorts them by query ID, and then
// calculates the aggregate report for each query using either the weighted-median or weighted-mode method.
// TODO: Add support for other aggregation methods.
// Rewards are allocated to the reporters based on the query tip amount, and time-based rewards are also
// allocated to the reporters.
func (k Keeper) SetAggregatedReport(ctx sdk.Context) error {
	// Access the store that holds reports.
	reportsStore := k.ReportsStore(ctx)
	// Get the current block height of the blockchain.
	currentHeight := ctx.BlockHeight()

	// Retrieve the stored reports for the current block height.
	bz := reportsStore.Get(types.NumKey(currentHeight))
	var revealedReports types.Reports
	k.cdc.Unmarshal(bz, &revealedReports)

	// Initialize a map to organize reports by their query ID.
	reportMapping := make(map[string][]types.MicroReport)

	// Sort the micro reports by their query ID.
	for _, s := range revealedReports.MicroReports {
		reportMapping[s.QueryId] = append(reportMapping[s.QueryId], *s)
	}

	// Prepare a list to keep track of reporters who are eligible for tbr.
	reportersToPay := make([]*types.AggregateReporter, 0)

	// Process each set of reports based on their aggregation method.
	for _, reports := range reportMapping {
		// Handle weighted-median aggregation method.
		if reports[0].AggregateMethod == "weighted-median" {
			// Calculate the aggregated report.
			report := k.WeightedMedian(ctx, reports)
			// Get the tip for this query.
			tip := k.GetQueryTip(ctx, report.QueryId)
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
			tip := k.GetQueryTip(ctx, report.QueryId)
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
	if rk.Has0xPrefix(report.QueryId) {
		report.QueryId = report.QueryId[2:]
	}
	queryId, err := hex.DecodeString(report.QueryId)
	if err != nil {
		panic(err)
	}
	nonce := k.GetMaxNonceForQueryId(ctx, queryId)
	nonce++
	report.Nonce = nonce
	currentTimestamp := ctx.BlockTime()
	k.SetMaxNonceForQueryId(ctx, queryId, nonce)
	k.AppendAvailableTimestamps(ctx, queryId, currentTimestamp)
	key := types.AggregateKey(queryId, currentTimestamp)
	store := k.AggregateStore(ctx)
	store.Set(key, k.cdc.MustMarshal(report))
	k.SetQueryIdAndTimestampPairByBlockHeight(ctx, report.QueryId, currentTimestamp)
}

func (k Keeper) getDataBefore(ctx sdk.Context, queryId []byte, timestamp time.Time) (*types.Aggregate, error) {
	availableTimestamps := k.GetAvailableTimestampsByQueryId(ctx, queryId)
	if len(availableTimestamps.Timestamps) == 0 {
		return nil, fmt.Errorf("no data available for query id %s", hex.EncodeToString(queryId))
	}
	found, index := FindTimestampBefore(availableTimestamps.Timestamps, timestamp)
	if found {
		key := types.AggregateKey(queryId, availableTimestamps.Timestamps[index])
		store := k.AggregateStore(ctx)
		bz := store.Get(key)
		var report types.Aggregate
		k.cdc.MustUnmarshal(bz, &report)
		return &report, nil
	}
	return nil, fmt.Errorf("no data before timestamp %v available for query id %s", timestamp, hex.EncodeToString(queryId))
}

func (k Keeper) GetAvailableTimestampsByQueryId(ctx sdk.Context, queryId []byte) types.AvailableTimestamps {
	store := k.AggregateStore(ctx)
	key := types.AvailableTimestampsKey(queryId)
	bz := store.Get(key)

	var availableTimestamps types.AvailableTimestamps
	k.cdc.MustUnmarshal(bz, &availableTimestamps)

	return availableTimestamps
}

func (k Keeper) AppendAvailableTimestamps(ctx sdk.Context, queryId []byte, timestamp time.Time) {
	store := k.AggregateStore(ctx)
	key := types.AvailableTimestampsKey(queryId)
	availableTimestamps := k.GetAvailableTimestampsByQueryId(ctx, queryId)

	availableTimestamps.Timestamps = append(availableTimestamps.Timestamps, timestamp)
	store.Set(key, k.cdc.MustMarshal(&availableTimestamps))

}

func (k Keeper) GetMaxNonceForQueryId(ctx sdk.Context, queryId []byte) int64 {
	store := k.AggregateStore(ctx)
	key := types.MaxNonceKey(queryId)
	bz := store.Get(key)
	if bz == nil {
		return 0
	}

	maxNonce := int64(sdk.BigEndianToUint64(bz))

	return maxNonce
}

func (k Keeper) SetMaxNonceForQueryId(ctx sdk.Context, queryId []byte, maxNonce int64) {
	store := k.AggregateStore(ctx)
	key := types.MaxNonceKey(queryId)
	store.Set(key, sdk.Uint64ToBigEndian(uint64(maxNonce)))
}

func (k Keeper) GetCurrentValueForQueryId(ctx sdk.Context, queryId []byte) *types.Aggregate {
	availableTimestamps := k.GetAvailableTimestampsByQueryId(ctx, queryId)
	if len(availableTimestamps.Timestamps) == 0 {
		return nil
	}
	mostRecentTimestamp := availableTimestamps.Timestamps[len(availableTimestamps.Timestamps)-1]
	key := types.AggregateKey(queryId, mostRecentTimestamp)

	store := k.AggregateStore(ctx)
	bz := store.Get(key)
	var report types.Aggregate
	k.cdc.MustUnmarshal(bz, &report)
	return &report
}

func FindTimestampBefore(timestamps []time.Time, target time.Time) (bool, int) {
	left, right := 0, len(timestamps)-1
	resultIndex := -1
	for left <= right {
		mid := left + (right-left)/2
		if timestamps[mid].Before(target) {
			resultIndex = mid
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return resultIndex != -1, resultIndex
}

func FindTimestampAfter(timestamps []time.Time, target time.Time) (bool, int) {
	left, right := 0, len(timestamps)-1
	resultIndex := -1
	for left <= right {
		mid := left + (right-left)/2
		if timestamps[mid].After(target) {
			resultIndex = mid
			right = mid - 1 // Search in the left half
		} else {
			left = mid + 1 // Search in the right half
		}
	}
	if resultIndex != -1 {
		return true, resultIndex
	}
	return false, -1
}

func (k Keeper) GetAggregateReport(ctx sdk.Context, queryId []byte, timestamp time.Time) (*types.Aggregate, error) {
	key := types.AggregateKey(queryId, timestamp)
	store := k.AggregateStore(ctx)
	bz := store.Get(key)
	var report types.Aggregate
	k.cdc.MustUnmarshal(bz, &report)
	return &report, nil
}

func (k Keeper) GetTimestampBefore(ctx sdk.Context, queryId []byte, timestamp time.Time) (time.Time, error) {
	availableTimestamps := k.GetAvailableTimestampsByQueryId(ctx, queryId)
	if len(availableTimestamps.Timestamps) == 0 {
		return time.Time{}, fmt.Errorf("no data available for query id %s", hex.EncodeToString(queryId))
	}
	found, index := FindTimestampBefore(availableTimestamps.Timestamps, timestamp)
	if found {
		return availableTimestamps.Timestamps[index], nil
	}
	return time.Time{}, fmt.Errorf("no data before timestamp %v available for query id %s", timestamp, hex.EncodeToString(queryId))
}

func (k Keeper) GetTimestampAfter(ctx sdk.Context, queryId []byte, timestamp time.Time) (time.Time, error) {
	availableTimestamps := k.GetAvailableTimestampsByQueryId(ctx, queryId)
	if len(availableTimestamps.Timestamps) == 0 {
		return time.Time{}, fmt.Errorf("no data available for query id %s", hex.EncodeToString(queryId))
	}
	found, index := FindTimestampAfter(availableTimestamps.Timestamps, timestamp)
	if found {
		return availableTimestamps.Timestamps[index], nil
	}
	return time.Time{}, fmt.Errorf("no data after timestamp %v available for query id %s", timestamp, hex.EncodeToString(queryId))
}
