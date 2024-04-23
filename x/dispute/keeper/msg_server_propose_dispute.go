package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k msgServer) ProposeDispute(goCtx context.Context, msg *types.MsgProposeDispute) (*types.MsgProposeDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Fee.Denom != layer.BondDenom {
		return nil, types.ErrInvalidFeeDenom.Wrapf("wrong fee denom: %s, expected: %s", msg.Fee.Denom, layer.BondDenom)
	}

	if msg.Fee.Amount.LT(layer.OnePercent) {
		return nil, types.ErrMinimumTRBrequired.Wrapf("fee %s doesn't meet minimum fee required", msg.Fee.Amount)
	}
	dispute, err := k.GetDisputeByReporter(ctx, *msg.Report, msg.DisputeCategory)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			if err := k.Keeper.SetNewDispute(ctx, *msg); err != nil {
				return nil, err
			}
			return &types.MsgProposeDisputeResponse{}, nil
		}
		return nil, err
	}
	// Add round to Existing Dispute
	if err := k.Keeper.AddDisputeRound(ctx, dispute, *msg); err != nil {
		return nil, err
	}
	return &types.MsgProposeDisputeResponse{}, nil
}
