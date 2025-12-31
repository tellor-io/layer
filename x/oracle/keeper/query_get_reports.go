package keeper

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"math"

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

// use GetReportsByReporterQid for ordered results
func (k Querier) GetReportsbyQid(ctx context.Context, req *types.QueryGetReportsbyQidRequest) (*types.QueryMicroReportsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	qId, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, err
	}

	if req.Pagination == nil {
		req.Pagination = &query.PageRequest{}
	}

	defaultLimit := uint64(10)
	if req.Pagination.Limit == 0 {
		req.Pagination.Limit = defaultLimit
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

	if req.Pagination == nil {
		req.Pagination = &query.PageRequest{}
	}

	defaultLimit := uint64(10)
	if req.Pagination.Limit == 0 {
		req.Pagination.Limit = defaultLimit
	}

	reporter := sdk.MustAccAddressFromBech32(req.Reporter)

	// TODO: add max limit to prevent abuse
	pageRes := &query.PageResponse{
		NextKey: nil,
		Total:   uint64(0),
	}

	startKey, endKey, err := k.keeper.GetStartEndKey(ctx, reporter.Bytes(), req.Pagination.Key, req.Pagination.Reverse)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get start and end keys: "+err.Error())
	}
	rng := types.NewPrefixInBetween[[]byte, collections.Triple[[]byte, []byte, uint64]](startKey, endKey)
	if req.Pagination.Reverse {
		rng.Descending()
	}

	iter, err := k.keeper.Reports.Indexes.Reporter.Iterate(ctx, rng)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer iter.Close()
	if req.Pagination.Offset > 0 {
		if !advanceIter(iter, req.Pagination.Offset) {
			return nil, status.Error(codes.InvalidArgument, "invalid pagination offset")
		}
	}
	reports := make([]types.MicroReportStrings, 0)
	counter := uint64(0)
	for ; iter.Valid() && counter < req.Pagination.Limit; iter.Next() {
		fullKey, err := iter.FullKey()
		if err != nil {
			return nil, err
		}

		report, err := k.keeper.Reports.Get(ctx, fullKey.K2())
		if err != nil {
			return nil, err
		}
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
		counter++

	}
	if iter.Valid() {
		fullKeys, err := iter.FullKey()
		if err != nil {
			return nil, err
		}
		pageRes.NextKey = fullKeys.K1()
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

	if req.Pagination == nil {
		req.Pagination = &query.PageRequest{}
	}

	defaultLimit := uint64(10)
	if req.Pagination.Limit == 0 {
		req.Pagination.Limit = defaultLimit
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

func (k Keeper) GetStartEndKey(ctx context.Context, reporter, nextKey []byte, reverse bool) (startKey, endKey []byte, err error) {
	uint64BytesMin := make([]byte, 8)
	binary.BigEndian.PutUint64(uint64BytesMin, uint64(0))
	uint64BytesMax := make([]byte, 8)
	binary.BigEndian.PutUint64(uint64BytesMax, math.MaxUint64)

	// First page: nextKey is nil, so return full range for this reporter
	if nextKey == nil {
		// Create startKey: reporter prefix + 0 (lowest possible value)
		startKey = append(reporter, uint64BytesMin...)
		// Create endKey: reporter prefix + MaxUint64 (highest possible value)
		endKey = append(reporter, uint64BytesMax...)

		return startKey, endKey, nil
	} else {
		// Subsequent pages: nextKey provided from previous page
		if reverse {
			// Descending: nextKey becomes END boundary
			// Start from beginning of reporter range, end at nextKey
			startKey = append(reporter, uint64BytesMin...)
			endKey = nextKey // nextKey becomes the upper bound
		} else {
			// Ascending: nextKey becomes START boundary
			// Start from nextKey, go to end of reporter range
			startKey = nextKey // nextKey becomes the lower bound
			endKey = append(reporter, uint64BytesMax...)
		}
		return startKey, endKey, nil
	}
}

func advanceIter[I interface {
	Next()
	Valid() bool
}](iter I, offset uint64,
) bool {
	for i := uint64(0); i < offset; i++ {
		if !iter.Valid() {
			return false
		}
		iter.Next()
	}
	return true
}
