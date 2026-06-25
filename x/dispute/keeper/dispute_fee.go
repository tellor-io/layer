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
func (k Keeper) PayFromBond(ctx sdk.Context, reporterAddr sdk.AccAddress, fee sdk.Coin, hashId []byte, isFirstRound bool) error {
	return k.reporterKeeper.FeefromReporterStake(ctx, reporterAddr, fee.Amount, hashId, isFirstRound)
}

// Pay dispute fee
func (k Keeper) PayDisputeFee(ctx sdk.Context, proposer sdk.AccAddress, fee sdk.Coin, fromBond bool, hashId []byte, isFirstRound bool) error {
	if fromBond {
		// pay fee from given validator
		err := k.PayFromBond(ctx, proposer, fee, hashId, isFirstRound)
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
	bondedAmt, unbondedAmt, err := k.reporterKeeper.ReturnSlashedTokens(ctx, dispute.SlashAmount, dispute.HashId)
	if err != nil {
		return err
	}

	// Route the dispute-module coins to each pool separately. stakingKeeper.Delegate
	// with tokenSrc=Bonded&&validator.IsBonded() or tokenSrc=Unbonded&&!IsBonded() performs
	// no pool transfer itself, so the bonded and not-bonded pools each receive exactly
	// what was delegated into them across possibly mixed bonded/unbonded origins.
	if bondedAmt.IsPositive() {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, bondedAmt))); err != nil {
			return err
		}
	}
	if unbondedAmt.IsPositive() {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.NotBondedPoolName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, unbondedAmt))); err != nil {
			return err
		}
	}
	// The reporter keeper refunds each origin as a truncated integer share of
	// dispute.SlashAmount (and, when the reporter wins, a proportional winning
	// purse), so bondAmt across origins can sum to slightly less than
	// dispute.SlashAmount. Route the truncation dust to the bonded pool so no
	// coins are stranded in the dispute module; this matches the prior
	// single-pool behavior where the full slash amount moved to one (bonded)
	// pool.
	if dust := dispute.SlashAmount.Sub(bondedAmt).Sub(unbondedAmt); dust.IsPositive() {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, dust))); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) ReturnFeetoStake(ctx context.Context, hashId []byte, remainingAmt math.Int) error {
	err := k.reporterKeeper.FeeRefund(ctx, hashId, remainingAmt)
	if err != nil {
		return err
	}

	coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, remainingAmt))
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, coins)
}
