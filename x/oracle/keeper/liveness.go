package keeper

import (
	"context"
	"errors"
	"fmt"

	layer "github.com/tellor-io/layer/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IncrementQueryOpportunities increments the opportunity count for a queryId
func (k Keeper) IncrementQueryOpportunities(ctx context.Context, queryId []byte) error {
	current, err := k.QueryOpportunities.Get(ctx, queryId)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		current = 0
	}
	return k.QueryOpportunities.Set(ctx, queryId, current+1)
}

// IncrementTotalQueriesInPeriod increments the count of cyclelist queries in the current period
func (k Keeper) IncrementTotalQueriesInPeriod(ctx context.Context) error {
	current, err := k.TotalQueriesInPeriod.Get(ctx)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		current = 0
	}
	return k.TotalQueriesInPeriod.Set(ctx, current+1)
}

// TrackReporterQuery records that a reporter reported on a specific queryId
func (k Keeper) TrackReporterQuery(ctx context.Context, reporter, queryId []byte) error {
	return k.ReporterQueriesInPeriod.Set(ctx, collections.Join(reporter, queryId), true)
}

// UpdateReporterLiveness updates a reporter's liveness record for the current period
// Called when processing cyclelist reports during aggregation
func (k Keeper) UpdateReporterLiveness(ctx context.Context, reporter, queryId []byte, power uint64) error {
	// Track which query this reporter reported on
	if err := k.TrackReporterQuery(ctx, reporter, queryId); err != nil {
		return err
	}

	// Update accumulated power (useful especially if reporter power changes mid-cycle)
	record, err := k.LivenessRecords.Get(ctx, reporter)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		// initialize new record
		record = types.LivenessRecord{
			QueriesReported:  0,
			AccumulatedPower: 0,
		}
	}

	record.QueriesReported++
	record.AccumulatedPower += power

	return k.LivenessRecords.Set(ctx, reporter, record)
}

// CheckAndDistributeLivenessRewards checks if a full cycle is complete and triggers distribution if needed
func (k Keeper) CheckAndDistributeLivenessRewards(ctx context.Context) error {
	// increment cycle count
	cycleCount, err := k.CycleCount.Next(ctx)
	if err != nil {
		return err
	}

	// get params to check LivenessCycles
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	// check if it's time to distribute (every N cycles)
	if cycleCount%params.LivenessCycles != 0 {
		return nil
	}

	// time to distribute liveness rewards
	return k.DistributeLivenessRewards(ctx)
}

// DistributeLivenessRewards distributes TBR based on liveness
func (k Keeper) DistributeLivenessRewards(ctx context.Context) error {
	// get current TBR balance (entire amount to distribute)
	tbr := k.GetTimeBasedRewards(ctx)

	// if no TBR to distribute, just reset and return
	if tbr.IsZero() {
		return k.ResetLivenessData(ctx)
	}

	// get cyclelist length for max liveness calculation
	cyclelist, err := k.GetCyclelist(ctx)
	if err != nil {
		return err
	}
	maxLiveness := math.LegacyNewDec(int64(len(cyclelist)))

	// if no cyclelist queries, reset and return
	if len(cyclelist) == 0 {
		return k.ResetLivenessData(ctx)
	}

	// collect all reporters and calculate their weighted liveness
	type reporterWeight struct {
		reporter []byte
		weight   math.LegacyDec
	}
	var reporters []reporterWeight
	totalWeight := math.LegacyZeroDec()

	// iterate through liveness records
	livenessIter, err := k.LivenessRecords.Iterate(ctx, nil)
	if err != nil {
		return err
	}
	defer livenessIter.Close()

	for ; livenessIter.Valid(); livenessIter.Next() {
		kv, err := livenessIter.KeyValue()
		if err != nil {
			return err
		}
		reporter := kv.Key
		record := kv.Value

		// calculate weighted liveness for this reporter
		// liveness = sum(1 / opportunities) for each query they reported on
		weightedLiveness := math.LegacyZeroDec()

		// iterate through queries this reporter reported on
		reporterQueryPrefix := collections.NewPrefixedPairRange[[]byte, []byte](reporter)
		queryIter, err := k.ReporterQueriesInPeriod.Iterate(ctx, reporterQueryPrefix)
		if err != nil {
			return err
		}

		for ; queryIter.Valid(); queryIter.Next() {
			queryKey, err := queryIter.Key()
			if err != nil {
				queryIter.Close()
				return err
			}
			queryId := queryKey.K2()

			// get opportunities for this query
			opportunities, err := k.QueryOpportunities.Get(ctx, queryId)
			if err != nil {
				queryIter.Close()
				return err
			}

			// add weighted contribution: 1 / opportunities
			if opportunities > 0 {
				contribution := math.LegacyOneDec().Quo(math.LegacyNewDec(int64(opportunities)))
				weightedLiveness = weightedLiveness.Add(contribution)
			}
		}
		queryIter.Close()

		// calculate final liveness ratio: weightedLiveness / maxLiveness
		livenessRatio := weightedLiveness.Quo(maxLiveness)

		// calculate weight: accumulatedPower × livenessRatio
		weight := math.LegacyNewDec(int64(record.AccumulatedPower)).Mul(livenessRatio)

		if weight.IsPositive() {
			reporters = append(reporters, reporterWeight{reporter: reporter, weight: weight})
			totalWeight = totalWeight.Add(weight)
		}
	}

	// if no reporters or no weight, reset and return
	if len(reporters) == 0 || totalWeight.IsZero() {
		return k.ResetLivenessData(ctx)
	}

	// transfer TBR from TimeBasedRewards to TipsEscrowPool
	err = k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		minttypes.TimeBasedRewards,
		reportertypes.TipsEscrowPool,
		sdk.NewCoins(sdk.NewCoin(layer.BondDenom, tbr)),
	)
	if err != nil {
		return err
	}

	// get dust from previous period
	dust, err := k.Dust.Get(ctx)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		dust = math.ZeroInt()
	}

	// add dust to TBR for distribution
	totalToDistribute := tbr.ToLegacyDec().Add(dust.ToLegacyDec())
	distributed := math.LegacyZeroDec()

	// distribute rewards based on weight
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	for _, r := range reporters {
		// reward = (weight / totalWeight) × totalToDistribute
		reward := r.weight.Quo(totalWeight).Mul(totalToDistribute)
		distributed = distributed.Add(reward)

		// allocate TBR to reporter
		err = k.AllocateTBR(ctx, r.reporter, reward)
		if err != nil {
			return err
		}
	}

	// calculate new dust (leftover from rounding)
	newDust := totalToDistribute.Sub(distributed)
	if newDust.IsPositive() {
		err = k.Dust.Set(ctx, newDust.TruncateInt())
		if err != nil {
			return err
		}
	} else {
		err = k.Dust.Set(ctx, math.ZeroInt())
		if err != nil {
			return err
		}
	}

	// emit event
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"liveness_rewards_distributed",
			sdk.NewAttribute("total_distributed", tbr.String()),
			sdk.NewAttribute("reporter_count", fmt.Sprintf("%d", len(reporters))),
		),
	})

	return k.ResetLivenessData(ctx)
}

// ResetLivenessData resets all liveness tracking data for the new period
func (k Keeper) ResetLivenessData(ctx context.Context) error {
	// reset total queries
	if err := k.TotalQueriesInPeriod.Set(ctx, 0); err != nil {
		return err
	}

	// clear all liveness records
	if err := k.LivenessRecords.Clear(ctx, nil); err != nil {
		return err
	}

	// clear query opportunities
	if err := k.QueryOpportunities.Clear(ctx, nil); err != nil {
		return err
	}

	// clear reporter queries in period
	if err := k.ReporterQueriesInPeriod.Clear(ctx, nil); err != nil {
		return err
	}

	return nil
}
