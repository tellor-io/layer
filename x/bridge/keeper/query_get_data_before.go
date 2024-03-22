package keeper

import (
	"context"
	"encoding/hex"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetDataBefore(goCtx context.Context, req *types.QueryGetDataBeforeRequest) (*types.QueryGetDataBeforeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	queryIdBytes, err := hex.DecodeString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query id")
	}
	aggregate, timestamp, err := k.oracleKeeper.GetAggregateBefore(ctx, queryIdBytes, time.Unix(req.Timestamp, 0))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get aggregate before")
	}
	if aggregate == nil {
		return nil, status.Error(codes.NotFound, "aggregate before not found")
	}
	timeUnix := timestamp.Unix()

	return &types.QueryGetDataBeforeResponse{
		Aggregate: aggregate,
		// Aggregate: &bridgeAggregate,
		Timestamp: uint64(timeUnix),
	}, nil
}
