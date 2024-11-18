package keeper

import (
	"context"
	"errors"
	"strconv"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
	// check if fee has been already met
	if dispute.FeeTotal.GTE(dispute.SlashAmount) {
		return nil, types.ErrDisputeFeeAlreadyMet
	}

	// validate fee amount
	if dispute.FeeTotal.Add(msg.Amount.Amount).GT(dispute.SlashAmount) {
		msg.Amount.Amount = dispute.SlashAmount.Sub(dispute.FeeTotal)
	}

	// Pay fee
	if err := k.Keeper.PayDisputeFee(ctx, sender, msg.Amount, msg.PayFromBond, dispute.HashId); err != nil {
		return nil, err
	}
	// Don't take payment more than slash amount
	fee := dispute.SlashAmount.Sub(dispute.FeeTotal)
	if msg.Amount.Amount.GT(fee) {
		msg.Amount.Amount = fee
	}
	// dispute fee payer
	if err := k.Keeper.DisputeFeePayer.Set(ctx, collections.Join(dispute.DisputeId, sender.Bytes()), types.PayerInfo{
		Amount:   msg.Amount.Amount,
		FromBond: msg.PayFromBond,
	}); err != nil {
		return nil, err
	}

	dispute.FeeTotal = dispute.FeeTotal.Add(msg.Amount.Amount)
	if dispute.FeeTotal.Equal(dispute.SlashAmount) {
		if err := k.Keeper.SlashAndJailReporter(ctx, dispute.InitialEvidence, dispute.DisputeCategory, dispute.InitialEvidence.QueryId, dispute.HashId); err != nil {
			return nil, err
		}
		// begin voting immediately
		dispute.DisputeEndTime = ctx.BlockTime().Add(THREE_DAYS)
		dispute.DisputeStatus = types.Voting
		if err := k.Keeper.SetStartVote(ctx, dispute.DisputeId); err != nil {
			return nil, err
		}
	}
	if err := k.Keeper.Disputes.Set(ctx, dispute.DisputeId, dispute); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"fee_added_to_dispute",
			sdk.NewAttribute("payer", msg.Creator),
			sdk.NewAttribute("dispute_id", strconv.FormatUint(dispute.DisputeId, 10)),
			sdk.NewAttribute("amount", msg.Amount.Amount.String()),
			sdk.NewAttribute("paid_from_bond", strconv.FormatBool(msg.PayFromBond)),
		),
	})
	return &types.MsgAddFeeToDisputeResponse{}, nil
}
