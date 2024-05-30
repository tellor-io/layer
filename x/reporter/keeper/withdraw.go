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
func (k Keeper) FeefromReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, amt math.Int, hashId []byte) error {
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return err
	}

	// Calculate each delegator's share (including the reporter as a self-delegator)
	repAddr := sdk.AccAddress(reporter.Reporter)

	delAddrs, err := k.Delegators.Indexes.Reporter.MatchExact(ctx, repAddr)
	if err != nil {
		return err
	}
	totaltokens := math.LegacyNewDecFromInt(reporter.TotalTokens)
	defer delAddrs.Close()
	for ; delAddrs.Valid(); delAddrs.Next() {
		key, err := delAddrs.PrimaryKey()
		if err != nil {
			return err
		}

		rng := collections.NewPrefixedPairRange[[]byte, []byte](key)
		iter, err := k.TokenOrigin.Iterate(ctx, rng)
		if err != nil {
			return err
		}
		delegatorSources, err := iter.KeyValues()
		if err != nil {
			return err
		}
		feeTracker := make([]*types.TokenOriginInfo, 0, len(delegatorSources))
		for _, source := range delegatorSources {
			srcAmt := math.LegacyNewDecFromInt(source.Value)
			share := srcAmt.Quo(totaltokens).Mul(math.LegacyNewDecFromInt(amt))
			_, err = k.feeFromStake(ctx, source.Key.K1(), source.Key.K2(), share)
			if err != nil {
				return err
			}
			feeTracker = append(feeTracker, &types.TokenOriginInfo{
				DelegatorAddress: source.Key.K1(),
				ValidatorAddress: source.Key.K2(),
				Amount:           share.TruncateInt(),
			})
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
		if err := k.FeePaidFromStake.Set(ctx, hashId, types.DelegationsPreUpdate{TokenOrigins: feeTracker}); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) feeFromStake(
	ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount math.LegacyDec,
) (math.Int, error) {
	returnAmount, err := k.stakingKeeper.Unbond(ctx, delAddr, valAddr, sharesAmount)
	if err != nil {
		return math.Int{}, err
	}
	if err := k.moveTokensFromValidator(ctx, valAddr, returnAmount); err != nil {
		return math.Int{}, err
	}

	return returnAmount, nil
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

func (k Keeper) EscrowReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, power, height int64, amt math.Int, hashId []byte) error {
	// get origins at height
	rng := collections.NewPrefixedPairRange[[]byte, int64](reporterAddr).EndInclusive(height).Descending()
	var firstValue *types.DelegationsPreUpdate

	err := k.TokenOriginSnapshot.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.DelegationsPreUpdate) (stop bool, err error) {
		firstValue = &value
		return true, nil
	})
	if err != nil {
		return err
	}

	totalTokens := layertypes.PowerReduction.MulRaw(power)
	disputeTokens := make([]*types.TokenOriginInfo, 0)
	leftover := amt
	for i, del := range firstValue.TokenOrigins {

		delegatorShare := math.LegacyNewDecFromInt(del.Amount).Quo(math.LegacyNewDecFromInt(totalTokens)).Mul(math.LegacyNewDecFromInt(amt))

		leftover = leftover.Sub(delegatorShare.TruncateInt())

		if i == len(firstValue.TokenOrigins)-1 {
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
	return k.DisputedDelegationAmounts.Set(ctx, hashId, types.DelegationsPreUpdate{TokenOrigins: disputeTokens})
}

func (k Keeper) tokensToDispute(ctx context.Context, fromPool string, amount math.Int) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromPool, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, amount)))
}
