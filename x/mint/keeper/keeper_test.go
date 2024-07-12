package keeper_test

import (
	"testing"

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
