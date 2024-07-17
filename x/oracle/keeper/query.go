package keeper

import (
	"context"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
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
