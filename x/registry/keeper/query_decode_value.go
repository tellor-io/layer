package keeper

import (
	"context"
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	decodedValue, err := types.DecodeValue(req.Value, spec.ValueType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to decode value")
	}
	// convert interface to json
	decodedValueJSON, err := json.Marshal(decodedValue)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal decoded value")
	}
	return &types.QueryDecodeValueResponse{DecodedValue: string(decodedValueJSON)}, nil
}
