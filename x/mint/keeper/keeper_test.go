package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	auth "github.com/cosmos/cosmos-sdk/x/auth"
	bank "github.com/cosmos/cosmos-sdk/x/bank"
	staking "github.com/cosmos/cosmos-sdk/x/staking"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/tellor-io/layer/app/config"
	"github.com/tellor-io/layer/x/mint/keeper"
	"github.com/tellor-io/layer/x/mint/mocks"
	"github.com/tellor-io/layer/x/mint/types"

	keepertest "github.com/tellor-io/layer/testutil/keeper"
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

func (s *KeeperTestSuite) TestNewKeeper(t *testing.T) {
	s.SetupTest()

	s.accountKeeper.On("GetModuleAddress", types.ModuleName).Return(authtypes.NewModuleAddress(types.ModuleName))
	s.accountKeeper.On("GetModuleAddress", types.MintToTeam).Return(authtypes.NewModuleAddress(types.MintToTeam))
	s.accountKeeper.On("GetModuleAddress", types.TimeBasedRewards).Return(authtypes.NewModuleAddress(types.TimeBasedRewards))

	appCodec := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, staking.AppModuleBasic{}).Codec
	keys := storetypes.NewKVStoreKeys(types.StoreKey)

	//kvStoreService := storetypes.NewKVStoreKey("mint")(registrytypes.StoreKey).(*storetypes.KVStoreKey)
	keeper := keeper.NewKeeper(appCodec, keys[types.StoreKey], s.accountKeeper, s.bankKeeper)
	s.NotNil(keeper)
}

func (s *KeeperTestSuite) TestLogger(t *testing.T) {
	s.SetupTest()

	logger := s.mintKeeper.Logger(s.ctx)
	s.NotNil(logger)
}

func (s *KeeperTestSuite) TestGetMinter(t *testing.T) {
	s.SetupTest()

	minter := s.mintKeeper.GetMinter(s.ctx)
	s.ctx.Logger().Info("Minter: %v", minter)

	s.NotNil(minter)
	s.Equal("loya", minter.BondDenom)
}

func (s *KeeperTestSuite) TestSetMinter(t *testing.T) {
	s.SetupTest()

	minter := types.NewMinter("loya")
	s.mintKeeper.SetMinter(s.ctx, minter)

	returnedMinter := s.mintKeeper.GetMinter(s.ctx)
	s.Equal(minter, returnedMinter)
}

func (s *KeeperTestSuite) TestMintCoins(t *testing.T) {
	s.SetupTest()
	coins := sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(100*1e6)))

	err := s.mintKeeper.MintCoins(s.ctx, coins)
	s.NoError(err)
}

func (s *KeeperTestSuite) TestSendCoinsToTeam(t *testing.T) {
	s.SetupTest()
	coins := sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(100*1e6)))

	err := s.mintKeeper.SendCoinsToTeam(s.ctx, coins)
	s.NoError(err)
}

func (s *KeeperTestSuite) TestInflationaryRewards(t *testing.T) {
	s.SetupTest()
	coins := sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(100*1e6)))

	err := s.mintKeeper.SendInflationaryRewards(s.ctx, coins)
	s.NoError(err)
}
