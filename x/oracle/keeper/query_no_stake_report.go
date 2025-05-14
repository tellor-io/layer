package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetReportersNoStakeReports(ctx context.Context, req *types.QueryGetReportersNoStakeReportsRequest) (*types.QueryGetReportersNoStakeReportsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reports, err := q.keeper.GetNoStakeReportsByReporter(ctx, req.Reporter)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetReportersNoStakeReportsResponse{NoStakeReports: reports}, nil
}
