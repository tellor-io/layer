package keeper

import (
	"context"
	"errors"
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

// feeRoundingDust is the largest gap between escrowed slash and fee coins on
// hand that share rounding can explain. One unbond truncates by at most a
// loya; this covers a payer whose fee came from many delegations.
const feeRoundingDust int64 = 100

// adjustDisputeFeeToCollectedSlash runs after round-1 escrow. When the coins
// actually escrowed are below the quote, the dispute is rewritten to the coins
// that can back that slash and each round-1 payer's excess is returned in this
// transaction. From-bond payers are scaled on the coins actually unbonded, so a
// few loya of share rounding does not fail the dispute. A collected total at or
// above the quote leaves the quoted fee unchanged.
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
	available := make([]math.Int, len(payers))
	availableTotal := math.ZeroInt()
	for i, payer := range payers {
		have, err := k.roundPayerAvailable(ctx, dispute.HashId, payer)
		if err != nil {
			return err
		}
		available[i] = have
		availableTotal = availableTotal.Add(have)
	}
	if !availableTotal.IsPositive() {
		return fmt.Errorf("collected slash %s cannot be scaled across round-1 fees %s", collected, availableTotal)
	}

	// The slash can only be matched by coins the module actually received.
	// A shortfall of a few loya is share rounding: keep every fee coin and size
	// the dispute to that. The escrow snapshot stays at the real collected total.
	retain := collected
	if !collected.LT(availableTotal) {
		shortfall := collected.Sub(availableTotal)
		if shortfall.GT(math.NewInt(feeRoundingDust)) {
			return fmt.Errorf("collected slash %s exceeds round-1 fees on hand %s", collected, availableTotal)
		}
		retain = availableTotal
	}
	keptShares := available
	if retain.LT(availableTotal) {
		keptShares = proportionalKeptAmounts(available, availableTotal, retain)
	}
	burn := math.LegacyNewDecFromInt(retain).Mul(math.LegacyNewDec(1)).Quo(math.LegacyNewDec(20)).TruncateInt()
	dispute.SlashAmount = retain
	dispute.DisputeFee = retain
	dispute.FeeTotal = retain
	dispute.BurnAmount = burn

	attrs := []sdk.Attribute{
		sdk.NewAttribute("dispute_id", strconv.FormatUint(dispute.DisputeId, 10)),
		sdk.NewAttribute("quote", quote.String()),
		sdk.NewAttribute("collected", collected.String()),
		sdk.NewAttribute("burn_amount", burn.String()),
	}
	for i, payer := range payers {
		kept := keptShares[i]
		have := available[i]
		if kept.GT(have) {
			return fmt.Errorf("scaled dispute fee share %s exceeds payer coins %s", kept, have)
		}
		excess := have.Sub(kept)
		refunded := excess
		if excess.IsPositive() {
			if payer.info.FromBond {
				bondedAmt, unbondedAmt, err := k.reporterKeeper.ReturnFeeExcess(ctx, dispute.HashId, payer.addr, excess, kept)
				if err != nil {
					return err
				}
				if err := k.sendStakeReturnsToPools(ctx, bondedAmt, unbondedAmt); err != nil {
					return err
				}
				remaining, err := k.reporterKeeper.FeePaidFromStakeByPayer(ctx, dispute.HashId, payer.addr)
				if err != nil {
					if !errors.Is(err, collections.ErrNotFound) {
						return err
					}
					kept = math.ZeroInt()
				} else if remaining.Total.IsNil() || !remaining.Total.IsPositive() {
					kept = math.ZeroInt()
				} else {
					kept = remaining.Total
				}
				refunded = have.Sub(kept)
			} else if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payer.addr, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, excess))); err != nil {
				return err
			}
		}
		attrs = append(attrs,
			sdk.NewAttribute("payer", payer.addr.String()),
			sdk.NewAttribute("refund", refunded.String()),
		)
		payerKey := collections.Join(dispute.DisputeId, payer.addr.Bytes())
		if !kept.IsPositive() {
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

// roundPayerAvailable is the coins a round-1 payer actually put in the module.
// An account payment is exact. A from-bond payment is the unbonded tracker, which
// can differ by a few loya from the amount recorded on the payer.
func (k Keeper) roundPayerAvailable(ctx sdk.Context, hashId []byte, payer roundPayer) (math.Int, error) {
	if !payer.info.FromBond {
		if payer.info.Amount.IsNil() || payer.info.Amount.IsNegative() {
			return math.Int{}, fmt.Errorf("round-1 fee payer %s has a negative amount", payer.addr)
		}
		return payer.info.Amount, nil
	}
	tracked, err := k.reporterKeeper.FeePaidFromStakeByPayer(ctx, hashId, payer.addr)
	if err != nil {
		return math.Int{}, err
	}
	if tracked.Total.IsNil() || tracked.Total.IsNegative() {
		return math.Int{}, fmt.Errorf("from-bond fee tracker for payer %s is negative", payer.addr)
	}
	return tracked.Total, nil
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
