package v4_test

import (
	"context"
	"strconv"
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	v4 "github.com/tellor-io/layer/x/bridge/migrations/v4"
	"github.com/tellor-io/layer/x/bridge/mocks"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkStore "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// ValidatorCheckpointParamsLegacy represents the old ValidatorCheckpointParams struct without BlockHeight
type ValidatorCheckpointParamsLegacy struct {
	Checkpoint     []byte `protobuf:"bytes,1,opt,name=checkpoint,proto3"`
	ValsetHash     []byte `protobuf:"bytes,2,opt,name=valset_hash,json=valsetHash,proto3"`
	Timestamp      uint64 `protobuf:"varint,3,opt,name=timestamp,proto3"`
	PowerThreshold uint64 `protobuf:"varint,4,opt,name=power_threshold,json=powerThreshold,proto3"`
}

var _ proto.Message = &ValidatorCheckpointParamsLegacy{}

func (*ValidatorCheckpointParamsLegacy) Reset() {}
func (m *ValidatorCheckpointParamsLegacy) String() string {
	return proto.CompactTextString(m)
}
func (*ValidatorCheckpointParamsLegacy) ProtoMessage() {}

// ParamsLegacy represents the old empty Params struct before v4
type ParamsLegacy struct {
	// Empty struct - no parameters existed before v4
}

var _ proto.Message = &ParamsLegacy{}

func (*ParamsLegacy) Reset() {}
func (m *ParamsLegacy) String() string {
	return proto.CompactTextString(m)
}
func (*ParamsLegacy) ProtoMessage() {}

func setupTest(t *testing.T) (context.Context, store.KVStoreService, codec.Codec, keeper.Keeper) {
	t.Helper()
	// create in-memory store
	storeKey := storetypes.NewKVStoreKey(bridgetypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(bridgetypes.MemStoreKey)
	db := cosmosdb.NewMemDB()

	stateStore := sdkStore.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	// create store service
	storeService := runtime.NewKVStoreService(storeKey)

	// create codec
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

func createLegacyValidatorCheckpointParams(t *testing.T, ctx context.Context, storeService store.KVStoreService, cdc codec.Codec) []ValidatorCheckpointParamsLegacy {
	t.Helper()
	// create sample legacy ValidatorCheckpointParams data
	legacyData := []ValidatorCheckpointParamsLegacy{
		{
			Checkpoint:     []byte("checkpoint1"),
			ValsetHash:     []byte("valset_hash1"),
			Timestamp:      1000,
			PowerThreshold: 5000,
		},
		{
			Checkpoint:     []byte("checkpoint2"),
			ValsetHash:     []byte("valset_hash2"),
			Timestamp:      2000,
			PowerThreshold: 6000,
		},
		{
			Checkpoint:     []byte("checkpoint3"),
			ValsetHash:     []byte("valset_hash3"),
			Timestamp:      3000,
			PowerThreshold: 7000,
		},
	}

	// store legacy data
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	checkpointStore := prefix.NewStore(store, bridgetypes.ValidatorCheckpointParamsMapKey)

	for _, data := range legacyData {
		// use timestamp as key (uint64 -> []byte)
		key := sdk.Uint64ToBigEndian(data.Timestamp)
		value, err := cdc.Marshal(&data)
		require.NoError(t, err)
		checkpointStore.Set(key, value)
	}

	return legacyData
}

// TestEVMAddressMigration tests the migration of OperatorToEVMAddressMap to EVMToOperatorAddressMap
func TestEVMAddressMigration(t *testing.T) {
	ctx, storeService, cdc, _ := setupTest(t)

	// Generate valid validator addresses for testing
	pk1 := secp256k1.GenPrivKey()
	pk2 := secp256k1.GenPrivKey()
	pk3 := secp256k1.GenPrivKey()

	addr1 := sdk.AccAddress(pk1.PubKey().Address())
	addr2 := sdk.AccAddress(pk2.PubKey().Address())
	addr3 := sdk.AccAddress(pk3.PubKey().Address())

	valAddr1 := sdk.ValAddress(addr1)
	valAddr2 := sdk.ValAddress(addr2)
	valAddr3 := sdk.ValAddress(addr3)

	evmHex1 := "1234567890abcdef1234567890abcdef12345678"
	evmHex2 := "abcdef1234567890abcdef1234567890abcdef12"
	evmHex3 := "fedcba0987654321fedcba0987654321fedcba09"

	// Test data: operator addresses and corresponding EVM addresses
	testData := []struct {
		operatorAddr string
		evmAddrHex   string
		evmAddrBytes []byte
	}{
		{
			operatorAddr: valAddr1.String(),
			evmAddrHex:   evmHex1,
			evmAddrBytes: common.FromHex(evmHex1),
		},
		{
			operatorAddr: valAddr2.String(),
			evmAddrHex:   evmHex2,
			evmAddrBytes: common.FromHex(evmHex2),
		},
		{
			operatorAddr: valAddr3.String(),
			evmAddrHex:   evmHex3,
			evmAddrBytes: common.FromHex(evmHex3),
		},
	}

	// Create and store test data in OperatorToEVMAddressMap
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	operatorToEVMStore := prefix.NewStore(store, bridgetypes.OperatorToEVMAddressMapKey)

	for _, data := range testData {
		evmAddr := bridgetypes.EVMAddress{
			EVMAddress: data.evmAddrBytes,
		}
		value, err := cdc.Marshal(&evmAddr)
		require.NoError(t, err)

		operatorToEVMStore.Set([]byte(data.operatorAddr), value)
	}

	// Run migration
	err := v4.MigrateStore(ctx, storeService, cdc)
	require.NoError(t, err, "Migration should succeed")

	// Verify EVMToOperatorAddressMap was populated correctly
	evmToOperatorStore := prefix.NewStore(store, bridgetypes.EVMToOperatorAddressMapKey)

	for i, data := range testData {
		// Key should be the hex string (without 0x prefix)
		evmKey := []byte(data.evmAddrHex)

		// Verify the reverse mapping exists
		require.True(t, evmToOperatorStore.Has(evmKey),
			"Reverse mapping should exist for EVM address %s", data.evmAddrHex)

		// Get and verify the value
		value := evmToOperatorStore.Get(evmKey)
		require.NotNil(t, value, "Value should not be nil for EVM address %s", data.evmAddrHex)

		var operatorAddr bridgetypes.OperatorAddress
		err := cdc.Unmarshal(value, &operatorAddr)
		require.NoError(t, err, "Should unmarshal operator address")

		// Verify the operator address matches
		expectedValAddr, err := sdk.ValAddressFromBech32(data.operatorAddr)
		require.NoError(t, err)
		require.Equal(t, expectedValAddr.Bytes(), operatorAddr.OperatorAddress,
			"Operator address should match for entry %d", i)
	}
}

// TestEVMAddressMigrationWithKeeper tests the migration through the keeper interface
func TestEVMAddressMigrationWithKeeper(t *testing.T) {
	ctx, storeService, cdc, bk := setupTest(t)

	// Generate valid validator address
	pk := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(pk.PubKey().Address())
	valAddr := sdk.ValAddress(addr)

	operatorAddr := valAddr.String()
	evmAddrBytes := common.FromHex("0x1234567890abcdef1234567890abcdef12345678")
	evmAddrHex := "1234567890abcdef1234567890abcdef12345678"

	// Create initial mapping using the store directly (simulating pre-migration state)
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	operatorToEVMStore := prefix.NewStore(store, bridgetypes.OperatorToEVMAddressMapKey)

	evmAddr := bridgetypes.EVMAddress{EVMAddress: evmAddrBytes}
	value, err := cdc.Marshal(&evmAddr)
	require.NoError(t, err)
	operatorToEVMStore.Set([]byte(operatorAddr), value)

	// Run migration through keeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	m := keeper.NewMigrator(bk)
	err = m.Migrate3to4(sdkCtx)
	require.NoError(t, err)

	// Test that the keeper can now access the reverse mapping
	// verify original mapping still exists
	storedEVMAddr, err := bk.OperatorToEVMAddressMap.Get(ctx, operatorAddr)
	require.NoError(t, err)
	require.Equal(t, evmAddrBytes, storedEVMAddr.EVMAddress)

	// Verify reverse mapping exists and is accessible via raw store access
	// (since the keeper doesn't have a direct method for EVMToOperatorAddressMap access in the test)
	evmToOperatorStore := prefix.NewStore(store, bridgetypes.EVMToOperatorAddressMapKey)
	evmKey := []byte(evmAddrHex)
	require.True(t, evmToOperatorStore.Has(evmKey))

	value = evmToOperatorStore.Get(evmKey)
	var operatorAddrResult bridgetypes.OperatorAddress
	err = cdc.Unmarshal(value, &operatorAddrResult)
	require.NoError(t, err)

	expectedValAddr, err := sdk.ValAddressFromBech32(operatorAddr)
	require.NoError(t, err)
	require.Equal(t, expectedValAddr.Bytes(), operatorAddrResult.OperatorAddress)

	storedOperatorAddr, err := bk.EVMToOperatorAddressMap.Get(ctx, evmAddrHex)
	require.NoError(t, err)
	require.Equal(t, operatorAddr, sdk.ValAddress(storedOperatorAddr.OperatorAddress).String())
}

// TestEVMAddressMigrationEmpty tests migration with empty stores
func TestEVMAddressMigrationEmpty(t *testing.T) {
	ctx, storeService, cdc, _ := setupTest(t)

	// Run migration on empty store
	err := v4.MigrateStore(ctx, storeService, cdc)
	require.NoError(t, err, "Migration should succeed even with empty OperatorToEVMAddressMap")

	// Verify EVMToOperatorAddressMap is still empty
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	evmToOperatorStore := prefix.NewStore(store, bridgetypes.EVMToOperatorAddressMapKey)

	iter := evmToOperatorStore.Iterator(nil, nil)
	defer iter.Close()
	require.False(t, iter.Valid(), "EVMToOperatorAddressMap should be empty")
}

func TestMigrateStore(t *testing.T) {
	// setup
	ctx, storeService, cdc, _ := setupTest(t)
	legacyData := createLegacyValidatorCheckpointParams(t, ctx, storeService, cdc)

	// run migration directly on store
	err := v4.MigrateStore(ctx, storeService, cdc)
	require.NoError(t, err, "Migration should succeed")

	// verify ValidatorCheckpointParams migration results
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	checkpointStore := prefix.NewStore(store, bridgetypes.ValidatorCheckpointParamsMapKey)

	// check each key to ensure data was properly migrated
	for _, data := range legacyData {
		key := sdk.Uint64ToBigEndian(data.Timestamp)

		// ensure key exists
		hasKey := checkpointStore.Has(key)
		require.True(t, hasKey, "Key should exist after migration for timestamp %d", data.Timestamp)

		// get and unmarshal the new value
		var newParams bridgetypes.ValidatorCheckpointParams
		err := cdc.Unmarshal(checkpointStore.Get(key), &newParams)
		require.NoError(t, err, "Should unmarshal new ValidatorCheckpointParams without error")

		// verify fields were properly migrated
		require.Equal(t, data.Checkpoint, newParams.Checkpoint, "Checkpoint should match")
		require.Equal(t, data.ValsetHash, newParams.ValsetHash, "ValsetHash should match")
		require.Equal(t, data.Timestamp, newParams.Timestamp, "Timestamp should match")
		require.Equal(t, data.PowerThreshold, newParams.PowerThreshold, "PowerThreshold should match")

		// verify new field was set to 0 (as specified in the migration)
		require.Equal(t, uint64(0), newParams.BlockHeight, "BlockHeight should be set to 0 for existing entries")
	}
}

func TestMigrateStoreEmpty(t *testing.T) {
	// setup with no existing data
	ctx, storeService, cdc, _ := setupTest(t)

	// run migration on empty store
	err := v4.MigrateStore(ctx, storeService, cdc)
	require.NoError(t, err, "Migration should succeed even with empty store")
}

func TestMigrateStoreMalformedData(t *testing.T) {
	// setup
	ctx, storeService, cdc, _ := setupTest(t)

	// create malformed data
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	checkpointStore := prefix.NewStore(store, bridgetypes.ValidatorCheckpointParamsMapKey)
	checkpointStore.Set([]byte("malformed"), []byte("not a valid proto"))

	// run migration and expect panic
	require.Panics(t, func() {
		_ = v4.MigrateStore(ctx, storeService, cdc)
	}, "Migration should panic with malformed data")
}

func BenchmarkMigrateStore(b *testing.B) {
	// setup
	t := &testing.T{}
	ctx, storeService, cdc, _ := setupTest(t)

	// create large dataset
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	checkpointStore := prefix.NewStore(store, bridgetypes.ValidatorCheckpointParamsMapKey)

	for i := 0; i < 1000; i++ {
		data := ValidatorCheckpointParamsLegacy{
			Checkpoint:     []byte("checkpoint" + strconv.Itoa(i)),
			ValsetHash:     []byte("valset_hash" + strconv.Itoa(i)),
			Timestamp:      uint64(1000 + i),
			PowerThreshold: uint64(5000 + i),
		}

		key := sdk.Uint64ToBigEndian(data.Timestamp)
		value, _ := cdc.Marshal(&data)
		checkpointStore.Set(key, value)
	}

	b.ResetTimer()

	// run benchmark
	for i := 0; i < b.N; i++ {
		err := v4.MigrateStore(ctx, storeService, cdc)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestMigrateStoreIntegration(t *testing.T) {
	// setup
	ctx, storeService, cdc, bk := setupTest(t)
	legacyData := createLegacyValidatorCheckpointParams(t, ctx, storeService, cdc)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// run migration
	m := keeper.NewMigrator(bk)
	err := m.Migrate3to4(sdkCtx)
	require.NoError(t, err, "Migration should succeed")

	// test that the keeper can now read the migrated data properly
	for _, data := range legacyData {
		// use the keeper's ValidatorCheckpointParamsMap to read the migrated data
		params, err := bk.ValidatorCheckpointParamsMap.Get(ctx, data.Timestamp)
		require.NoError(t, err, "Should be able to read migrated data via keeper")

		// verify all fields
		require.Equal(t, data.Checkpoint, params.Checkpoint)
		require.Equal(t, data.ValsetHash, params.ValsetHash)
		require.Equal(t, data.Timestamp, params.Timestamp)
		require.Equal(t, data.PowerThreshold, params.PowerThreshold)
		require.Equal(t, uint64(0), params.BlockHeight, "BlockHeight should be 0 for migrated data")
	}

	// test that new data can be written with BlockHeight
	newParams := bridgetypes.ValidatorCheckpointParams{
		Checkpoint:     []byte("new_checkpoint"),
		ValsetHash:     []byte("new_valset_hash"),
		Timestamp:      9999,
		PowerThreshold: 8000,
		BlockHeight:    100, // new field with actual value
	}

	err = bk.ValidatorCheckpointParamsMap.Set(ctx, newParams.Timestamp, newParams)
	require.NoError(t, err, "Should be able to write new data with BlockHeight")

	// verify the new data can be read back
	retrievedParams, err := bk.ValidatorCheckpointParamsMap.Get(ctx, newParams.Timestamp)
	require.NoError(t, err, "Should be able to read new data")
	require.Equal(t, newParams.BlockHeight, retrievedParams.BlockHeight, "BlockHeight should be preserved for new data")
}

func TestLegacyParamsMigration(t *testing.T) {
	// Test the migration from legacy empty params to new populated params
	ctx, storeService, cdc, bk := setupTest(t)

	// 1. Create and store legacy empty params (simulating current chain state)
	legacyParams := ParamsLegacy{}
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	paramsKey := bridgetypes.ParamsKey.Bytes()
	legacyValue, err := cdc.Marshal(&legacyParams)
	require.NoError(t, err)
	store.Set(paramsKey, legacyValue)

	// 2. Verify legacy params are stored correctly (empty struct)
	storedLegacyValue := store.Get(paramsKey)
	require.NotNil(t, storedLegacyValue, "Legacy params should be stored")
	var retrievedLegacy ParamsLegacy
	err = cdc.Unmarshal(storedLegacyValue, &retrievedLegacy)
	require.NoError(t, err, "Should be able to unmarshal legacy params")

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 3. Run migration
	m := keeper.NewMigrator(bk)
	err = m.Migrate3to4(sdkCtx)
	require.NoError(t, err, "Migration should succeed")

	// 4. Verify that new default parameters were set via Collections API
	migratedParams, err := bk.Params.Get(ctx)
	require.NoError(t, err, "Should be able to read migrated params")

	// 5. Check all default parameters are set correctly
	expectedParams := bridgetypes.DefaultParams()
	require.Equal(t, expectedParams.AttestSlashPercentage, migratedParams.AttestSlashPercentage)
	require.Equal(t, expectedParams.AttestRateLimitWindow, migratedParams.AttestRateLimitWindow)
	require.Equal(t, expectedParams.ValsetSlashPercentage, migratedParams.ValsetSlashPercentage)
	require.Equal(t, expectedParams.ValsetRateLimitWindow, migratedParams.ValsetRateLimitWindow)
	require.Equal(t, expectedParams.AttestPenaltyTimeCutoff, migratedParams.AttestPenaltyTimeCutoff)

	// 6. Verify runtime KVStore reading method
	storedNewValue := store.Get(paramsKey)
	require.NotNil(t, storedNewValue, "New params should be stored")
	var retrievedNew bridgetypes.Params
	err = cdc.Unmarshal(storedNewValue, &retrievedNew)
	require.NoError(t, err, "Should be able to unmarshal new params")
	require.Equal(t, bridgetypes.DefaultParams(), retrievedNew)
}

func TestFullMigration(t *testing.T) {
	// Test the complete migration including both ValidatorCheckpointParams and Params
	ctx, storeService, cdc, bk := setupTest(t)
	legacyData := createLegacyValidatorCheckpointParams(t, ctx, storeService, cdc)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Run full migration via keeper
	m := keeper.NewMigrator(bk)
	err := m.Migrate3to4(sdkCtx)
	require.NoError(t, err, "Full migration should succeed")

	// Verify ValidatorCheckpointParams migration
	for _, data := range legacyData {
		params, err := bk.ValidatorCheckpointParamsMap.Get(ctx, data.Timestamp)
		require.NoError(t, err, "Should be able to read migrated checkpoint data")

		require.Equal(t, data.Checkpoint, params.Checkpoint)
		require.Equal(t, data.ValsetHash, params.ValsetHash)
		require.Equal(t, data.Timestamp, params.Timestamp)
		require.Equal(t, data.PowerThreshold, params.PowerThreshold)
		require.Equal(t, uint64(0), params.BlockHeight, "BlockHeight should be 0 for migrated data")
	}

	// Verify Params migration via Collections API
	migratedParams, err := bk.Params.Get(ctx)
	require.NoError(t, err, "Should be able to read migrated params")

	expectedParams := bridgetypes.DefaultParams()
	require.Equal(t, expectedParams.AttestSlashPercentage, migratedParams.AttestSlashPercentage)
	require.Equal(t, expectedParams.AttestRateLimitWindow, migratedParams.AttestRateLimitWindow)
	require.Equal(t, expectedParams.ValsetSlashPercentage, migratedParams.ValsetSlashPercentage)
	require.Equal(t, expectedParams.ValsetRateLimitWindow, migratedParams.ValsetRateLimitWindow)
	require.Equal(t, expectedParams.AttestPenaltyTimeCutoff, migratedParams.AttestPenaltyTimeCutoff)
}
