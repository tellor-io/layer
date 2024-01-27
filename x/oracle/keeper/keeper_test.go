package keeper_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/mocks"
	"github.com/tellor-io/layer/x/oracle/types"
	r "github.com/tellor-io/layer/x/registry"
	registryk "github.com/tellor-io/layer/x/registry/keeper"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
)

var (
	PrivKey cryptotypes.PrivKey
	PubKey  cryptotypes.PubKey
	Addr    sdk.AccAddress
)

type KeeperTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	oracleKeeper   keeper.Keeper
	registryKeeper registryk.Keeper
	stakingKeeper  *mocks.StakingKeeper
	accountKeeper  *mocks.AccountKeeper
	distrKeeper    *mocks.DistrKeeper
	queryClient    types.QueryClient
	msgServer      types.MsgServer
}

func (s *KeeperTestSuite) SetupTest() {
	require := s.Require()
	config.SetupConfig()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	rStoreKey := storetypes.NewKVStoreKey(registrytypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())

	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(rStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	s.stakingKeeper = new(mocks.StakingKeeper)
	s.accountKeeper = new(mocks.AccountKeeper)
	s.distrKeeper = new(mocks.DistrKeeper)

	s.registryKeeper = registryk.NewKeeper(
		cdc,
		runtime.NewKVStoreService(rStoreKey),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	s.oracleKeeper = keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		s.accountKeeper,
		nil,
		s.distrKeeper,
		s.stakingKeeper,
		s.registryKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	s.ctx = sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	genesisState := registrytypes.GenesisState{
		Params:   registrytypes.DefaultParams(),
		Dataspec: registrytypes.GenesisDataSpec(),
	}
	r.InitGenesis(s.ctx, s.registryKeeper, genesisState)
	// Initialize params
	s.oracleKeeper.SetParams(s.ctx, types.DefaultParams())
	s.msgServer = keeper.NewMsgServerImpl(s.oracleKeeper)
	KeyTestPubAddr()
	val, _ := stakingtypes.NewValidator(Addr.String(), PubKey, stakingtypes.Description{Moniker: "test"})
	val.Jailed = false
	val.Status = stakingtypes.Bonded
	val.Tokens = math.NewInt(1000000000000000000)
	s.stakingKeeper.On("Validator", mock.Anything, mock.Anything).Return(val, nil)
	account := authtypes.NewBaseAccount(Addr, PubKey, 0, 0)
	s.accountKeeper.On("GetAccount", mock.Anything, mock.Anything).Return(account, nil)
}

func KeyTestPubAddr() {
	PrivKey = secp256k1.GenPrivKey()
	PubKey = PrivKey.PubKey()
	Addr = sdk.AccAddress(PubKey.Address())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
