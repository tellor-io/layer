package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetQueryDataLimit(ctx context.Context, req *types.QueryGetQueryDataLimitRequest) (*types.QueryGetQueryDataLimitResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	limit, err := q.keeper.QueryDataLimit.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetQueryDataLimitResponse{
		Limit: limit.Limit,
	}, nil
}

func (k Keeper) SetQueryDataLimit(ctx context.Context, limit uint64) error {
	return k.QueryDataLimit.Set(ctx, types.QueryDataLimit{Limit: limit})
}
