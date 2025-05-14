package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetReportersNoStakeReports(ctx context.Context, req *types.QueryGetReportersNoStakeReportsRequest) (*types.QueryGetReportersNoStakeReportsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporter, err := sdk.AccAddressFromBech32(req.Reporter)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid reporter address")
	}
	reports, err := q.keeper.GetNoStakeReportsByReporter(ctx, reporter)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetReportersNoStakeReportsResponse{NoStakeReports: reports}, nil
}
