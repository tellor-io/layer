package keeper

import (
	"context"
	"time"

	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetDataBefore(ctx context.Context, req *types.QueryGetDataBeforeRequest) (*types.QueryGetDataBeforeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	aggregate, timestamp, err := k.oracleKeeper.GetAggregateBefore(ctx, req.QueryId, time.Unix(req.Timestamp, 0))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get aggregate before")
	}
	if aggregate == nil {
		return nil, status.Error(codes.NotFound, "aggregate before not found")
	}
	timeUnix := timestamp.Unix()

	return &types.QueryGetDataBeforeResponse{
		Aggregate: aggregate,
		Timestamp: uint64(timeUnix),
	}, nil
}
