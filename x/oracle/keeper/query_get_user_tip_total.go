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
	store := k.TipStore(ctx)
	if rk.Has0xPrefix(req.QueryData) {
		req.QueryData = req.QueryData[2:]
	}
	tips := k.GetUserTips(ctx, store, req.Tipper, req.QueryData)

	return &types.QueryGetUserTipTotalResponse{Tips: &tips}, nil
}
