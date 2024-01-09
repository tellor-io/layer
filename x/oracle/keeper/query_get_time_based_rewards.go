package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetTimeBasedRewards(goCtx context.Context, req *types.QueryGetTimeBasedRewardsRequest) (*types.QueryGetTimeBasedRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	rewards := k.getTimeBasedRewards(ctx)

	return &types.QueryGetTimeBasedRewardsResponse{Reward: rewards}, nil
}
