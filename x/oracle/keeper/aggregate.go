package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tellor-io/layer/lib/metrics"
	layer "github.com/tellor-io/layer/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"
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
type rewards struct {
	aggregateReport types.Aggregate
	reward          math.LegacyDec
}

func (k Keeper) SetAggregatedReport(ctx context.Context) (err error) {
	// aggregate
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())
	cyclelist := make([]types.Aggregate, 0)
	transferAmt := math.ZeroInt()
	totalPowerForTbr := uint64(0)
	tipRewardAllocation := make(map[uint64]rewards)
	var tipRewardKeys []uint64
	// rng for queries that have expired and have revealed reports
	// ranger is inclusive and descending
	rng := collections.NewPrefixUntilPairRange[collections.Pair[bool, uint64], collections.Pair[[]byte, uint64]](collections.Join(true, blockHeight)).Descending()
	idsIterator, err := k.Query.Indexes.Expiration.Iterate(ctx, rng)
	if err != nil {
		return err
	}
	// no queries to aggregate ie no queries in the store
	if !idsIterator.Valid() {
		return nil
	}

	defer idsIterator.Close()
	for ; idsIterator.Valid(); idsIterator.Next() {
		fullKey, err := idsIterator.FullKey()
		if err != nil {
			return err
		}
		if !fullKey.K1().K1() {
			break
		}
		query, err := k.Query.Get(ctx, fullKey.K2())
		if err != nil {
			return err
		}

		aggregateReport, isCyclelist, err := k.AggregateReport(ctx, query.Id, query.QueryData)
		if err != nil {
			return err
		}

		if !query.Amount.IsZero() {
			tipRewardAllocation[query.Id] = rewards{aggregateReport: aggregateReport, reward: query.Amount.ToLegacyDec()}
			tipRewardKeys = append(tipRewardKeys, query.Id)
			transferAmt = transferAmt.Add(query.Amount)
		}

		if isCyclelist {
			totalPowerForTbr = aggregateReport.AggregatePower
			cyclelist = append(cyclelist, aggregateReport)
		}
		err = k.Query.Remove(ctx, fullKey.K2())
		if err != nil {
			return err
		}
	}
	if totalPowerForTbr > 0 {
		tbr := k.GetTimeBasedRewards(ctx)
		tipRewardAllocation, tipRewardKeys = k.DistributeTbr(
			ctx,
			tipRewardKeys,
			cyclelist,
			tipRewardAllocation,
			totalPowerForTbr,
			tbr,
		)
		err = k.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			minttypes.TimeBasedRewards,
			reportertypes.TipsEscrowPool,
			sdk.NewCoins(sdk.NewCoin(layer.BondDenom, tbr)),
		)
		if err != nil {
			return err
		}
	}
	err = k.DistributeRewards(ctx, tipRewardAllocation, tipRewardKeys)
	if err != nil {
		return err
	}
	if transferAmt.GT(math.ZeroInt()) {
		err = k.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.ModuleName,
			reportertypes.TipsEscrowPool,
			sdk.NewCoins(sdk.NewCoin(layer.BondDenom, transferAmt)),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetAggregate increments the queryId's report index plus sets the timestamp and blockHeight and stores the aggregate report
func (k Keeper) SetAggregate(ctx context.Context, report *types.Aggregate, queryData []byte, queryType string) error {
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

	// if bridge report, set in deposit queue
	if strings.EqualFold(queryType, TRBBridgeQueryType) {
		err = k.BridgeDepositQueue.Set(ctx, collections.Join(currentTimestamp, report.MetaId), queryData)
		if err != nil {
			return err
		}
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"aggregate_report",
			sdk.NewAttribute("query_id", hex.EncodeToString(report.QueryId)),
			sdk.NewAttribute("query_data", hex.EncodeToString(queryData)),
			sdk.NewAttribute("value", report.AggregateValue),
			sdk.NewAttribute("aggregate_power", strconv.FormatUint(report.AggregatePower, 10)),
			sdk.NewAttribute("micro_report_height", fmt.Sprintf("%d", report.MicroHeight)),
			sdk.NewAttribute("micro_report_type", queryType),
		),
	})
	telemetry.SetGaugeWithLabels([]string{"reporter_power_in_aggregates"}, float32(report.AggregatePower), []metrics.Label{{Name: "chain_id", Value: sdkCtx.ChainID()}, {Name: "query_id", Value: hex.EncodeToString(report.QueryId)}})
	telemetry.IncrCounterWithLabels([]string{"reports_in_aggregate", "aggregate_created"}, 1, []metrics.Label{{Name: "chain_id", Value: sdkCtx.ChainID()}, {Name: "query_id", Value: hex.EncodeToString(report.QueryId)}})
	return k.Aggregates.Set(ctx, collections.Join(report.QueryId, currentTimestamp), *report)
}

func (k Keeper) AggregateReport(ctx context.Context, id uint64, queryData []byte) (types.Aggregate, bool, error) {
	median, err := k.AggregateValue.Get(ctx, id)
	if err != nil {
		return types.Aggregate{}, false, err // return nil and log error ?
	}
	aggregateValue, err := k.Values.Get(ctx, collections.Join(id, median.Value))
	if err != nil {
		return types.Aggregate{}, false, err // return nil and log error ?
	}
	tPower, err := k.ValuesWeightSum.Get(ctx, id)
	if err != nil {
		// print error
		return types.Aggregate{}, false, err
	}

	microReport := aggregateValue.MicroReport
	aggregateReport := &types.Aggregate{
		QueryId:           microReport.QueryId,
		AggregateValue:    microReport.Value,
		AggregateReporter: microReport.Reporter,
		AggregatePower:    tPower,
		MicroHeight:       microReport.BlockNumber,
		MetaId:            id,
	}
	err = k.SetAggregate(ctx, aggregateReport, queryData, microReport.QueryType)
	if err != nil {
		return types.Aggregate{}, false, err
	}
	return *aggregateReport, microReport.Cyclelist, nil
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

func (k Keeper) GetAggregateAfter(ctx context.Context, queryId []byte, timestampAfter time.Time) (aggregate *types.Aggregate, timestamp time.Time, err error) {
	// Convert the timestampAfter to Unix time and create a range that starts just after this timestamp
	rng := collections.NewPrefixedPairRange[[]byte, uint64](queryId).StartExclusive(uint64(timestampAfter.UnixMilli()))

	var oldest *types.Aggregate
	var oldestTimestamp uint64

	err = k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.Aggregate) (stop bool, err error) {
		if !value.Flagged {
			oldest = &value
			oldestTimestamp = key.K2()
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, time.Time{}, err
	}

	if oldest == nil {
		return nil, time.Time{}, fmt.Errorf("no aggregate report found after timestamp %v for query id %s", timestampAfter, hex.EncodeToString(queryId))
	}

	// Convert the Unix timestamp back to time.Time
	timestamp = time.UnixMilli(int64(oldestTimestamp))
	return oldest, timestamp, nil
}

func (k Keeper) GetAggregateByTimestamp(ctx context.Context, queryId []byte, timestamp uint64) (types.Aggregate, error) {
	agg, err := k.Aggregates.Get(ctx, collections.Join(queryId, timestamp))
	if err != nil {
		return types.Aggregate{}, err
	}
	return agg, nil
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
