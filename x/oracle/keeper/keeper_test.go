package keeper_test

import (
	"testing"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/mocks"
	"github.com/tellor-io/layer/x/oracle/keeper"
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
	config := sdk.GetConfig()
	accountPubKeyPrefix := app.AccountAddressPrefix + "pub"
	config.SetBech32PrefixForAccount(app.AccountAddressPrefix, accountPubKeyPrefix)

	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	rStoreKey := sdk.NewKVStoreKey(registrytypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(rStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	s.stakingKeeper = new(mocks.StakingKeeper)
	s.accountKeeper = new(mocks.AccountKeeper)
	s.distrKeeper = new(mocks.DistrKeeper)
	rmemStoreKey := storetypes.NewMemoryStoreKey(registrytypes.MemStoreKey)
	rparamsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"RegistryParams",
	)
	s.registryKeeper = *registryk.NewKeeper(
		cdc,
		rStoreKey,
		rmemStoreKey,
		rparamsSubspace,
	)

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		nil,
		nil,
		s.accountKeeper,
		nil,
		s.distrKeeper,
		s.stakingKeeper,
		s.registryKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	s.oracleKeeper = *k
	s.ctx = sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	genesisState := registrytypes.GenesisState{
		Params: registrytypes.DefaultParams(),
	}
	r.InitGenesis(s.ctx, s.registryKeeper, genesisState)
	// Initialize params
	k.SetParams(s.ctx, types.DefaultParams())
	s.msgServer = keeper.NewMsgServerImpl(*k)
	KeyTestPubAddr()
	addy, _ := sdk.AccAddressFromBech32(Addr.String())
	val, _ := stakingtypes.NewValidator(sdk.ValAddress(addy), PubKey, stakingtypes.Description{Moniker: "test"})
	val.Jailed = false
	val.Status = stakingtypes.Bonded
	val.Tokens = sdk.NewInt(1000000000000000000)
	s.stakingKeeper.On("Validator", mock.Anything, mock.Anything).Return(val, true)
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
