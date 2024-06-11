package keeper

import (
	"context"
	"errors"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

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
	dispute.FeePayers = append(dispute.FeePayers, types.PayerInfo{
		PayerAddress: sender.Bytes(),
		Amount:       msg.Amount.Amount,
		FromBond:     msg.PayFromBond,
		BlockNumber:  ctx.BlockHeight(),
	})
	dispute.FeeTotal = dispute.FeeTotal.Add(msg.Amount.Amount)
	if dispute.FeeTotal.Equal(dispute.SlashAmount) {
		if err := k.Keeper.SlashAndJailReporter(ctx, dispute.ReportEvidence, dispute.DisputeCategory, dispute.HashId); err != nil {
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

	return &types.MsgAddFeeToDisputeResponse{}, nil
}
