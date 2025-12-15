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
// called in dispute module after dispute is resolved with result invalid or reporter wins
func (k Keeper) ReturnSlashedTokens(ctx context.Context, amt math.Int, hashId []byte) (string, error) {
	var pool string
	// get the snapshot of the metadata of the tokens that were slashed ie selectors' shares amounts and validator they were delegated to
	snapshot, err := k.DisputedDelegationAmounts.Get(ctx, hashId)
	if err != nil {
		return "", err
	}

	// winningpurse represents the amount of tokens that a disputed reporter possibly receives for winning a dispute
	winningpurse := amt.Sub(snapshot.Total)
	// for each selector-validator pair, bond the tokens back to the validator, if the validator still exists
	// if not, then find a bonded validator to bond the tokens to
	for _, source := range snapshot.TokenOrigins {
		valAddr := sdk.ValAddress(source.ValidatorAddress)

		var val stakingtypes.Validator
		val, err = k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			if !errors.Is(err, stakingtypes.ErrNoValidatorFound) {
				return "", err
			}
			vals, err := k.GetBondedValidators(ctx, 1)
			if err != nil {
				return "", err
			}
			// this should never happen since there should always be a bonded validator
			if len(vals) == 0 {
				return "", errors.New("no validators found in staking module to return tokens to")
			}
			val = vals[0]
		}
		delAddr := sdk.AccAddress(source.DelegatorAddress)

		// the refund amount is either the amount of tokens that were slashed
		// or the proportion of the slashed tokens plus the winning purse
		shareAmt := math.LegacyNewDecFromInt(source.Amount)
		if winningpurse.IsPositive() {
			// convert args needed for calculations to legacy decimals
			shareAmt = shareAmt.Quo(math.LegacyNewDecFromInt(snapshot.Total)).Mul(math.LegacyNewDecFromInt(amt))
		}
		// set token source to bonded if validator is bonded
		// if not, set to unbonded
		// this causes the delegate method (in staking module) to not transfer tokens since tokens
		// are transferred via dispute module where ReturnSlashedTokens is called
		var tokenSrc stakingtypes.BondStatus
		if val.IsBonded() {
			tokenSrc = stakingtypes.Bonded
			pool = stakingtypes.BondedPoolName
		} else {
			tokenSrc = stakingtypes.Unbonded
			pool = stakingtypes.NotBondedPoolName
		}
		newShares, err := k.stakingKeeper.Delegate(ctx, delAddr, shareAmt.TruncateInt(), tokenSrc, val, false) // false means to not subtract tokens from an account
		if err != nil {
			return "", err
		}
		// TODO: emit event for each delegation
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
	return pool, k.DisputedDelegationAmounts.Remove(ctx, hashId)
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

	for _, source := range trackedFees.TokenOrigins {
		val, err := k.stakingKeeper.GetValidator(ctx, sdk.ValAddress(source.ValidatorAddress))
		if err != nil {
			if !errors.Is(err, stakingtypes.ErrNoValidatorFound) {
				return err
			}
			vals, err := k.GetBondedValidators(ctx, 1)
			if err != nil {
				return err
			}
			if len(vals) == 0 {
				return errors.New("no validators found in staking module to return tokens to")
			}
			val = vals[0]
		} else if !val.IsBonded() {
			vals, err := k.GetBondedValidators(ctx, 1)
			if err != nil {
				return err
			}
			val = vals[0]
		}
		// since fee paid is returned minus the voter/burned amount, calculate by accordingly
		// convert args needed for calculations to legacy decimals
		sourceAmountDec := math.LegacyNewDecFromInt(source.Amount)
		trackedFeesTotalDec := math.LegacyNewDecFromInt(trackedFees.Total)
		amtDec := math.LegacyNewDecFromInt(amt)
		shareAmtDec := sourceAmountDec.Mul(amtDec).Quo(trackedFeesTotalDec)
		shareAmt := shareAmtDec.TruncateInt()
		newShares, err := k.stakingKeeper.Delegate(ctx, sdk.AccAddress(source.DelegatorAddress), shareAmt, stakingtypes.Bonded, val, false)
		if err != nil {
			return err
		}
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
	// Flag to check if the account is already delegated to a bonded validator
	isDelegated := false

	// iterate through delegations to find if account is delegated to a bonded validator
	err := k.stakingKeeper.IterateDelegatorDelegations(ctx, acc, func(delegation stakingtypes.Delegation) (stop bool) {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			return true
		}
		val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			return true
		}
		// if val is bonded, delegate winnings to that validator and exit iteration
		if val.IsBonded() {
			isDelegated = true
			newShares, err := k.stakingKeeper.Delegate(ctx, acc, amt, stakingtypes.Bonded, val, false)
			if err != nil {
				return true
			}
			sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
				sdk.NewEvent(
					"tokens_added_to_stake",
					sdk.NewAttribute("delegator", acc.String()),
					sdk.NewAttribute("validator", val.OperatorAddress),
					sdk.NewAttribute("amount", amt.String()),
					sdk.NewAttribute("shares", newShares.String()),
				),
			})
			return true
		}
		return false // continue iteration if not bonded
	})
	if err != nil {
		return err
	}

	// if account is not delegated to any bonded validator, then delegate winnings to a bonded validator
	if !isDelegated {
		vals, err := k.GetBondedValidators(ctx, 1)
		if err != nil {
			return err
		}
		validator := vals[0]
		newShares, err := k.stakingKeeper.Delegate(ctx, acc, amt, stakingtypes.Bonded, validator, false)
		if err != nil {
			return err
		}
		// TODO: emit event for delegation
		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				"tokens_added_to_stake",
				sdk.NewAttribute("delegator", acc.String()),
				sdk.NewAttribute("validator", validator.OperatorAddress),
				sdk.NewAttribute("amount", amt.String()),
				sdk.NewAttribute("shares", newShares.String()),
			),
		})
	}

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
