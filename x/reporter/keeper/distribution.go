package keeper

import (
	"context"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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

// ReturnSlashedTokens returns slashed tokens to delegators after a dispute resolves
// invalid or in the reporter's favor. Returns per-pool amounts (bonded, unbonded)
// the dispute caller must move out of the dispute module.
func (k Keeper) ReturnSlashedTokens(ctx context.Context, amt math.Int, hashId []byte) (math.Int, math.Int, error) {
	bondedReturn := math.ZeroInt()
	unbondedReturn := math.ZeroInt()
	snapshot, err := k.DisputedDelegationAmounts.Get(ctx, hashId)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	maxShare, err := k.maxValidatorPowerShare(ctx)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}
	projectedBondedDelta := math.ZeroInt()

	winningpurse := amt.Sub(snapshot.Total)
	for _, source := range snapshot.TokenOrigins {
		valAddr := sdk.ValAddress(source.ValidatorAddress)
		delAddr := sdk.AccAddress(source.DelegatorAddress)

		shareAmt := math.LegacyNewDecFromInt(source.Amount)
		if winningpurse.IsPositive() {
			shareAmt = shareAmt.Quo(math.LegacyNewDecFromInt(snapshot.Total)).Mul(math.LegacyNewDecFromInt(amt))
		}
		bondAmt := shareAmt.TruncateInt()

		dest, err := k.resolveDelegationDestination(ctx, valAddr, bondAmt, maxShare, projectedBondedDelta, routingPreserveOriginalPool)
		if err != nil {
			return math.ZeroInt(), math.ZeroInt(), err
		}
		if _, err := k.delegateToStakeAndEmit(ctx, delAddr, bondAmt, dest.TokenSrc, dest.Validator); err != nil {
			return math.ZeroInt(), math.ZeroInt(), err
		}
		if dest.TokenSrc == stakingtypes.Bonded {
			bondedReturn = bondedReturn.Add(bondAmt)
			projectedBondedDelta = projectedBondedDelta.Add(bondAmt)
		} else {
			unbondedReturn = unbondedReturn.Add(bondAmt)
		}
	}
	// route truncation dust to the bonded pool so callers never own dust policy
	if dust := amt.Sub(bondedReturn).Sub(unbondedReturn); dust.IsPositive() {
		bondedReturn = bondedReturn.Add(dust)
	}
	return bondedReturn, unbondedReturn, k.DisputedDelegationAmounts.Remove(ctx, hashId)
}

// delegateToStakeAndEmit delegates amount to validator and emits a tokens_added_to_stake event.
func (k Keeper) delegateToStakeAndEmit(ctx context.Context, delegator sdk.AccAddress, amount math.Int, tokenSrc stakingtypes.BondStatus, validator stakingtypes.Validator) (math.LegacyDec, error) {
	newShares, err := k.stakingKeeper.Delegate(ctx, delegator, amount, tokenSrc, validator, false)
	if err != nil {
		return math.LegacyZeroDec(), err
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"tokens_added_to_stake",
			sdk.NewAttribute("delegator", delegator.String()),
			sdk.NewAttribute("validator", validator.OperatorAddress),
			sdk.NewAttribute("shares", newShares.String()),
			sdk.NewAttribute("amount", amount.String()),
		),
	})
	return newShares, nil
}

// FeeRefund refunds the fee paid (minus the burned amount) from the stake to selectors.
func (k Keeper) FeeRefund(ctx context.Context, hashId []byte, amt math.Int) error {
	trackedFees, err := k.FeePaidFromStake.Get(ctx, hashId)
	if err != nil {
		return err
	}

	maxShare, err := k.maxValidatorPowerShare(ctx)
	if err != nil {
		return err
	}
	projectedBondedDelta := math.ZeroInt()

	for _, source := range trackedFees.TokenOrigins {
		shareAmt := math.LegacyNewDecFromInt(source.Amount).Mul(math.LegacyNewDecFromInt(amt)).Quo(math.LegacyNewDecFromInt(trackedFees.Total)).TruncateInt()

		dest, err := k.resolveDelegationDestination(ctx, sdk.ValAddress(source.ValidatorAddress), shareAmt, maxShare, projectedBondedDelta, routingForceBonded)
		if err != nil {
			return err
		}
		if _, err := k.delegateToStakeAndEmit(ctx, sdk.AccAddress(source.DelegatorAddress), shareAmt, dest.TokenSrc, dest.Validator); err != nil {
			return err
		}
		projectedBondedDelta = projectedBondedDelta.Add(shareAmt)
	}
	return k.FeePaidFromStake.Remove(ctx, hashId)
}

// GetBondedValidators returns a list of BONDED validators up to a given maximum number.
func (k Keeper) GetBondedValidators(ctx context.Context, max uint32) ([]stakingtypes.Validator, error) {
	validators := make([]stakingtypes.Validator, max)

	iterator, err := k.stakingKeeper.ValidatorsPowerStoreIterator(ctx)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	i := 0
	for ; iterator.Valid() && i < int(max); iterator.Next() {
		address := iterator.Value()
		validator, err := k.stakingKeeper.GetValidator(ctx, address)
		if err != nil {
			return nil, fmt.Errorf("validator record not found for address: %X", address)
		}

		if validator.IsBonded() {
			validators[i] = validator
			i++
		}
	}

	return validators[:i], nil // trim
}

// AddAmountToStake stakes amt to a bonded validator for acc. If acc is already
// delegated to a bonded validator, winnings go there; otherwise scan for one.
func (k Keeper) AddAmountToStake(ctx context.Context, acc sdk.AccAddress, amt math.Int) error {
	maxShare, err := k.maxValidatorPowerShare(ctx)
	if err != nil {
		return err
	}

	var (
		delegated   bool
		callbackErr error
	)

	// capture the first error in callbackErr since the iterator callback returns only stop
	err = k.stakingKeeper.IterateDelegatorDelegations(ctx, acc, func(delegation stakingtypes.Delegation) (stop bool) {
		if callbackErr != nil {
			return true
		}

		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			callbackErr = err
			return true
		}

		val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			callbackErr = err
			return true
		}

		if !val.IsBonded() {
			return false
		}

		// Single delegation per call; iterator stops at first bonded match, so no
		// cross-origin projectedBondedDelta accumulation is needed here.
		dst, err := k.bondedValidatorForDelegation(ctx, val, amt, maxShare, math.ZeroInt())
		if err != nil {
			callbackErr = err
			return true
		}
		if _, err := k.delegateToStakeAndEmit(ctx, acc, amt, stakingtypes.Bonded, dst); err != nil {
			callbackErr = err
			return true
		}
		delegated = true
		return true
	})
	if err != nil {
		return err
	}
	if callbackErr != nil {
		return callbackErr
	}
	if delegated {
		return nil
	}

	// no bonded delegation found: scan for an under-cap bonded validator
	dst, err := k.pickUnderCapBondedValidator(ctx, amt, maxShare, math.ZeroInt())
	if err != nil {
		return err
	}
	_, err = k.delegateToStakeAndEmit(ctx, acc, amt, stakingtypes.Bonded, dst)
	return err
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
