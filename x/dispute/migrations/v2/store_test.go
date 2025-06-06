package v2_test

// import (
// 	"context"
// 	"strconv"
// 	"testing"
// 	"time"

// 	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
// 	cosmosdb "github.com/cosmos/cosmos-db"
// 	"github.com/stretchr/testify/require"
// 	"github.com/tellor-io/layer/x/dispute/keeper"
// 	"github.com/tellor-io/layer/x/dispute/mocks"
// 	disputetypes "github.com/tellor-io/layer/x/dispute/types"
// 	oracletypes "github.com/tellor-io/layer/x/oracle/types"

// 	"cosmossdk.io/collections"
// 	"cosmossdk.io/core/store"
// 	"cosmossdk.io/log"
// 	"cosmossdk.io/math"
// 	sdkStore "cosmossdk.io/store"
// 	"cosmossdk.io/store/metrics"
// 	"cosmossdk.io/store/prefix"
// 	storetypes "cosmossdk.io/store/types"

// 	"github.com/cosmos/cosmos-sdk/codec"
// 	"github.com/cosmos/cosmos-sdk/codec/types"
// 	"github.com/cosmos/cosmos-sdk/runtime"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// )

// func setupTest(t *testing.T) (context.Context, store.KVStoreService, codec.Codec, keeper.Keeper) {
// 	t.Helper()
// 	// Create in-memory store
// 	storeKey := storetypes.NewKVStoreKey(disputetypes.StoreKey)
// 	db := cosmosdb.NewMemDB()

// 	stateStore := sdkStore.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
// 	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
// 	require.NoError(t, stateStore.LoadLatestVersion())

// 	// Create store service
// 	storeService := runtime.NewKVStoreService(storeKey)

// 	// Create codec
// 	interfaceRegistry := types.NewInterfaceRegistry()
// 	cdc := codec.NewProtoCodec(interfaceRegistry)

// 	bankKeeper := new(mocks.BankKeeper)
// 	oracleKeeper := new(mocks.OracleKeeper)
// 	reporterKeeper := new(mocks.ReporterKeeper)
// 	accountKeeper := new(mocks.AccountKeeper)

// 	k := keeper.NewKeeper(
// 		cdc,
// 		storeService,
// 		accountKeeper,
// 		bankKeeper,
// 		oracleKeeper,
// 		reporterKeeper,
// 	)

// 	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

// 	return ctx, storeService, cdc, k
// }

// func createLegacyData(t *testing.T, ctx context.Context, storeService store.KVStoreService, cdc codec.Codec) []disputetypes.Dispute {
// 	t.Helper()
// 	// Create sample dispute data
// 	disputes := []disputetypes.Dispute{
// 		{
// 			HashId:            []byte("Ua/iRdqdo2WmTK6PJLXL960x9hHZdYg/R/g76lnis4k="),
// 			DisputeId:         1,
// 			DisputeCategory:   disputetypes.DisputeCategory(1), // WARNING
// 			DisputeFee:        math.NewInt(50000),
// 			DisputeStatus:     disputetypes.DisputeStatus(2), // RESOLVED
// 			DisputeStartTime:  time.Date(2025, 3, 24, 15, 0, 25, 898659018, time.UTC),
// 			DisputeEndTime:    time.Date(2025, 3, 27, 15, 0, 25, 898659018, time.UTC),
// 			DisputeStartBlock: 220201,
// 			DisputeRound:      1,
// 			SlashAmount:       math.NewInt(50000),
// 			BurnAmount:        math.NewInt(2500),
// 			InitialEvidence: oracletypes.MicroReport{
// 				AggregateMethod: "weighted-median",
// 				BlockNumber:     218261,
// 				MetaId:          109136,
// 				Power:           uint64(5),
// 				QueryId:         []byte("DRKtSRkxY7u+/05tuClM7SP/hgU1n9ZmeZ1OJaOqDjo="),
// 				QueryType:       "AmpleforthCustomSpotPrice",
// 				Reporter:        "tellor17gc67q05d5rgsz9caznm0s7s5eazwg2e3fkk8e",
// 				Timestamp:       time.Date(2025, 3, 24, 14, 11, 35, 566750439, time.UTC),
// 				Value:           "00000000000000000000000000000000000000000000000010eb2f7f19c6e1ed",
// 			},
// 			FeeTotal:         math.NewInt(50000),
// 			PrevDisputeIds:   []uint64{1},
// 			BlockNumber:      220201,
// 			Open:             false,
// 			VoterReward:      math.NewInt(1250),
// 			PendingExecution: false,
// 		},
// 		{
// 			HashId:            []byte("VnysCz6tGz4e0UiahRFpuDljJY4V3h7k47qMQVsfNBs="),
// 			DisputeId:         2,
// 			DisputeCategory:   disputetypes.DisputeCategory(1), // WARNING
// 			DisputeFee:        math.NewInt(2130000),
// 			DisputeStatus:     disputetypes.DisputeStatus(2), // RESOLVED
// 			DisputeStartTime:  time.Date(2025, 4, 8, 17, 19, 56, 96798398, time.UTC),
// 			DisputeEndTime:    time.Date(2025, 4, 11, 17, 19, 56, 96798398, time.UTC),
// 			DisputeStartBlock: 1058447,
// 			DisputeRound:      1,
// 			SlashAmount:       math.NewInt(4153500),
// 			BurnAmount:        math.NewInt(106500),
// 			InitialEvidence: oracletypes.MicroReport{
// 				AggregateMethod: "weighted-median",
// 				BlockNumber:     1058420,
// 				MetaId:          529293,
// 				Power:           uint64(213),
// 				QueryId:         []byte("g6fz1IeGrCZnUDph6MQVQ47Ski64aikG5O5m2aLOSZI="),
// 				QueryType:       "SpotPrice",
// 				Reporter:        "tellor1m6tuy2ek06yptcyhtd3t7h54uvv3wxgvmjckt9",
// 				Timestamp:       time.Date(2025, 4, 8, 17, 19, 10, 752174575, time.UTC),
// 				Value:           "00000000000000000000000000000000000000000000004fc3c6e8304dc40000",
// 			},
// 			FeeTotal:         math.NewInt(2130000),
// 			PrevDisputeIds:   []uint64{2},
// 			BlockNumber:      1058447,
// 			Open:             true,
// 			VoterReward:      math.NewInt(53250),
// 			PendingExecution: false,
// 		},
// 		{
// 			HashId:            []byte("7dk4AGHoUM8qN92cx3QkXrOpYNpqZ8d0cwp2jJpX8A0="),
// 			DisputeId:         3,
// 			DisputeCategory:   disputetypes.DisputeCategory(1), // WARNING
// 			DisputeFee:        math.NewInt(750000),
// 			DisputeStatus:     disputetypes.DisputeStatus(2), // RESOLVED
// 			DisputeStartTime:  time.Date(2025, 4, 9, 17, 20, 54, 293150565, time.UTC),
// 			DisputeEndTime:    time.Date(2025, 4, 12, 17, 20, 54, 293150565, time.UTC),
// 			DisputeStartBlock: 1109642,
// 			DisputeRound:      1,
// 			SlashAmount:       math.NewInt(1462500),
// 			BurnAmount:        math.NewInt(37500),
// 			InitialEvidence: oracletypes.MicroReport{
// 				AggregateMethod: "weighted-median",
// 				BlockNumber:     1109402,
// 				MetaId:          554786,
// 				Power:           uint64(75),
// 				QueryId:         []byte("g6fz1IeGrCZnUDph6MQVQ47Ski64aikG5O5m2aLOSZI="),
// 				QueryType:       "SpotPrice",
// 				Reporter:        "tellor1mkl462wk4q64825ce5escjcg0l3p5j2n3hu8h9",
// 				Timestamp:       time.Date(2025, 4, 9, 17, 14, 9, 724152861, time.UTC),
// 				Value:           "00000000000000000000000000000000000000000000005181cf1fe756d20000",
// 			},
// 			FeeTotal:         math.NewInt(750000),
// 			PrevDisputeIds:   []uint64{3},
// 			BlockNumber:      1109642,
// 			Open:             true,
// 			VoterReward:      math.NewInt(18750),
// 			PendingExecution: false,
// 		},
// 		{
// 			HashId:            []byte("YDS3IimZyqJZrkBXQcqyuqXnfojWMlMZYvx8J/LkoXw="),
// 			DisputeId:         4,
// 			DisputeCategory:   disputetypes.DisputeCategory(1), // WARNING
// 			DisputeFee:        math.NewInt(10090000),
// 			DisputeStatus:     disputetypes.DisputeStatus(0), // PREVOTE
// 			DisputeStartTime:  time.Date(2025, 4, 15, 17, 17, 22, 999578927, time.UTC),
// 			DisputeEndTime:    time.Date(2025, 4, 16, 17, 17, 22, 999578927, time.UTC),
// 			DisputeStartBlock: 1403347,
// 			DisputeRound:      1,
// 			SlashAmount:       math.NewInt(10090000),
// 			BurnAmount:        math.NewInt(504500),
// 			InitialEvidence: oracletypes.MicroReport{
// 				AggregateMethod: "weighted-median",
// 				BlockNumber:     1403176,
// 				MetaId:          701681,
// 				Power:           uint64(1009),
// 				QueryId:         []byte("pvAT7iNoBIJ7d2ltNQ6fCsPoeTKPKjAh1HOgt3iteKw="),
// 				QueryType:       "SpotPrice",
// 				Reporter:        "tellor1gkstv3n4cpmdskh4m8mhr2mlljf540sqsm7gnp",
// 				Timestamp:       time.Date(2025, 4, 15, 17, 12, 12, 643158828, time.UTC),
// 				Value:           "0000000000000000000000000000000000000000000011fd8ba8e0d3c5740000",
// 			},
// 			FeeTotal:         math.NewInt(1000000),
// 			PrevDisputeIds:   []uint64{4},
// 			BlockNumber:      1403347,
// 			Open:             true,
// 			VoterReward:      math.NewInt(0),
// 			PendingExecution: false,
// 		},
// 	}

// 	// Store dispute data
// 	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
// 	disputeStore := prefix.NewStore(store, disputetypes.DisputesPrefix)

// 	for _, dispute := range disputes {
// 		key := make([]byte, 8)
// 		_, err := collections.Uint64Key.Encode(key, dispute.DisputeId)
// 		require.NoError(t, err)
// 		value, err := cdc.Marshal(&dispute)
// 		require.NoError(t, err)
// 		disputeStore.Set(key, value)
// 	}

// 	return disputes
// }

// func TestMigrateStore(t *testing.T) {
// 	// Setup
// 	ctx, storeService, cdc, bk := setupTest(t)
// 	legacyData := createLegacyData(t, ctx, storeService, cdc)

// 	sdkCtx := sdk.UnwrapSDKContext(ctx)
// 	// create migrator
// 	m := keeper.NewMigrator(bk)
// 	// Run migration
// 	err := m.Migrate1to2(sdkCtx)
// 	require.NoError(t, err, "Migration should succeed")

// 	// Verify migration results
// 	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
// 	disputeStore := prefix.NewStore(store, disputetypes.DisputesPrefix)

// 	// Check each key to ensure data was properly migrated
// 	for _, data := range legacyData {
// 		if data.DisputeId == 2 || data.DisputeId == 3 {
// 			data.Open = false
// 		}
// 		key := make([]byte, 8)
// 		_, err := collections.Uint64Key.Encode(key, data.DisputeId)
// 		require.NoError(t, err)
// 		// Ensure key exists
// 		hasKey := disputeStore.Has(key)
// 		require.True(t, hasKey, "Key should exist after migration")

// 		// Get and unmarshal the new value
// 		var newData disputetypes.Dispute
// 		err = cdc.Unmarshal(disputeStore.Get(key), &newData)
// 		require.NoError(t, err, "Should unmarshal new data without error")

// 		// Verify fields were properly migrated
// 		require.Equal(t, data.HashId, newData.HashId)
// 		require.Equal(t, data.DisputeId, newData.DisputeId)
// 		require.Equal(t, data.DisputeCategory, newData.DisputeCategory)
// 		require.Equal(t, data.DisputeFee, newData.DisputeFee)
// 		require.Equal(t, data.DisputeStatus, newData.DisputeStatus)
// 		require.Equal(t, data.DisputeStartTime, newData.DisputeStartTime)
// 		require.Equal(t, data.DisputeEndTime, newData.DisputeEndTime)
// 		require.Equal(t, data.DisputeStartBlock, newData.DisputeStartBlock)
// 		require.Equal(t, data.DisputeRound, newData.DisputeRound)
// 		require.Equal(t, data.SlashAmount, newData.SlashAmount)
// 		require.Equal(t, data.BurnAmount, newData.BurnAmount)
// 		require.Equal(t, data.InitialEvidence, newData.InitialEvidence)
// 		require.Equal(t, data.FeeTotal, newData.FeeTotal)
// 		require.Equal(t, data.PrevDisputeIds, newData.PrevDisputeIds)
// 		require.Equal(t, data.BlockNumber, newData.BlockNumber)
// 		require.Equal(t, data.Open, newData.Open)
// 		require.Equal(t, data.VoterReward, newData.VoterReward)
// 		require.Equal(t, data.PendingExecution, newData.PendingExecution)
// 	}
// }

// func TestMigrateStoreMalformedData(t *testing.T) {
// 	// Setup
// 	ctx, storeService, _, bk := setupTest(t)
// 	sdkCtx := sdk.UnwrapSDKContext(ctx)

// 	// Create malformed data
// 	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
// 	disputeStore := prefix.NewStore(store, disputetypes.DisputesPrefix)
// 	key := make([]byte, 8)
// 	_, err := collections.Uint64Key.Encode(key, 2)
// 	require.NoError(t, err)
// 	disputeStore.Set(key, []byte("not a valid proto"))

// 	m := keeper.NewMigrator(bk)
// 	// Run migration and expect panic
// 	require.Error(t, m.Migrate1to2(sdkCtx), "Migration should error with malformed data")
// }

// func BenchmarkMigrateStore(b *testing.B) {
// 	// Setup
// 	t := &testing.T{}
// 	ctx, storeService, cdc, bk := setupTest(t)
// 	sdkCtx := sdk.UnwrapSDKContext(ctx)

// 	// Create large dataset
// 	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
// 	attestStore := prefix.NewStore(store, disputetypes.DisputesPrefix)

// 	for i := 0; i < 1000; i++ {
// 		data := disputetypes.Dispute{
// 			HashId:            []byte("checkpoint" + strconv.Itoa(i)),
// 			DisputeId:         uint64(i),
// 			DisputeCategory:   disputetypes.DisputeCategory(1), // WARNING
// 			DisputeFee:        math.NewInt(50000),
// 			DisputeStatus:     disputetypes.DisputeStatus(2), // RESOLVED
// 			DisputeStartTime:  time.Date(2025, 3, 24, 15, 0, 25, 898659018, time.UTC),
// 			DisputeEndTime:    time.Date(2025, 3, 27, 15, 0, 25, 898659018, time.UTC),
// 			DisputeStartBlock: 220201,
// 			DisputeRound:      1,
// 			SlashAmount:       math.NewInt(50000),
// 			BurnAmount:        math.NewInt(2500),
// 			InitialEvidence: oracletypes.MicroReport{
// 				AggregateMethod: "weighted-median",
// 				BlockNumber:     218261,
// 				MetaId:          109136,
// 				Power:           uint64(5),
// 				QueryId:         []byte("DRKtSRkxY7u+/05tuClM7SP/hgU1n9ZmeZ1OJaOqDjo="),
// 				QueryType:       "AmpleforthCustomSpotPrice",
// 				Reporter:        "tellor17gc67q05d5rgsz9caznm0s7s5eazwg2e3fkk8e",
// 				Timestamp:       time.Date(2025, 3, 24, 14, 11, 35, 566750439, time.UTC),
// 				Value:           "00000000000000000000000000000000000000000000000010eb2f7f19c6e1ed",
// 			},
// 			FeeTotal:         math.NewInt(50000),
// 			PrevDisputeIds:   []uint64{1},
// 			BlockNumber:      220201,
// 			Open:             true,
// 			VoterReward:      math.NewInt(1250),
// 			PendingExecution: false,
// 		}

// 		key := make([]byte, 8)
// 		_, err := collections.Uint64Key.Encode(key, data.DisputeId)
// 		require.NoError(t, err)
// 		value, _ := cdc.Marshal(&data)
// 		attestStore.Set(key, value)
// 	}

// 	b.ResetTimer()

// 	// Run benchmark
// 	for i := 0; i < b.N; i++ {
// 		m := keeper.NewMigrator(bk)
// 		err := m.Migrate1to2(sdkCtx)
// 		if err != nil {
// 			b.Fatal(err)
// 		}
// 	}
// }
