package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	layertypes "github.com/tellor-io/layer/types"
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
	return k.reporterKeeper.FeefromReporterStake(ctx, reporterAddr, fee.Amount)
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

// return slashed tokens when reporter either wins dispute or dispute is invalid
func (k Keeper) ReturnSlashedTokens(ctx sdk.Context, dispute types.Dispute) error {

	err := k.reporterKeeper.ReturnSlashedTokens(ctx, dispute.ReportEvidence.Reporter, dispute.ReportEvidence.BlockNumber, dispute.SlashAmount)
	if err != nil {
		return err
	}

	coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, dispute.SlashAmount))
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, coins)
}

func (k Keeper) ReturnFeetoStake(ctx sdk.Context, repAddr string, height int64, remainingAmt math.Int) error {

	err := k.reporterKeeper.ReturnSlashedTokens(ctx, repAddr, height, remainingAmt)
	if err != nil {
		return err
	}

	coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, remainingAmt))
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, coins)
}
