package bridge

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/mocks"
	"github.com/tellor-io/layer/x/bridge/types"
)

func SetupBridgeApp(t *testing.T) (app AppModule, k keeper.Keeper, ctx sdk.Context) {
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

	accountKeeper := new(mocks.AccountKeeper)
	bankKeeper := new(mocks.BankKeeper)
	oracleKeeper := new(mocks.OracleKeeper)
	reporterKeeper := new(mocks.ReporterKeeper)
	stakingKeeper := new(mocks.StakingKeeper)

	k = keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		stakingKeeper,
		accountKeeper,
		oracleKeeper,
		bankKeeper,
		reporterKeeper,
	)

	app = NewAppModule(
		cdc,
		k,
		accountKeeper,
		bankKeeper,
	)

	ctx = sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	return app, k, ctx
}

func TestEndBlock(t *testing.T) {
	app, _, ctx := SetupBridgeApp(t)

	sk := new(mocks.StakingKeeper)
	sk.On("GetAllValidators", ctx).Return([]stakingtypes.Validator{}, nil)

	err := app.EndBlock(ctx)
	require.NoError(t, err)
}
