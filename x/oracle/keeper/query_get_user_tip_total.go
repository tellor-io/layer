package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	rk "github.com/tellor-io/layer/x/registry/keeper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetUserTipTotal(goCtx context.Context, req *types.QueryGetUserTipTotalRequest) (*types.QueryGetUserTipTotalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	tipper := sdk.MustAccAddressFromBech32(req.Tipper)

	var totalTips types.UserTipTotal
	store := k.TipStore(ctx)
	if req.QueryData == "" {
		totalTips = k.GetUserTips(ctx, store, tipper)
		return &types.QueryGetUserTipTotalResponse{TotalTips: &totalTips}, nil
	}
	if rk.Has0xPrefix(req.QueryData) {
		req.QueryData = req.QueryData[2:]
	}
	totalTips = k.GetUserQueryTips(ctx, store, tipper.String(), req.QueryData)

	return &types.QueryGetUserTipTotalResponse{TotalTips: &totalTips}, nil
}
