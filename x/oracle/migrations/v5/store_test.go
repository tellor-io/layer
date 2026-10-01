package v5_test

import (
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	v5 "github.com/tellor-io/layer/x/oracle/migrations/v5"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	sdkStore "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMigrateStore_DeletesMaxBatchSize(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := cosmosdb.NewMemDB()
	stateStore := sdkStore.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	storeService := runtime.NewKVStoreService(storeKey)

	sb := collections.NewSchemaBuilder(storeService)
	maxBatchSize := collections.NewItem(sb, types.DeprecatedMaxBatchSizePrefix, "max_batch_size", collections.Uint32Value)
	queryDataLimit := collections.NewItem(sb, types.QueryDataLimitPrefix, "query_data_limit", collections.Uint64Value)

	require.NoError(t, maxBatchSize.Set(ctx, 20))
	require.NoError(t, queryDataLimit.Set(ctx, 512))

	got, err := maxBatchSize.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint32(20), got)

	require.NoError(t, v5.MigrateStore(ctx, storeService))

	_, err = maxBatchSize.Get(ctx)
	require.ErrorIs(t, err, collections.ErrNotFound)

	limit, err := queryDataLimit.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(512), limit)
}

func TestMigrateStore_NoopWhenMissing(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := cosmosdb.NewMemDB()
	stateStore := sdkStore.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	storeService := runtime.NewKVStoreService(storeKey)

	require.NoError(t, v5.MigrateStore(ctx, storeService))
}
