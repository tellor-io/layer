package keeper

import (
	"context"
	"encoding/hex"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) RequestAttestations(ctx context.Context, msg *types.MsgRequestAttestations) (*types.MsgRequestAttestationsResponse, error) {
	k.Keeper.Logger(sdk.UnwrapSDKContext(ctx)).Info("@RequestAttestations", "queryId", msg.QueryId, "timestamp", msg.Timestamp)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	queryId, err := hex.DecodeString(msg.QueryId)
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to decode query id", "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	timestampInt, err := strconv.ParseUint(msg.Timestamp, 10, 64)
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to parse timestamp", "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	timestamp := time.Unix(int64(timestampInt), 0)
	err = k.Keeper.CreateSnapshot(sdkCtx, queryId, timestamp)
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to create snapshot", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.MsgRequestAttestationsResponse{}, nil
}
