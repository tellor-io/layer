package keeper

import (
	"testing"

	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/mocks"
	r "github.com/tellor-io/layer/x/registry"
	rkeeper "github.com/tellor-io/layer/x/registry/keeper"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
)

func OracleKeeper(t testing.TB) (*keeper.Keeper, *mocks.StakingKeeper, *mocks.AccountKeeper, sdk.Context) {
	accountPubKeyPrefix := app.AccountAddressPrefix + "pub"
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(app.AccountAddressPrefix, accountPubKeyPrefix)

	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	rStoreKey := sdk.NewKVStoreKey(registrytypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(rStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	sk := new(mocks.StakingKeeper)
	ak := new(mocks.AccountKeeper)
	rmemStoreKey := storetypes.NewMemoryStoreKey(registrytypes.MemStoreKey)
	rparamsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"RegistryParams",
	)
	rk := rkeeper.NewKeeper(
		cdc,
		rStoreKey,
		rmemStoreKey,
		rparamsSubspace,
	)

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		ak,
		nil,
		nil,
		sk,
		rk,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	genesisState := registrytypes.GenesisState{
		Params: registrytypes.DefaultParams(),
	}
	r.InitGenesis(ctx, *rk, genesisState)
	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, sk, ak, ctx
}
