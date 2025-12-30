package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/mint/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (q Querier) GetExtraRewardsPoolBalance(ctx context.Context, req *types.QueryGetExtraRewardsPoolBalanceRequest) (*types.QueryGetExtraRewardsPoolBalanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	moduleAddr := q.keeper.accountKeeper.GetModuleAddress(types.ExtraRewardsPool)
	params := q.keeper.GetExtraRewardRateParams(ctx)
	balance := q.keeper.bankKeeper.GetBalance(ctx, moduleAddr, params.BondDenom)

	return &types.QueryGetExtraRewardsPoolBalanceResponse{Balance: balance}, nil
}

// GetExtraRewardsPoolAddress returns the address of the extra rewards pool module account.
func (q Querier) GetExtraRewardsPoolAddress() sdk.AccAddress {
	return q.keeper.accountKeeper.GetModuleAddress(types.ExtraRewardsPool)
}
