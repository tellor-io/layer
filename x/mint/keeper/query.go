package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/mint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	keeper Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{keeper: keeper}
}

func (q Querier) GetExtraRewardsRate(ctx context.Context, req *types.QueryGetExtraRewardsRateRequest) (*types.QueryGetExtraRewardsRateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	effectiveRate := q.keeper.GetEffectiveExtraRewardsRate(ctx)

	return &types.QueryGetExtraRewardsRateResponse{DailyExtraRewards: effectiveRate}, nil
}
