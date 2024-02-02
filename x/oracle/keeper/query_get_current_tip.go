package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetCurrentTip(goCtx context.Context, req *types.QueryGetCurrentTipRequest) (*types.QueryGetCurrentTipResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := k.TipStore(ctx)

	req.QueryData = regtypes.Remove0xPrefix(req.QueryData)
	tips, _ := k.GetQueryTips(ctx, store, req.QueryData)

	return &types.QueryGetCurrentTipResponse{Tips: &tips}, nil
}
