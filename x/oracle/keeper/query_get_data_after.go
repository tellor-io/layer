package keeper

import (
	"context"
	"time"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Querier) GetDataAfter(ctx context.Context, req *types.QueryGetDataAfterRequest) (*types.QueryGetDataAfterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	qIdBz, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid queryId")
	}

	aggregate, timestamp, err := k.keeper.GetAggregateAfter(ctx, qIdBz, time.UnixMilli(int64(req.Timestamp)))
	if err != nil {
		return nil, err
	}

	timeUnix := timestamp.UnixMilli()

	return &types.QueryGetDataAfterResponse{
		Aggregate: aggregate,
		Timestamp: uint64(timeUnix),
	}, nil
}
