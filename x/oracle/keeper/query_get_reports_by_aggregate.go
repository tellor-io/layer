package keeper

import (
	"bytes"
	"context"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"

	"github.com/cosmos/cosmos-sdk/types/query"
)

func (k Querier) GetReportsByAggregate(ctx context.Context, req *types.QueryGetReportsByAggregateRequest) (*types.QueryGetReportsByAggregateResponse, error) {
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

	// Handle offset
	if req.Pagination.Offset > 0 {
		if !advanceIter(iter, req.Pagination.Offset) {
			return nil, status.Error(codes.InvalidArgument, "invalid pagination offset")
		}
	}

	if req.Pagination.Key != nil {
		if !advanceIterToNextKey(&iter, req.Pagination.Key) {
			return nil, status.Error(codes.InvalidArgument, "invalid pagination key")
		}
	}

	counter := uint64(0)
	for ; iter.Valid() && counter < req.Pagination.Limit; iter.Next() {
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
		counter++
	}

	// Set next key for pagination
	if iter.Valid() {
		// For the next page, we need to encode the current position
		// Since we're using MatchExact with a specific metaId+queryId combination,
		// the next key should contain information about where we left off
		// We'll use the reporter address from the last processed report as the next key
		if len(microreports) > 0 {
			iter.Next()
			// Get the primary key for the next page
			key, err := iter.PrimaryKey()
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			// Encode the reporter address as the next key
			pageRes.NextKey = key.K1()
		}
	}

	pageRes.Total = uint64(len(microreports))

	return &types.QueryGetReportsByAggregateResponse{MicroReports: microreports, Pagination: pageRes}, nil
}

func advanceIterToNextKey(iter *indexes.MultiIterator[collections.Pair[uint64, []byte], collections.Triple[[]byte, []byte, uint64]], nextKey []byte) bool {
	for iter.Valid() {
		key, err := iter.PrimaryKey()
		if err != nil {
			return false
		}
		if bytes.Equal(key.K1(), nextKey) {
			return true
		}
		iter.Next()
	}
	return false
}
