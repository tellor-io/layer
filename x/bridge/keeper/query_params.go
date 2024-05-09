package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
)

func (q Querier) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := q.k.Params.Get(ctx)
	return &types.QueryParamsResponse{Params: params}, err
}
