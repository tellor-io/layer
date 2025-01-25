package keeper

import (
	"context"
	"errors"
	"strings"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) ProposeDispute(goCtx context.Context, msg *types.MsgProposeDispute) (*types.MsgProposeDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	sender, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	err = validateProposeDispute(msg)
	if err != nil {
		return nil, err
	}

	if msg.Fee.Amount.LT(layer.OnePercent) {
		return nil, types.ErrMinimumTRBrequired.Wrapf("fee %s doesn't meet minimum fee required", msg.Fee.Amount)
	}
	// return an error if the proposer attempts to create a dispute on themselves while paying from their bond
	if msg.PayFromBond && strings.EqualFold(msg.Creator, msg.Report.Reporter) {
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

func validateProposeDispute(msg *types.MsgProposeDispute) error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	// ensure that the fee matches the layer.BondDenom and the amount is a positive number
	if msg.Fee.Denom != layer.BondDenom || msg.Fee.Amount.IsZero() || msg.Fee.Amount.IsNegative() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "invalid fee amount (%s)", msg.Fee.Amount.String())
	}
	if msg.Report == nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "report should not be nil")
	}
	if msg.DisputeCategory != types.Warning && msg.DisputeCategory != types.Minor && msg.DisputeCategory != types.Major {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "dispute category should be either Warning, Minor, or Major")
	}
	return nil
}
