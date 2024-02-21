package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/reporter/types"
)

func (k Keeper) ValidateAmount(ctx context.Context, delegator sdk.AccAddress, originAmounts []*types.TokenOrigin, amount math.Int) error {
	var _amt = math.ZeroInt()
	for _, origin := range originAmounts {
		_amt = _amt.Add(origin.Amount)
	}

	if !amount.Equal(_amt) {
		return errorsmod.Wrapf(types.ErrTokenAmountMismatch, "got %v as amount, but sum of token origins is %v", amount, _amt)
	}
	for _, origin := range originAmounts {
		valAddr, err := sdk.ValAddressFromBech32(origin.ValidatorAddress)
		if err != nil {
			return err
		}
		tokenSource, err := k.TokenOrigin.Get(ctx, collections.Join(delegator, valAddr))
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return errorsmod.Wrapf(err, "unable to fetch token origin")
			} else {
				// not found so initialize
				tokenSource.ValidatorAddress = origin.ValidatorAddress
				tokenSource.Amount = math.ZeroInt()
			}
		}
		validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to fetch validator for source tokens %v", origin)
		}
		delegation, err := k.stakingKeeper.Delegation(ctx, delegator, valAddr)
		if err != nil {
			return err
		}
		// check if the validator has enough tokens bonded with validator, this would be the sum
		// of what is currently delegated to reporter plus the amount being added in this transaction
		sum := tokenSource.Amount.Add(origin.Amount)
		if validator.TokensFromShares(delegation.GetShares()).TruncateInt().LT(sum) {
			return errorsmod.Wrapf(types.ErrInsufficientTokens, "insufficient tokens bonded with validator %v", valAddr)
		}
		tokenSource.Amount = sum
		if err := k.TokenOrigin.Set(ctx, collections.Join(delegator, valAddr), tokenSource); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) UpdateOrRemoveDelegator(ctx context.Context, delAddr sdk.AccAddress, del types.Delegation, reporter types.OracleReporter, amt math.Int) error {
	if err := k.BeforeDelegationModified(ctx, delAddr, del, reporter); err != nil {
		return err
	}
	if del.Amount.LTE(amt) {
		return k.Delegators.Remove(ctx, delAddr)
	}
	del.Amount = del.Amount.Sub(amt)
	err := k.Delegators.Set(ctx, delAddr, del)
	if err != nil {
		return err
	}
	reporterVal := sdk.ValAddress(sdk.MustAccAddressFromBech32(reporter.GetReporter()))
	return k.AfterDelegationModified(ctx, delAddr, reporterVal, del.Amount)
}

func (k Keeper) UpdateOrRemoveReporter(ctx context.Context, key sdk.AccAddress, rep types.OracleReporter, amt math.Int) error {
	if rep.TotalTokens.LTE(amt) {
		if err := k.Reporters.Remove(ctx, key); err != nil {
			return err
		}
		reporterVal := sdk.ValAddress(key)
		return k.AfterReporterRemoved(ctx, reporterVal)
	}
	rep.TotalTokens = rep.TotalTokens.Sub(amt)
	return k.Reporters.Set(ctx, key, rep)

}

func (k Keeper) UpdateOrRemoveSource(ctx context.Context, key collections.Pair[sdk.AccAddress, sdk.ValAddress], s types.TokenOrigin, amt math.Int) (err error) {
	if s.Amount.LTE(amt) {
		return k.TokenOrigin.Remove(ctx, key)
	}
	s.Amount = s.Amount.Sub(amt)
	return k.TokenOrigin.Set(ctx, key, s)
}
