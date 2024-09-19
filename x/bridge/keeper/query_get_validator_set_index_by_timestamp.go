package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetValidatorSetIndexByTimestamp(ctx context.Context, req *types.QueryGetValidatorSetIndexByTimestampRequest) (*types.QueryGetValidatorSetIndexByTimestampResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	index, err := q.k.GetValidatorSetIndexByTimestamp(ctx, req.Timestamp)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get validator checkpoint")
	}

	return &types.QueryGetValidatorSetIndexByTimestampResponse{Index: index}, nil
}
