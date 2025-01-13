package keeper

import (
	"context"
	"errors"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
	// return an error if the proposer attempts to create a dispute on themselves while paying from their bond
	if msg.PayFromBond && sender.String() == msg.Creator {
		return nil, types.ErrSelfDisputeFromBond.Wrapf("proposer cannot pay from their bond when creating a dispute on themselves")
	}
	dispute, err := k.GetDisputeByReporter(ctx, *msg.Report, msg.DisputeCategory)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// event gets emitted in SetNewDispute
			if err := k.Keeper.SetNewDispute(ctx, sender, *msg); err != nil {
				return nil, err
			}
			return &types.MsgProposeDisputeResponse{}, nil
		}
		return nil, err
	}
	// Add round to Existing Dispute - emits event
	if err := k.Keeper.AddDisputeRound(ctx, sender, dispute, *msg); err != nil {
		return nil, err
	}
	return &types.MsgProposeDisputeResponse{}, nil
}
