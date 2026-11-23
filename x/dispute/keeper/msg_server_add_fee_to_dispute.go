package keeper

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// After this affordable tier, a new partial payer must provide at least
// 1% of the quote. Dust cannot exhaust a finite set of contribution slots.
const freeRoundOneFeePayers = 32

// Contribute to the first round payment of a proposed dispute if the proposer did not pay the full fee.
func (k msgServer) AddFeeToDispute(goCtx context.Context,
	msg *types.MsgAddFeeToDispute,
) (*types.MsgAddFeeToDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	sender, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	if msg.Amount.Denom != layer.BondDenom {
		return nil, errors.New("fee must be paid in loya")
	}
	// assert fee is greater than zero
	if msg.Amount.Amount.LTE(math.ZeroInt()) {
		return nil, errors.New("fee must be a positive amount")
	}
	dispute, err := k.Disputes.Get(ctx, msg.DisputeId)
	if err != nil {
		return nil, err
	}
	// if disputed reporter wants to add to fee, they have to use free floating tokens
	if sender.Equals(sdk.MustAccAddressFromBech32(dispute.InitialEvidence.Reporter)) && msg.PayFromBond {
		return nil, errors.New("disputed reporter can't add fee from bond")
	}
	// check if time to add fee has expired
	if ctx.BlockTime().After(dispute.DisputeEndTime) {
		return nil, types.ErrDisputeTimeExpired
	}
	// Only a first round awaiting funding can accept another contribution.
	if dispute.DisputeStatus != types.Prevote || dispute.FeeTotal.GTE(dispute.SlashAmount) {
		return nil, types.ErrDisputeFeeAlreadyMet
	}

	remaining := dispute.SlashAmount.Sub(dispute.FeeTotal)
	if msg.Amount.Amount.GT(remaining) {
		msg.Amount.Amount = remaining
	}

	payerKey := collections.Join(dispute.DisputeId, sender.Bytes())
	payer, err := k.DisputeFeePayer.Get(ctx, payerKey)
	newPayer := errors.Is(err, collections.ErrNotFound)
	switch {
	case newPayer:
		payer = types.PayerInfo{Amount: math.ZeroInt(), FromBond: msg.PayFromBond}
	case err != nil:
		return nil, err
	case payer.FromBond != msg.PayFromBond:
		return nil, errors.New("PayFromBond must match previously paid fee origin")
	}
	if newPayer && !msg.PayFromBond {
		if err := k.checkNewRoundOnePayer(ctx, dispute.DisputeId, dispute.SlashAmount, msg.Amount.Amount, remaining); err != nil {
			return nil, err
		}
	}
	received, err := k.payDisputeFee(ctx, sender, msg.Amount, msg.PayFromBond, dispute.HashId, true)
	if err != nil {
		return nil, err
	}
	if newPayer && msg.PayFromBond {
		if err := k.checkNewRoundOnePayer(ctx, dispute.DisputeId, dispute.SlashAmount, received, remaining); err != nil {
			return nil, err
		}
	}
	payer.Amount = payer.Amount.Add(received)
	if err := k.DisputeFeePayer.Set(ctx, payerKey, payer); err != nil {
		return nil, err
	}

	dispute.FeeTotal = dispute.FeeTotal.Add(received)
	if err := k.startRoundOneIfFunded(ctx, &dispute); err != nil {
		return nil, err
	}
	if err := k.Keeper.Disputes.Set(ctx, dispute.DisputeId, dispute); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"fee_added_to_dispute",
			sdk.NewAttribute("payer", msg.Creator),
			sdk.NewAttribute("dispute_id", strconv.FormatUint(dispute.DisputeId, 10)),
			sdk.NewAttribute("amount", received.String()),
			sdk.NewAttribute("paid_from_bond", strconv.FormatBool(msg.PayFromBond)),
		),
	})
	return &types.MsgAddFeeToDisputeResponse{}, nil
}

func (k Keeper) checkNewRoundOnePayer(ctx sdk.Context, id uint64, quote, credit, remaining math.Int) error {
	// An exact finish must still be possible after the priced-tier floor
	// exceeds the remaining balance, including for legacy payer sets.
	minimum := quote.SubRaw(1).QuoRaw(100).AddRaw(1)
	if credit.Equal(remaining) || credit.GTE(minimum) {
		return nil
	}
	iter, err := k.DisputeFeePayer.Iterate(ctx, collections.NewPrefixedPairRange[uint64, []byte](id))
	if err != nil {
		return err
	}
	defer iter.Close()
	for count := 1; iter.Valid(); iter.Next() {
		if count == freeRoundOneFeePayers {
			return fmt.Errorf("minimum fee for a new dispute payer is %s loya unless completing funding", minimum)
		}
		count++
	}
	return nil
}
