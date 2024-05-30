package keeper

import (
	"context"
	"time"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetDataBefore(goCtx context.Context, req *types.QueryGetDataBeforeRequest) (*types.QueryGetAggregatedReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	qId, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, err
	}

	t := time.Unix(req.Timestamp, 0)
	report, err := q.keeper.GetDataBefore(goCtx, qId, t)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetAggregatedReportResponse{Report: report}, nil
}
