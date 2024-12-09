package keeper_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	minttypes "github.com/tellor-io/layer/x/mint/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

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
