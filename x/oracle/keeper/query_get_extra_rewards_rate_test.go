package keeper_test

import (
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestQueryGetExtraRewardsRate() {
	require := s.Require()
	q := s.queryClient
	mk := s.mintKeeper
	ctx := s.ctx

	// nil request
	res, err := q.GetExtraRewardsRate(s.ctx, nil)
	require.ErrorContains(err, "invalid request")
	require.Nil(res)

	// mock mint keeper to return extra reward params
	extraRewardParams := minttypes.ExtraRewardParams{
		DailyExtraRewards: 1_000_000,
		BondDenom:         "loya",
	}
	mk.On("GetExtraRewardRateParams", ctx).Return(extraRewardParams).Once()
	res, err = q.GetExtraRewardsRate(ctx, &types.QueryGetExtraRewardsRateRequest{})
	require.NoError(err)
	require.Equal(int64(1_000_000), res.DailyExtraRewards)

	// test with zero extra rewards
	extraRewardParams.DailyExtraRewards = 0
	mk.On("GetExtraRewardRateParams", ctx).Return(extraRewardParams).Once()
	res, err = q.GetExtraRewardsRate(ctx, &types.QueryGetExtraRewardsRateRequest{})
	require.NoError(err)
	require.Equal(int64(0), res.DailyExtraRewards)
}
