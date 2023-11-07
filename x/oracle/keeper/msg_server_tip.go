package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/tellor-io/layer/x/oracle/types"
	rk "github.com/tellor-io/layer/x/registry/keeper"
)

func (k msgServer) Tip(goCtx context.Context, msg *types.MsgTip) (*types.MsgTipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Amount.Denom != sdk.DefaultBondDenom {
		return nil, sdkerrors.ErrInvalidRequest
	}
	if rk.Has0xPrefix(msg.QueryData) {
		msg.QueryData = msg.QueryData[2:]
	}
	tipper := sdk.MustAccAddressFromBech32(msg.Tipper)

	if err := k.Keeper.transfer(ctx, tipper, msg.Amount); err != nil {
		return nil, err
	}
	k.Keeper.SetTip(ctx, tipper, msg.QueryData, msg.Amount)

	return &types.MsgTipResponse{}, nil
}
