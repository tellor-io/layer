package keeper

import (
	"bytes"
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

// TRBBridgeMarkerQueryId is a special queryId used to track all TRBBridge
// queries as a single "slot" for TBR distribution. All TRBBridge aggregates share
// one slot's worth of rewards, split proportionally among reporters.
// TRBBridge is always treated as non-standard.
var TRBBridgeMarkerQueryId = []byte("TRBBridge")

// TrackLivenessForAggregate iterates through all reporters in an aggregate and tracks their liveness
// Called at aggregation time when we know of who reported and with what power
func (k Keeper) TrackLivenessForAggregate(ctx context.Context, aggregate types.Aggregate) error {
	return k.TrackLivenessForAggregateWithQueryId(ctx, aggregate, aggregate.QueryId)
}

// TrackLivenessForTRBBridge tracks liveness for TRBBridge queries
func (k Keeper) TrackLivenessForTRBBridge(ctx context.Context, aggregate types.Aggregate) error {
	// TRBBridge is marked as non-standard
	if err := k.NonStandardQueries.Set(ctx, TRBBridgeMarkerQueryId, true); err != nil {
		return err
	}
	return k.TrackLivenessForAggregateWithQueryId(ctx, aggregate, TRBBridgeMarkerQueryId)
}

// TrackLivenessForAggregateWithQueryId tracks liveness for an aggregate given a queryId
func (k Keeper) TrackLivenessForAggregateWithQueryId(ctx context.Context, aggregate types.Aggregate, queryIdForTracking []byte) error {
	isNonStandard, err := k.NonStandardQueries.Has(ctx, queryIdForTracking)
	if err != nil {
		return err
	}

	// Iterate through all micro-reports in this aggregate
	iter, err := k.Reports.Indexes.IdQueryId.MatchExact(ctx, collections.Join(aggregate.MetaId, aggregate.QueryId))
	if err != nil {
		return err
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		reportKey, err := iter.PrimaryKey()
		if err != nil {
			return err
		}

		report, err := k.Reports.Get(ctx, reportKey)
		if err != nil {
			return err
		}

		reporter := reportKey.K2() // reporter address bytes

		// Update liveness with both reporter power and aggregate total power
		err = k.UpdateReporterLiveness(ctx, reporter, queryIdForTracking, report.Power, aggregate.AggregatePower, isNonStandard)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateReporterLiveness updates a reporter's liveness record for the current period
// Called at aggregation time when we know both the reporter's power and aggregate total power
func (k Keeper) UpdateReporterLiveness(ctx context.Context, reporter, queryId []byte, reporterPower, aggregateTotalPower uint64, isNonStandard bool) error {
	if aggregateTotalPower == 0 {
		return nil
	}

	// Calculate share: reporterPower / aggregateTotalPower
	share := math.LegacyNewDec(int64(reporterPower)).Quo(math.LegacyNewDec(int64(aggregateTotalPower)))

	// Always store per-query share (needed for potential demotion and non-standard calculation)
	if err := k.AddReporterQueryShareSum(ctx, reporter, queryId, reporterPower, aggregateTotalPower); err != nil {
		return err
	}

	// If standard query, also add to the fast-path standard share sum
	if !isNonStandard {
		currentStandard, err := k.ReporterStandardShareSum.Get(ctx, reporter)
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return err
			}
			currentStandard = math.LegacyZeroDec()
		}
		if err := k.ReporterStandardShareSum.Set(ctx, reporter, currentStandard.Add(share)); err != nil {
			return err
		}
	}

	return nil
}

// AddReporterQueryShareSum adds a power share for a reporter on a specific queryId
// share = reporterPower / aggregateTotalPower (stored as LegacyDec)
func (k Keeper) AddReporterQueryShareSum(ctx context.Context, reporter, queryId []byte, reporterPower, aggregateTotalPower uint64) error {
	if aggregateTotalPower == 0 {
		return nil
	}

	key := collections.Join(reporter, queryId)

	// reporterPower / aggregateTotalPower
	share := math.LegacyNewDec(int64(reporterPower)).Quo(math.LegacyNewDec(int64(aggregateTotalPower)))

	// Get current sum
	currentSum, err := k.ReporterQueryShareSum.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		currentSum = math.LegacyZeroDec()
	}

	// Add the new share to the sum
	return k.ReporterQueryShareSum.Set(ctx, key, currentSum.Add(share))
}

// IncrementQueryOpportunities increments the opportunity count for a queryId
// For non-standard queries, this tracks their individual opportunity count
// For standard queries, this is still called but the count is used for demotion detection
func (k Keeper) IncrementQueryOpportunities(ctx context.Context, queryId []byte) error {
	current, err := k.QueryOpportunities.Get(ctx, queryId)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		current = 0
	}

	// Set the new opportunity count
	if err := k.QueryOpportunities.Set(ctx, queryId, current+1); err != nil {
		return err
	}

	return nil
}

// IncrementStandardOpportunities increments the standard opportunities count
// Called when a cycle completes (all standard queries get one more opportunity)
func (k Keeper) IncrementStandardOpportunities(ctx context.Context) error {
	current, err := k.StandardOpportunities.Get(ctx)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		current = 0
	}
	return k.StandardOpportunities.Set(ctx, current+1)
}

// DemoteQueryToNonStandard moves a query from standard to non-standard
// This is called when an out-of-turn tip happens for a cyclelist query
// It moves all existing shares for this query from ReporterStandardShareSum to ReporterQueryShareSum
func (k Keeper) DemoteQueryToNonStandard(ctx context.Context, queryId []byte) error {
	// Check if already non-standard
	isNonStandard, err := k.NonStandardQueries.Has(ctx, queryId)
	if err != nil {
		return err
	}
	if isNonStandard {
		return nil
	}

	if err := k.NonStandardQueries.Set(ctx, queryId, true); err != nil {
		return err
	}

	// Get current standard opportunities to set as the query's base opportunities
	standardOpp, err := k.StandardOpportunities.Get(ctx)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		standardOpp = 0
	}

	// Set the query's opportunities to current standard + 1 (for this new tip)
	// Note: The caller (msg_server_tip) will also call IncrementQueryOpportunities,
	// so we set to standardOpp here, and it will become standardOpp+1 after
	if err := k.QueryOpportunities.Set(ctx, queryId, standardOpp); err != nil {
		return err
	}

	// Move existing shares from standard to non-standard
	// We need to iterate all reporters who have shares in this query
	// and subtract from their standard sum
	// TODO: Consider flipping the key structure so we can more easily query by queryId
	// rng := collections.NewPrefixedPairRange[[]byte, []byte](queryId)
	iter, err := k.ReporterQueryShareSum.Iterate(ctx, nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		kv, err := iter.KeyValue()
		if err != nil {
			return err
		}

		reporter := kv.Key.K1()
		entryQueryId := kv.Key.K2()

		// Check if this entry is for the query we're demoting
		if !bytes.Equal(entryQueryId, queryId) {
			continue
		}

		shareSum := kv.Value

		// Subtract from reporter's standard share sum
		currentStandard, err := k.ReporterStandardShareSum.Get(ctx, reporter)
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return err
			}
			currentStandard = math.LegacyZeroDec()
		}

		newStandard := currentStandard.Sub(shareSum)
		if newStandard.IsNegative() {
			newStandard = math.LegacyZeroDec()
		}

		if err := k.ReporterStandardShareSum.Set(ctx, reporter, newStandard); err != nil {
			return err
		}
	}

	return nil
}

// CheckAndDistributeLivenessRewards checks if a full cycle is complete and triggers distribution if needed
func (k Keeper) CheckAndDistributeLivenessRewards(ctx context.Context) error {
	// Increment standard opportunities (all standard queries get one more opportunity per cycle)
	if err := k.IncrementStandardOpportunities(ctx); err != nil {
		return err
	}

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

// DistributeLivenessRewards distributes TBR based on per-aggregate power share
// O(reporters) for standard queries + O(reporters × non_standard_queries) for non-standard
// Formula: reward = (shareSum / opportunities) * (TBR / numSlots)
func (k Keeper) DistributeLivenessRewards(ctx context.Context) error {
	// get current TBR balance (entire amount to distribute)
	tbr := k.GetTimeBasedRewards(ctx)

	// if no TBR to distribute, just reset and return
	if tbr.IsZero() {
		return k.ResetLivenessData(ctx)
	}

	cyclelist, err := k.GetCyclelist(ctx)
	if err != nil {
		return err
	}

	// Check if TRBBridge has opportunities (gets 1 slot if there are any TRBBridge reports)
	trbBridgeOpportunities, err := k.QueryOpportunities.Get(ctx, TRBBridgeMarkerQueryId)
	hasTRBBridge := err == nil && trbBridgeOpportunities > 0

	// Calculate total slots: cyclelist queries + TRBBridge (if active)
	numSlots := int64(len(cyclelist))
	if hasTRBBridge {
		numSlots++
	}

	// if no slots, reset and return
	if numSlots == 0 {
		return k.ResetLivenessData(ctx)
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

	// rewardPerSlot = TBR / numSlots (cyclelist queries + TRBBridge if active)
	rewardPerSlot := totalToDistribute.Quo(math.LegacyNewDec(numSlots))

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get standard opportunities
	standardOpportunities, err := k.StandardOpportunities.Get(ctx)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		standardOpportunities = 0
	}

	// Collect all non-standard queryIds and their opportunities
	nonStandardQueries := make(map[string]uint64)
	nonStandardIter, err := k.NonStandardQueries.Iterate(ctx, nil)
	if err != nil {
		return err
	}
	defer nonStandardIter.Close()
	for ; nonStandardIter.Valid(); nonStandardIter.Next() {
		queryId, err := nonStandardIter.Key()
		if err != nil {
			return err
		}

		opp, err := k.QueryOpportunities.Get(ctx, queryId)
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return err
			}
			opp = 0
		}
		if opp > 0 {
			nonStandardQueries[string(queryId)] = opp
		}
	}

	// Build reporter rewards map
	reporterRewards := make(map[string]math.LegacyDec)

	// FAST PATH: Process standard shares (O(reporters))
	if standardOpportunities > 0 {
		standardIter, err := k.ReporterStandardShareSum.Iterate(ctx, nil)
		if err != nil {
			return err
		}
		defer standardIter.Close()
		for ; standardIter.Valid(); standardIter.Next() {
			kv, err := standardIter.KeyValue()
			if err != nil {
				return err
			}
			reporter := kv.Key
			standardShareSum := kv.Value

			// reward = (standardShareSum / standardOpportunities) * rewardPerSlot
			reward := standardShareSum.
				Quo(math.LegacyNewDec(int64(standardOpportunities))).
				Mul(rewardPerSlot)

			reporterKey := string(reporter)
			if existing, ok := reporterRewards[reporterKey]; ok {
				reporterRewards[reporterKey] = existing.Add(reward)
			} else {
				reporterRewards[reporterKey] = reward
			}
		}
	}

	// SLOW PATH: Process non-standard queries (O(reporters × non_standard_queries))
	if len(nonStandardQueries) > 0 {
		shareIter, err := k.ReporterQueryShareSum.Iterate(ctx, nil)
		if err != nil {
			return err
		}
		for ; shareIter.Valid(); shareIter.Next() {
			kv, err := shareIter.KeyValue()
			if err != nil {
				shareIter.Close()
				return err
			}

			reporter := kv.Key.K1()
			queryId := kv.Key.K2()
			shareSum := kv.Value

			// Check if this query is non-standard
			opportunities, isNonStandard := nonStandardQueries[string(queryId)]
			if !isNonStandard || opportunities == 0 {
				continue
			}

			// reward = (shareSum / opportunities) * rewardPerSlot
			reward := shareSum.
				Quo(math.LegacyNewDec(int64(opportunities))).
				Mul(rewardPerSlot)

			reporterKey := string(reporter)
			if existing, ok := reporterRewards[reporterKey]; ok {
				reporterRewards[reporterKey] = existing.Add(reward)
			} else {
				reporterRewards[reporterKey] = reward
			}
		}
		shareIter.Close()
	}

	reporterCount := 0

	// Distribute rewards to each reporter
	for reporterKey, reward := range reporterRewards {
		reporter := []byte(reporterKey)

		if reward.IsPositive() {
			err = k.AllocateTBR(ctx, reporter, reward)
			if err != nil {
				return err
			}
			distributed = distributed.Add(reward)
			reporterCount++
		}
	}

	// Now transfer TBR from TimeBasedRewards to TipsEscrowPool
	err = k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		minttypes.TimeBasedRewards,
		reportertypes.TipsEscrowPool,
		sdk.NewCoins(sdk.NewCoin(layer.BondDenom, tbr)),
	)
	if err != nil {
		return err
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
			sdk.NewAttribute("reporter_count", fmt.Sprintf("%d", reporterCount)),
			sdk.NewAttribute("standard_opportunities", fmt.Sprintf("%d", standardOpportunities)),
			sdk.NewAttribute("non_standard_queries", fmt.Sprintf("%d", len(nonStandardQueries))),
		),
	})

	return k.ResetLivenessData(ctx)
}

// ResetLivenessData resets all liveness tracking data for the new period
func (k Keeper) ResetLivenessData(ctx context.Context) error {
	// clear query opportunities
	if err := k.QueryOpportunities.Clear(ctx, nil); err != nil {
		return err
	}

	// clear reporter query share sums
	if err := k.ReporterQueryShareSum.Clear(ctx, nil); err != nil {
		return err
	}

	// clear reporter standard share sums
	if err := k.ReporterStandardShareSum.Clear(ctx, nil); err != nil {
		return err
	}

	// clear non-standard queries
	if err := k.NonStandardQueries.Clear(ctx, nil); err != nil {
		return err
	}

	// reset standard opportunities
	if err := k.StandardOpportunities.Set(ctx, 0); err != nil {
		return err
	}

	return nil
}
