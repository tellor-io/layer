package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetCurrentValidatorSetTimestamp(ctx context.Context, req *types.QueryGetCurrentValidatorSetTimestampRequest) (*types.QueryGetCurrentValidatorSetTimestampResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	timestamp, err := q.k.GetCurrentValidatorSetTimestamp(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get validator checkpoint")
	}

	return &types.QueryGetCurrentValidatorSetTimestampResponse{Timestamp: timestamp}, nil
}
