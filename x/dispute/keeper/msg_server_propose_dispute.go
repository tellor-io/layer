package keeper

import (
	"context"
	"errors"

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

// want to catch bad median values / add evidence that this guy is bad
// add evidence function
// if already open dispute of that trype on the same reporter, just adde wvidence
// loop through evidence reports, can flag additional reports made by the same reporter
// if signifigane player is emdian, submits bad value, can flag all of his reports without putting up all of tyhe capital

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
