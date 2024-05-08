package keeper

import (
	"context"
	"encoding/hex"

	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetValidatorCheckpoint(ctx context.Context, req *types.QueryGetValidatorCheckpointRequest) (*types.QueryGetValidatorCheckpointResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	checkpoint, err := q.k.GetValidatorCheckpointFromStorage(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get validator checkpoint")
	}

	checkpointHexString := hex.EncodeToString(checkpoint.Checkpoint)

	return &types.QueryGetValidatorCheckpointResponse{ValidatorCheckpoint: checkpointHexString}, nil
}
