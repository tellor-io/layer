package keeper

import (
	"context"
	"time"

	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetDataBefore(goCtx context.Context, req *types.QueryGetDataBeforeRequest) (*types.QueryGetAggregatedReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	t := time.Unix(req.Timestamp, 0)
	report, err := k.getDataBefore(goCtx, req.QueryId, t)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetAggregatedReportResponse{Report: report}, nil
}
