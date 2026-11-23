package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// payDisputeFee returns the coins transferred, not the requested amount.
func (k Keeper) payDisputeFee(ctx sdk.Context, payer sdk.AccAddress, fee sdk.Coin, fromBond bool, hashId []byte, isFirstRound bool) (math.Int, error) {
	if fromBond {
		return k.reporterKeeper.FeefromReporterStake(ctx, payer, fee.Amount, hashId, isFirstRound)
	}
	if !k.bankKeeper.HasBalance(ctx, payer, fee) {
		return math.Int{}, types.INSUFFICIENT_BALANCE
	}
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, payer, types.ModuleName, sdk.NewCoins(fee)); err != nil {
		return math.Int{}, fmt.Errorf("fee payment failed: %w", err)
	}
	return fee.Amount, nil
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
	addr     sdk.AccAddress
	info     types.PayerInfo
	received math.Int
}

func (k Keeper) startRoundOneIfFunded(ctx sdk.Context, dispute *types.Dispute) error {
	quote := dispute.SlashAmount
	if dispute.FeeTotal.LT(quote) {
		return nil
	}

	// Pre-upgrade Prevote disputes can show the quote paid even though the
	// reporter module received less. Reopen the difference before any slash.
	payers, paid, err := k.roundOneReceipts(ctx, dispute.DisputeId, dispute.HashId, true)
	if err != nil {
		return err
	}
	dispute.FeeTotal = paid
	if paid.LT(quote) {
		return k.applyRoundOneReceipts(ctx, dispute, payers, paid, paid)
	}

	if err := k.SlashAndJailReporter(ctx, dispute.InitialEvidence, dispute.DisputeCategory, dispute.HashId); err != nil {
		return err
	}
	collected, err := k.reporterKeeper.DisputedDelegationTotal(ctx, dispute.HashId)
	if err != nil {
		return err
	}
	retain := math.MinInt(quote, collected)
	if err := k.applyRoundOneReceipts(ctx, dispute, payers, paid, retain); err != nil {
		return err
	}
	burn := retain.QuoRaw(20)
	if !retain.Equal(quote) || paid.GT(retain) {
		ctx.EventManager().EmitEvent(sdk.NewEvent("dispute_fee_adjusted",
			sdk.NewAttribute("dispute_id", strconv.FormatUint(dispute.DisputeId, 10)),
			sdk.NewAttribute("quote", quote.String()),
			sdk.NewAttribute("collected", collected.String()),
			sdk.NewAttribute("retained_fee", retain.String()),
			sdk.NewAttribute("burn_amount", burn.String()),
		))
	}
	dispute.SlashAmount = retain
	dispute.DisputeFee = retain
	dispute.FeeTotal = retain
	dispute.BurnAmount = burn
	dispute.DisputeEndTime = ctx.BlockTime().Add(THREE_DAYS)
	dispute.DisputeStatus = types.Voting
	return k.SetStartVote(ctx, dispute.DisputeId)
}

// applyRoundOneReceipts reconciles credits and returns any surplus over the
// retained fee. When funding is still short, retain is the full receipt total.
func (k Keeper) applyRoundOneReceipts(ctx sdk.Context, dispute *types.Dispute, payers []roundPayer, paid, retain math.Int) error {
	var keptShares []math.Int
	var disputeID string
	if retain.LT(paid) {
		keptShares = proportionalKeptAmounts(payers, paid, retain)
		disputeID = strconv.FormatUint(dispute.DisputeId, 10)
	}

	for i, payer := range payers {
		kept := payer.received
		if keptShares != nil {
			kept = keptShares[i]
		}
		if kept.IsPositive() && keptShares == nil && kept.Equal(payer.info.Amount) {
			continue
		}
		excess := payer.received.Sub(kept)
		if payer.info.FromBond && (excess.IsPositive() || kept.IsZero()) {
			bonded, unbonded, err := k.reporterKeeper.ReturnFeeExcess(ctx, dispute.HashId, payer.addr, excess, kept)
			if err != nil {
				return err
			}
			if err := k.sendStakeReturnsToPools(ctx, bonded, unbonded); err != nil {
				return err
			}
		} else if excess.IsPositive() {
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payer.addr, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, excess))); err != nil {
				return err
			}
		}
		if keptShares != nil {
			ctx.EventManager().EmitEvent(sdk.NewEvent("dispute_fee_payer_adjusted",
				sdk.NewAttribute("dispute_id", disputeID),
				sdk.NewAttribute("payer", payer.addr.String()),
				sdk.NewAttribute("refund", excess.String()),
				sdk.NewAttribute("retained_fee", kept.String()),
			))
		}
		if kept.IsPositive() && kept.Equal(payer.info.Amount) {
			continue
		}
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
	return nil
}

// roundOneReceipts attributes only coins held for this dispute. Funding
// collects isolated payer receipts so it can rewrite them. Settlement needs
// only the total and can read a hash-only tracker for a sole bonded payer.
func (k Keeper) roundOneReceipts(ctx context.Context, disputeID uint64, hashId []byte, collectPayers bool) ([]roundPayer, math.Int, error) {
	rng := collections.NewPrefixedPairRange[uint64, []byte](disputeID)
	iter, err := k.DisputeFeePayer.Iterate(ctx, rng)
	if err != nil {
		return nil, math.Int{}, err
	}
	defer iter.Close()
	if !iter.Valid() {
		return nil, math.Int{}, fmt.Errorf("no round-one fee receipts for dispute %d", disputeID)
	}

	var payers []roundPayer
	var hashOnlyPayer sdk.AccAddress
	paid := math.ZeroInt()
	bondPayers := 0
	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return nil, math.Int{}, err
		}
		info, err := iter.Value()
		if err != nil {
			return nil, math.Int{}, err
		}
		addr := sdk.AccAddress(key.K2())
		received := info.Amount
		if info.FromBond {
			bondPayers++
			tracked, err := k.reporterKeeper.FeePaidFromStakeByPayer(ctx, hashId, addr)
			if err != nil {
				if collectPayers || hashOnlyPayer != nil || !errors.Is(err, collections.ErrNotFound) {
					return nil, math.Int{}, fmt.Errorf("round-one stake receipt for dispute %d, payer %s: %w", disputeID, addr, err)
				}
				hashOnlyPayer = bytes.Clone(addr)
				continue
			}
			received = tracked.Total
		}
		if received.IsNil() || received.IsNegative() {
			return nil, math.Int{}, fmt.Errorf("invalid round-one fee receipt for dispute %d, payer %s", disputeID, addr)
		}
		paid = paid.Add(received)
		if collectPayers {
			// BytesKey decodes a view of the iterator key; retain it after Close.
			payers = append(payers, roundPayer{addr: bytes.Clone(addr), info: info, received: received})
		}
	}
	if hashOnlyPayer != nil {
		if bondPayers != 1 {
			return nil, math.Int{}, fmt.Errorf("cannot attribute hash-only stake receipt for dispute %d with multiple bonded payers", disputeID)
		}
		received, err := k.reporterKeeper.FeePaidFromStakeTotalByPayer(ctx, hashId, hashOnlyPayer)
		if err != nil {
			return nil, math.Int{}, err
		}
		if received.IsNil() || received.IsNegative() {
			return nil, math.Int{}, fmt.Errorf("invalid hash-only stake receipt for dispute %d", disputeID)
		}
		paid = paid.Add(received)
	}
	return payers, paid, nil
}

// proportionalKeptAmounts splits collected across payers in proportion to what
// they paid. Largest-remainder rounding makes the kept shares sum to collected;
// ties keep the payer's order in the store.
func proportionalKeptAmounts(payers []roundPayer, paidTotal, collected math.Int) []math.Int {
	type rankedShare struct {
		index     int
		remainder math.Int
	}
	out := make([]math.Int, len(payers))
	ranked := make([]rankedShare, 0, len(payers))
	distributed := math.ZeroInt()
	for i, payer := range payers {
		scaled := payer.received.Mul(collected)
		out[i] = scaled.Quo(paidTotal)
		distributed = distributed.Add(out[i])
		if remainder := scaled.Mod(paidTotal); remainder.IsPositive() {
			ranked = append(ranked, rankedShare{index: i, remainder: remainder})
		}
	}
	dust := collected.Sub(distributed).Uint64()
	if dust == 0 {
		return out
	}
	slices.SortFunc(ranked, func(a, b rankedShare) int {
		switch {
		case a.remainder.GT(b.remainder):
			return -1
		case a.remainder.LT(b.remainder):
			return 1
		default:
			return a.index - b.index
		}
	})
	// Each positive remainder contributes less than one loya to dust, so
	// there are enough distinct ranked payers to assign it without rescans.
	for _, payer := range ranked[:dust] {
		out[payer.index] = out[payer.index].AddRaw(1)
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
