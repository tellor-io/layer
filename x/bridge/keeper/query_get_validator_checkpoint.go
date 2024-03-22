package keeper

import (
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetValidatorCheckpoint(goCtx context.Context, req *types.QueryGetValidatorCheckpointRequest) (*types.QueryGetValidatorCheckpointResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	checkpoint, err := k.GetValidatorCheckpointFromStorage(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get validator checkpoint")
	}

	checkpointHexString := hex.EncodeToString(checkpoint.Checkpoint)

	return &types.QueryGetValidatorCheckpointResponse{ValidatorCheckpoint: checkpointHexString}, nil
}
