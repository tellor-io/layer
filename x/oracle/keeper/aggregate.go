package keeper

import (
	"encoding/hex"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	rk "github.com/tellor-io/layer/x/registry/keeper"
)

func (k Keeper) SetAggregatedReport(ctx sdk.Context) error {
	reportsStore := k.ReportsStore(ctx)
	currentHeight := ctx.BlockHeight()

	bz := reportsStore.Get(types.NumKey(currentHeight))
	var revealedReports types.Reports
	k.cdc.Unmarshal(bz, &revealedReports)

	reportMapping := make(map[string][]types.MicroReport)

	// sort by query id
	for _, s := range revealedReports.MicroReports {
		reportMapping[s.QueryId] = append(reportMapping[s.QueryId], *s)
	}
	reportersToPay := make([]*types.AggregateReporter, 0)
	for _, reports := range reportMapping {
		if reports[0].AggregateMethod == "weighted-median" {
			reportersToPay = append(reportersToPay, k.WeightedMedian(ctx, reports).Reporters...)
		}
		if reports[0].AggregateMethod == "weighted-mode" {
			reportersToPay = append(reportersToPay, k.WeightedMode(ctx, reports).Reporters...)
		}
	}
	// pay reporters, time based rewards
	k.AllocateTimeBasedRewards(ctx, reportersToPay)
	return nil
}

func (k Keeper) SetAggregate(ctx sdk.Context, report *types.Aggregate) {
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
	switch len(timestamps) {
	case 0:
		return false, 0

	case 1:
		if timestamps[0].Before(target) || timestamps[0].Equal(target) {
			return true, 0
		}
		return false, 0

	default:
		midIdx := len(timestamps) / 2
		midTimestamp := timestamps[midIdx]

		if target.Before(midTimestamp) {
			return FindTimestampBefore(timestamps[:midIdx], target)
		} else {
			if midIdx == len(timestamps)-1 || timestamps[midIdx+1].After(target) {
				return true, midIdx
			}
			found, idx := FindTimestampBefore(timestamps[midIdx+1:], target)
			if found {
				return true, midIdx + 1 + idx
			}
			return true, midIdx
		}
	}
}
