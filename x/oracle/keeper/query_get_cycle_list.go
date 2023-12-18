package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetCycleList(goCtx context.Context, req *types.QueryGetCycleListRequest) (*types.QueryGetCycleListResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	cycleList := k.GetCyclList(ctx)

	return &types.QueryGetCycleListResponse{CycleList: &cycleList}, nil
}
