package keeper

import (
	"context"
	"encoding/hex"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

// WithCollectionPaginationTriplePrefix applies a prefix to a collection, whose key is a collection.Triple,
// being paginated that needs prefixing.
func WithCollectionPaginationTriplePrefix[K1, K2, K3 any](prefix K1) func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
	return func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
		prefix := collections.TriplePrefix[K1, K2, K3](prefix)
		o.Prefix = &prefix
	}
}

func WithCollectionPaginationTripleSuperPrefix[K1, K2, K3 any](prefix1 K1, prefix2 K2) func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
	return func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
		prefix := collections.TripleSuperPrefix[K1, K2, K3](prefix1, prefix2)
		o.Prefix = &prefix
	}
}

func (k Querier) GetReportsbyQid(ctx context.Context, req *types.QueryGetReportsbyQidRequest) (*types.QueryMicroReportsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	qId, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, err
	}
	microreports := make([]types.MicroReportStrings, 0)
	_, pageRes, err := query.CollectionPaginate(
		ctx, k.keeper.Reports, req.Pagination, func(_ collections.Triple[[]byte, []byte, uint64], rep types.MicroReport) (types.MicroReport, error) {
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
			return rep, nil
		}, WithCollectionPaginationTriplePrefix[[]byte, []byte, uint64](qId),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryMicroReportsResponse{MicroReports: microreports, Pagination: pageRes}, nil
}

func (k Querier) GetReportsbyReporter(ctx context.Context, req *types.QueryGetReportsbyReporterRequest) (*types.QueryMicroReportsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporter := sdk.MustAccAddressFromBech32(req.Reporter)

	// Retrieve the stored reports for the current block height.
	iter, err := k.keeper.Reports.Indexes.Reporter.MatchExact(ctx, reporter.Bytes())
	if err != nil {
		return nil, err
	}

	reports, err := indexes.CollectValues(ctx, k.keeper.Reports, iter)
	if err != nil {
		return nil, err
	}

	microreports := make([]types.MicroReportStrings, 0)
	for _, rep := range reports {
		microReport := types.MicroReportStrings{
			Reporter:        rep.Reporter,
			Power:           rep.Power,
			QueryType:       rep.QueryType,
			QueryId:         hex.EncodeToString(rep.QueryId),
			AggregateMethod: rep.AggregateMethod,
			Value:           rep.Value,
			Timestamp:       uint64(rep.Timestamp.UnixMilli()),
			Cyclelist:       rep.Cyclelist,
			BlockNumber:     rep.BlockNumber,
			MetaId:          rep.MetaId,
		}
		microreports = append(microreports, microReport)
	}

	return &types.QueryMicroReportsResponse{MicroReports: microreports}, nil
}

func (k Querier) GetReportsbyReporterQid(ctx context.Context, req *types.QueryGetReportsbyReporterQidRequest) (*types.QueryMicroReportsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporterAdd, err := sdk.AccAddressFromBech32(req.Reporter)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to decode reporter address")
	}

	qId, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, err
	}

	microreports := make([]types.MicroReportStrings, 0)
	_, pageRes, err := query.CollectionPaginate(
		ctx, k.keeper.Reports, req.Pagination, func(_ collections.Triple[[]byte, []byte, uint64], rep types.MicroReport) (types.MicroReport, error) {
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
			return rep, nil
		}, WithCollectionPaginationTripleSuperPrefix[[]byte, []byte, uint64](qId, reporterAdd.Bytes()),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryMicroReportsResponse{MicroReports: microreports, Pagination: pageRes}, nil
}
