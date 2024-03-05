package keeper

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	layer "github.com/tellor-io/layer/types"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
)

func (k Keeper) EscrowReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, amt math.Int) error {
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
	totaltokens := reporter.TotalTokens
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
		// total amount to remove from the delegator
		delegatorShare := del.Amount.Quo(totaltokens).Mul(amt)

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
			srcAmt := source.Value
			_share := srcAmt.Quo(del.Amount).Mul(delegatorShare)
			_, err = k.Undelegate(ctx, source.Key.K1(), source.Key.K2(), math.LegacyNewDecFromInt(_share))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Keeper) Undelegate(
	ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount math.LegacyDec,
) (math.Int, error) {
	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return math.Int{}, err
	}

	returnAmount, err := k.Unbond(ctx, delAddr, valAddr, sharesAmount)
	if err != nil {
		return math.Int{}, err
	}

	// transfer the validator tokens to the not bonded pool
	if validator.IsBonded() {
		err = k.bondedTokensToDispute(ctx, returnAmount)
		if err != nil {
			return math.Int{}, err
		}
	}

	return returnAmount, nil
}

func (k Keeper) bondedTokensToDispute(ctx context.Context, amount math.Int) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, stakingtypes.BondedPoolName, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, amount)))
}

func (k Keeper) Unbond(
	ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, shares math.LegacyDec,
) (amount math.Int, err error) {
	del, err := k.stakingKeeper.Delegation(ctx, delAddr, valAddr)
	if err != nil {
		return amount, err
	}
	delegation := stakingtypes.Delegation{
		DelegatorAddress: del.GetDelegatorAddr(),
		ValidatorAddress: del.GetValidatorAddr(),
		Shares:           del.GetShares(),
	}
	// call the before-delegation-modified hook
	if err := k.Hooks().BeforeDelegationSharesModified(ctx, delAddr, valAddr); err != nil {
		return amount, err
	}

	// ensure that we have enough shares to remove
	if delegation.Shares.LT(shares) {
		return amount, errorsmod.Wrap(stakingtypes.ErrNotEnoughDelegationShares, delegation.Shares.String())
	}

	// get validator
	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return amount, err
	}

	// subtract shares from delegation
	delegation.Shares = delegation.Shares.Sub(shares)

	delegatorAddress, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
	if err != nil {
		return amount, err
	}

	valbz, err := sdk.ValAddressFromBech32(validator.GetOperator())
	if err != nil {
		return amount, err
	}

	isValidatorOperator := bytes.Equal(delegatorAddress, valbz)

	// If the delegation is the operator of the validator and undelegating will decrease the validator's
	// self-delegation below their minimum, we jail the validator.
	if isValidatorOperator && !validator.Jailed &&
		validator.TokensFromShares(delegation.Shares).TruncateInt().LT(validator.MinSelfDelegation) {
		consAddr, err := validator.GetConsAddr()
		if err != nil {
			return amount, err
		}
		err = k.stakingKeeper.Jail(ctx, consAddr)
		if err != nil {
			return amount, fmt.Errorf("failed to jail validator: %v", err)
		}
		validator, err = k.stakingKeeper.GetValidator(ctx, valbz)
		if err != nil {
			return amount, fmt.Errorf("validator record not found for address: %X", valbz)
		}
	}

	if delegation.Shares.IsZero() {
		err = k.stakingKeeper.RemoveDelegation(ctx, delegation)
	} else {
		if err = k.stakingKeeper.SetDelegation(ctx, delegation); err != nil {
			return amount, err
		}

		valAddr, err = sdk.ValAddressFromBech32(delegation.GetValidatorAddr())
		if err != nil {
			return amount, err
		}

		// call the after delegation modification hook
		err = k.Hooks().AfterDelegationModified(ctx, delegatorAddress, valAddr)
		if err != nil {
			return amount, err
		}
	}

	if err != nil {
		return amount, err
	}

	// remove the shares and coins from the validator
	// NOTE that the amount is later (in keeper.Delegation) moved between staking module pools
	validator, amount, err = k.stakingKeeper.RemoveValidatorTokensAndShares(ctx, validator, shares)
	if err != nil {
		return amount, err
	}

	if validator.DelegatorShares.IsZero() && validator.IsUnbonded() {
		// if not unbonded, we must instead remove validator in EndBlocker once it finishes its unbonding period
		if err = k.stakingKeeper.RemoveValidator(ctx, valbz); err != nil {
			return amount, err
		}
	}

	return amount, nil
}
