package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

var reward = math.NewInt(100)

func TestCalculateRewardAmount(t *testing.T) {
	testCases := []struct {
		name           string
		reporter       []keeper.ReportersReportCount
		reporterPowers []uint64
		totalPower     uint64
		reportsCount   uint64
		expectedAmount []reportertypes.BigUint
	}{
		{
			name:           "Test all reporters report",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 1}, {Power: 20, Reports: 1}, {Power: 30, Reports: 1}, {Power: 40, Reports: 1}},
			expectedAmount: []reportertypes.BigUint{{Value: math.NewUint(10 * 1e6)}, {Value: math.NewUint(20 * 1e6)}, {Value: math.NewUint(30 * 1e6)}, {Value: math.NewUint(40 * 1e6)}},
			totalPower:     100, // 40 + 30 + 20 + 10

		},
		{
			name:           "only 1 reports",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 1}, {Power: 20, Reports: 0}, {Power: 30, Reports: 0}, {Power: 40, Reports: 0}},
			expectedAmount: []reportertypes.BigUint{{Value: math.NewUint(100 * 1e6)}, {Value: math.NewUint(0)}, {Value: math.NewUint(0)}, {Value: math.NewUint(0)}},
			totalPower:     10,
		},
		{
			name:           "only 1 and 3 reports one report, a single queryId",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 1}, {Power: 20, Reports: 0}, {Power: 30, Reports: 1}, {Power: 40, Reports: 0}},
			expectedAmount: []reportertypes.BigUint{{Value: math.NewUint(25 * 1e6)}, {Value: math.NewUint(0)}, {Value: math.NewUint(75 * 1e6)}, {Value: math.NewUint(0)}},
			totalPower:     40, // 30 + 10
		},
		{
			name:           "all reporters report, a two queryIds",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 2}, {Power: 20, Reports: 2}, {Power: 30, Reports: 2}, {Power: 40, Reports: 2}},
			expectedAmount: []reportertypes.BigUint{{Value: math.NewUint(10 * 1e6)}, {Value: math.NewUint(20 * 1e6)}, {Value: math.NewUint(30 * 1e6)}, {Value: math.NewUint(40 * 1e6)}},
			totalPower:     200,
		},
		{
			name:     "all reporters report single, and 1 reports two queryIds",
			reporter: []keeper.ReportersReportCount{{Power: 10, Reports: 2}, {Power: 20, Reports: 1}, {Power: 30, Reports: 1}, {Power: 40, Reports: 1}},
			expectedAmount: []reportertypes.BigUint{
				// power*reports/totalPower*reward
				{Value: math.NewUint((10 * 2 * 100) * 1e6).Quo(math.NewUint(110))},
				{Value: math.NewUint((20 * 100) * 1e6).Quo(math.NewUint(110))},
				{Value: math.NewUint((30 * 100) * 1e6).Quo(math.NewUint(110))},
				{Value: math.NewUint((40 * 100) * 1e6).Quo(math.NewUint(110))},
			},
			totalPower: 110, // 40 + 30 + 20 + (10 * 2)
		},
		{
			name:     "all reporters report single, 1 and 3 report a second queryId",
			reporter: []keeper.ReportersReportCount{{Power: 10, Reports: 2}, {Power: 20, Reports: 1}, {Power: 30, Reports: 2}, {Power: 40, Reports: 1}},
			expectedAmount: []reportertypes.BigUint{
				{Value: math.NewUint(14285714)},
				{Value: math.NewUint(14285714)}, // 285714285700
				{Value: math.NewUint(42857142)}, // 857142857100
				{Value: math.NewUint(28571428)}, // 571428571400
			},
			totalPower: 140, // 40 + (30 * 2) + 20 + (10 * 2)
		},
	}
	for _, tc := range testCases {
		fmt.Println("Start of test case")
		expectedTotalReward := math.ZeroUint()
		totaldist := math.ZeroUint()
		t.Run(tc.name, func(t *testing.T) {
			for i, r := range tc.reporter {
				amount := keeper.CalculateRewardAmount(r.Power, r.Reports, tc.totalPower, reward)
				fmt.Println("Reward amount: ", amount.Value.String())
				totaldist = totaldist.Add(amount.Value)
				fmt.Println("TotalDist: ", totaldist.String())
				require.Equal(t, amount, tc.expectedAmount[i])
				if i == len(tc.reporter)-1 {
					amount.Value = amount.Value.Add(math.NewUint(reward.Uint64() * 1e6)).Sub(totaldist)
				}
				expectedTotalReward = expectedTotalReward.Add(amount.Value)
				fmt.Println("Expected total reward: ", expectedTotalReward.String())
			}
		})
		require.True(t, expectedTotalReward.Equal(math.NewUint(reward.Uint64()*1e6)), "reward amount should be within tolerance")
	}
}

func (s *KeeperTestSuite) TestAllocateRewards() {
	require := s.Require()
	k := s.oracleKeeper
	bk := s.bankKeeper
	rk := s.reporterKeeper
	ctx := s.ctx

	// zero reward
	reports := []*types.Aggregate{}
	reward := math.ZeroInt()
	require.NoError(k.AllocateRewards(ctx, reports, reward, types.ModuleName))

	// 2 reporters, bad addresses
	reporters := []*types.Aggregate{
		{
			QueryId: []byte{},
			Reporters: []*types.AggregateReporter{
				{Reporter: "bad address", Power: 10, BlockNumber: 0},
				{Reporter: "bad address", Power: 10, BlockNumber: 0},
			},
		},
	}
	reward = math.NewInt(100)
	require.Error(k.AllocateRewards(ctx, reporters, reward, types.ModuleName))

	// 2 reporters, good addresses
	rep1 := sample.AccAddress()
	rep2 := sample.AccAddress()
	reporters = []*types.Aggregate{
		{
			QueryId: []byte{},
			Reporters: []*types.AggregateReporter{
				{Reporter: rep1, Power: 10, BlockNumber: 0},
				{Reporter: rep2, Power: 10, BlockNumber: 0},
			},
		},
	}

	reward = math.NewInt(100)
	rep1Addr, err := sdk.AccAddressFromBech32(rep1)
	require.NoError(err)
	rep2Addr, err := sdk.AccAddressFromBech32(rep2)
	require.NoError(err)
	rk.On("DivvyingTips", ctx, rep1Addr, reportertypes.BigUint{Value: math.NewUint(50000000)}, []byte{}, uint64(0)).Return(nil).Once()
	rk.On("DivvyingTips", ctx, rep2Addr, reportertypes.BigUint{Value: math.NewUint(50000000)}, []byte{}, uint64(0)).Return(nil).Once()
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
	amount := reportertypes.BigUint{Value: math.NewUint(100)}
	rk.On("DivvyingTips", ctx, addr, amount, []byte{}, uint64(ctx.BlockHeight())).Return(nil).Once()
	require.NoError(k.AllocateTip(ctx, addr, []byte{}, amount, uint64(ctx.BlockHeight())))
}
