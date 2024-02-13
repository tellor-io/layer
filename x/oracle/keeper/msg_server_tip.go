package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/tellor-io/layer/x/oracle/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"
)

func (k msgServer) Tip(goCtx context.Context, msg *types.MsgTip) (*types.MsgTipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Amount.Denom != types.DefaultBondDenom || msg.Amount.Amount.IsZero() || msg.Amount.Amount.IsNegative() {
		return nil, sdkerrors.ErrInvalidRequest
	}
	msg.QueryData = regtypes.Remove0xPrefix(msg.QueryData)
	tipper := sdk.MustAccAddressFromBech32(msg.Tipper)

	tip, err := k.Keeper.transfer(ctx, tipper, msg.Amount)
	if err != nil {
		return nil, err
	}
	k.Keeper.SetTip(ctx, tipper, msg.QueryData, tip)

	return &types.MsgTipResponse{}, nil
}
