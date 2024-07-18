package keeper_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (s *KeeperTestSuite) TestQueryGetTimeBasedRewards() {
	require := s.Require()
	q := s.queryClient
	ak := s.accountKeeper
	bk := s.bankKeeper
	ctx := s.ctx

	// nil request
	res, err := q.GetTimeBasedRewards(s.ctx, nil)
	require.ErrorContains(err, "invalid request")
	require.Nil(res)

	addr := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(addr)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)

	ak.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(testModuleAccount)
	bk.On("GetBalance", ctx, addr, "loya").Return(sdk.NewCoin("loya", math.NewInt(1_000*1e6))).Once()
	res, err = q.GetTimeBasedRewards(ctx, &types.QueryGetTimeBasedRewardsRequest{})
	require.NoError(err)
	require.Equal(res.Reward.Amount, math.NewInt(1_000*1e6))
	require.Equal(res.Reward.Denom, "loya")

	bk.On("GetBalance", ctx, addr, "loya").Return(sdk.NewCoin("loya", math.NewInt(0))).Once()
	res, err = q.GetTimeBasedRewards(ctx, &types.QueryGetTimeBasedRewardsRequest{})
	require.NoError(err)
	require.Equal(res.Reward.Amount, math.NewInt(0))
	require.Equal(res.Reward.Denom, "loya")
}
