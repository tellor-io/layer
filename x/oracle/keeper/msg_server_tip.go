package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k msgServer) Tip(goCtx context.Context, msg *types.MsgTip) (*types.MsgTipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Amount.Denom != layer.BondDenom || msg.Amount.Amount.IsZero() || msg.Amount.Amount.IsNegative() {
		return nil, sdkerrors.ErrInvalidRequest
	}
	tipper := sdk.MustAccAddressFromBech32(msg.Tipper)

	tip, err := k.Keeper.transfer(ctx, tipper, msg.Amount)
	if err != nil {
		return nil, err
	}

	queryId, err := utils.QueryIDFromDataString(msg.QueryData)
	if err != nil {
		return nil, err
	}

	err = k.Keeper.AddtoTotalTips(ctx, tip.Amount)
	if err != nil {
		return nil, err
	}

	err = k.Keeper.SetQueryTip(ctx, queryId, tip.Amount)
	if err != nil {
		return nil, err
	}
	prevTip, err := k.Keeper.Tips.Get(ctx, collections.Join(queryId, tipper.Bytes()))
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, fmt.Errorf("failed to get previous tip: %w", err)
	}

	if !prevTip.IsNil() {
		tip = tip.AddAmount(prevTip)
	}
	err = k.Keeper.Tips.Set(ctx, collections.Join(queryId, tipper.Bytes()), tip.Amount)
	if err != nil {
		return nil, err
	}

	return &types.MsgTipResponse{}, nil
}
