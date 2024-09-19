package keeper

import (
	"context"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

func (k Querier) GetReportsByAggregate(ctx context.Context, req *types.QueryGetReportsByAggregateRequest) (*types.QueryGetReportsByAggregateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	queryId, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, err
	}
	agg, err := k.keeper.Aggregates.Get(ctx, collections.Join(queryId, int64(req.Timestamp)))
	if err != nil {
		return nil, err
	}

	metaId := agg.MetaId
	reporters := agg.Reporters

	microreports := make([]types.MicroReport, 0)
	pageRes := &query.PageResponse{
		NextKey: nil,
		Total:   uint64(len(reporters)),
	}

	for _, reporter := range reporters {
		reporterAddr := sdk.MustAccAddressFromBech32(reporter.Reporter)
		key := collections.Join3(queryId, reporterAddr.Bytes(), metaId)
		rep, err := k.keeper.Reports.Get(ctx, key)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		microreports = append(microreports, rep)

		if uint64(len(microreports)) >= req.Pagination.Limit {
			break
		}
	}

	return &types.QueryGetReportsByAggregateResponse{MicroReports: microreports, Pagination: pageRes}, nil
}
