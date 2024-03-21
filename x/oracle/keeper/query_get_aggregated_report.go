package keeper

import (
	"context"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetAggregatedReport(goCtx context.Context, req *types.QueryGetCurrentAggregatedReportRequest) (*types.QueryGetAggregatedReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryId, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		panic(err)
	}

	mostRecent, err := k.GetCurrentValueForQueryId(goCtx, queryId)
	if err != nil {
		return nil, err
	}
	if mostRecent == nil {
		return nil, types.ErrNoAvailableReports.Wrapf("no reports available for query id %s", req.QueryId)
	}
	return &types.QueryGetAggregatedReportResponse{Report: mostRecent}, nil
}
