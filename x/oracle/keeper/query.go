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

// gets an aggregate report by query id and timestamp
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

// gets the current aggregate report for a query id
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

// gets the last aggregate report before a timestamp by query id and reporter
func (k Querier) GetAggregateBeforeByReporter(ctx context.Context, req *types.QueryGetAggregateBeforeByReporterRequest) (*types.QueryGetAggregateBeforeByReporterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryId, err := hex.DecodeString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query id")
	}
	reporterAddr := sdk.MustAccAddressFromBech32(req.Reporter)
	aggregate, err := k.keeper.GetAggregateBeforeByReporter(ctx, queryId, time.UnixMilli(int64(req.Timestamp)), reporterAddr)
	if err != nil {
		return nil, err
	}
	return &types.QueryGetAggregateBeforeByReporterResponse{Aggregate: aggregate}, nil
}

// gets a query by query id and id
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

// returns a list of queries that are not expired and have a tip available
func (k Querier) TippedQueriesForDaemon(ctx context.Context, req *types.QueryTippedQueriesForDaemonRequest) (*types.QueryTippedQueriesForDaemonResponse, error) {
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
		if queryMeta.Expiration > uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()) && queryMeta.Amount.GT(math.ZeroInt()) {
			queries = append(queries, &queryMeta)
		}

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTippedQueriesForDaemonResponse{Queries: queries}, nil
}

// returns a list of reported ids by reporter
func (k Querier) ReportedIdsByReporter(ctx context.Context, req *types.QueryReportedIdsByReporterRequest) (*types.QueryReportedIdsByReporterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	store := runtime.KVStoreAdapter(k.keeper.storeService.OpenKVStore(ctx))
	reportsStore := prefix.NewStore(store, types.ReportsPrefix)
	ids := make([]uint64, 0)
	queryIds := make([][]byte, 0)

	pageRes, err := query.Paginate(reportsStore, req.Pagination, func(key, value []byte) error {
		keycdc := collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.Uint64Key)
		_, pk, err := keycdc.Decode(key)
		if err != nil {
			return err
		}
		ids = append(ids, pk.K3())
		queryIds = append(queryIds, pk.K1())

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryReportedIdsByReporterResponse{Ids: ids, QueryIds: queryIds, Pagination: pageRes}, nil
}

// gets the timestamp before a given timestamp by query id
func (k Querier) GetTimestampBefore(ctx context.Context, req *types.QueryGetTimestampBeforeRequest) (*types.QueryGetTimestampBeforeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryId, err := hex.DecodeString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query id")
	}
	timestamp, err := k.keeper.GetTimestampBefore(ctx, queryId, time.UnixMilli(int64(req.Timestamp)))
	if err != nil {
		return nil, err
	}

	return &types.QueryGetTimestampBeforeResponse{Timestamp: uint64(timestamp.UnixMilli())}, nil
}

// gets the timestamp after a given timestamp by query id
func (k Querier) GetTimestampAfter(ctx context.Context, req *types.QueryGetTimestampAfterRequest) (*types.QueryGetTimestampAfterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryId, err := hex.DecodeString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query id")
	}
	timestamp, err := k.keeper.GetTimestampAfter(ctx, queryId, time.UnixMilli(int64(req.Timestamp)))
	if err != nil {
		return nil, err
	}
	return &types.QueryGetTimestampAfterResponse{Timestamp: uint64(timestamp.UnixMilli())}, nil
}

// returns a list of queries that are not expired and have a tip available and the query data as a string
func (k Querier) GetTippedQueries(ctx context.Context, req *types.QueryGetTippedQueriesRequest) (*types.QueryGetTippedQueriesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	store := runtime.KVStoreAdapter(k.keeper.storeService.OpenKVStore(ctx))
	queryStore := prefix.NewStore(store, types.QueryTipPrefix)
	queries := make([]*types.QueryMetaButString, 0)
	_, err := query.Paginate(queryStore, req.Pagination, func(queryId, value []byte) error {
		// pull querymeta from store
		var queryMeta types.QueryMeta
		err := k.keeper.cdc.Unmarshal(value, &queryMeta)
		if err != nil {
			return err
		}
		if queryMeta.Expiration > uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()) && queryMeta.Amount.GT(math.ZeroInt()) {
			// write querymeta to querymetabutstring
			queryMetaButString := types.QueryMetaButString{
				Id:                      queryMeta.Id,
				Amount:                  queryMeta.Amount,
				Expiration:              queryMeta.Expiration,
				RegistrySpecBlockWindow: queryMeta.RegistrySpecBlockWindow,
				HasRevealedReports:      queryMeta.HasRevealedReports,
				QueryData:               hex.EncodeToString(queryMeta.QueryData),
				QueryType:               queryMeta.QueryType,
				CycleList:               queryMeta.CycleList,
			}
			queries = append(queries, &queryMetaButString)
		}

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetTippedQueriesResponse{Queries: queries}, nil
}
