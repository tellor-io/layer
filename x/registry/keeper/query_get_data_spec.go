package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Querier) GetDataSpec(goCtx context.Context, req *types.QueryGetDataSpecRequest) (*types.QueryGetDataSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if req.QueryType == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid request; query type cannot be empty")
	}

	exists, err := k.Keeper.HasSpec(ctx, req.QueryType)
	if !exists {
		return nil, status.Error(codes.NotFound, "data spec not registered")
	}
	dataSpec, err := k.Keeper.GetSpec(ctx, req.QueryType)
	if err != nil {
		return nil, err
	}
	return &types.QueryGetDataSpecResponse{Spec: &dataSpec}, nil
}
