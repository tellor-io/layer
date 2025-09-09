package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/mint/keeper"
	"github.com/tellor-io/layer/x/mint/mocks"
	"github.com/tellor-io/layer/x/mint/types"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	auth "github.com/cosmos/cosmos-sdk/x/auth"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	mintKeeper    keeper.Keeper
	accountKeeper *mocks.AccountKeeper
	bankKeeper    *mocks.BankKeeper
}

func (s *KeeperTestSuite) SetupTest() {
	config.SetupConfig()

	s.mintKeeper,
		s.accountKeeper,
		s.bankKeeper,
		s.ctx = keepertest.MintKeeper(s.T())
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestNewKeeper() {
	ak := s.accountKeeper
	ak.On("GetModuleAddress", types.ModuleName).Return(authtypes.NewModuleAddress(types.ModuleName)).Once()
	ak.On("GetModuleAddress", types.TimeBasedRewards).Return(authtypes.NewModuleAddress(types.TimeBasedRewards)).Once()

	appCodec := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, staking.AppModuleBasic{}).Codec
	keys := storetypes.NewKVStoreKey(types.StoreKey)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	k := keeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys), s.accountKeeper, s.bankKeeper, authority)
	s.NotNil(k)

	// invalid authority
	require.Panics(s.T(), func() {
		keeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys), s.accountKeeper, s.bankKeeper, "badAuthority")
	})
}

func (s *KeeperTestSuite) TestLogger() {
	logger := s.mintKeeper.Logger(s.ctx)
	s.NotNil(logger)
}

func (s *KeeperTestSuite) TestMintCoins() {
	require := s.Require()
	bk := s.bankKeeper
	coins := sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(100*1e6)))

	bk.On("MintCoins", s.ctx, types.ModuleName, coins).Return(nil)
	err := s.mintKeeper.MintCoins(s.ctx, coins)
	require.NoError(err)

	emptyCoins := sdk.NewCoins()
	bk.On("MintCoins", s.ctx, types.ModuleName, emptyCoins).Return(nil)
	err = s.mintKeeper.MintCoins(s.ctx, emptyCoins)
	require.Nil(err)
}

func (s *KeeperTestSuite) TestSendInflationaryRewards() {
	require := s.Require()
	bk := s.bankKeeper
	threeQuartersCoins := sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(75*1e6)))
	oneQuarterCoins := sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(25*1e6)))
	totalCoins := sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(100*1e6)))

	input := banktypes.NewInput(authtypes.NewModuleAddress(types.ModuleName), totalCoins)
	outputs := []banktypes.Output{
		{
			Address: authtypes.NewModuleAddressOrBech32Address(types.TimeBasedRewards).String(),
			Coins:   threeQuartersCoins,
		},
		{
			Address: authtypes.NewModuleAddressOrBech32Address(authtypes.FeeCollectorName).String(),
			Coins:   oneQuarterCoins,
		},
	}
	bk.On("InputOutputCoins", s.ctx, input, outputs).Return(nil)
	err := s.mintKeeper.SendInflationaryRewards(s.ctx, totalCoins)
	require.NoError(err)

	emptyCoins := sdk.NewCoins()
	err = s.mintKeeper.SendInflationaryRewards(s.ctx, emptyCoins)
	require.Nil(err)
}

func (s *KeeperTestSuite) TestGetAuthority() {
	require := s.Require()
	k := s.mintKeeper

	authority := k.GetAuthority()
	require.Equal(authority, authtypes.NewModuleAddress(govtypes.ModuleName).String())
}

func (s *KeeperTestSuite) TestSendExtraRewards() {
	require := s.Require()
	k := s.mintKeeper
	bk := s.bankKeeper
	ak := s.accountKeeper

	extraRewardsAddr := authtypes.NewModuleAddress(types.ExtraRewardsPool)
	ak.On("GetModuleAddress", types.ExtraRewardsPool).Return(extraRewardsAddr)

	// Zero balance in extra rewards pool should return nil
	bk.On("GetBalance", s.ctx, extraRewardsAddr, types.DefaultBondDenom).Return(sdk.NewCoin(types.DefaultBondDenom, math.ZeroInt())).Once()
	err := k.SendExtraRewards(s.ctx)
	require.Nil(err)

	// First time call with no previous block time should return nil
	params := types.ExtraRewardParams{
		DailyExtraRewards: types.DailyMintRate,
		PreviousBlockTime: nil,
		BondDenom:         types.DefaultBondDenom,
	}
	err = k.ExtraRewardParams.Set(s.ctx, params)
	require.NoError(err)
	err = k.SendExtraRewards(s.ctx)
	require.NoError(err)

	// Sufficient balance with valid previous block time
	currentTime := time.Now().UTC()
	s.ctx = s.ctx.WithBlockTime(currentTime)
	previousTime := currentTime.Add(-24 * time.Hour) // 1 day ago
	params.PreviousBlockTime = &previousTime
	err = k.ExtraRewardParams.Set(s.ctx, params)
	require.NoError(err)

	// Calculate expected reward amount (1 day worth)
	expectedRewardAmount := math.NewInt(types.DailyMintRate)

	// Pool has sufficient balance
	poolBalance := sdk.NewCoin(types.DefaultBondDenom, expectedRewardAmount.MulRaw(2))
	bk.On("GetBalance", s.ctx, extraRewardsAddr, types.DefaultBondDenom).Return(poolBalance).Once()

	// Set up expected InputOutputCoins call
	banktypesInput := banktypes.NewInput(extraRewardsAddr, sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, expectedRewardAmount)))
	banktypesOutputs := []banktypes.Output{
		{
			Address: authtypes.NewModuleAddressOrBech32Address(types.TimeBasedRewards).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, expectedRewardAmount.QuoRaw(4).MulRaw(3))),
		},
		{
			Address: authtypes.NewModuleAddressOrBech32Address(authtypes.FeeCollectorName).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, expectedRewardAmount.QuoRaw(4))),
		},
	}
	bk.On("InputOutputCoins", s.ctx, banktypesInput, banktypesOutputs).Return(nil).Once()

	err = k.SendExtraRewards(s.ctx)
	require.NoError(err)

	// Verify previous block time was updated
	updatedParams, err := k.ExtraRewardParams.Get(s.ctx)
	require.NoError(err)
	require.Equal(currentTime, *updatedParams.PreviousBlockTime)

	// Insufficient balance in pool should return nil
	params.PreviousBlockTime = &previousTime
	err = k.ExtraRewardParams.Set(s.ctx, params)
	require.NoError(err)

	insufficientBalance := sdk.NewCoin(types.DefaultBondDenom, math.NewInt(100)) // Very small balance
	bk.On("GetBalance", s.ctx, extraRewardsAddr, types.DefaultBondDenom).Return(insufficientBalance).Once()

	err = k.SendExtraRewards(s.ctx)
	require.NoError(err)

	// Verify previous block time was still updated even though no rewards sent
	updatedParams, err = k.ExtraRewardParams.Get(s.ctx)
	require.NoError(err)
	require.Equal(currentTime, *updatedParams.PreviousBlockTime)

	// Zero time elapsed - should handle gracefully
	params.PreviousBlockTime = &currentTime // Same as current time
	err = k.ExtraRewardParams.Set(s.ctx, params)
	require.NoError(err)

	bk.On("GetBalance", s.ctx, extraRewardsAddr, types.DefaultBondDenom).Return(poolBalance).Once()

	err = k.SendExtraRewards(s.ctx)
	require.NoError(err)
}

func (s *KeeperTestSuite) TestGetExtraRewardRateParams() {
	require := s.Require()
	k := s.mintKeeper

	// Get params when not set should return defults
	params := k.GetExtraRewardRateParams(s.ctx)
	require.Equal(types.DefaultBondDenom, params.BondDenom)
	require.Equal(int64(0), params.DailyExtraRewards)
	require.Nil(params.PreviousBlockTime)

	// Set and get params
	currentTime := s.ctx.BlockTime()
	expectedParams := types.ExtraRewardParams{
		DailyExtraRewards: 1000000,
		PreviousBlockTime: &currentTime,
		BondDenom:         "testdenom",
	}
	err := k.ExtraRewardParams.Set(s.ctx, expectedParams)
	require.NoError(err)

	retrievedParams := k.GetExtraRewardRateParams(s.ctx)
	require.Equal(expectedParams.DailyExtraRewards, retrievedParams.DailyExtraRewards)
	require.Equal(expectedParams.BondDenom, retrievedParams.BondDenom)
	require.NotNil(retrievedParams.PreviousBlockTime)
	require.Equal(*expectedParams.PreviousBlockTime, *retrievedParams.PreviousBlockTime)
}
