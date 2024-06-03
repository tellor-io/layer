package keeper

import (
	"context"
	"errors"
	"fmt"

	layertypes "github.com/tellor-io/layer/types"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// FeefromReporterStake deducts the fee from the reporter's stake used mainly for paying dispute from bond
// checks total tokens of delegators given to reporter and deducts the fee proportionally among them
func (k Keeper) FeefromReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, amt math.Int, hashId []byte) error {
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return err
	}

	if amt.GT(reporter.TotalTokens) {
		return errors.New("insufficient stake to pay fee")
	}

	// Calculate each delegator's share (including the reporter as a self-delegator)
	delAddrs, err := k.Delegators.Indexes.Reporter.MatchExact(ctx, reporterAddr)
	if err != nil {
		return err
	}
	totaltokens := math.LegacyNewDecFromInt(reporter.TotalTokens)

	feeTracker := make([]*types.TokenOriginInfo, 0)
	totalTracked := math.ZeroInt()

	defer delAddrs.Close()

	for ; delAddrs.Valid(); delAddrs.Next() {
		key, err := delAddrs.PrimaryKey()
		if err != nil {
			return err
		}
		del, err := k.Delegators.Get(ctx, key)
		if err != nil {
			return err
		}
		srcAmt := math.LegacyNewDecFromInt(del.Amount)
		shareAmt := srcAmt.Quo(totaltokens).Mul(math.LegacyNewDecFromInt(amt))
		fees, err := k.DeductTokensFromDelegator(ctx, key, shareAmt)
		if err != nil {
			return err
		}
		totalTracked = totalTracked.Add(shareAmt.TruncateInt())
		feeTracker = append(feeTracker, fees...)
	}
	has, err := k.FeePaidFromStake.Has(ctx, hashId)
	if err != nil {
		return err
	}
	if has {
		preFeeTracker, err := k.FeePaidFromStake.Get(ctx, hashId)
		if err != nil {
			return err
		}

		feeTracker = append(feeTracker, preFeeTracker.TokenOrigins...)
	}
	if err := k.FeePaidFromStake.Set(ctx, hashId, types.DelegationsAmounts{TokenOrigins: feeTracker, Total: totalTracked}); err != nil {
		return err
	}
	return nil
}

// get dst validator for a redelegated delegator
func (k Keeper) getDstValidator(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.ValAddress, error) {
	reds, err := k.stakingKeeper.GetRedelegationsFromSrcValidator(ctx, valAddr)
	if err != nil {
		return nil, err
	}
	for _, red := range reds {
		if red.DelegatorAddress == delAddr.String() {
			valAddr, err := sdk.ValAddressFromBech32(red.ValidatorDstAddress)
			if err != nil {
				return nil, err
			}
			return valAddr, nil
		}
	}
	return nil, errors.New("redelegation to destination validator not found")
}

func (k Keeper) deductUnbondingDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, tokens math.Int) (math.Int, error) {
	ubd, err := k.stakingKeeper.GetUnbondingDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return math.Int{}, err
	}
	if len(ubd.Entries) == 0 {
		return math.Int{}, types.ErrNoUnbondingDelegationEntries
	}
	for i, u := range ubd.Entries {
		if u.Balance.LT(tokens) {
			tokens = tokens.Sub(u.Balance)
			ubd.RemoveEntry(int64(i))
		} else {
			u.Balance = u.Balance.Sub(tokens)
			u.InitialBalance = u.InitialBalance.Sub(tokens)
			ubd.Entries[i] = u
			tokens = math.ZeroInt()
			break
		}
	}

	if len(ubd.Entries) == 0 {
		err = k.stakingKeeper.RemoveUnbondingDelegation(ctx, ubd)
	} else {
		err = k.stakingKeeper.SetUnbondingDelegation(ctx, ubd)
	}
	if err != nil {
		return math.Int{}, err
	}
	return tokens, nil
}

func (k Keeper) deductFromdelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, delTokens math.LegacyDec) (math.LegacyDec, error) {
	// get delegation
	del, err := k.stakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return math.LegacyDec{}, err
	}
	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return math.LegacyDec{}, err
	}

	// convert current delegation shares to tokens
	currentTokens := validator.TokensFromShares(del.Shares)

	if currentTokens.GTE(delTokens) {
		shares, err := validator.SharesFromTokens(delTokens.TruncateInt())
		if err != nil {
			return math.LegacyDec{}, err
		}
		_, err = k.stakingKeeper.Unbond(ctx, delAddr, valAddr, shares)
		if err != nil {
			return math.LegacyDec{}, err
		}
		return math.LegacyZeroDec(), nil
	} else {
		delTokens = delTokens.Sub(currentTokens)
		_, err := k.stakingKeeper.Unbond(ctx, delAddr, valAddr, del.Shares)
		if err != nil {
			return math.LegacyDec{}, err
		}
		return delTokens, nil
	}
}

func (k Keeper) moveTokensFromValidator(ctx context.Context, valAddr sdk.ValAddress, amount math.Int) error {
	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return err
	}
	var fromPool string
	switch {
	case validator.IsBonded():
		fromPool = stakingtypes.BondedPoolName
	case validator.IsUnbonding():
		fromPool = stakingtypes.NotBondedPoolName
	default:
		return fmt.Errorf("unknown validator status: %s", validator.GetStatus())
	}
	return k.tokensToDispute(ctx, fromPool, amount)
}

func (k Keeper) MoveTokensFromValidator(ctx context.Context, validator stakingtypes.Validator, amount math.Int) error {
	var fromPool string
	switch {
	case validator.IsBonded():
		fromPool = stakingtypes.BondedPoolName
	case validator.IsUnbonding():
		fromPool = stakingtypes.NotBondedPoolName
	default:
		return fmt.Errorf("unknown validator status: %s", validator.GetStatus())
	}
	return k.tokensToDispute(ctx, fromPool, amount)
}

func (k Keeper) DeductTokensFromDelegator(ctx context.Context, delAddr sdk.AccAddress, amt math.LegacyDec) ([]*types.TokenOriginInfo, error) {
	feeTracker := make([]*types.TokenOriginInfo, 0)
	remainder := amt
	err := k.stakingKeeper.IterateDelegatorDelegations(ctx, delAddr, func(delegation stakingtypes.Delegation) bool {
		if remainder.IsZero() {
			return false
		}

		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			panic(err)
		}
		tokens := val.TokensFromShares(delegation.Shares)

		unbondAmt := tokens
		if tokens.GTE(remainder) {
			unbondAmt = remainder
		}

		shares, err := val.SharesFromTokens(unbondAmt.TruncateInt())
		if err != nil {
			panic(err)
		}

		// Unbond and move tokens from validator
		_, err = k.stakingKeeper.Unbond(ctx, delAddr, valAddr, shares)
		if err != nil {
			panic(err)
		}
		err = k.MoveTokensFromValidator(ctx, val, unbondAmt.TruncateInt())
		if err != nil {
			panic(err)
		}
		// Reduce the remainder by the amount unbonded
		remainder = remainder.Sub(unbondAmt)
		// Add the unbonded amount to the fee tracker
		feeTracker = append(feeTracker, &types.TokenOriginInfo{
			DelegatorAddress: delAddr.Bytes(),
			ValidatorAddress: valAddr.Bytes(),
			Amount:           unbondAmt.TruncateInt(),
		})
		return false
	})
	return feeTracker, err
}

func (k Keeper) tokensToDispute(ctx context.Context, fromPool string, amount math.Int) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromPool, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, amount)))
}

func (k Keeper) SlashUnbondingDelegation(ctx context.Context, delAddrs sdk.AccAddress, slashAmount math.Int) (totalSlashAmount math.Int, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockHeader().Time
	totalSlashAmount = math.ZeroInt()
	burnedAmount := math.ZeroInt()
	remainder := slashAmount
	err = k.stakingKeeper.IterateDelegatorUnbondingDelegations(ctx, delAddrs, func(ubd stakingtypes.UnbondingDelegation) (stop bool) {

		// perform slashing on all entries within the unbonding delegation
		for i, entry := range ubd.Entries {
			if remainder.IsZero() {
				return true
			}
			if entry.IsMature(now) && !entry.OnHold() {
				// Unbonding delegation no longer eligible for slashing, skip it
				continue
			}

			slashAmount := math.MinInt(remainder, entry.Balance)
			remainder = remainder.Sub(slashAmount)
			totalSlashAmount = totalSlashAmount.Add(slashAmount)

			// Don't slash more tokens than held
			// Possible since the unbonding delegation may already
			// have been slashed, and slash amounts are calculated
			// according to stake held at time of infraction
			unbondingSlashAmount := slashAmount

			// Update unbonding delegation if necessary
			if unbondingSlashAmount.IsZero() {
				continue
			}

			burnedAmount = burnedAmount.Add(unbondingSlashAmount)
			entry.Balance = entry.Balance.Sub(unbondingSlashAmount)
			ubd.Entries[i] = entry
			if err = k.stakingKeeper.SetUnbondingDelegation(ctx, ubd); err != nil {
				panic(err)
			}
		}
		return false
	})
	if err := k.tokensToDispute(ctx, stakingtypes.NotBondedPoolName, burnedAmount); err != nil {
		return math.ZeroInt(), err
	}

	return totalSlashAmount, nil
}

func (k Keeper) EscrowReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, power, height int64, amt math.Int, hashId []byte) error {
	report, err := k.Report.Get(ctx, collections.Join(reporterAddr.Bytes(), height))
	if err != nil {
		return err
	}

	totalTokens := layertypes.PowerReduction.MulRaw(power)
	disputeTokens := make([]*types.TokenOriginInfo, 0)
	leftover := amt
	for i, del := range report.TokenOrigins {

		delegatorShare := math.LegacyNewDecFromInt(del.Amount).Quo(math.LegacyNewDecFromInt(totalTokens)).Mul(math.LegacyNewDecFromInt(amt))

		leftover = leftover.Sub(delegatorShare.TruncateInt())

		if i == len(report.TokenOrigins)-1 {
			delegatorShare = delegatorShare.Add(leftover.ToLegacyDec())
		}

		delAddr := sdk.AccAddress(del.DelegatorAddress)
		valAddr := sdk.ValAddress(del.ValidatorAddress)

		remaining, err := k.undelegate(ctx, delAddr, valAddr, delegatorShare)
		if err != nil {
			return err
		}
		disputeTokens = append(disputeTokens, &types.TokenOriginInfo{
			DelegatorAddress: del.DelegatorAddress,
			ValidatorAddress: del.ValidatorAddress,
			Amount:           delegatorShare.TruncateInt().Sub(remaining),
		})
		if !remaining.IsZero() {
			dstVAl, err := k.getDstValidator(ctx, delAddr, valAddr)
			if err != nil {
				return err
			}
			_, err = k.undelegate(ctx, delAddr, dstVAl, math.LegacyNewDecFromInt(remaining))
			if err != nil {
				return err
			}
			disputeTokens = append(disputeTokens, &types.TokenOriginInfo{
				DelegatorAddress: del.DelegatorAddress,
				ValidatorAddress: dstVAl,
				Amount:           remaining,
			})
		}
	}

	// after escrow you should keep a new snapshot of the amounts from each that were taken instead of relying on the original snapshot
	// then you can delete it after the slashed tokens are returned
	return k.DisputedDelegationAmounts.Set(ctx, hashId, types.DelegationsAmounts{TokenOrigins: disputeTokens})
}
func (k Keeper) undelegate(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, delTokens math.LegacyDec) (math.Int, error) {
	remainingFromdel, err := k.deductFromdelegation(ctx, delAddr, valAddr, delTokens)
	if err != nil {
		if !errors.Is(err, stakingtypes.ErrNoDelegation) {
			return math.Int{}, err
		}
	}

	if remainingFromdel.IsZero() {
		if err := k.moveTokensFromValidator(ctx, valAddr, delTokens.TruncateInt()); err != nil {
			return math.Int{}, err
		}
		return remainingFromdel.TruncateInt(), nil

	} else {
		remainingUnbonding, err := k.deductUnbondingDelegation(ctx, delAddr, valAddr, remainingFromdel.TruncateInt())
		if err != nil {
			return math.Int{}, err
		}
		if remainingUnbonding.IsZero() {
			if err := k.tokensToDispute(ctx, stakingtypes.NotBondedPoolName, remainingFromdel.TruncateInt()); err != nil {
				return math.Int{}, err
			}
		}
		return remainingUnbonding, nil
	}
}
