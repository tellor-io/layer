package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k msgServer) ProposeDispute(goCtx context.Context, msg *types.MsgProposeDispute) (*types.MsgProposeDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute := k.GetDisputeByReporter(ctx, *msg.Report, msg.DisputeCategory)

	if dispute == nil {
		// Set New Dispute
		if err := k.Keeper.SetNewDispute(ctx, *msg); err != nil {
			return nil, err
		}
	} else {
		// Add round to Existing Dispute
		if err := k.Keeper.AddDisputeRound(ctx, *dispute); err != nil {
			return nil, err
		}
	}
	return &types.MsgProposeDisputeResponse{}, nil
}
