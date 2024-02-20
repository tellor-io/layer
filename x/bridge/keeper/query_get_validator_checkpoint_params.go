package keeper

import (
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetValidatorCheckpointParams(goCtx context.Context, req *types.QueryGetValidatorCheckpointParamsRequest) (*types.QueryGetValidatorCheckpointParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	checkpointParams, err := k.GetValidatorCheckpointParamsFromStorage(ctx, uint64(req.Timestamp))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get validator checkpoint")
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
