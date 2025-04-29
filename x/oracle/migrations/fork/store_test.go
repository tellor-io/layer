package fork_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"cosmossdk.io/math"
	sdkStore "cosmossdk.io/store"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/mocks"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

type TipperTotalData struct {
	TipperTotal string `json:"tipper_total"`
	Address     []byte `json:"address"`
	Block       uint64 `json:"block"`
}

type TotalTipsData struct {
	TotalTips string `json:"total_tips"`
	Block     uint64 `json:"block"`
}

type ModuleStateData struct {
	TipperTotal     []TipperTotalData       `json:"tipper_total"`
	LatestTotalTips TotalTipsData           `json:"latest_total_tips"`
	TippedQueries   []oracletypes.QueryMeta `json:"tipped_queries"`
}

func setupTest(t *testing.T) (context.Context, store.KVStoreService, codec.Codec, keeper.Keeper) {
	t.Helper()
	// Create in-memory store
	storeKey := storetypes.NewKVStoreKey(oracletypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(oracletypes.MemStoreKey)
	db := cosmosdb.NewMemDB()

	stateStore := sdkStore.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	storeService := runtime.NewKVStoreService(storeKey)
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	accountKeeper := new(mocks.AccountKeeper)
	bankKeeper := new(mocks.BankKeeper)
	reporterKeeper := new(mocks.ReporterKeeper)
	registryKeeper := new(mocks.RegistryKeeper)
	k := keeper.NewKeeper(cdc, storeService, accountKeeper, bankKeeper, registryKeeper, reporterKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	return ctx, storeService, cdc, k
}

func TestMigrateStore(t *testing.T) {
	ctx, _, _, k := setupTest(t)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	m := keeper.NewMigrator(k)

	err := m.MigrateFork(sdkCtx)
	require.NoError(t, err)

	// Read the test data file
	file, err := os.Open("oracle_module_state.json")
	require.NoError(t, err)
	defer file.Close()

	// Read the file contents
	fileData, err := io.ReadAll(file)
	require.NoError(t, err)
	fmt.Println("Length of file data: ", len(fileData))

	// Parse the JSON data
	var moduleState ModuleStateData
	err = json.Unmarshal(fileData, &moduleState)
	require.NoError(t, err)
	fmt.Println("Module state: ", moduleState)

	// Verify tipper_total was migrated correctly
	for _, entry := range moduleState.TipperTotal {
		tipperTotal, err := k.TipperTotal.Get(sdkCtx, collections.Join(entry.Address, entry.Block))
		require.NoError(t, err)
		tipperTotalInt, ok := math.NewIntFromString(entry.TipperTotal)
		require.True(t, ok)
		require.Equal(t, tipperTotalInt, tipperTotal)
	}

	// Verify total_tips was migrated correctly
	fmt.Println("moduleState.LatestTotalTips.Block: ", moduleState.LatestTotalTips.Block)
	totalTips, err := k.TotalTips.Get(sdkCtx, moduleState.LatestTotalTips.Block)
	require.NoError(t, err)
	totalTipsFromFile, ok := math.NewIntFromString(moduleState.LatestTotalTips.TotalTips)
	require.True(t, ok)
	require.Equal(t, totalTipsFromFile, totalTips)

	// Verify tipped_queries was migrated correctly
	for _, entry := range moduleState.TippedQueries {
		fmt.Println("entry: ", entry)
		queryId := utils.QueryIDFromData(entry.QueryData)
		fmt.Println("queryId: ", hex.EncodeToString(queryId))
		fmt.Println("entry.Id: ", entry.Id)
		fmt.Println("key: ", collections.Join(queryId, entry.Id))
		tippedQuery, err := k.Query.Get(sdkCtx, collections.Join(queryId, entry.Id))
		require.NoError(t, err)
		fmt.Println("tippedQuery: ", tippedQuery)
		require.Equal(t, tippedQuery, entry)
	}
}
