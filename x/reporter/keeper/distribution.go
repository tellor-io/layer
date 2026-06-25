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

// ReturnSlashedTokens returns the slashed tokens to the delegators,
// called in dispute module after dispute is resolved with result invalid or reporter wins.
// It returns the per-pool amounts (bonded, unbonded) the dispute caller must move
// out of the dispute module so the bonded and not-bonded pools each receive exactly
// what was delegated into them. A single pool name was previously returned, which
// mis-routed the full slash amount to the last-selected pool when origins mixed bonded
// and unbonded destinations.
//
// Per-origin destination selection:
//   - missing original validator: scan for an under-cap bonded fallback (bonded pool,
//     cap-checked, contributes to bondedReturn);
//   - existing but not-bonded original (M4): refund to the original validator with
//     tokenSrc=Unbonded (not-bonded pool, not cap-checked, contributes to
//     unbondedReturn) — this does not create immediate active bonded stake;
//   - bonded original: preserve when under cap, otherwise scan to an under-cap
//     bonded fallback (bonded pool, cap-checked, contributes to bondedReturn).
func (k Keeper) ReturnSlashedTokens(ctx context.Context, amt math.Int, hashId []byte) (math.Int, math.Int, error) {
	bondedReturn := math.ZeroInt()
	unbondedReturn := math.ZeroInt()
	// get the snapshot of the metadata of the tokens that were slashed ie selectors' shares amounts and validator they were delegated to
	snapshot, err := k.DisputedDelegationAmounts.Get(ctx, hashId)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// capState threads projected bonded and per-validator deltas across origins.
	// The dispute caller moves bonded-pool coins only after this keeper returns, so
	// TotalBondedTokens and each validator.Tokens would otherwise drift apart
	// across origins and under-count a destination's post-refund share.
	capState := newValidatorPowerCapState()

	// winningpurse represents the amount of tokens that a disputed reporter possibly receives for winning a dispute
	winningpurse := amt.Sub(snapshot.Total)
	// for each selector-validator pair, bond the tokens back to the validator, if the validator still exists
	// if not, then find a bonded validator to bond the tokens to
	for _, source := range snapshot.TokenOrigins {
		valAddr := sdk.ValAddress(source.ValidatorAddress)
		delAddr := sdk.AccAddress(source.DelegatorAddress)

		// the refund amount is either the amount of tokens that were slashed
		// or the proportion of the slashed tokens plus the winning purse
		shareAmt := math.LegacyNewDecFromInt(source.Amount)
		if winningpurse.IsPositive() {
			// convert args needed for calculations to legacy decimals
			shareAmt = shareAmt.Quo(math.LegacyNewDecFromInt(snapshot.Total)).Mul(math.LegacyNewDecFromInt(amt))
		}
		bondAmt := shareAmt.TruncateInt()

		// Choose a refund destination per origin. Three cases:
		//   - missing original validator (ErrNoValidatorFound): the original is
		//     gone, so scan for an under-cap bonded fallback and refund through the
		//     bonded pool. This is the only path that creates immediate active
		//     bonded stake, so it is cap-checked.
		//   - existing but not-bonded original (M4): refund to the original
		//     validator itself with tokenSrc=Unbonded. This restores the
		//     not-bonded pool's coins to the not-bonded pool and does not create
		//     immediate active bonded stake, so the validator cap is intentionally
		//     not enforced here. unbondedReturn is incremented so the dispute
		//     caller routes these coins to the NotBondedPoolName.
		//   - bonded original: preserve when under cap, otherwise scan to an
		//     under-cap bonded fallback; refund through the bonded pool.
		original, getErr := k.stakingKeeper.GetValidator(ctx, valAddr)
		var (
			tokenSrc stakingtypes.BondStatus
			val      stakingtypes.Validator
		)
		switch {
		case getErr != nil && !errors.Is(getErr, stakingtypes.ErrNoValidatorFound):
			return math.ZeroInt(), math.ZeroInt(), getErr
		case errors.Is(getErr, stakingtypes.ErrNoValidatorFound):
			// missing original: scan for an under-cap bonded validator
			val, err = k.pickUnderCapBondedValidator(ctx, bondAmt, capState)
			if err != nil {
				return math.ZeroInt(), math.ZeroInt(), err
			}
			tokenSrc = stakingtypes.Bonded
			bondedReturn = bondedReturn.Add(bondAmt)
		case !original.IsBonded():
			// M4: existing but not-bonded original. Refund to the original
			// validator with tokenSrc=Unbonded so the not-bonded pool receives
			// exactly what was delegated into it. The validator cap is not
			// enforced because this does not create immediate active bonded
			// stake. capState is not touched because no bonded-pool acquisition
			// happens.
			val = original
			tokenSrc = stakingtypes.Unbonded
			unbondedReturn = unbondedReturn.Add(bondAmt)
		default:
			// bonded original: preserve when under cap, otherwise scan.
			val, err = k.bondedValidatorForDelegation(ctx, original, bondAmt, capState)
			if err != nil {
				return math.ZeroInt(), math.ZeroInt(), err
			}
			tokenSrc = stakingtypes.Bonded
			bondedReturn = bondedReturn.Add(bondAmt)
		}
		newShares, err := k.stakingKeeper.Delegate(ctx, delAddr, bondAmt, tokenSrc, val, false) // false means to not subtract tokens from an account
		if err != nil {
			return math.ZeroInt(), math.ZeroInt(), err
		}
		// record the bonded-pool acquisition so later origins in this call are
		// checked against the projected post-refund totals. Unbonded refunds do
		// not touch the bonded pool and are intentionally excluded here.
		if tokenSrc == stakingtypes.Bonded {
			capState.applyBondedDelegation(bondAmt)
		}
		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				"tokens_added_to_stake",
				sdk.NewAttribute("delegator", delAddr.String()),
				sdk.NewAttribute("validator", val.OperatorAddress),
				sdk.NewAttribute("shares", newShares.String()),
				sdk.NewAttribute("amount", shareAmt.String()),
			),
		})

	}
	return bondedReturn, unbondedReturn, k.DisputedDelegationAmounts.Remove(ctx, hashId)
}

// called in dispute module after dispute is resolved
// returns the fee to the selectors that passively paid minus the burn amount
// refunds the fee paid (minus the burned amount) from the stake to the selectors.
// It retrieves the tracked fees using the provided hashId and calculates the share
// of the amount to be refunded to each selector based on their contribution.
// If the validator associated with the selector is not found or not bonded, it
// selects a bonded validator to delegate the refund amount to.
func (k Keeper) FeeRefund(ctx context.Context, hashId []byte, amt math.Int) error {
	trackedFees, err := k.FeePaidFromStake.Get(ctx, hashId)
	if err != nil {
		return err
	}

	// capState threads projected bonded and per-validator deltas across origins;
	// see ReturnSlashedTokens for why the denominator must be projected.
	capState := newValidatorPowerCapState()

	for _, source := range trackedFees.TokenOrigins {
		// since fee paid is returned minus the voter/burned amount, calculate by accordingly
		// convert args needed for calculations to legacy decimals
		sourceAmountDec := math.LegacyNewDecFromInt(source.Amount)
		trackedFeesTotalDec := math.LegacyNewDecFromInt(trackedFees.Total)
		amtDec := math.LegacyNewDecFromInt(amt)
		shareAmt := sourceAmountDec.Mul(amtDec).Quo(trackedFeesTotalDec).TruncateInt()

		original, getErr := k.stakingKeeper.GetValidator(ctx, sdk.ValAddress(source.ValidatorAddress))
		var val stakingtypes.Validator
		switch {
		case getErr != nil && !errors.Is(getErr, stakingtypes.ErrNoValidatorFound):
			return getErr
		case errors.Is(getErr, stakingtypes.ErrNoValidatorFound) || !original.IsBonded():
			// missing or not-bonded original: scan for an under-cap bonded validator
			val, err = k.pickUnderCapBondedValidator(ctx, shareAmt, capState)
			if err != nil {
				return err
			}
		default:
			// bonded original: preserve when under cap, otherwise scan
			val, err = k.bondedValidatorForDelegation(ctx, original, shareAmt, capState)
			if err != nil {
				return err
			}
		}
		newShares, err := k.stakingKeeper.Delegate(ctx, sdk.AccAddress(source.DelegatorAddress), shareAmt, stakingtypes.Bonded, val, false)
		if err != nil {
			return err
		}
		capState.applyBondedDelegation(shareAmt)
		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				"tokens_added_to_stake",
				sdk.NewAttribute("delegator", sdk.AccAddress(source.DelegatorAddress).String()),
				sdk.NewAttribute("validator", val.OperatorAddress),
				sdk.NewAttribute("shares", newShares.String()),
				sdk.NewAttribute("amount", shareAmt.String()),
			),
		})
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

// TODO: this should be in dispute module, no reason for it to be in reporter module
// Stakes a given amount of tokens to a BONDED validator from a given address
// first looks up if given acount is delegated to a validator
// if they are delgated, then delegate winnings to that validator
// if not, then delegate winnings to a bonded validator
func (k Keeper) AddAmountToStake(ctx context.Context, acc sdk.AccAddress, amt math.Int) error {
	var (
		delegated   bool
		callbackErr error
	)

	// iterate through delegations to find if account is delegated to a bonded
	// validator; the iterator callback returns only stop, so capture the first
	// error in callbackErr (M3) and never swallow it.
	err := k.stakingKeeper.IterateDelegatorDelegations(ctx, acc, func(delegation stakingtypes.Delegation) (stop bool) {
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
			return false // continue iteration if not bonded
		}

		// bonded delegation found: preserve when under cap, otherwise scan.
		// AddAmountToStake delegates at most once (the iterator returns on the
		// first bonded delegation), so a fresh capState with no cross-origin
		// projection is correct here.
		dst, err := k.bondedValidatorForDelegation(ctx, val, amt, newValidatorPowerCapState())
		if err != nil {
			callbackErr = err
			return true
		}

		newShares, err := k.stakingKeeper.Delegate(ctx, acc, amt, stakingtypes.Bonded, dst, false)
		if err != nil {
			callbackErr = err
			return true
		}
		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				"tokens_added_to_stake",
				sdk.NewAttribute("delegator", acc.String()),
				sdk.NewAttribute("validator", dst.OperatorAddress),
				sdk.NewAttribute("amount", amt.String()),
				sdk.NewAttribute("shares", newShares.String()),
			),
		})
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

	// no bonded delegation found: scan for an under-cap bonded validator.
	// capState is fresh because AddAmountToStake makes a single decision for
	// one account; a projected delta across the iterator's earlier bonded
	// delegations is not needed (the iterator returns on the first bonded hit).
	capState := newValidatorPowerCapState()
	dst, err := k.pickUnderCapBondedValidator(ctx, amt, capState)
	if err != nil {
		return err
	}
	newShares, err := k.stakingKeeper.Delegate(ctx, acc, amt, stakingtypes.Bonded, dst, false)
	if err != nil {
		return err
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"tokens_added_to_stake",
			sdk.NewAttribute("delegator", acc.String()),
			sdk.NewAttribute("validator", dst.OperatorAddress),
			sdk.NewAttribute("amount", amt.String()),
			sdk.NewAttribute("shares", newShares.String()),
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
