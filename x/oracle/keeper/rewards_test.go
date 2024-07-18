package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"
)

var reward = math.NewInt(100)

func TestCalculateRewardAmount(t *testing.T) {
	testCases := []struct {
		name           string
		reporter       []keeper.ReportersReportCount
		reporterPowers []int64
		totalPower     int64
		reportsCount   int64
		expectedAmount []math.Int
	}{
		{
			name:           "Test all reporters report",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 1}, {Power: 20, Reports: 1}, {Power: 30, Reports: 1}, {Power: 40, Reports: 1}},
			expectedAmount: []math.Int{math.NewInt(10), math.NewInt(20), math.NewInt(30), math.NewInt(40)},
			totalPower:     100, // 40 + 30 + 20 + 10

		},
		{
			name:           "only 1 reports",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 1}, {Power: 20, Reports: 0}, {Power: 30, Reports: 0}, {Power: 40, Reports: 0}},
			expectedAmount: []math.Int{math.NewInt(100), math.NewInt(0), math.NewInt(0), math.NewInt(0)},
			totalPower:     10,
		},
		{
			name:           "only 1 and 3 reports one report, a single queryId",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 1}, {Power: 20, Reports: 0}, {Power: 30, Reports: 1}, {Power: 40, Reports: 0}},
			expectedAmount: []math.Int{math.NewInt(25), math.NewInt(0), math.NewInt(75), math.NewInt(0)},
			totalPower:     40, // 30 + 10
		},
		{
			name:           "all reporters report, a two queryIds",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 2}, {Power: 20, Reports: 2}, {Power: 30, Reports: 2}, {Power: 40, Reports: 2}},
			expectedAmount: []math.Int{math.NewInt(10), math.NewInt(20), math.NewInt(30), math.NewInt(40)},
			totalPower:     200,
		},
		{
			name:           "all reporters report single, and 1 reports two queryIds",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 2}, {Power: 20, Reports: 1}, {Power: 30, Reports: 1}, {Power: 40, Reports: 1}},
			expectedAmount: []math.Int{math.NewInt(18), math.NewInt(18), math.NewInt(27), math.NewInt(36)},
			totalPower:     110, // 40 + 30 + 20 + (10 * 2)
		},
		{
			name:           "all reporters report single, 1 and 3 report a second queryId",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 2}, {Power: 20, Reports: 1}, {Power: 30, Reports: 2}, {Power: 40, Reports: 1}},
			expectedAmount: []math.Int{math.NewInt(14), math.NewInt(14), math.NewInt(43), math.NewInt(29)},
			totalPower:     140, // 40 + (30 * 2) + 20 + (10 * 2)
		},
	}
	for _, tc := range testCases {
		expectedTotalReward := math.ZeroInt()
		t.Run(tc.name, func(t *testing.T) {
			for i, r := range tc.reporter {
				amount := keeper.CalculateRewardAmount(r.Power, int64(r.Reports), tc.totalPower, reward)
				require.Equal(t, amount, tc.expectedAmount[i])
				expectedTotalReward = expectedTotalReward.Add(amount)
			}
		})
		tolerance := reward.Sub(math.OneInt()) // when rounded
		withinTolerance := expectedTotalReward.Equal(reward) || expectedTotalReward.Equal(tolerance)
		require.True(t, withinTolerance, "reward amount should be within tolerance")
	}
}

func (s *KeeperTestSuite) TestAllocateRewards() {
	require := s.Require()
	k := s.oracleKeeper
	bk := s.bankKeeper
	rk := s.reporterKeeper
	ctx := s.ctx

	// zero reward
	reporters := []*types.AggregateReporter{}
	reward := math.ZeroInt()
	require.NoError(k.AllocateRewards(ctx, reporters, reward, types.ModuleName))

	// 2 reporters, bad addresses
	reporters = []*types.AggregateReporter{
		{Reporter: "bad address", Power: 10, BlockNumber: 0},
		{Reporter: "bad address", Power: 10, BlockNumber: 0},
	}
	reward = math.NewInt(100)
	require.Error(k.AllocateRewards(ctx, reporters, reward, types.ModuleName))

	// 2 reporters, good addresses
	rep1 := sample.AccAddress()
	rep2 := sample.AccAddress()
	reporters = []*types.AggregateReporter{
		{Reporter: rep1, Power: 10, BlockNumber: 0},
		{Reporter: rep2, Power: 10, BlockNumber: 0},
	}
	reward = math.NewInt(100)
	rep1Addr, err := sdk.AccAddressFromBech32(rep1)
	require.NoError(err)
	rep2Addr, err := sdk.AccAddressFromBech32(rep2)
	require.NoError(err)
	rk.On("DivvyingTips", ctx, rep1Addr, math.NewInt(50), int64(0)).Return(nil).Once()
	rk.On("DivvyingTips", ctx, rep2Addr, math.NewInt(50), int64(0)).Return(nil).Once()
	bk.On("SendCoinsFromModuleToModule", ctx, "oracle", "tips_escrow_pool", sdk.NewCoins(sdk.NewCoin("loya", reward))).Return(nil)
	require.NoError(k.AllocateRewards(ctx, reporters, reward, types.ModuleName))
}

func (s *KeeperTestSuite) TestGetTimeBasedRewards() {
	require := s.Require()
	k := s.oracleKeeper
	ak := s.accountKeeper
	bk := s.bankKeeper
	ctx := s.ctx

	add := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(add)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)
	ak.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(testModuleAccount))
	bk.On("GetBalance", ctx, testModuleAccount.GetAddress(), "loya").Return(sdk.Coin{Amount: math.NewInt(100), Denom: "loya"}).Once()
	tbr := k.GetTimeBasedRewards(ctx)
	require.Equal(tbr, math.NewInt(100))

	bk.On("GetBalance", ctx, testModuleAccount.GetAddress(), "loya").Return(sdk.Coin{Amount: math.NewInt(0), Denom: "loya"}).Once()
	tbr = k.GetTimeBasedRewards(ctx)
	require.Equal(tbr, math.ZeroInt())
}

func (s *KeeperTestSuite) TestGetTimeBasedRewardsAccount() {
	require := s.Require()
	k := s.oracleKeeper
	ak := s.accountKeeper
	ctx := s.ctx

	ak.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(nil)).Once()
	require.Equal(k.GetTimeBasedRewardsAccount(ctx), nil)

	add := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(add)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)
	ak.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(testModuleAccount)).Once()
	require.Equal(k.GetTimeBasedRewardsAccount(ctx), testModuleAccount)
}

func (s *KeeperTestSuite) TestAllocateTips() {
	require := s.Require()
	k := s.oracleKeeper
	rk := s.reporterKeeper
	ctx := s.ctx

	addr := sample.AccAddressBytes()
	amount := math.NewInt(100)
	rk.On("DivvyingTips", ctx, addr, amount, ctx.BlockHeight()).Return(nil).Once()
	require.NoError(k.AllocateTip(ctx, addr, amount, ctx.BlockHeight()))
}
