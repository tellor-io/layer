package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) DecodeValue(goCtx context.Context, req *types.QueryDecodeValueRequest) (*types.QueryDecodeValueResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	// get spec from store
	spec, err := k.GetSpec(ctx, req.QueryType)
	if err != nil {
		return nil, status.Error(codes.NotFound, "data spec not found")
	}

	value, err := spec.DecodeValue(req.Value)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to decode value")
	}

	return &types.QueryDecodeValueResponse{DecodedValue: value}, nil
}
