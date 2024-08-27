package migrations_test

import (
	"testing"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/oracle/migrations"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMigrateStore(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(storeKey)
	registry := codectypes.NewInterfaceRegistry()
	db := dbm.NewMemDB()

	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)

	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	cdc := codec.NewProtoCodec(registry)
	sb := collections.NewSchemaBuilder(storeService)

	oldquery := collections.NewIndexedMap(sb,
		types.QueryTipPrefix,
		"query",
		collections.BytesKey,
		codec.CollValue[types.QueryMeta](cdc),
		migrations.NewQueryIndex(sb),
	)

	newquery := collections.NewIndexedMap(sb,
		types.QueryTipPrefix,
		"query",
		collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
		codec.CollValue[types.QueryMeta](cdc),
		types.NewQueryIndex(sb),
	)

	oldquery.Set(ctx, []byte("key"), types.QueryMeta{Id: 1, Expiration: ctx.BlockTime()})
	oldquery.Set(ctx, []byte("key2"), types.QueryMeta{Id: 2, Expiration: ctx.BlockTime().Add(time.Hour)})
	oldquery.Set(ctx, []byte("key3"), types.QueryMeta{Id: 3, Expiration: ctx.BlockTime().Add(2 * time.Hour)})
	oldquery.Set(ctx, []byte("key4"), types.QueryMeta{Id: 4, Expiration: ctx.BlockTime().Add(-3 * time.Hour)})
	err := migrations.MigrateStore(ctx, storeService, cdc, newquery)
	require.NoError(t, err)

	q, err := newquery.Get(ctx, collections.Join([]byte("key"), uint64(1)))
	require.NoError(t, err)
	require.Equal(t, types.QueryMeta{Id: 1, Expiration: ctx.BlockTime(), Amount: math.ZeroInt()}, q)

	q, err = newquery.Get(ctx, collections.Join([]byte("key2"), uint64(2)))
	require.NoError(t, err)
	require.Equal(t, types.QueryMeta{Id: 2, Expiration: ctx.BlockTime().Add(time.Hour), Amount: math.ZeroInt()}, q)

	q, err = newquery.Get(ctx, collections.Join([]byte("key3"), uint64(3)))
	require.NoError(t, err)
	require.Equal(t, types.QueryMeta{Id: 3, Expiration: ctx.BlockTime().Add(2 * time.Hour), Amount: math.ZeroInt()}, q)

	_, err = newquery.Get(ctx, collections.Join([]byte("key4"), uint64(4)))
	require.ErrorIs(t, err, collections.ErrNotFound)

}
