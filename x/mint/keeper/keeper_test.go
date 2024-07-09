package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/mint/keeper"
	"github.com/tellor-io/layer/x/mint/mocks"
	"github.com/tellor-io/layer/x/mint/types"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	auth "github.com/cosmos/cosmos-sdk/x/auth"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank"
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
	s.accountKeeper.On("GetModuleAddress", types.ModuleName).Return(authtypes.NewModuleAddress(types.ModuleName))
	s.accountKeeper.On("GetModuleAddress", types.TimeBasedRewards).Return(authtypes.NewModuleAddress(types.TimeBasedRewards))

	appCodec := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, staking.AppModuleBasic{}).Codec
	keys := storetypes.NewKVStoreKey(types.StoreKey)

	keeper := keeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys), s.accountKeeper, s.bankKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	s.NotNil(keeper)
}

func (s *KeeperTestSuite) TestLogger() {
	logger := s.mintKeeper.Logger(s.ctx)
	s.NotNil(logger)
}

func (s *KeeperTestSuite) TestMintCoins() {
	// coins := sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(100*1e6)))

	// err := s.mintKeeper.MintCoins(s.ctx, coins)
	// s.NoError(err)
}

func (s *KeeperTestSuite) TestSendInflationaryRewards() {
	// coins := sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(100*1e6)))

	// err := s.mintKeeper.SendInflationaryRewards(s.ctx, coins)
	// s.NoError(err)
}

func (s *KeeperTestSuite) TestGetAuthority() {
	require := s.Require()
	k := s.mintKeeper
	authority := k.GetAuthority()
	require.Equal(authority, authtypes.NewModuleAddress(govtypes.ModuleName).String())
}
