package keeper

import (
	"context"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) RequestAttestations(ctx context.Context, msg *types.MsgRequestAttestations) (*types.MsgRequestAttestationsResponse, error) {
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
	timestamp := time.UnixMilli(int64(timestampInt))
	err = k.Keeper.CreateSnapshot(sdkCtx, queryId, timestamp)
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to create snapshot", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"attestations_request",
			sdk.NewAttribute("query_id", msg.QueryId),
			sdk.NewAttribute("timestamp", msg.Timestamp),
		),
	})
	return &types.MsgRequestAttestationsResponse{}, nil
}
