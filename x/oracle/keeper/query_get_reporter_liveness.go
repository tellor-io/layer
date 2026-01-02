package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetReporterLiveness returns the liveness of a reporter.
// This includes:
// - Number of reports submitted and total aggregates (for percent liveness)
// - Last report timestamp
func (q Querier) GetReporterLiveness(ctx context.Context, req *types.QueryGetReporterLivenessRequest) (*types.QueryGetReporterLivenessResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporterAddr, err := sdk.AccAddressFromBech32(req.Reporter)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid reporter address")
	}

	reporterBytes := reporterAddr.Bytes()

	// Get percent liveness (reports/aggregates)
	reported, total, percent, err := q.keeper.GetReporterPercentLiveness(ctx, reporterBytes)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Get last report time
	lastReportTime, err := q.keeper.GetReporterLastReportTime(ctx, reporterBytes)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetReporterLivenessResponse{
		ReporterReports: reported,
		TotalAggregates: total,
		PercentLiveness: percent,
		LastReportTime:  lastReportTime,
	}, nil
}
