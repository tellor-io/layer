package keeper

import (
	"context"
	"fmt"

	"github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GenerateQuerydata(ctx context.Context, req *types.QueryGenerateQuerydataRequest) (*types.QueryGenerateQuerydataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// fetch data spec from store
	exists, err := k.HasSpec(ctx, req.QueryType)
	if !exists {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("can't generate query data for query type which doesn't exist in store: %v", err))
	}
	dataSpec, err := k.GetSpec(ctx, req.QueryType)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to get data spec: %v", err))
	}

	querydata, err := dataSpec.EncodeData(req.QueryType, req.Parameters)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to encode query data: %v", err))
	}

	return &types.QueryGenerateQuerydataResponse{QueryData: querydata}, nil
}
