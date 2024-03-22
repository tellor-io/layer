package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetValidatorTimestampByIndex(goCtx context.Context, req *types.QueryGetValidatorTimestampByIndexRequest) (*types.QueryGetValidatorTimestampByIndexResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	validatorTimestamp, err := k.GetValidatorTimestampByIdxFromStorage(ctx, uint64(req.Index))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get validator timestamp by index")
	}

	return &types.QueryGetValidatorTimestampByIndexResponse{
		Timestamp: int64(validatorTimestamp.Timestamp),
	}, nil
}
