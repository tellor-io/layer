package keeper

import (
	"context"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetAggregatedReport(ctx context.Context, req *types.QueryGetCurrentAggregatedReportRequest) (*types.QueryGetAggregatedReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryID, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, err
	}

	mostRecent, err := q.keeper.GetCurrentValueForQueryId(ctx, queryID)
	if err != nil {
		return nil, err
	}
	if mostRecent == nil {
		return nil, types.ErrNoAvailableReports.Wrapf("no reports available for query id %s", req.QueryId)
	}
	return &types.QueryGetAggregatedReportResponse{Report: mostRecent}, nil
}
