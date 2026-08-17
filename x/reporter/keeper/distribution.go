package keeper

import (
	"context"
	"errors"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DivvyingTips accumulates rewards for a reporter's current period.
// Actual distribution to selectors happens when:
// 1. The delegation state changes (queued via ReporterStake)
// 2. A selector calls WithdrawTip (forces settlement)
// 3. The distribution queue is processed in EndBlocker
//
// Commission is added directly to the reporter's SelectorTips.
// Net reward (after commission) accumulates in the period data.
func (k Keeper) DivvyingTips(ctx context.Context, reporterAddr sdk.AccAddress, reward math.LegacyDec) error {
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}

	// Calculate commission for the reporter
	commission := reward.Mul(reporter.CommissionRate)
	// Calculate net reward for selectors
	netReward := reward.Sub(commission)

	// Add commission directly to reporter's tips
	if commission.IsPositive() {
		oldTips, err := k.SelectorTips.Get(ctx, reporterAddr)
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return err
			}
			oldTips = math.LegacyZeroDec()
		}
		if err := k.SelectorTips.Set(ctx, reporterAddr, oldTips.Add(commission)); err != nil {
			return err
		}
	}

	// Accumulate net reward to current period
	periodData, err := k.ReporterPeriodData.Get(ctx, reporterAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// No period data yet - this shouldn't happen if ReporterStake was called first
			return nil
		}
		return err
	}

	periodData.RewardAmount = periodData.RewardAmount.Add(netReward)
	if err := k.ReporterPeriodData.Set(ctx, reporterAddr, periodData); err != nil {
		return err
	}

	// Emit event for reward accumulation
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"rewards_accumulated",
			sdk.NewAttribute("reporter", reporterAddr.String()),
			sdk.NewAttribute("commission", commission.String()),
			sdk.NewAttribute("net_reward", netReward.String()),
			sdk.NewAttribute("period_total", periodData.RewardAmount.String()),
		),
	})

	return nil
}

// ProcessDistributionQueue processes up to maxItems from the distribution queue.
// Each queue item distributes rewards to up to 100 selectors (the max per reporter).
func (k Keeper) ProcessDistributionQueue(ctx context.Context, maxItems int) error {
	counter, err := k.DistributionQueueCounter.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// No queue yet
			return nil
		}
		return err
	}

	processed := 0
	for counter.Head < counter.Tail && processed < maxItems {
		item, err := k.DistributionQueue.Get(ctx, counter.Head)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				// Item missing, skip
				counter.Head++
				continue
			}
			return err
		}

		// Distribute rewards to selectors
		if err := k.distributeQueueItem(ctx, item); err != nil {
			return err
		}

		// Remove from queue
		if err := k.DistributionQueue.Remove(ctx, counter.Head); err != nil {
			return err
		}

		counter.Head++
		processed++
	}

	return k.DistributionQueueCounter.Set(ctx, counter)
}

// distributeQueueItem distributes rewards from a queued period to its selectors.
func (k Keeper) distributeQueueItem(ctx context.Context, item types.DistributionQueueItem) error {
	if item.RewardAmount.IsZero() || item.Total.IsZero() {
		return nil
	}

	for _, sel := range item.Selectors {
		// Calculate selector's share: (selector_amount / total) * reward
		shareRatio := sel.Amount.ToLegacyDec().Quo(item.Total.ToLegacyDec())
		selectorReward := item.RewardAmount.Mul(shareRatio)

		if selectorReward.IsZero() {
			continue
		}

		// Add to selector's tips
		oldTips, err := k.SelectorTips.Get(ctx, sel.SelectorAddress)
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return err
			}
			oldTips = math.LegacyZeroDec()
		}

		newTips := oldTips.Add(selectorReward)
		if err := k.SelectorTips.Set(ctx, sel.SelectorAddress, newTips); err != nil {
			return err
		}

		// Emit event
		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				"rewards_distributed",
				sdk.NewAttribute("selector", sdk.AccAddress(sel.SelectorAddress).String()),
				sdk.NewAttribute("amount", selectorReward.String()),
				sdk.NewAttribute("total_tips", newTips.String()),
			),
		})
	}

	return nil
}

// SettleReporter forces settlement of a reporter's current period.
// Called when a selector wants to withdraw and needs their rewards settled first.
func (k Keeper) SettleReporter(ctx context.Context, reporterAddr sdk.AccAddress) error {
	periodData, err := k.ReporterPeriodData.Get(ctx, reporterAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// No period data, nothing to settle
			return nil
		}
		return err
	}

	// Only settle if there are rewards to distribute
	if !periodData.RewardAmount.IsPositive() {
		return nil
	}

	// Create queue item from current period
	item := types.DistributionQueueItem{
		Reporter:     reporterAddr,
		Selectors:    periodData.Selectors,
		Total:        periodData.Total,
		RewardAmount: periodData.RewardAmount,
	}

	// Distribute immediately (not queued)
	if err := k.distributeQueueItem(ctx, item); err != nil {
		return err
	}

	// Reset period reward amount
	periodData.RewardAmount = math.LegacyZeroDec()
	return k.ReporterPeriodData.Set(ctx, reporterAddr, periodData)
}
