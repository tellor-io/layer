package v3_0_4_test

import (
	"context"
	"strconv"
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
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
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type AttestationSnapshotDataLegacy struct {
	ValidatorCheckpoint  []byte `protobuf:"bytes,1,rep,name=validator_checkpoint,proto3"`
	AttestationTimestamp uint64 `protobuf:"varint,2,rep,name=attestation_timestamp,proto3"`
	PrevReportTimestamp  uint64 `protobuf:"varint,3,rep,name=prev_report_timestamp,proto3"`
	NextReportTimestamp  uint64 `protobuf:"varint,4,rep,name=next_report_timestamp,proto3"`
	QueryId              []byte `protobuf:"bytes,5,rep,name=query_id,proto3"`
	Timestamp            uint64 `protobuf:"varint,6,rep,name=timestamp,proto3"`
}

var _ proto.Message = &AttestationSnapshotDataLegacy{}

func (*AttestationSnapshotDataLegacy) Reset() {}
func (m *AttestationSnapshotDataLegacy) String() string {
	return proto.CompactTextString(m)
}
func (*AttestationSnapshotDataLegacy) ProtoMessage() {}

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

func createLegacyData(t *testing.T, ctx context.Context, storeService store.KVStoreService, cdc codec.Codec) []AttestationSnapshotDataLegacy {
	t.Helper()
	// Create sample legacy data
	legacyData := []AttestationSnapshotDataLegacy{
		{
			ValidatorCheckpoint:  []byte("checkpoint1"),
			AttestationTimestamp: 1000,
			PrevReportTimestamp:  900,
			NextReportTimestamp:  1100,
			QueryId:              []byte("query1"),
			Timestamp:            950,
		},
		{
			ValidatorCheckpoint:  []byte("checkpoint2"),
			AttestationTimestamp: 2000,
			PrevReportTimestamp:  1900,
			NextReportTimestamp:  2100,
			QueryId:              []byte("query2"),
			Timestamp:            1950,
		},
	}

	// Store legacy data
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	attestStore := prefix.NewStore(store, bridgetypes.AttestSnapshotDataMapKey)

	for _, data := range legacyData {
		key := data.QueryId
		value, err := cdc.Marshal(&data)
		require.NoError(t, err)
		attestStore.Set(key, value)
	}

	return legacyData
}

func TestMigrateStore(t *testing.T) {
	// Setup
	ctx, storeService, cdc, bk := setupTest(t)
	legacyData := createLegacyData(t, ctx, storeService, cdc)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// create migrator
	m := keeper.NewMigrator(bk)
	// Run migration
	err := m.Migrate3to4(sdkCtx)
	require.NoError(t, err, "Migration should succeed")

	// Verify migration results
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	attestStore := prefix.NewStore(store, bridgetypes.AttestSnapshotDataMapKey)

	// Check each key to ensure data was properly migrated
	for _, data := range legacyData {
		key := data.QueryId

		// Ensure key exists
		hasKey := attestStore.Has(key)
		require.True(t, hasKey, "Key should exist after migration")

		// Get and unmarshal the new value
		var newData bridgetypes.AttestationSnapshotData
		err := cdc.Unmarshal(attestStore.Get(key), &newData)
		require.NoError(t, err, "Should unmarshal new data without error")

		// Verify fields were properly migrated
		require.Equal(t, data.ValidatorCheckpoint, newData.ValidatorCheckpoint)
		require.Equal(t, data.AttestationTimestamp, newData.AttestationTimestamp)
		require.Equal(t, data.PrevReportTimestamp, newData.PrevReportTimestamp)
		require.Equal(t, data.NextReportTimestamp, newData.NextReportTimestamp)
		require.Equal(t, data.QueryId, newData.QueryId)
		require.Equal(t, data.Timestamp, newData.Timestamp)

		// Verify new field was set to 0 (as specified in the migration)
		require.Equal(t, uint64(0), newData.LastConsensusTimestamp)
	}
}

func TestMigrateStoreMalformedData(t *testing.T) {
	// Setup
	ctx, storeService, _, bk := setupTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create malformed data
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	attestStore := prefix.NewStore(store, bridgetypes.AttestSnapshotDataMapKey)
	attestStore.Set([]byte("malformed"), []byte("not a valid proto"))

	// Run migration and expect panic
	require.Panics(t, func() {
		m := keeper.NewMigrator(bk)
		_ = m.Migrate3to4(sdkCtx)
	}, "Migration should panic with malformed data")
}

func BenchmarkMigrateStore(b *testing.B) {
	// Setup
	t := &testing.T{}
	ctx, storeService, cdc, bk := setupTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create large dataset
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	attestStore := prefix.NewStore(store, bridgetypes.AttestSnapshotDataMapKey)

	for i := 0; i < 1000; i++ {
		data := AttestationSnapshotDataLegacy{
			ValidatorCheckpoint:  []byte("checkpoint" + strconv.Itoa(i)),
			AttestationTimestamp: uint64(1000 + i),
			PrevReportTimestamp:  uint64(900 + i),
			NextReportTimestamp:  uint64(1100 + i),
			QueryId:              []byte("query" + strconv.Itoa(i)),
			Timestamp:            uint64(950 + i),
		}

		key := data.QueryId
		value, _ := cdc.Marshal(&data)
		attestStore.Set(key, value)
	}

	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		m := keeper.NewMigrator(bk)
		err := m.Migrate3to4(sdkCtx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
