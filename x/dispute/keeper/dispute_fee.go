package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

// Pay fee from account
func (k Keeper) PayFromAccount(ctx sdk.Context, addr sdk.AccAddress, fee sdk.Coin) error {
	if !k.bankKeeper.HasBalance(ctx, addr, fee) {
		return types.INSUFFICIENT_BALANCE
	}
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, sdk.NewCoins(fee)); err != nil {
		return fmt.Errorf("fee payment failed: %w", err)
	}
	return nil
}

// Pay fee from validator's bond can only be called by the validator itself
func (k Keeper) PayFromBond(ctx sdk.Context, delAddr sdk.AccAddress, fee sdk.Coin) error {
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
	switch validator.GetStatus() {
	case stakingtypes.Bonded:
		poolName = stakingtypes.BondedPoolName
	case stakingtypes.Unbonded, stakingtypes.Unbonding:
		poolName = stakingtypes.NotBondedPoolName
	default:
		panic("invalid validator status")
	}

	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, poolName, types.ModuleName, sdk.NewCoins(fee)); err != nil {
		return err
	}
	return nil
}

func (k Keeper) RefundToAccount(ctx sdk.Context, addr sdk.AccAddress, fee sdk.Coin) error {
	err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(fee))
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) RefundToBond(ctx sdk.Context, refundTo string, fee sdk.Coin) error {
	// TODO: loophole bypassing redelegation MaxEntries check
	// k.GetLastRefundBlockTime(ctx, delAddr)
	// k.SetLastRefundBlockTime(ctx, delAddr, currentBlock)
	delAddr := sdk.MustAccAddressFromBech32(refundTo)
	validator, found := k.stakingKeeper.GetValidator(ctx, sdk.ValAddress(delAddr))
	if !found {
		return stakingtypes.ErrNoValidatorFound
	}
	validator, _ = k.stakingKeeper.AddValidatorTokensAndShares(ctx, validator, fee.Amount)

	if validator.IsBonded() {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(fee)); err != nil {
			return err
		}
	} else {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.NotBondedPoolName, sdk.NewCoins(fee)); err != nil {
			return err
		}
	}
	return nil
}

// Returns Validtor object from validator address string
func (k Keeper) validator(ctx sdk.Context, valAddr string) stakingtypes.Validator {
	valAddress, err := sdk.ValAddressFromBech32(valAddr)
	if err != nil {
		panic(err)
	}
	val, found := k.stakingKeeper.GetValidator(ctx, valAddress)
	if !found {
		panic(fmt.Errorf("validator %s not found", valAddr))
	}
	return val
}
