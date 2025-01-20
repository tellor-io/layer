package keeper

import (
	"context"
	"fmt"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
func (k Keeper) PayFromBond(ctx sdk.Context, reporterAddr sdk.AccAddress, fee sdk.Coin, hashId []byte) error {
	return k.reporterKeeper.FeefromReporterStake(ctx, reporterAddr, fee.Amount, hashId)
}

// Pay dispute fee
func (k Keeper) PayDisputeFee(ctx sdk.Context, proposer sdk.AccAddress, fee sdk.Coin, fromBond bool, hashId []byte) error {
	if fromBond {
		// pay fee from given validator
		err := k.PayFromBond(ctx, proposer, fee, hashId)
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

// return slashed tokens when reporter either wins dispute or dispute is invalid
func (k Keeper) ReturnSlashedTokens(ctx context.Context, dispute types.Dispute) error {
	pool, err := k.reporterKeeper.ReturnSlashedTokens(ctx, dispute.SlashAmount, dispute.HashId)
	if err != nil {
		return err
	}

	coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, dispute.SlashAmount))
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, pool, coins)
}

func (k Keeper) ReturnFeetoStake(ctx context.Context, hashId []byte, remainingAmt math.Int) error {
	err := k.reporterKeeper.FeeRefund(ctx, hashId, remainingAmt)
	if err != nil {
		return err
	}

	coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, remainingAmt))
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, coins)
}
