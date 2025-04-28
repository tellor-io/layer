package fork_test

import (
	"context"
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
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/mocks"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

type TipperTotalData struct {
	TipperTotal math.Int
	Address     []byte
	Block       uint64
}

type TotalTipsData struct {
	TotalTips math.Int
	Block     uint64
}

type ModuleStateData struct {
	TipperTotal     []TipperTotalData       `json:"tipper_total"`
	LatestTotalTips TotalTipsData           `json:"total_tips"`
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
	k := keeper.NewKeeper(cdc, storeService, accountKeeper, bankKeeper, registryKeeper, reporterKeeper, "oracle")

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
	file, err := os.Open("dispute_module_state.json")
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

	// Verify tipper_total was migrated correctly
	for _, entry := range moduleState.TipperTotal {
		tipperTotal, err := k.TipperTotal.Get(sdkCtx, collections.Join(entry.Address, entry.Block))
		require.NoError(t, err)
		require.Equal(t, tipperTotal, entry.TipperTotal)
	}

	// Verify total_tips was migrated correctly
	totalTips, err := k.TotalTips.Get(sdkCtx, moduleState.LatestTotalTips.Block)
	require.NoError(t, err)
	require.Equal(t, totalTips, moduleState.LatestTotalTips.TotalTips)

	// Verify tipped_queries was migrated correctly
	for _, entry := range moduleState.TippedQueries {
		queryId := utils.QueryIDFromData(entry.QueryData)
		tippedQuery, err := k.Query.Get(sdkCtx, collections.Join(queryId, entry.Id))
		require.NoError(t, err)
		require.Equal(t, tippedQuery, entry)
	}
}
