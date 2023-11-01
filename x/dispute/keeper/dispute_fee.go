package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k Keeper) PayFromAccount(ctx sdk.Context, addr sdk.AccAddress, fee sdk.Coin) error {
	if !k.bankKeeper.HasBalance(ctx, addr, fee) {
		return types.INSUFFICIENT_BALANCE
	}
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, sdk.NewCoins(fee)); err != nil {
		return fmt.Errorf("fee payment failed: %w", err)
	}
	return nil
}

func (k Keeper) PayFromStake(ctx sdk.Context, delAddr sdk.AccAddress, fee sdk.Coin) error {
	// TODO:
	// 1. Should validator bond/jail status be checked?
	// 2. Should only validator's own self delegation be checked?
	remainingAmount := fee.Amount

	var toUpdate []stakingtypes.Delegation
	var toRemove []stakingtypes.Delegation

	k.stakingKeeper.IterateDelegatorDelegations(ctx, delAddr, func(delegation stakingtypes.Delegation) (stop bool) {
		validator := k.validator(ctx, delegation.ValidatorAddress)
		if remainingAmount.IsZero() {
			return true // Break from delegation iterator
		}

		tokens := validator.TokensFromShares(delegation.Shares)
		updatedDelegation := delegation
		if tokens.GTE(remainingAmount.ToLegacyDec()) {
			shares, err := validator.SharesFromTokens(remainingAmount)
			if err != nil {
				panic(err)
			}
			updatedDelegation.Shares = updatedDelegation.Shares.Sub(shares)
			if updatedDelegation.Shares.IsZero() {
				toRemove = append(toRemove, updatedDelegation)
			} else {
				toUpdate = append(toUpdate, updatedDelegation)
			}
			remainingAmount = sdk.ZeroInt()
		} else {
			sharesToRemove, err := validator.SharesFromTokens(tokens.TruncateInt())
			if err != nil {
				panic(err)
			}
			updatedDelegation.Shares = updatedDelegation.Shares.Sub(sharesToRemove)
			toUpdate = append(toUpdate, updatedDelegation)
			remainingAmount = remainingAmount.Sub(tokens.TruncateInt())
		}
		return false
	})

	// Update and remove delegations outside of the iterator
	for _, delegation := range toUpdate {
		// Assuming you have an UpdateDelegation function
		k.stakingKeeper.SetDelegation(ctx, delegation)
	}

	for _, delegation := range toRemove {
		k.stakingKeeper.RemoveDelegation(ctx, delegation)
	}

	if !remainingAmount.IsZero() {
		return fmt.Errorf("insufficient stake for the specified fee")
	}
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, stakingtypes.BondedPoolName, types.ModuleName, sdk.NewCoins(fee))
	if err != nil {
		panic(err)
	}

	return nil
}

func (k Keeper) validator(ctx sdk.Context, valAddr string) stakingtypes.Validator {
	valAddress, err := sdk.ValAddressFromBech32(valAddr)
	if err != nil {
		panic(err)
	}
	validator, found := k.stakingKeeper.GetValidator(ctx, valAddress)
	if !found {
		panic(fmt.Errorf("validator %s not found", valAddr))
	}
	return validator
}

func (k Keeper) PayFromGivenValidator(ctx sdk.Context, delAddr sdk.AccAddress, operAddr sdk.ValAddress, fee sdk.Coin) error {
	// Fetch the delegation
	delegation, found := k.stakingKeeper.GetDelegation(ctx, delAddr, operAddr)
	if !found {
		return fmt.Errorf("no delegation found between delegator %s and validator %s", delAddr, operAddr)
	}
	// Get the validator info
	validator, found := k.stakingKeeper.GetValidator(ctx, operAddr)
	if !found {
		return fmt.Errorf("validator not found: %s", operAddr)
	}

	// Calculate tokens from the delegation's shares
	tokens := validator.TokensFromShares(delegation.Shares).TruncateInt()

	// Ensure there are enough tokens to pay the fee
	if tokens.LT(fee.Amount) {
		return fmt.Errorf("not enough stake to pay the fee")
	}

	// Calculate shares equivalent to the fee
	sharesDeducted, err := validator.SharesFromTokens(fee.Amount)
	if err != nil {
		return err
	}

	// Deduct tokens and shares
	delegation.Shares = delegation.Shares.Sub(sharesDeducted)
	validator, _ = k.stakingKeeper.RemoveValidatorTokensAndShares(ctx, validator, sharesDeducted)

	// Check if the delegation should be removed (because it's now zero)
	if delegation.Shares.IsZero() {
		k.stakingKeeper.RemoveDelegation(ctx, delegation)
	} else {
		k.stakingKeeper.SetDelegation(ctx, delegation)
	}
	var poolName string
	if validator.IsBonded() {
		poolName = stakingtypes.BondedPoolName
	} else {
		poolName = stakingtypes.NotBondedPoolName
	}
	err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, poolName, types.ModuleName, sdk.NewCoins(fee))
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) ReturnFeesToAccount(ctx sdk.Context, addr sdk.AccAddress, fee sdk.Coin) error {
	err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(fee))
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) ReturnFeesAsStaked(ctx sdk.Context, delAddr sdk.AccAddress, val sdk.ValAddress, fee sdk.Coin) error {
	// TODO: loophole bypassing redelegation MaxEntries check
	// k.GetLastRefundBlockTime(ctx, delAddr)
	// k.SetLastRefundBlockTime(ctx, delAddr, currentBlock)
	validator, found := k.stakingKeeper.GetValidator(ctx, val)
	if !found {
		return stakingtypes.ErrNoValidatorFound
	}
	_, err := k.stakingKeeper.Delegate(ctx, delAddr, fee.Amount, stakingtypes.Unbonded, validator, true)
	if err != nil {
		return err
	}
	return nil
}

// Pay fee from validator's bond can only be called by the validator itself
func (k Keeper) ValidatorPayFeeFromBond(ctx sdk.Context, delAddr sdk.AccAddress, fee sdk.Coin) error {
	validator, found := k.stakingKeeper.GetValidator(ctx, sdk.ValAddress(delAddr))
	if !found {
		return stakingtypes.ErrNoValidatorFound
	}

	// Check if validator has tokens to pay the fee
	if fee.Amount.GT(validator.GetBondedTokens()) {
		return fmt.Errorf("not enough stake to pay the fee")
	}

	// Deduct tokens from validator
	validator = k.stakingKeeper.RemoveValidatorTokens(ctx, validator, fee.Amount)

	// Send fee to module account
	var poolName string
	if validator.IsBonded() {
		poolName = stakingtypes.BondedPoolName
	} else {
		poolName = stakingtypes.NotBondedPoolName
	}
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, poolName, types.ModuleName, sdk.NewCoins(fee)); err != nil {
		return err
	}
	return nil
}
