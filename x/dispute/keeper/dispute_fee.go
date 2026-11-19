package keeper

import (
	"context"
	"fmt"
	"strconv"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
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

type roundPayer struct {
	addr sdk.AccAddress
	info types.PayerInfo
}

// adjustDisputeFeeToCollectedSlash runs after round-1 escrow. When the coins
// actually escrowed are below the quote, the dispute is rewritten to that
// amount and each round-1 payer's excess is returned in this transaction.
// A collected total at or above the quote leaves the quoted fee unchanged.
func (k Keeper) adjustDisputeFeeToCollectedSlash(ctx sdk.Context, dispute *types.Dispute) error {
	collected, err := k.reporterKeeper.DisputedDelegationTotal(ctx, dispute.HashId)
	if err != nil {
		return err
	}
	// Equal is the usual escrow result. Return before loading payers or refunding.
	// A collected total above the quote keeps the quoted fee too.
	if collected.IsNil() || collected.Equal(dispute.SlashAmount) || collected.GT(dispute.SlashAmount) {
		return nil
	}

	quote := dispute.SlashAmount
	payers, err := k.roundOnePayers(ctx, dispute.DisputeId)
	if err != nil {
		return err
	}
	paid := make([]math.Int, len(payers))
	paidTotal := math.ZeroInt()
	for i, payer := range payers {
		paid[i] = payer.info.Amount
		paidTotal = paidTotal.Add(payer.info.Amount)
	}
	if !paidTotal.IsPositive() || !collected.LT(paidTotal) {
		return fmt.Errorf("collected slash %s cannot be scaled across round-1 fees %s", collected, paidTotal)
	}

	keptShares := proportionalKeptAmounts(paid, paidTotal, collected)
	burn := math.LegacyNewDecFromInt(collected).Mul(math.LegacyNewDec(1)).Quo(math.LegacyNewDec(20)).TruncateInt()
	dispute.SlashAmount = collected
	dispute.DisputeFee = collected
	dispute.FeeTotal = collected
	dispute.BurnAmount = burn

	attrs := []sdk.Attribute{
		sdk.NewAttribute("dispute_id", strconv.FormatUint(dispute.DisputeId, 10)),
		sdk.NewAttribute("quote", quote.String()),
		sdk.NewAttribute("collected", collected.String()),
		sdk.NewAttribute("burn_amount", burn.String()),
	}
	for i, payer := range payers {
		kept := keptShares[i]
		if kept.GT(payer.info.Amount) {
			return fmt.Errorf("scaled dispute fee share %s exceeds payer amount %s", kept, payer.info.Amount)
		}
		excess := payer.info.Amount.Sub(kept)
		if excess.IsPositive() {
			if payer.info.FromBond {
				bondedAmt, unbondedAmt, err := k.reporterKeeper.ReturnFeeExcess(ctx, dispute.HashId, payer.addr, excess, kept)
				if err != nil {
					return err
				}
				if err := k.sendStakeReturnsToPools(ctx, bondedAmt, unbondedAmt); err != nil {
					return err
				}
			} else if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payer.addr, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, excess))); err != nil {
				return err
			}
		}
		attrs = append(attrs,
			sdk.NewAttribute("payer", payer.addr.String()),
			sdk.NewAttribute("refund", excess.String()),
		)
		payerKey := collections.Join(dispute.DisputeId, payer.addr.Bytes())
		if kept.IsZero() {
			if err := k.DisputeFeePayer.Remove(ctx, payerKey); err != nil {
				return err
			}
			continue
		}
		payer.info.Amount = kept
		if err := k.DisputeFeePayer.Set(ctx, payerKey, payer.info); err != nil {
			return err
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent("dispute_fee_adjusted", attrs...))
	return nil
}

func (k Keeper) roundOnePayers(ctx sdk.Context, disputeID uint64) ([]roundPayer, error) {
	rng := collections.NewPrefixedPairRange[uint64, []byte](disputeID)
	iter, err := k.DisputeFeePayer.Iterate(ctx, rng)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	payers := make([]roundPayer, 0)
	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return nil, err
		}
		info, err := iter.Value()
		if err != nil {
			return nil, err
		}
		addrBytes := make([]byte, len(key.K2()))
		copy(addrBytes, key.K2())
		payers = append(payers, roundPayer{addr: addrBytes, info: info})
	}
	return payers, nil
}

// proportionalKeptAmounts splits collected across payers in proportion to what
// they paid. Largest-remainder rounding makes the kept shares sum to collected.
func proportionalKeptAmounts(paid []math.Int, paidTotal, collected math.Int) []math.Int {
	type share struct {
		amount    math.Int
		remainder math.Int
	}
	shares := make([]share, len(paid))
	distributed := math.ZeroInt()
	for i, amount := range paid {
		scaled := amount.Mul(collected)
		shares[i] = share{
			amount:    scaled.Quo(paidTotal),
			remainder: scaled.Mod(paidTotal),
		}
		distributed = distributed.Add(shares[i].amount)
	}

	for dust := collected.Sub(distributed); dust.IsPositive(); dust = dust.Sub(math.OneInt()) {
		best := -1
		for i := range shares {
			if !shares[i].remainder.IsPositive() {
				continue
			}
			if best == -1 || shares[i].remainder.GT(shares[best].remainder) {
				best = i
			}
		}
		if best == -1 {
			shares[0].amount = shares[0].amount.Add(dust)
			break
		}
		shares[best].amount = shares[best].amount.Add(math.OneInt())
		shares[best].remainder = math.ZeroInt()
	}

	out := make([]math.Int, len(shares))
	for i, payerShare := range shares {
		out[i] = payerShare.amount
	}
	return out
}

func (k Keeper) ReturnFeetoStake(ctx context.Context, hashId []byte, feePayer sdk.AccAddress, remainingAmt math.Int) error {
	bondedAmt, unbondedAmt, err := k.reporterKeeper.FeeRefund(ctx, hashId, feePayer, remainingAmt)
	if err != nil {
		return err
	}

	return k.sendStakeReturnsToPools(ctx, bondedAmt, unbondedAmt)
}
