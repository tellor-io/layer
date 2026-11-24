package keeper

import (
	"context"
	"fmt"
	"strconv"

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
func (k Keeper) ReturnSlashedTokens(ctx context.Context, dispute types.Dispute, extraReturn math.Int) error {
	bondedAmt, unbondedAmt, err := k.reporterKeeper.ReturnSlashedTokens(ctx, dispute.HashId, extraReturn)
	if err != nil {
		return err
	}

	return k.sendStakeReturnsToPools(ctx, bondedAmt, unbondedAmt)
}

func (k Keeper) sendStakeReturnsToPools(ctx context.Context, bondedAmt, unbondedAmt math.Int) error {
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
	return nil
}

// adjustDisputeFeeToCollectedSlash runs after round-1 escrow. When the coins
// actually escrowed are below the quote, only the retained fee and its burn are
// resized. The gross round-one payer receipts and escrow tracker are left
// untouched for the final claim.
func (k Keeper) adjustDisputeFeeToCollectedSlash(ctx sdk.Context, dispute *types.Dispute) error {
	collected, err := k.reporterKeeper.DisputedDelegationTotal(ctx, dispute.HashId)
	if err != nil {
		return err
	}
	// Equal is the usual escrow result. A collected total at or above the quote,
	// or a zero result, keeps the quoted fee unchanged. A positive quote with
	// zero recovery is rejected upstream by EscrowReporterStake.
	if !collected.IsPositive() || !collected.LT(dispute.SlashAmount) {
		return nil
	}

	quote := dispute.SlashAmount
	burn := collected.QuoRaw(20)
	dispute.DisputeFee = collected
	dispute.BurnAmount = burn

	ctx.EventManager().EmitEvent(sdk.NewEvent("dispute_fee_adjusted",
		sdk.NewAttribute("dispute_id", strconv.FormatUint(dispute.DisputeId, 10)),
		sdk.NewAttribute("quote", quote.String()),
		sdk.NewAttribute("collected", collected.String()),
		sdk.NewAttribute("burn_amount", burn.String()),
	))
	return nil
}

func (k Keeper) ReturnFeetoStake(ctx context.Context, hashId []byte, feePayer sdk.AccAddress, remainingAmt math.Int) error {
	bondedAmt, unbondedAmt, err := k.reporterKeeper.FeeRefund(ctx, hashId, feePayer, remainingAmt)
	if err != nil {
		return err
	}

	return k.sendStakeReturnsToPools(ctx, bondedAmt, unbondedAmt)
}
