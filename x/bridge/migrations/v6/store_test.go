package v6_test

import (
	"context"
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	bridgekeeper "github.com/tellor-io/layer/x/bridge/keeper"
	v6 "github.com/tellor-io/layer/x/bridge/migrations/v6"
	bridgemocks "github.com/tellor-io/layer/x/bridge/mocks"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oraclemocks "github.com/tellor-io/layer/x/oracle/mocks"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
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

func setupKeepers(t *testing.T) (context.Context, bridgekeeper.Keeper, oraclekeeper.Keeper) {
	t.Helper()

	bridgeStoreKey := storetypes.NewKVStoreKey(bridgetypes.StoreKey)
	oracleStoreKey := storetypes.NewKVStoreKey(oracletypes.StoreKey)
	db := cosmosdb.NewMemDB()

	stateStore := sdkStore.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(bridgeStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(oracleStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	interfaceRegistry := types.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)
	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	oracleK := oraclekeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(oracleStoreKey),
		new(oraclemocks.AccountKeeper),
		new(oraclemocks.BankKeeper),
		new(oraclemocks.RegistryKeeper),
		new(oraclemocks.ReporterKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	require.NoError(t, oracleK.SetParams(ctx, oracletypes.DefaultParams()))

	bridgeK := bridgekeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(bridgeStoreKey),
		new(bridgemocks.StakingKeeper),
		oracleK,
		new(bridgemocks.BankKeeper),
		new(bridgemocks.ReporterKeeper),
		new(bridgemocks.DisputeKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	require.NoError(t, bridgeK.Params.Set(ctx, bridgetypes.DefaultParams()))
	require.NoError(t, bridgeK.SnapshotLimit.Set(ctx, bridgetypes.SnapshotLimit{Limit: bridgetypes.DefaultSnapshotLimit}))

	return ctx, bridgeK, oracleK
}

func seedPowerThreshold(t *testing.T, ctx context.Context, k bridgekeeper.Keeper, threshold uint64) {
	t.Helper()
	require.NoError(t, k.LatestCheckpointIdx.Set(ctx, bridgetypes.CheckpointIdx{Index: 1}))
	require.NoError(t, k.ValidatorCheckpointIdxMap.Set(ctx, 1, bridgetypes.CheckpointTimestamp{Timestamp: 100}))
	require.NoError(t, k.ValidatorCheckpointParamsMap.Set(ctx, 100, bridgetypes.ValidatorCheckpointParams{
		Checkpoint:     []byte("checkpoint"),
		ValsetHash:     []byte("valsetHash"),
		Timestamp:      100,
		PowerThreshold: threshold,
	}))
}

func seedAggregate(t *testing.T, ctx context.Context, ok oraclekeeper.Keeper, queryId []byte, ts, power uint64, flagged bool) {
	t.Helper()
	require.NoError(t, ok.Aggregates.Set(ctx, collections.Join(queryId, ts), oracletypes.Aggregate{
		QueryId:           queryId,
		AggregateValue:    "1",
		AggregateReporter: sample.AccAddress(),
		AggregatePower:    power,
		Flagged:           flagged,
	}))
}

func TestMigrateStore_NewestConsensusPerQueryId(t *testing.T) {
	ctx, bridgeK, oracleK := setupKeepers(t)
	seedPowerThreshold(t, ctx, bridgeK, 100)

	q1 := []byte("query-one")
	q2 := []byte("query-two")
	seedAggregate(t, ctx, oracleK, q1, 10, 50, false)
	seedAggregate(t, ctx, oracleK, q1, 20, 150, false)
	seedAggregate(t, ctx, oracleK, q1, 30, 40, false)
	seedAggregate(t, ctx, oracleK, q2, 11, 200, false)
	seedAggregate(t, ctx, oracleK, q2, 21, 80, false)
	seedAggregate(t, ctx, oracleK, q2, 31, 180, true)

	err := v6.MigrateStore(ctx, bridgeK, oracleK)
	require.NoError(t, err)

	ts1, err := bridgeK.LastConsensusTimestampByQueryId.Get(ctx, q1)
	require.NoError(t, err)
	require.Equal(t, uint64(20), ts1)

	ts2, err := bridgeK.LastConsensusTimestampByQueryId.Get(ctx, q2)
	require.NoError(t, err)
	require.Equal(t, uint64(31), ts2)
}

func TestMigrateStore_OnlyNonConsensusAggregates(t *testing.T) {
	ctx, bridgeK, oracleK := setupKeepers(t)
	seedPowerThreshold(t, ctx, bridgeK, 100)

	q := []byte("query-none")
	seedAggregate(t, ctx, oracleK, q, 10, 10, false)
	seedAggregate(t, ctx, oracleK, q, 20, 99, false)

	err := v6.MigrateStore(ctx, bridgeK, oracleK)
	require.NoError(t, err)

	_, err = bridgeK.LastConsensusTimestampByQueryId.Get(ctx, q)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestMigrateStore_EmptyAggregates(t *testing.T) {
	ctx, bridgeK, oracleK := setupKeepers(t)
	seedPowerThreshold(t, ctx, bridgeK, 100)

	err := v6.MigrateStore(ctx, bridgeK, oracleK)
	require.NoError(t, err)

	has := false
	err = bridgeK.LastConsensusTimestampByQueryId.Walk(ctx, nil, func(_ []byte, _ uint64) (bool, error) {
		has = true
		return true, nil
	})
	require.NoError(t, err)
	require.False(t, has)
}

func TestMigrateStore_DoesNotDependOnAttestSnapshotDataMap(t *testing.T) {
	ctx, bridgeK, oracleK := setupKeepers(t)
	seedPowerThreshold(t, ctx, bridgeK, 100)

	q := []byte("query-no-snapshots")
	seedAggregate(t, ctx, oracleK, q, 42, 200, false)

	err := v6.MigrateStore(ctx, bridgeK, oracleK)
	require.NoError(t, err)

	ts, err := bridgeK.LastConsensusTimestampByQueryId.Get(ctx, q)
	require.NoError(t, err)
	require.Equal(t, uint64(42), ts)

	empty := true
	err = bridgeK.AttestSnapshotDataMap.Walk(ctx, nil, func(_ []byte, _ bridgetypes.AttestationSnapshotData) (bool, error) {
		empty = false
		return true, nil
	})
	require.NoError(t, err)
	require.True(t, empty)
}

func TestMigrateStore_FlaggedConsensusIsEligible(t *testing.T) {
	ctx, bridgeK, oracleK := setupKeepers(t)
	seedPowerThreshold(t, ctx, bridgeK, 100)

	q := []byte("query-flagged")
	seedAggregate(t, ctx, oracleK, q, 10, 50, false)
	seedAggregate(t, ctx, oracleK, q, 20, 150, true)

	err := v6.MigrateStore(ctx, bridgeK, oracleK)
	require.NoError(t, err)

	ts, err := bridgeK.LastConsensusTimestampByQueryId.Get(ctx, q)
	require.NoError(t, err)
	require.Equal(t, uint64(20), ts)
}
