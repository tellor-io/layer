package keeper

import (
	"context"
	"errors"
	"fmt"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// propose a single dispute on an array of reports
//
// extra field evidence that is list of other reports
// iterate through those reports, see if it affects median, dispute those reports
// geter for if bridge depots have been claimed, bridge deposit id as input
// getter for a bunch of dispute info

func (k msgServer) ProposeDispute(goCtx context.Context, msg *types.MsgProposeDispute) (*types.MsgProposeDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	sender, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	if msg.Fee.Denom != layer.BondDenom {
		return nil, types.ErrInvalidFeeDenom.Wrapf("wrong fee denom: %s, expected: %s", msg.Fee.Denom, layer.BondDenom)
	}

	if msg.Fee.Amount.LT(layer.OnePercent) {
		return nil, types.ErrMinimumTRBrequired.Wrapf("fee %s doesn't meet minimum fee required", msg.Fee.Amount)
	}
	dispute, err := k.GetDisputeByReporter(ctx, *msg.Report, msg.DisputeCategory)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			if err := k.Keeper.SetNewDispute(ctx, sender, *msg); err != nil {
				return nil, err
			}
			// event for new dispute
			sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
				sdk.NewEvent(
					"new_dispute",
					sdk.NewAttribute("dispute_id", fmt.Sprintf("%d", dispute.DisputeId)),
					sdk.NewAttribute("creator", msg.Creator),
					sdk.NewAttribute("dispute_category", msg.DisputeCategory.String()),
					sdk.NewAttribute("fee", msg.Fee.String()),
					sdk.NewAttribute("report", msg.Report.String()),
				),
			})
			return &types.MsgProposeDisputeResponse{}, nil
		}
		return nil, err
	}
	// Add round to Existing Dispute
	if err := k.Keeper.AddDisputeRound(ctx, sender, dispute, *msg); err != nil {
		return nil, err
	}
	// event for new dispute round
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent("dispute_round_added",
			sdk.NewAttribute("dispute_id", fmt.Sprintf("%d", dispute.DisputeId)),
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("dispute_category", msg.DisputeCategory.String()),
			sdk.NewAttribute("fee", msg.Fee.String()),
			sdk.NewAttribute("report", msg.Report.String()),
		),
	})
	return &types.MsgProposeDisputeResponse{}, nil
}
