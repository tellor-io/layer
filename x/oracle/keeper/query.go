package keeper

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	keeper Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{keeper: keeper}
}

func (k Querier) RetrieveData(ctx context.Context, req *types.QueryRetrieveDataRequest) (*types.QueryRetrieveDataResponse, error) {
	queryId, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid queryId")
	}
	agg, err := k.keeper.Aggregates.Get(ctx, collections.Join(queryId, req.Timestamp))
	if err != nil {
		return nil, err
	}
	return &types.QueryRetrieveDataResponse{Aggregate: &agg}, nil
}

func (k Querier) GetCurrentAggregateReport(ctx context.Context, req *types.QueryGetCurrentAggregateReportRequest) (*types.QueryGetCurrentAggregateReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryId, err := hex.DecodeString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query id")
	}
	aggregate, timestamp, err := k.keeper.GetCurrentAggregateReport(ctx, queryId)
	if err != nil {
		return nil, err
	}
	timeUnix := timestamp.UnixMilli()

	return &types.QueryGetCurrentAggregateReportResponse{
		Aggregate: aggregate,
		Timestamp: uint64(timeUnix),
	}, nil
}

func (k Querier) GetAggregateBeforeByReporter(ctx context.Context, req *types.QueryGetAggregateBeforeByReporterRequest) (*types.QueryGetAggregateBeforeByReporterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryId, err := hex.DecodeString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query id")
	}
	reporterAddr := sdk.MustAccAddressFromBech32(req.Reporter)
	aggregate, err := k.keeper.GetAggregateBeforeByReporter(ctx, queryId, time.UnixMilli(req.Timestamp), reporterAddr)
	if err != nil {
		return nil, err
	}
	return &types.QueryGetAggregateBeforeByReporterResponse{Aggregate: aggregate}, nil
}

func (k Querier) GetQuery(ctx context.Context, req *types.QueryGetQueryRequest) (*types.QueryGetQueryResponse, error) {
	queryId, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid queryId")
	}
	query, err := k.keeper.Query.Get(ctx, collections.Join(queryId, req.Id))
	if err != nil {
		return nil, err
	}
	return &types.QueryGetQueryResponse{Query: &query}, nil
}

func (k Querier) TippedQueries(ctx context.Context, req *types.QueryTippedQueriesRequest) (*types.QueryTippedQueriesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	store := runtime.KVStoreAdapter(k.keeper.storeService.OpenKVStore(ctx))
	queryStore := prefix.NewStore(store, types.QueryTipPrefix)
	queries := make([]*types.QueryMeta, 0)
	_, err := query.Paginate(queryStore, req.Pagination, func(queryId, value []byte) error {
		var queryMeta types.QueryMeta
		err := k.keeper.cdc.Unmarshal(value, &queryMeta)
		if err != nil {
			return err
		}
		if queryMeta.Expiration.Add(offset).After(sdk.UnwrapSDKContext(ctx).BlockTime()) && queryMeta.Amount.GT(math.ZeroInt()) {
			queries = append(queries, &queryMeta)
		}

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTippedQueriesResponse{Queries: queries}, nil
}
