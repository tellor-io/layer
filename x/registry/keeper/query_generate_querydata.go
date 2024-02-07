package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GenerateQuerydata(goCtx context.Context, req *types.QueryGenerateQuerydataRequest) (*types.QueryGenerateQuerydataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	// fetch data spec from store
	exists, err := k.HasSpec(ctx, req.Querytype)
	if !exists {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("can't generate query data for query type which doesn't exist in store: %v", err))
	}
	dataSpec, err := k.GetSpec(ctx, req.Querytype)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to get data spec: %v", err))
	}

	querydata, err := dataSpec.EncodeData(req.Querytype, req.Parameters)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to encode query data: %v", err))
	}

	return &types.QueryGenerateQuerydataResponse{Querydata: querydata}, nil
}
