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

// ReporterRange implements Ranger[collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]] for efficient reporter queries
type ReporterRange struct {
	reporterAddr []byte
	order        collections.Order
	startKey     *collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]
}

// NewReporterRange creates a new ReporterRange for the given reporter address
func NewReporterRange(reporterAddr []byte) *ReporterRange {
	return &ReporterRange{
		reporterAddr: reporterAddr,
		order:        collections.OrderAscending,
	}
}

// Descending sets the range to iterate in descending order
func (r *ReporterRange) Descending() *ReporterRange {
	r.order = collections.OrderDescending
	return r
}

// StartInclusive sets the starting key for pagination
func (r *ReporterRange) StartInclusive(key collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]) *ReporterRange {
	r.startKey = &key
	return r
}

// RangeValues implements the Ranger interface
func (r *ReporterRange) RangeValues() (start, end *collections.RangeKey[collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]], order collections.Order, err error) {
	// The Reporter index key is: reporter_address + encoded_metaId
	// We want to create a range that covers all entries for this reporter

	// Create start bound: reporter_address + 0 (minimum metaId)
	startBuffer := make([]byte, 8)
	_, err = collections.Uint64Key.Encode(startBuffer, 0)
	if err != nil {
		return nil, nil, 0, err
	}
	startIndexKey := append(r.reporterAddr, startBuffer...)

	// Create end bound: reporter_address + maxUint64 (maximum metaId)
	endBuffer := make([]byte, 8)
	_, err = collections.Uint64Key.Encode(endBuffer, ^uint64(0))
	if err != nil {
		return nil, nil, 0, err
	}
	endIndexKey := append(r.reporterAddr, endBuffer...)

	// For the Pair type, we need to construct pairs with the index key and a dummy primary key
	// The actual primary key will be determined by the iteration
	dummyPrimaryKey := collections.Join3([]byte{}, []byte{}, uint64(0))

	var startPair collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]
	var endPair collections.Pair[[]byte, collections.Triple[[]byte, []byte, uint64]]

	if r.startKey != nil {
		// Use the provided start key for pagination
		startPair = *r.startKey
	} else {
		// Start from the beginning of this reporter's data
		startPair = collections.Join(startIndexKey, dummyPrimaryKey)
	}

	endPair = collections.Join(endIndexKey, dummyPrimaryKey)

	start = collections.RangeKeyExact(startPair)
	end = collections.RangeKeyNext(endPair) // Next to make it inclusive

	return start, end, r.order, nil
}

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

// GetReportsbyReporter uses a custom range to efficiently query reports by reporter
func (k Querier) GetReportsbyReporter(ctx context.Context, req *types.QueryGetReportsbyReporterRequest) (*types.QueryMicroReportsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporter, err := sdk.AccAddressFromBech32(req.Reporter)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid reporter address")
	}

	defaultLimit := 10
	limit := uint64(defaultLimit)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}

	// Create custom range for this reporter
	reporterRange := NewReporterRange(reporter.Bytes())

	// Apply ordering
	if req.Pagination != nil && req.Pagination.Reverse {
		reporterRange = reporterRange.Descending()
	}

	// Handle pagination start key
	if req.Pagination != nil && len(req.Pagination.Key) > 0 {
		tripleKeyCodec := collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.Uint64Key)
		_, startKey, err := tripleKeyCodec.Decode(req.Pagination.Key)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid pagination key")
		}
		// Convert the primary key to an index pair for range starting
		// The index key is constructed from reporter address + metaId
		size := collections.Uint64Key.Size(startKey.K3())
		buffer := make([]byte, size)
		_, err = collections.Uint64Key.Encode(buffer, startKey.K3())
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to encode metaId")
		}
		indexKey := append(startKey.K2(), buffer...)
		startPair := collections.Join(indexKey, startKey)
		reporterRange = reporterRange.StartInclusive(startPair)
	}

	// Use the custom range with the Reporter index
	iter, err := k.keeper.Reports.Indexes.Reporter.Iterate(ctx, reporterRange)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer iter.Close()

	reports := make([]types.MicroReportStrings, 0)
	pageRes := &query.PageResponse{
		NextKey: nil,
		Total:   uint64(0),
	}

	for ; iter.Valid(); iter.Next() {
		pk, err := iter.PrimaryKey()
		if err != nil {
			return nil, err
		}

		// Check if limit is reached
		if uint64(len(reports)) >= limit {
			tripleKeyCodec := collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.Uint64Key)
			buffer := make([]byte, tripleKeyCodec.Size(pk))
			_, err = tripleKeyCodec.Encode(buffer, pk)
			if err != nil {
				return nil, status.Error(codes.Internal, "failed to encode pagination key")
			}
			pageRes.NextKey = buffer
			break
		}

		report, err := k.keeper.Reports.Get(ctx, pk)
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
