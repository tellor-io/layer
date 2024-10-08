package keeper

import (
	"context"
	"encoding/hex"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Querier) CurrentCyclelistQuery(ctx context.Context, req *types.QueryCurrentCyclelistQueryRequest) (*types.QueryCurrentCyclelistQueryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	querydata, err := k.keeper.GetCurrentQueryInCycleList(ctx)
	if err != nil {
		return nil, err
	}
	queryId := utils.QueryIDFromData(querydata)
	query, err := k.keeper.CurrentQuery(ctx, queryId)
	if err != nil {
		return nil, err
	}
	return &types.QueryCurrentCyclelistQueryResponse{QueryData: hex.EncodeToString(querydata), QueryMeta: &query}, nil
}

func (k Querier) NextCyclelistQuery(ctx context.Context, req *types.QueryNextCyclelistQueryRequest) (*types.QueryNextCyclelistQueryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	querydata, err := k.keeper.GetNextCurrentQueryInCycleList(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryNextCyclelistQueryResponse{QueryData: hex.EncodeToString(querydata)}, nil
}
