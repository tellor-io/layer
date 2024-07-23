package keeper

import (
	"context"
	"time"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Querier) GetDataBefore(ctx context.Context, req *types.QueryGetDataBeforeRequest) (*types.QueryGetDataBeforeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	qIdBz, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid queryId")
	}

	aggregate, timestamp, err := k.keeper.GetAggregateBefore(ctx, qIdBz, time.UnixMilli(req.Timestamp))
	if err != nil {
		return nil, err
	}

	timeUnix := timestamp.UnixMilli()

	return &types.QueryGetDataBeforeResponse{
		Aggregate: aggregate,
		Timestamp: uint64(timeUnix),
	}, nil
}
