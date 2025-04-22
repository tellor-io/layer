package keeper

import (
	"context"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"

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
	agg, err := k.keeper.Aggregates.Get(ctx, collections.Join(queryId, req.Timestamp))
	if err != nil {
		return nil, err
	}

	metaId := agg.MetaId

	microreports := make([]types.MicroReportStrings, 0)
	pageRes := &query.PageResponse{
		NextKey: nil,
		Total:   uint64(0),
	}

	iter, err := k.keeper.Reports.Indexes.IdQueryId.MatchExact(ctx, collections.Join(metaId, queryId))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key, err := iter.PrimaryKey()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		rep, err := k.keeper.Reports.Get(ctx, key)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		microReport := types.MicroReportStrings{
			Reporter:        rep.Reporter,
			Power:           rep.Power,
			QueryType:       rep.QueryType,
			QueryId:         req.QueryId,
			AggregateMethod: rep.AggregateMethod,
			Value:           rep.Value,
			Timestamp:       uint64(rep.Timestamp.UnixMilli()),
			Cyclelist:       rep.Cyclelist,
			BlockNumber:     rep.BlockNumber,
			MetaId:          rep.MetaId,
		}
		microreports = append(microreports, microReport)

		if uint64(len(microreports)) >= req.Pagination.Limit {
			break
		}
	}
	pageRes.Total = uint64(len(microreports))

	return &types.QueryGetReportsByAggregateResponse{MicroReports: microreports, Pagination: pageRes}, nil
}
