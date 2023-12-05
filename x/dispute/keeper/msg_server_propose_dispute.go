package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k msgServer) ProposeDispute(goCtx context.Context, msg *types.MsgProposeDispute) (*types.MsgProposeDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Exponent for the denomination (e.g., 6 for 1 trb = 1e6 loya)
	oneTRB := math.NewInt(1_000_000)
	if msg.Fee.IsLT(sdk.NewCoin(sdk.DefaultBondDenom, oneTRB)) {
		return nil, types.ErrMinimumTRBrequired
	}
	if msg.Fee.Denom != sdk.DefaultBondDenom {
		return nil, types.ErrInvalidFeeDenom
	}
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
