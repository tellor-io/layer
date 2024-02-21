package keeper

import (
	"context"
	"fmt"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetAggregatedReport(goCtx context.Context, req *types.QueryGetCurrentAggregatedReportRequest) (*types.QueryGetAggregatedReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryId, err := utils.QueryIDFromString(req.QueryId)
	if err != nil {
		panic(err)
	}

	mostRecent := k.GetCurrentValueForQueryId(goCtx, queryId)
	if mostRecent == nil {
		return nil, fmt.Errorf("no available timestamps")
	}
	return &types.QueryGetAggregatedReportResponse{Report: mostRecent}, nil
}
