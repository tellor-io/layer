package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	layertypes "github.com/tellor-io/layer/types"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	"github.com/tellor-io/layer/x/reporter/types"
)

// FeefromReporterStake deducts the fee from the reporter's stake used mainly for paying dispute from bond
func (k Keeper) FeefromReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, amt math.Int) error {
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return err
	}

	// Calculate each delegator's share (including the reporter as a self-delegator)
	repAddr, err := sdk.AccAddressFromBech32(reporter.Reporter)
	if err != nil {
		return err
	}
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

		rng := collections.NewPrefixedPairRange[sdk.AccAddress, sdk.ValAddress](key)
		iter, err := k.TokenOrigin.Iterate(ctx, rng)
		if err != nil {
			return err
		}
		delegatorSources, err := iter.KeyValues()
		if err != nil {
			return err
		}
		for _, source := range delegatorSources {
			srcAmt := math.LegacyNewDecFromInt(source.Value)
			share := srcAmt.Quo(totaltokens).Mul(math.LegacyNewDecFromInt(amt))
			_, err = k.feeFromStake(ctx, source.Key.K1(), source.Key.K2(), share)
			if err != nil {
				return err
			}
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

func (k Keeper) deductUnbondingDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, shares math.Int) (math.Int, error) {
	ubd, err := k.stakingKeeper.GetUnbondingDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return math.Int{}, err
	}
	if len(ubd.Entries) == 0 {
		return math.Int{}, types.ErrNoUnbondingDelegationEntries
	}
	for i, u := range ubd.Entries {
		if u.Balance.LT(shares) {
			shares = shares.Sub(u.Balance)
			ubd.RemoveEntry(int64(i))
		} else {
			u.Balance = u.Balance.Sub(shares)
			u.InitialBalance = u.InitialBalance.Sub(shares)
			ubd.Entries[i] = u
			shares = math.ZeroInt()
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
	return shares, nil

}

func (k Keeper) deductFromdelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, shares math.LegacyDec) (math.LegacyDec, error) {
	// get delegation
	del, err := k.stakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return shares, err
	}
	if del.Shares.GTE(shares) {
		_, err := k.stakingKeeper.Unbond(ctx, delAddr, valAddr, shares)
		if err != nil {
			return shares, err
		}
		return math.LegacyZeroDec(), nil
	} else {
		shares = shares.Sub(del.Shares)
		_, err := k.stakingKeeper.Unbond(ctx, delAddr, valAddr, del.Shares)
		if err != nil {
			return shares, err
		}
		return shares, nil
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
func (k Keeper) undelegate(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, shares math.LegacyDec) (math.Int, error) {
	remainingFromdel, err := k.deductFromdelegation(ctx, delAddr, valAddr, shares)
	if err != nil {
		if !errors.Is(err, stakingtypes.ErrNoDelegation) {
			return math.Int{}, err
		}
	}

	if remainingFromdel.IsZero() {
		if err := k.moveTokensFromValidator(ctx, valAddr, shares.TruncateInt()); err != nil {
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

func (k Keeper) EscrowReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, power, height int64, amt math.Int) error {
	// get origins at height
	rng := collections.NewPrefixedPairRange[sdk.AccAddress, int64](reporterAddr).StartInclusive(height)
	var firstValue *types.DelegationsPreUpdate

	err := k.TokenOriginSnapshot.Walk(ctx, rng, func(key collections.Pair[sdk.AccAddress, int64], value types.DelegationsPreUpdate) (stop bool, err error) {
		firstValue = &value
		return true, nil
	})
	if err != nil {
		return err
	}

	totalTokens := layertypes.PowerReduction.MulRaw(power)
	for _, del := range firstValue.TokenOrigins {
		delegatorShare := math.LegacyNewDecFromInt(del.Amount).Quo(math.LegacyNewDecFromInt(totalTokens)).Mul(math.LegacyNewDecFromInt(amt))
		delAddr, err := sdk.AccAddressFromBech32(del.DelegatorAddress)
		if err != nil {
			return err
		}
		valAddr, err := sdk.ValAddressFromBech32(del.ValidatorAddress)
		if err != nil {
			return err
		}
		remaining, err := k.undelegate(ctx, delAddr, valAddr, delegatorShare)
		if err != nil {
			return err
		}
		if !remaining.IsZero() {
			dstVAl, err := k.getDstValidator(ctx, delAddr, valAddr)
			if err != nil {
				return err
			}
			_, err = k.undelegate(ctx, delAddr, dstVAl, math.LegacyNewDecFromInt(remaining))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Keeper) tokensToDispute(ctx context.Context, fromPool string, amount math.Int) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromPool, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, amount)))
}
