package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k msgServer) AddFeeToDispute(goCtx context.Context,
	msg *types.MsgAddFeeToDispute) (*types.MsgAddFeeToDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute := k.GetDisputeById(ctx, msg.DisputeId)
	if dispute == nil {
		return nil, types.ErrDisputeDoesNotExist
	}
	// check if time to add fee has expired
	if ctx.BlockTime().After(dispute.DisputeEndTime) {
		return nil, types.ErrDisputeTimeExpired
	}
	// check if fee has been already met
	if dispute.FeeTotal.GTE(dispute.SlashAmount) {
		return nil, types.ErrDisputeFeeAlreadyMet
	}
	// Pay fee
	if err := k.Keeper.PayDisputeFee(ctx, msg.Creator, msg.ValidatorAddress, msg.Amount, msg.PayFromBond); err != nil {
		return nil, err
	}
	// Don't take payment more than slash amount
	fee := dispute.SlashAmount.Sub(dispute.FeeTotal)
	if msg.Amount.Amount.GT(fee) {
		msg.Amount.Amount = fee
	}
	dispute.FeePayers = append(dispute.FeePayers, types.PayerInfo{
		PayerAddress: msg.Creator,
		Amount:       msg.Amount,
		FromBond:     msg.PayFromBond,
	})
	dispute.FeeTotal = dispute.FeeTotal.Add(msg.Amount.Amount)
	if dispute.FeeTotal.Equal(dispute.SlashAmount) {
		k.SlashAndJailReporter(ctx, dispute.ReportEvidence, dispute.DisputeCategory)
	}
	k.Keeper.SetDisputeById(ctx, dispute.DisputeId, *dispute)
	k.Keeper.SetDisputeByReporter(ctx, *dispute)

	return &types.MsgAddFeeToDisputeResponse{}, nil
}
