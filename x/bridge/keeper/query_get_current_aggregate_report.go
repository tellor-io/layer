package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetCurrentAggregateReport(ctx context.Context, req *types.QueryGetCurrentAggregateReportRequest) (*types.QueryGetCurrentAggregateReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	aggregate, timestamp := k.oracleKeeper.GetCurrentAggregateReport(ctx, req.QueryId)
	if aggregate == nil {
		return nil, status.Error(codes.NotFound, "aggregate not found")
	}
	timeUnix := timestamp.Unix()

	// convert oracletypes.Reporters to bridgetypes.Reporters
	convertedReporters := make([]*types.AggregateReporter, len(aggregate.Reporters))
	for i, reporter := range aggregate.Reporters {
		convertedReporters[i] = &types.AggregateReporter{
			Reporter: reporter.Reporter,
			Power:    reporter.Power,
		}
	}

	// convert oracletypes.Aggregate to bridgetypes.Aggregate
	bridgeAggregate := types.Aggregate{
		QueryId:              req.QueryId,
		AggregateValue:       aggregate.AggregateValue,
		AggregateReporter:    aggregate.AggregateReporter,
		ReporterPower:        aggregate.ReporterPower,
		StandardDeviation:    aggregate.StandardDeviation,
		Reporters:            convertedReporters,
		Flagged:              aggregate.Flagged,
		Nonce:                int64(aggregate.Nonce),
		AggregateReportIndex: aggregate.AggregateReportIndex,
		Height:               aggregate.Height,
	}

	return &types.QueryGetCurrentAggregateReportResponse{
		Aggregate: &bridgeAggregate,
		Timestamp: uint64(timeUnix),
	}, nil
}
