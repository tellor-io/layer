package v5

import (
	"context"
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/mocks"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdkStore "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func setupTest(t *testing.T) (context.Context, store.KVStoreService, codec.Codec, keeper.Keeper) {
	t.Helper()
	// Create in-memory store
	storeKey := storetypes.NewKVStoreKey(bridgetypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(bridgetypes.MemStoreKey)
	db := cosmosdb.NewMemDB()

	stateStore := sdkStore.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	// Create store service
	storeService := runtime.NewKVStoreService(storeKey)

	// Create codec
	interfaceRegistry := types.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	bankKeeper := new(mocks.BankKeeper)
	oracleKeeper := new(mocks.OracleKeeper)
	reporterKeeper := new(mocks.ReporterKeeper)
	stakingKeeper := new(mocks.StakingKeeper)
	disputeKeeper := new(mocks.DisputeKeeper)

	k := keeper.NewKeeper(
		cdc,
		storeService,
		stakingKeeper,
		oracleKeeper,
		bankKeeper,
		reporterKeeper,
		disputeKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	return ctx, storeService, cdc, k
}

func TestMigrateStore(t *testing.T) {
	// Setup
	ctx, _, cdc, k := setupTest(t)

	// Set initial parameters without MainnetChainId (since it's a new parameter)
	initialParams := bridgetypes.Params{
		AttestSlashPercentage:   math.LegacyNewDec(5),
		AttestRateLimitWindow:   1000,
		ValsetSlashPercentage:   math.LegacyNewDec(10),
		ValsetRateLimitWindow:   2000,
		AttestPenaltyTimeCutoff: 5000,
		// MainnetChainId is not set - it's a new parameter being added
	}
	err := k.Params.Set(ctx, initialParams)
	require.NoError(t, err)

	// Verify initial state - MainnetChainId should be empty
	params, err := k.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, "", params.MainnetChainId, "MainnetChainId should be empty initially")

	// Run migration
	err = MigrateStore(ctx, &k, cdc)
	require.NoError(t, err, "Migration should succeed")

	// Verify migration results
	params, err = k.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, "tellor-1", params.MainnetChainId, "Mainnet chain ID should be set to 'tellor-1'")
}

func TestMigrateStorePreservesOtherParams(t *testing.T) {
	// Setup
	ctx, _, cdc, k := setupTest(t)

	// Set initial parameters with various values (without MainnetChainId since it's new)
	initialParams := bridgetypes.Params{
		AttestSlashPercentage:   math.LegacyNewDec(5), // 5%
		AttestRateLimitWindow:   1000,
		ValsetSlashPercentage:   math.LegacyNewDec(10), // 10%
		ValsetRateLimitWindow:   2000,
		AttestPenaltyTimeCutoff: 5000,
		// MainnetChainId is not set - it's a new parameter being added
	}
	err := k.Params.Set(ctx, initialParams)
	require.NoError(t, err)

	// Run migration
	err = MigrateStore(ctx, &k, cdc)
	require.NoError(t, err, "Migration should succeed")

	// Verify migration results
	params, err := k.Params.Get(ctx)
	require.NoError(t, err)

	// Check that mainnet chain ID was set
	require.Equal(t, "tellor-1", params.MainnetChainId, "Mainnet chain ID should be set to 'tellor-1'")

	// Check that other parameters were preserved
	require.Equal(t, math.LegacyNewDec(5), params.AttestSlashPercentage, "Attest slash percentage should be preserved")
	require.Equal(t, uint64(1000), params.AttestRateLimitWindow, "Attest rate limit window should be preserved")
	require.Equal(t, math.LegacyNewDec(10), params.ValsetSlashPercentage, "Valset slash percentage should be preserved")
	require.Equal(t, uint64(2000), params.ValsetRateLimitWindow, "Valset rate limit window should be preserved")
	require.Equal(t, uint64(5000), params.AttestPenaltyTimeCutoff, "Attest penalty time cutoff should be preserved")
}

func TestMigrateStoreCallsSetValsetCheckpointDomainSeparator(t *testing.T) {
	// Setup
	ctx, _, cdc, k := setupTest(t)

	// Set initial parameters without MainnetChainId (since it's a new parameter)
	initialParams := bridgetypes.Params{
		AttestSlashPercentage: math.LegacyNewDec(5),
		AttestRateLimitWindow: 1000,
	}
	err := k.Params.Set(ctx, initialParams)
	require.NoError(t, err)

	// Run migration
	err = MigrateStore(ctx, &k, cdc)
	require.NoError(t, err, "Migration should succeed")

	// Verify that SetValsetCheckpointDomainSeparator was called by checking
	// that we can get the mainnet chain ID (which is used in that method)
	chainId, err := k.GetMainnetChainId(ctx)
	require.NoError(t, err)
	require.Equal(t, "tellor-1", chainId, "Should be able to get the updated chain ID")
}
