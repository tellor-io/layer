package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetValidatorTimestampByIndex(ctx context.Context, req *types.QueryGetValidatorTimestampByIndexRequest) (*types.QueryGetValidatorTimestampByIndexResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	validatorTimestamp, err := q.k.ValidatorCheckpointIdxMap.Get(ctx, uint64(req.Index))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get validator timestamp by index")
	}

	return &types.QueryGetValidatorTimestampByIndexResponse{
		Timestamp: validatorTimestamp.Timestamp,
	}, nil
}
