package keeper

import (
	"context"
	"time"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetDataBefore(ctx context.Context, req *types.QueryGetDataBeforeRequest) (*types.QueryGetDataBeforeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	qIdBz, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid queryId")
	}

	aggregate, timestamp, err := q.k.oracleKeeper.GetAggregateBefore(ctx, qIdBz, time.Unix(req.Timestamp, 0))
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
