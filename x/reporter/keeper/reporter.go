package keeper

import (
	"context"
	"errors"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) ValidateAndSetAmount(ctx context.Context, delegator sdk.AccAddress, originAmounts []*types.TokenOrigin, amount math.Int) error {
	_amt := math.ZeroInt()
	for _, origin := range originAmounts {
		_amt = _amt.Add(origin.Amount)
	}

	if !amount.Equal(_amt) {
		return errorsmod.Wrapf(types.ErrTokenAmountMismatch, "got %v as amount, but sum of token origins is %v", amount, _amt)
	}
	for _, origin := range originAmounts {
		valAddr := sdk.ValAddress(origin.ValidatorAddress)

		tokenSource, err := k.TokenOrigin.Get(ctx, collections.Join(delegator.Bytes(), valAddr.Bytes()))
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return errorsmod.Wrapf(err, "unable to fetch token origin")
			} else {
				// not found so initialize
				tokenSource = math.ZeroInt()
			}
		}
		validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to fetch validator for source tokens %v", origin)
		}
		delegation, err := k.stakingKeeper.GetDelegation(ctx, delegator, valAddr)
		if err != nil {
			return err
		}
		// check if the delegator has enough tokens bonded with validator, this would be the sum
		// of what is currently delegated to reporter plus the amount being added in this transaction
		sum := tokenSource.Add(origin.Amount)
		tokensFromShares := validator.TokensFromShares(delegation.GetShares()).TruncateInt()
		if tokensFromShares.LT(sum) {
			return errorsmod.Wrapf(types.ErrInsufficientTokens, "insufficient tokens bonded with validator %v", valAddr)
		}
		tokenSource = sum
		if err := k.TokenOrigin.Set(ctx, collections.Join(delegator.Bytes(), valAddr.Bytes()), tokenSource); err != nil {
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
	if err := k.DelegatorCheckpoint.Set(ctx, collections.Join(delAddr.Bytes(), sdk.UnwrapSDKContext(ctx).BlockHeight()), del.Amount); err != nil {
		return err
	}
	err := k.Delegators.Set(ctx, delAddr, del)
	if err != nil {
		return err
	}
	reporterVal := sdk.ValAddress(reporter.GetReporter())
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
	if err := k.ReporterCheckpoint.Set(ctx, collections.Join(key.Bytes(), sdk.UnwrapSDKContext(ctx).BlockHeight()), rep.TotalTokens); err != nil {
		return err
	}

	if err := k.Reporters.Set(ctx, key, rep); err != nil {
		return err
	}

	if err := k.UpdateTotalPower(ctx, amt, true); err != nil {
		return err
	}
	return k.AfterReporterModified(ctx, key)
}

func (k Keeper) UpdateOrRemoveSource(ctx context.Context, key collections.Pair[[]byte, []byte], srcAmount, amt math.Int) (err error) {
	// amount is the current staked amount in staking mod
	// so if current amount is zero remove the source
	if amt.IsZero() {
		return k.TokenOrigin.Remove(ctx, key)
	}
	return k.TokenOrigin.Set(ctx, key, amt)
}

func (k Keeper) UndelegateSource(ctx context.Context, key collections.Pair[[]byte, []byte], currentAmount, newAmount math.Int) error {
	if newAmount.GTE(currentAmount) {
		return k.TokenOrigin.Remove(ctx, key)
	}
	return k.TokenOrigin.Set(ctx, key, currentAmount.Sub(newAmount))
}

func (k Keeper) Reporter(ctx context.Context, repAddr sdk.AccAddress) (*types.OracleReporter, error) {
	reporter, err := k.Reporters.Get(ctx, repAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, types.ErrReporterDoesNotExist
		}
		return nil, err
	}
	return &reporter, nil
}

func (k Keeper) TotalReporterPower(ctx context.Context) (math.Int, error) {
	return k.TotalPowerAtBlock(ctx, sdk.UnwrapSDKContext(ctx).BlockHeight())
}

func (k Keeper) TotalPowerAtBlock(ctx context.Context, blockHeight int64) (math.Int, error) {
	totalPower := math.ZeroInt()
	rng := new(collections.Range[int64]).EndInclusive(blockHeight).Descending()
	err := k.TotalPower.Walk(ctx, rng, func(key int64, value math.Int) (stop bool, err error) {
		totalPower = value
		return true, nil
	})
	return totalPower, err
}

// alias
func (k Keeper) Delegation(ctx context.Context, delegator sdk.AccAddress) (types.Delegation, error) {
	return k.Delegators.Get(ctx, delegator)
}
