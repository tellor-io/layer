package keeper

import (
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmdb "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/mint/keeper"
	"github.com/tellor-io/layer/x/mint/mocks"
	"github.com/tellor-io/layer/x/mint/types"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func MintKeeper(tb testing.TB) (keeper.Keeper, *mocks.AccountKeeper, *mocks.BankKeeper, sdk.Context) {
	tb.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(tb, stateStore.LoadLatestVersion())
	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	accountKeeper := new(mocks.AccountKeeper)
	bankKeeper := new(mocks.BankKeeper)
	moduleAddr := authtypes.NewModuleAddress(types.ModuleName)
	timeBasedRewardsAddr := authtypes.NewModuleAddress(types.TimeBasedRewards)
	accountKeeper.On("GetModuleAddress", types.ModuleName).Return(moduleAddr)
	accountKeeper.On("GetModuleAddress", types.TimeBasedRewards).Return(timeBasedRewardsAddr)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		accountKeeper,
		bankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	return k, accountKeeper, bankKeeper, ctx
}
