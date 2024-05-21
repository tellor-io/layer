package keeper

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
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/mocks"
	"github.com/tellor-io/layer/x/bridge/types"
)

func BridgeKeeper(t testing.TB) (keeper.Keeper, *mocks.AccountKeeper, *mocks.BankKeeper, *mocks.OracleKeeper, *mocks.ReporterKeeper, *mocks.StakingKeeper, sdk.Context) {
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

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		stakingKeeper,
		accountKeeper,
		oracleKeeper,
		bankKeeper,
		reporterKeeper,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.Params.Set(ctx, types.DefaultParams())

	return k, accountKeeper, bankKeeper, oracleKeeper, reporterKeeper, stakingKeeper, ctx
}
