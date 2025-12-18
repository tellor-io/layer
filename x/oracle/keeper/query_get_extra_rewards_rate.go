package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetExtraRewardsRate(ctx context.Context, req *types.QueryGetExtraRewardsRateRequest) (*types.QueryGetExtraRewardsRateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	params := q.keeper.mintKeeper.GetExtraRewardRateParams(ctx)

	return &types.QueryGetExtraRewardsRateResponse{DailyExtraRewards: params.DailyExtraRewards}, nil
}
