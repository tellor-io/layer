package keeper

import (
	"context"
	"encoding/hex"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"

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

// gets the most recent n reports for a reporter
func (k Querier) GetReportsbyReporter(ctx context.Context, req *types.QueryGetReportsbyReporterRequest) (*types.QueryMicroReportsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporter := sdk.MustAccAddressFromBech32(req.Reporter)
	pageRes := &query.PageResponse{
		NextKey: nil,
		Total:   uint64(0),
	}

	// key is Bytes (reporter address) with bytes encoded max uint64 concatenated (reporterAddr...fff...)
	buffer := make([]byte, 8)
	_, err := collections.Uint64Key.Encode(buffer, ^uint64(0))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode start value")
	}

	// construct key: reporter_address + encoded_uint64
	key := append(reporter.Bytes(), buffer...)
	rng := collections.NewPrefixUntilPairRange[[]byte, collections.Triple[[]byte, []byte, uint64]](key)
	if req.Pagination != nil && req.Pagination.Reverse {
		rng.Descending()
	}
	tripleKeyCodec := collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.Uint64Key)
	if req.Pagination != nil && len(req.Pagination.Key) > 0 {
		_, startKey, err := tripleKeyCodec.Decode(req.Pagination.Key)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid pagination key")
		}
		rng.StartInclusive(startKey)
	}

	iter, err := k.keeper.Reports.Indexes.Reporter.Iterate(ctx, rng)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	reports := make([]types.MicroReportStrings, 0)
	for ; iter.Valid(); iter.Next() {
		pk, err := iter.PrimaryKey()
		if err != nil {
			return nil, err
		}

		report, err := k.keeper.Reports.Get(ctx, pk)
		if err != nil {
			return nil, err
		}
		if report.Reporter == reporter.String() {
			stringReport := types.MicroReportStrings{
				Reporter:        report.Reporter,
				Power:           report.Power,
				QueryType:       report.QueryType,
				QueryId:         hex.EncodeToString(report.QueryId),
				AggregateMethod: report.AggregateMethod,
				Value:           report.Value,
				Timestamp:       uint64(report.Timestamp.UnixMilli()),
				Cyclelist:       report.Cyclelist,
				BlockNumber:     report.BlockNumber,
				MetaId:          report.MetaId,
			}
			reports = append(reports, stringReport)
		}

		if req.Pagination != nil && uint64(len(reports)) >= req.Pagination.Limit {
			buffer := make([]byte, tripleKeyCodec.Size(pk))
			_, err = tripleKeyCodec.Encode(buffer, pk)
			if err != nil {
				return nil, status.Error(codes.Internal, "failed to encode pagination key")
			}
			pageRes.NextKey = buffer
			break
		}
	}
	pageRes.Total = uint64(len(reports))

	return &types.QueryMicroReportsResponse{MicroReports: reports, Pagination: pageRes}, nil
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
