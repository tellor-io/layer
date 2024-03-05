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
func (k Keeper) PayFromBond(ctx sdk.Context, reporterAddr sdk.AccAddress, fee sdk.Coin) error {
	return k.reporterKeeper.EscrowReporterStake(ctx, reporterAddr, fee.Amount)
}

// Refund coins to bonded pool
func (k Keeper) RefundToBond(ctx sdk.Context, refundTo string, fee sdk.Coin) error {
	reporterAddr, err := sdk.AccAddressFromBech32(refundTo)
	if err != nil {
		return err
	}
	if err := k.reporterKeeper.AllocateRewardsToStake(ctx, reporterAddr, fee.Amount); err != nil {
		return err
	}
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(fee))
}

// Pay dispute fee
func (k Keeper) PayDisputeFee(ctx sdk.Context, sender string, fee sdk.Coin, fromBond bool) error {
	proposer := sdk.MustAccAddressFromBech32(sender)
	if fromBond {
		// pay fee from given validator
		err := k.PayFromBond(ctx, proposer, fee)
		if err != nil {
			return err
		}
	} else {
		err := k.PayFromAccount(ctx, proposer, fee)
		if err != nil {
			return err
		}
	}
	return nil
}
