package reporter

import (
	"testing"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func SetupBridgeApp(t *testing.T) (AppModule, keeper.Keeper, *mocks.AccountKeeper, *mocks.BankKeeper, *mocks.RegistryKeeper, sdk.Context, types.MsgServer) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := cosmosdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	ak := new(mocks.AccountKeeper)
	bk := new(mocks.BankKeeper)
	rk := new(mocks.RegistryKeeper)
	sk := new(mocks.StakingKeeper)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority,
		ak,
		sk,
		bk,
		rk,
	)

	app := NewAppModule(
		cdc,
		k,
		ak,
		bk,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	require.NoError(t, k.Params.Set(ctx, types.DefaultParams()))

	msgServer := keeper.NewMsgServerImpl(k)

	return app, k, ak, bk, rk, ctx, msgServer
}

func TestNewAppModuleBasic(t *testing.T) {
	require := require.New(t)

	appCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	am := NewAppModuleBasic(appCodec)
	require.NotNil(am)
}

func TestName(t *testing.T) {
	require := require.New(t)

	appCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	am := NewAppModuleBasic(appCodec)
	require.Equal("reporter", am.Name())
}

func TestNewAppModule(t *testing.T) {
	require := require.New(t)

	am, k, ak, bk, rk, ctx, msgServer := SetupBridgeApp(t)
	require.NotNil(am)
	require.NotNil(k)
	require.NotNil(ak)
	require.NotNil(bk)
	require.NotNil(rk)
	require.NotNil(ctx)
	require.NotNil(msgServer)
}

func TestEndBlock(t *testing.T) {
	require := require.New(t)

	am, k, _, _, _, ctx, _ := SetupBridgeApp(t)

	expiration := ctx.BlockTime().Add(time.Hour)
	require.NoError(k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: &expiration,
		Amount:     math.NewInt(100 * 1e6),
	}))
	err := am.EndBlock(ctx)
	require.NoError(err)

	tracker, err := k.Tracker.Get(ctx)
	require.NoError(err)
	require.Equal(tracker, types.StakeTracker{
		Expiration: &expiration,
		Amount:     math.NewInt(100 * 1e6),
	})

	expiration = expiration.Add(time.Hour)
	require.NoError(k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: &expiration,
		Amount:     math.NewInt(200 * 1e6),
	}))
	err = am.EndBlock(ctx)
	require.NoError(err)

	tracker, err = k.Tracker.Get(ctx)
	require.NoError(err)
	require.Equal(tracker, types.StakeTracker{
		Expiration: &expiration,
		Amount:     math.NewInt(200 * 1e6),
	})
}
