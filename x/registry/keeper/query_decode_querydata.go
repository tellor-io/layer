package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Querier) DecodeQueryData(goCtx context.Context, req *types.QueryDecodeQueryDataRequest) (*types.QueryDecodeQueryDataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryType, fieldBytes, err := types.DecodeQueryType(req.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query data: %v", err))
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	// fetch data spec from store
	exists, err := k.Keeper.HasSpec(ctx, queryType)
	if !exists {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("can't decode query data for query type which doesn't exist in store: %v", queryType))
	}
	dataSpec, err := k.Keeper.GetSpec(ctx, queryType)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to get data spec: %v", err))
	}
	// convert field bytes to string
	fields, err := types.DecodeParamtypes(fieldBytes, dataSpec.AbiComponents)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query data fields: %v", err))
	}

	return &types.QueryDecodeQueryDataResponse{Spec: fmt.Sprintf("%s: %s", queryType, fields)}, nil
}
