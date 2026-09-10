package v6_test

import (
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	v6 "github.com/tellor-io/layer/x/oracle/migrations/v6"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkStore "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupMigrateStore(t *testing.T) (sdk.Context, store.KVStoreService, collections.Map[[]byte, []byte], collections.Sequence, collections.Item[[]byte]) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := cosmosdb.NewMemDB()
	stateStore := sdkStore.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	storeService := runtime.NewKVStoreService(storeKey)

	sb := collections.NewSchemaBuilder(storeService)
	cyclelist := collections.NewMap(sb, types.CyclelistPrefix, "cyclelist", collections.BytesKey, collections.BytesValue)
	sequencer := collections.NewSequence(sb, types.CycleSeqPrefix, "cycle_sequencer")
	current := collections.NewItem(sb, types.CurrentCycleListQueryPrefix, "current_cycle_list_query", collections.BytesValue)
	_, err := sb.Build()
	require.NoError(t, err)
	return ctx, storeService, cyclelist, sequencer, current
}

func TestMigrateStore_InBoundsIdxCopiesQAtIdx(t *testing.T) {
	ctx, storeService, cyclelist, sequencer, current := setupMigrateStore(t)

	require.NoError(t, cyclelist.Set(ctx, []byte{0}, []byte("q0")))
	require.NoError(t, cyclelist.Set(ctx, []byte{1}, []byte("q1")))
	require.NoError(t, cyclelist.Set(ctx, []byte{2}, []byte("q2")))
	require.NoError(t, sequencer.Set(ctx, 1))

	require.NoError(t, v6.MigrateStore(ctx, storeService))

	got, err := current.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, []byte("q1"), got)

	idx, err := sequencer.Peek(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), idx)
}

func TestMigrateStore_OOBIdxCopiesQ0(t *testing.T) {
	ctx, storeService, cyclelist, sequencer, current := setupMigrateStore(t)

	require.NoError(t, cyclelist.Set(ctx, []byte{0}, []byte("q0")))
	require.NoError(t, cyclelist.Set(ctx, []byte{1}, []byte("q1")))
	require.NoError(t, sequencer.Set(ctx, 99))

	require.NoError(t, v6.MigrateStore(ctx, storeService))

	got, err := current.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, []byte("q0"), got)

	idx, err := sequencer.Peek(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(99), idx)
}
