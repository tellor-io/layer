package keeper_test

import (
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
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
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
	// queryClient    types.QueryClient
	msgServer types.MsgServer
}

func (s *KeeperTestSuite) SetupTest() {
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
	// require.NoError(t, stateStore.LoadLatestVersion())
	stateStore.LoadLatestVersion()
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"OracleParams",
	)
	s.stakingKeeper = new(mocks.StakingKeeper)
	s.accountKeeper = new(mocks.AccountKeeper)
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
		paramsSubspace,
		s.accountKeeper,
		nil,
		s.stakingKeeper,
		s.registryKeeper,
	)

	s.ctx = sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	genesisState := registrytypes.GenesisState{
		Params: registrytypes.DefaultParams(),
	}
	r.InitGenesis(s.ctx, s.registryKeeper, genesisState)
	// Initialize params
	k.SetParams(s.ctx, types.DefaultParams())
	s.msgServer = keeper.NewMsgServerImpl(*k)
	PrivKey, PubKey, Addr = KeyTestPubAddr()
}

func KeyTestPubAddr() (cryptotypes.PrivKey, cryptotypes.PubKey, sdk.AccAddress) {
	key := secp256k1.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}
