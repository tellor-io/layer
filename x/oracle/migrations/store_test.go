package migrations_test

import (
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/oracle/migrations"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
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

	newquery := collections.NewIndexedMap(sb,
		types.QueryTipPrefix,
		"query",
		collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
		codec.CollValue[types.QueryMeta](cdc),
		types.NewQueryIndex(sb),
	)
	err := migrations.MigrateStore(ctx, storeService, cdc, newquery)
	require.NoError(t, err)
}
