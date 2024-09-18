package keeper

import (
	"context"
	"encoding/hex"

	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetValidatorCheckpointParams(ctx context.Context, req *types.QueryGetValidatorCheckpointParamsRequest) (*types.QueryGetValidatorCheckpointParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	checkpointParams, err := q.k.ValidatorCheckpointParamsMap.Get(ctx, req.Timestamp)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get validator checkpoint params")
	}

	checkpointHexString := hex.EncodeToString(checkpointParams.Checkpoint)
	valsetHashString := hex.EncodeToString(checkpointParams.ValsetHash)

	return &types.QueryGetValidatorCheckpointParamsResponse{
		Checkpoint:     checkpointHexString,
		ValsetHash:     valsetHashString,
		Timestamp:      checkpointParams.Timestamp,
		PowerThreshold: checkpointParams.PowerThreshold,
	}, nil
}
