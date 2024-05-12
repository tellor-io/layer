package keeper

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/mocks"
	"github.com/tellor-io/layer/x/dispute/types"
)

func DisputeKeeper(t testing.TB) (
	keeper.Keeper, *mocks.OracleKeeper, *mocks.ReporterKeeper, *mocks.AccountKeeper, *mocks.BankKeeper, sdk.Context,
) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	accountKeeper := new(mocks.AccountKeeper)
	bankKeeper := new(mocks.BankKeeper)
	oracleKeeper := new(mocks.OracleKeeper)
	reporterKeeper := new(mocks.ReporterKeeper)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		accountKeeper,
		bankKeeper,
		oracleKeeper,
		reporterKeeper,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	err := k.OpenDisputes.Set(ctx, types.OpenDisputes{Ids: make([]uint64, 0)})
	if err != nil {
		panic(err)
	}
	err = k.Params.Set(ctx, types.DefaultParams())
	if err != nil {
		panic(err)
	}
	return k, oracleKeeper, reporterKeeper, accountKeeper, bankKeeper, ctx
}
