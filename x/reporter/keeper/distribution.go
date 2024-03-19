package keeper

import (
	"context"
	"errors"
	"fmt"
	gomath "math"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/reporter/types"
)

// WithdrawReporterCommission withdraws the accumulated commission of a reporter.
// It fetches the reporter's accumulated commission from the storage and checks if it is zero.
// If the commission is zero, it returns an error.
// Otherwise, it truncates the commission and updates the remainder in the storage for later withdrawal.
// It then updates the outstanding rewards by subtracting the commission from the reporter's rewards.
// If the commission is not zero, it sends the commission coins from the module to the reporter's account.
// Finally, it emits an event to indicate the successful withdrawal of the commission.
// Returns the withdrawn commission coins and any error encountered.
func (k Keeper) WithdrawReporterCommission(ctx context.Context, reporterVal sdk.ValAddress) (sdk.Coins, error) {
	// fetch reporter accumulated commission
	accumCommission, err := k.ReportersAccumulatedCommission.Get(ctx, reporterVal)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}

	if accumCommission.Commission.IsZero() {
		return nil, types.ErrNoReporterCommission
	}

	commission, remainder := accumCommission.Commission.TruncateDecimal()
	err = k.ReportersAccumulatedCommission.Set(ctx, reporterVal, types.ReporterAccumulatedCommission{Commission: remainder}) // leave remainder to withdraw later
	if err != nil {
		return nil, err
	}
	// update outstanding
	outstanding, err := k.ReporterOutstandingRewards.Get(ctx, reporterVal)
	if err != nil {
		return nil, err
	}

	err = k.ReporterOutstandingRewards.Set(ctx, reporterVal, types.ReporterOutstandingRewards{Rewards: outstanding.Rewards.Sub(sdk.NewDecCoinsFromCoins(commission...))})
	if err != nil {
		return nil, err
	}

	if !commission.IsZero() {

		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, reporterVal.Bytes(), commission)
		if err != nil {
			return nil, err
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeWithdrawCommission,
			sdk.NewAttribute(sdk.AttributeKeyAmount, commission.String()),
		),
	)

	return commission, nil
}

// AllocateTokensToReporter allocate tokens to a particular reporter,
// splitting according to commission.
// AllocateTokensToReporter allocates tokens to a reporter and updates the commission, rewards, and outstanding rewards.
// It splits the tokens between the reporter and delegators according to the commission rate.
// Parameters:
// - ctx: The context of the current operation.
// - reporterAcc: The account address of the reporter as AccAddress type.
// - tokens: The tokens to be allocated.
// Returns:
// - error: An error if the operation fails, nil otherwise.
func (k Keeper) AllocateTokensToReporter(ctx context.Context, reporterVal sdk.ValAddress, tokens sdk.DecCoins) error {
	// split tokens between reporter and delegators according to commission
	rep, err := k.Reporters.Get(ctx, reporterVal.Bytes())
	if err != nil {
		return err
	}

	commission := tokens.MulDec(rep.Commission.Rate)
	shared := tokens.Sub(commission)

	// update current commission
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCommission,
			sdk.NewAttribute(sdk.AttributeKeyAmount, commission.String()),
			sdk.NewAttribute(types.AttributeKeyReporter, sdk.AccAddress(reporterVal).String()),
		),
	)
	currentCommission, err := k.ReportersAccumulatedCommission.Get(ctx, reporterVal)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	currentCommission.Commission = currentCommission.Commission.Add(commission...)
	err = k.ReportersAccumulatedCommission.Set(ctx, reporterVal, currentCommission)
	if err != nil {
		return err
	}

	// update current rewards
	currentRewards, err := k.ReporterCurrentRewards.Get(ctx, reporterVal)
	// if the rewards do not exist it's fine, we will just add to zero.
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	currentRewards.Rewards = currentRewards.Rewards.Add(shared...)
	err = k.ReporterCurrentRewards.Set(ctx, reporterVal, currentRewards)
	if err != nil {
		return err
	}

	// update outstanding rewards
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRewards,
			sdk.NewAttribute(sdk.AttributeKeyAmount, tokens.String()),
			sdk.NewAttribute(types.AttributeKeyReporter, sdk.AccAddress(reporterVal).String()),
		),
	)

	outstanding, err := k.ReporterOutstandingRewards.Get(ctx, reporterVal)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	outstanding.Rewards = outstanding.Rewards.Add(tokens...)
	return k.ReporterOutstandingRewards.Set(ctx, reporterVal, outstanding)
}

// WithdrawDelegationRewards withdraws the delegation rewards for a given delegator and reporter.
// It retrieves the reporter and delegator from the keeper and asserts that the reporter matches the delegator's reporter.
// Then, it calls the withdrawDelegationRewards function to actually withdraw the rewards.
// After that, it reinitializes the delegation by calling the initializeDelegation function.
// Finally, it returns the withdrawn rewards.
func (k Keeper) WithdrawDelegationRewards(ctx context.Context, reporterVal sdk.ValAddress, delAddr sdk.AccAddress) (sdk.Coins, error) {
	reporter, err := k.Reporters.Get(ctx, reporterVal.Bytes())
	if err != nil {
		return nil, err
	}

	del, err := k.Delegators.Get(ctx, delAddr)
	if err != nil {
		return nil, err
	}
	// assert the right reporter for sanity
	if del.GetReporter() != reporter.GetReporter() {
		return nil, types.ErrReporterMismatch
	}

	// withdraw rewards
	rewards, err := k.withdrawDelegationRewards(ctx, reporter, delAddr, del)
	if err != nil {
		return nil, err
	}

	// reinitialize the delegation
	err = k.initializeDelegation(ctx, reporterVal, delAddr, del.Amount)
	if err != nil {
		return nil, err
	}
	return rewards, nil
}

// initialize starting info for a new delegation
// initializeDelegation initializes a delegation by storing the period ended by the delegation action and updating the reference count for the period.
// It also sets the DelegatorStartingInfo for the delegation.
func (k Keeper) initializeDelegation(ctx context.Context, reporterVal sdk.ValAddress, delAddr sdk.AccAddress, stake math.Int) error {
	// period has already been incremented - we want to store the period ended by this delegation action
	repCurrentRewards, err := k.ReporterCurrentRewards.Get(ctx, reporterVal)
	if err != nil {
		return err
	}
	previousPeriod := repCurrentRewards.Period - 1

	// increment reference count for the period we're going to track
	err = k.incrementReferenceCount(ctx, reporterVal, previousPeriod)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return k.DelegatorStartingInfo.Set(ctx, collections.Join(reporterVal, delAddr), types.NewDelegatorStartingInfo(previousPeriod, stake, uint64(sdkCtx.BlockHeight())))
}

// increment the reference count for a historical rewards value
// incrementReferenceCount increments the reference count for a reporter's historical rewards for a specific period.
// It retrieves the historical rewards for the reporter and period from the store, increments the reference count,
// and updates the store with the modified historical rewards.
func (k Keeper) incrementReferenceCount(ctx context.Context, reporterVal sdk.ValAddress, period uint64) error {
	historical, err := k.ReporterHistoricalRewards.Get(ctx, collections.Join(reporterVal, period))
	if err != nil {
		return err
	}
	if historical.ReferenceCount > 2 {
		panic("reference count should never exceed 2")
	}
	historical.ReferenceCount++
	return k.ReporterHistoricalRewards.Set(ctx, collections.Join(reporterVal, period), historical)
}

// withdrawDelegationRewards withdraws the delegation rewards for a specific delegator.
// It calculates the rewards, truncates the decimal portion, adds the rewards to the delegator's account,
// updates the outstanding rewards, burns the remainder, decrements the reference count of the starting period,
// and removes the delegator starting info. Finally, it emits an event for the withdrawal of rewards.
func (k Keeper) withdrawDelegationRewards(ctx context.Context, reporter types.OracleReporter, delAddr sdk.AccAddress, del types.Delegation) (sdk.Coins, error) {
	reporterVal := sdk.ValAddress(sdk.MustAccAddressFromBech32(reporter.Reporter))

	// check existence of delegator starting info
	hasInfo, err := k.DelegatorStartingInfo.Has(ctx, collections.Join(reporterVal, delAddr))
	if err != nil {
		return nil, err
	}

	if !hasInfo {
		return nil, types.ErrEmptyDelegationDistInfo
	}
	// end current period and calculate rewards
	endingPeriod, err := k.IncrementReporterPeriod(ctx, reporter)
	if err != nil {
		return nil, err
	}

	rewardsRaw, err := k.CalculateDelegationRewards(ctx, reporterVal, delAddr, del, endingPeriod)
	if err != nil {
		return nil, err
	}

	outstanding, err := k.GetReporterOutstandingRewardsCoins(ctx, reporterVal)
	if err != nil {
		return nil, err
	}

	// defensive edge case may happen on the very final digits
	// of the decCoins due to operation order of the distribution mechanism.
	rewards := rewardsRaw.Intersect(outstanding)
	if !rewards.Equal(rewardsRaw) {
		logger := k.Logger()
		logger.Info(
			"rounding error withdrawing rewards from reporter",
			"delegator", delAddr.String(),
			"reporter", reporter.GetReporter(),
			"got", rewards.String(),
			"expected", rewardsRaw.String(),
		)
	}

	// truncate reward dec coins, return remainder to decimal pool
	finalRewards, remainder := rewards.TruncateDecimal()

	// add coins to user account
	if !finalRewards.IsZero() {
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, delAddr, finalRewards)
		if err != nil {
			return nil, err
		}
	}

	// update the outstanding rewards and the decimal pool only if the transaction was successful
	if err := k.ReporterOutstandingRewards.Set(ctx, reporterVal, types.ReporterOutstandingRewards{Rewards: outstanding.Sub(rewards)}); err != nil {
		return nil, err
	}
	// TODO: burn remainder
	_ = remainder

	// decrement reference count of starting period
	startingInfo, err := k.DelegatorStartingInfo.Get(ctx, collections.Join(reporterVal, delAddr))
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}

	startingPeriod := startingInfo.PreviousPeriod
	err = k.decrementReferenceCount(ctx, reporterVal, startingPeriod)
	if err != nil {
		return nil, err
	}

	// remove delegator starting info
	err = k.DelegatorStartingInfo.Remove(ctx, collections.Join(reporterVal, delAddr))
	if err != nil {
		return nil, err
	}

	if finalRewards.IsZero() {
		// Note, we do not call the NewCoins constructor as we do not want the zero
		// coin removed.
		finalRewards = sdk.Coins{sdk.NewCoin(types.Denom, math.ZeroInt())}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeWithdrawRewards,
			sdk.NewAttribute(sdk.AttributeKeyAmount, finalRewards.String()),
			sdk.NewAttribute(types.AttributeKeyReporter, reporter.GetReporter()),
			sdk.NewAttribute(types.AttributeKeyDelegator, delAddr.String()),
		),
	)

	return finalRewards, nil
}

// CalculateDelegationRewards calculates the rewards for a delegation based on the starting and ending period.
// It takes the context, reporter ValAddress, delegator AccAddress, delegation information, and the ending period as input.
// It returns the rewards as sdk.DecCoins and an error if any.
func (k Keeper) CalculateDelegationRewards(ctx context.Context, reporterVal sdk.ValAddress, delAddr sdk.AccAddress, del types.Delegation, endingPeriod uint64) (rewards sdk.DecCoins, err error) {
	// fetch starting info for delegation
	startingInfo, err := k.DelegatorStartingInfo.Get(ctx, collections.Join(reporterVal, delAddr))
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return sdk.DecCoins{}, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if startingInfo.Height == uint64(sdkCtx.BlockHeight()) {
		// started this height, no rewards yet
		return sdk.DecCoins{}, nil
	}

	startingPeriod := startingInfo.PreviousPeriod
	stake := math.LegacyNewDecFromInt(startingInfo.Stake)

	// Iterate through disputes and withdraw with calculated staking for
	// distribution periods. These period offsets are dependent on *when* disputes
	// happen
	startingHeight := startingInfo.Height
	// Disputes this block happened after reward allocation, but we have to account
	// for them for the stake sanity check below.
	endingHeight := uint64(sdkCtx.BlockHeight())
	var iterErr error
	if endingHeight > startingHeight {
		err = k.IterateReporterDisputeEventsBetween(ctx, reporterVal, startingHeight, endingHeight,
			func(height uint64, event types.ReporterDisputeEvent) (stop bool) {
				endingPeriod := event.ReporterPeriod
				if endingPeriod > startingPeriod {
					delRewards, err := k.calculateDelegationRewardsBetween(ctx, reporterVal, startingPeriod, endingPeriod, stake.TruncateInt())
					if err != nil {
						iterErr = err
						return true
					}
					rewards = rewards.Add(delRewards...)

					// Note: It is necessary to truncate so we don't allow withdrawing
					// more rewards than owed.
					stake = stake.MulTruncate(math.LegacyOneDec().Sub(event.Fraction))
					startingPeriod = endingPeriod
				}
				return false
			},
		)
		if iterErr != nil {
			return sdk.DecCoins{}, iterErr
		}
		if err != nil {
			return sdk.DecCoins{}, err
		}
	}

	// A total stake sanity check; Recalculated final stake should be less than or
	// equal to current stake here. We cannot use Equals because stake is truncated
	// when multiplied by slash fractions (see above). We could only use equals if
	// we had arbitrary-precision rationals.
	currentStake := math.LegacyNewDecFromInt(del.Amount)

	if stake.GT(currentStake) {
		// AccountI for rounding inconsistencies between:
		//
		//     currentStake: calculated as in staking with a single computation
		//     stake:        calculated as an accumulation of stake
		//                   calculations across reporter's distribution periods
		//
		// These inconsistencies are due to differing order of operations which
		// will inevitably have different accumulated rounding and may lead to
		// the smallest decimal place being one greater in stake than
		// currentStake. When we calculated slashing by period, even if we
		// round down for each slash fraction, it's possible due to how much is
		// being rounded that we slash less when slashing by period instead of
		// for when we slash without periods. In other words, the single slash,
		// and the slashing by period could both be rounding down but the
		// slashing by period is simply rounding down less, thus making stake >
		// currentStake
		//
		// A small amount of this error is tolerated and corrected for,
		// however any greater amount should be considered a breach in expected
		// behavior.
		marginOfErr := math.LegacySmallestDec().MulInt64(3)
		if stake.LTE(currentStake.Add(marginOfErr)) {
			stake = currentStake
		} else {
			return sdk.DecCoins{}, fmt.Errorf("calculated final stake for delegator %s greater than current stake"+
				"\n\tfinal stake:\t%s"+
				"\n\tcurrent stake:\t%s",
				del.GetReporter(), stake, currentStake)
		}
	}

	// calculate rewards for final period
	delRewards, err := k.calculateDelegationRewardsBetween(ctx, reporterVal, startingPeriod, endingPeriod, stake.TruncateInt())
	if err != nil {
		return sdk.DecCoins{}, err
	}

	rewards = rewards.Add(delRewards...)
	return rewards, nil
}

// increment reporter period, returning the period just ended
func (k Keeper) IncrementReporterPeriod(ctx context.Context, reporter types.OracleReporter) (uint64, error) {
	// fetch current rewards
	reporterVal := sdk.ValAddress(sdk.MustAccAddressFromBech32(reporter.Reporter))
	rewards, err := k.ReporterCurrentRewards.Get(ctx, reporterVal)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return 0, err
	}

	// calculate current ratio
	var current sdk.DecCoins
	if reporter.TotalTokens.IsZero() {

		// can't calculate ratio for zero-token reporters
		// ergo we instead add to ~~~the decimal pool~~ TODO: burn rewards.Rewards

		outstanding, err := k.ReporterOutstandingRewards.Get(ctx, reporterVal)
		if err != nil && !errors.Is(err, collections.ErrNotFound) {
			return 0, err
		}

		outstanding.Rewards = outstanding.GetRewards().Sub(rewards.Rewards)

		err = k.ReporterOutstandingRewards.Set(ctx, reporterVal, outstanding)
		if err != nil {
			return 0, err
		}

		current = sdk.DecCoins{}
	} else {
		// note: necessary to truncate so we don't allow withdrawing more rewards than owed
		current = rewards.Rewards.QuoDecTruncate(math.LegacyNewDecFromInt(reporter.TotalTokens))
	}

	// fetch historical rewards for last period
	historical, err := k.ReporterHistoricalRewards.Get(ctx, collections.Join(reporterVal, rewards.Period-1))
	if err != nil {
		return 0, err
	}

	cumRewardRatio := historical.CumulativeRewardRatio

	// decrement reference count
	err = k.decrementReferenceCount(ctx, reporterVal, rewards.Period-1)
	if err != nil {
		return 0, err
	}

	// set new historical rewards with reference count of 1
	err = k.ReporterHistoricalRewards.Set(ctx, collections.Join(reporterVal, rewards.Period), types.NewReporterHistoricalRewards(cumRewardRatio.Add(current...), 1))
	if err != nil {
		return 0, err
	}

	// set current rewards, incrementing period by 1
	err = k.ReporterCurrentRewards.Set(ctx, reporterVal, types.NewReporterCurrentRewards(sdk.DecCoins{}, rewards.Period+1))
	if err != nil {
		return 0, err
	}

	return rewards.Period, nil
}

// calculate the rewards accrued by a delegation between two periods
func (k Keeper) calculateDelegationRewardsBetween(ctx context.Context, reporterVal sdk.ValAddress,
	startingPeriod, endingPeriod uint64, stake math.Int,
) (sdk.DecCoins, error) {
	// sanity check
	if startingPeriod > endingPeriod {
		return sdk.DecCoins{}, fmt.Errorf("startingPeriod cannot be greater than endingPeriod")
	}

	// sanity check
	if stake.IsNegative() {
		return sdk.DecCoins{}, fmt.Errorf("stake should not be negative")
	}

	// return staking * (ending - starting)
	starting, err := k.ReporterHistoricalRewards.Get(ctx, collections.Join(reporterVal, startingPeriod))
	if err != nil {
		return sdk.DecCoins{}, err
	}

	ending, err := k.ReporterHistoricalRewards.Get(ctx, collections.Join(reporterVal, endingPeriod))
	if err != nil {
		return sdk.DecCoins{}, err
	}

	difference := ending.CumulativeRewardRatio.Sub(starting.CumulativeRewardRatio)
	if difference.IsAnyNegative() {
		return sdk.DecCoins{}, fmt.Errorf("negative rewards should not be possible")
	}
	// note: necessary to truncate so we don't allow withdrawing more rewards than owed
	rewards := difference.MulDecTruncate(math.LegacyNewDecFromInt(stake))
	return rewards, nil
}

// decrement the reference count for a historical rewards value, and delete if zero references remain
func (k Keeper) decrementReferenceCount(ctx context.Context, reporterAddr sdk.ValAddress, period uint64) error {
	historical, err := k.ReporterHistoricalRewards.Get(ctx, collections.Join(reporterAddr, period))
	if err != nil {
		return err
	}

	if historical.ReferenceCount == 0 {
		panic("cannot set negative reference count")
	}
	historical.ReferenceCount--
	if historical.ReferenceCount == 0 {
		return k.ReporterHistoricalRewards.Remove(ctx, collections.Join(reporterAddr, period))
	}

	return k.ReporterHistoricalRewards.Set(ctx, collections.Join(reporterAddr, period), historical)
}

// GetTotalRewards returns the total amount of fee distribution rewards held in the store
func (k Keeper) GetTotalRewards(ctx context.Context) (totalRewards sdk.DecCoins) {
	err := k.ReporterOutstandingRewards.Walk(ctx, nil, func(_ sdk.ValAddress, rewards types.ReporterOutstandingRewards) (stop bool, err error) {
		totalRewards = totalRewards.Add(rewards.Rewards...)
		return false, nil
	},
	)
	if err != nil {
		panic(err)
	}

	return totalRewards
}

// get outstanding rewards
func (k Keeper) GetReporterOutstandingRewardsCoins(ctx context.Context, reporterVal sdk.ValAddress) (sdk.DecCoins, error) {
	rewards, err := k.ReporterOutstandingRewards.Get(ctx, reporterVal)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}

	return rewards.Rewards, nil
}

// iterate over slash events between heights, inclusive
func (k Keeper) IterateReporterDisputeEventsBetween(ctx context.Context, reporterVal sdk.ValAddress, startingHeight, endingHeight uint64,
	handler func(height uint64, event types.ReporterDisputeEvent) (stop bool),
) error {
	rng := new(collections.Range[collections.Triple[sdk.ValAddress, uint64, uint64]]).
		StartInclusive(collections.Join3(reporterVal, startingHeight, uint64(0))).
		EndExclusive(collections.Join3(reporterVal, endingHeight+1, uint64(gomath.MaxUint64)))

	err := k.ReporterDisputeEvents.Walk(ctx, rng, func(k collections.Triple[sdk.ValAddress, uint64, uint64], ev types.ReporterDisputeEvent) (stop bool, err error) {
		height := k.K2()
		if handler(height, ev) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) initializeReporter(ctx context.Context, reporter types.OracleReporter) error {
	valBz := sdk.ValAddress(sdk.MustAccAddressFromBech32(reporter.Reporter))
	// set initial historical rewards (period 0) with reference count of 1
	err := k.ReporterHistoricalRewards.Set(ctx, collections.Join(valBz, uint64(0)), types.NewReporterHistoricalRewards(sdk.DecCoins{}, 1))
	if err != nil {
		return err
	}

	// set current rewards (starting at period 1)
	err = k.ReporterCurrentRewards.Set(ctx, valBz, types.NewReporterCurrentRewards(sdk.DecCoins{}, 1))
	if err != nil {
		return err
	}

	// set accumulated commission
	err = k.ReportersAccumulatedCommission.Set(ctx, valBz, types.ReporterAccumulatedCommission{})
	if err != nil {
		return err
	}

	// set outstanding rewards
	err = k.ReporterOutstandingRewards.Set(ctx, valBz, types.ReporterOutstandingRewards{Rewards: sdk.DecCoins{}})
	return err
}

func (k Keeper) updateReporterDisputeFraction(ctx context.Context, reporterVal sdk.ValAddress, fraction math.LegacyDec) error {
	if fraction.GT(math.LegacyOneDec()) || fraction.IsNegative() {
		return fmt.Errorf("fraction must be >=0 and <=1, current fraction: %v", fraction)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	reporter, err := k.Reporters.Get(ctx, sdk.AccAddress(reporterVal))
	if err != nil {
		return err
	}
	// increment current period
	newPeriod, err := k.IncrementReporterPeriod(ctx, reporter)
	if err != nil {
		return err
	}

	// increment reference count on period we need to track
	err = k.incrementReferenceCount(ctx, reporterVal, newPeriod)
	if err != nil {
		return err
	}

	slashEvent := types.NewReporterDisputeEvent(newPeriod, fraction)
	height := uint64(sdkCtx.BlockHeight())

	return k.ReporterDisputeEvents.Set(
		ctx,
		collections.Join3[sdk.ValAddress, uint64, uint64](
			reporterVal,
			height,
			newPeriod,
		),
		slashEvent,
	)
}

// Hooks to implement part of reporter crud operations
func (k Keeper) AfterReporterRemoved(ctx context.Context, reporterVal sdk.ValAddress) error {
	// fetch outstanding
	outstanding, err := k.GetReporterOutstandingRewardsCoins(ctx, reporterVal)
	if err != nil {
		return err
	}

	// force-withdraw commission
	reporterCommission, err := k.ReportersAccumulatedCommission.Get(ctx, reporterVal)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	commission := reporterCommission.Commission

	if !commission.IsZero() {
		// subtract from outstanding
		outstanding = outstanding.Sub(commission)

		// split into integral & remainder
		coins, remainder := commission.TruncateDecimal()
		// TODO: burn remainder (inflationary?)
		_ = remainder

		// add to reporter account
		if !coins.IsZero() {
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sdk.AccAddress(reporterVal), coins); err != nil {
				return err
			}
		}
	}

	// "TODO: burn outstanding dust"
	_ = outstanding

	// delete outstanding
	err = k.ReporterOutstandingRewards.Remove(ctx, reporterVal)
	if err != nil {
		return err
	}

	// remove commission record
	err = k.ReportersAccumulatedCommission.Remove(ctx, reporterVal)
	if err != nil {
		return err
	}

	// clear disputes
	err = k.ReporterDisputeEvents.Clear(ctx, collections.NewPrefixedTripleRange[sdk.ValAddress, uint64, uint64](reporterVal))
	if err != nil {
		return err
	}

	// clear historical rewards
	err = k.ReporterHistoricalRewards.Clear(ctx, collections.NewPrefixedPairRange[sdk.ValAddress, uint64](reporterVal))
	if err != nil {
		return err
	}
	// clear current rewards
	err = k.ReporterCurrentRewards.Remove(ctx, reporterVal)
	if err != nil {
		return err
	}

	return nil
}

// initialize reporter
func (k Keeper) AfterReporterCreated(ctx context.Context, reporter types.OracleReporter) error {
	return k.initializeReporter(ctx, reporter)
}

// increment period
func (k Keeper) BeforeDelegationCreated(ctx context.Context, reporter types.OracleReporter) error {
	_, err := k.IncrementReporterPeriod(ctx, reporter)
	return err
}

// withdraw delegation rewards (which also increments period)
func (k Keeper) BeforeDelegationModified(ctx context.Context, delAddr sdk.AccAddress, del types.Delegation, reporter types.OracleReporter) error {
	_, err := k.withdrawDelegationRewards(ctx, reporter, delAddr, del)
	return err
}

// create new delegation period record
func (k Keeper) AfterDelegationModified(ctx context.Context, delAddr sdk.AccAddress, reporterVal sdk.ValAddress, stake math.Int) error {
	return k.initializeDelegation(ctx, reporterVal, delAddr, stake)
}

// record the dispute event
func (k Keeper) BeforeReporterDisputed(ctx context.Context, valAddr sdk.ValAddress, fraction math.LegacyDec) error {
	return k.updateReporterDisputeFraction(ctx, valAddr, fraction)
}