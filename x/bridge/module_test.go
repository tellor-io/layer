package bridge

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/mocks"
	"github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func SetupBridgeApp(t *testing.T) (AppModule, keeper.Keeper, sdk.Context, *mocks.StakingKeeper, *mocks.OracleKeeper) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := cosmosdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	ak := new(mocks.AccountKeeper)
	bk := new(mocks.BankKeeper)
	ok := new(mocks.OracleKeeper)
	rk := new(mocks.ReporterKeeper)
	sk := new(mocks.StakingKeeper)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		sk,
		ak,
		ok,
		bk,
		rk,
	)

	app := NewAppModule(
		cdc,
		k,
		ak,
		bk,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	require.NoError(t, k.Params.Set(ctx, types.DefaultParams()))

	return app, k, ctx, sk, ok
}

func TestEndBlock(t *testing.T) {
	app, k, ctx, sk, ok := SetupBridgeApp(t)
	require.NotNil(t, app)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	require.NotNil(t, sk)

	// create validator set
	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(60 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: "operatorAddr1",
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(40 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: "operatorAddr2",
		},
	}
	evmAddresses := make([]types.EVMAddress, len(validators))
	for i, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)

		evmAddresses[i], err = k.OperatorToEVMAddressMap.Get(ctx, val.GetOperator())
		require.NoError(t, err)
		require.Equal(t, evmAddresses[i].EVMAddress, []byte(val.Description.Moniker))
	}
	sk.On("GetAllValidators", ctx).Return(validators, nil)

	ok.On("GetAggregatedReportsByHeight", ctx, int64(0)).Return([]oracletypes.Aggregate{
		{
			Height:         0,
			QueryId:        []byte("queryId"),
			AggregateValue: "5000",
			ReporterPower:  int64(100),
		},
	}, nil)
	queryId := []byte("queryId")
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	timestamp := sdkCtx.BlockTime()
	ok.On("GetTimestampBefore", ctx, queryId, timestamp).Return(timestamp, nil)

	ok.On("GetAggregateByTimestamp", ctx, queryId, timestamp).Return(&oracletypes.Aggregate{
		QueryId:        queryId,
		AggregateValue: "5000",
		ReporterPower:  int64(100),
	}, nil)

	err := k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{
		Checkpoint: []byte("checkpoint"),
	})
	require.NoError(t, err)
	ok.On("GetTimestampAfter", ctx, queryId, timestamp).Return(timestamp, nil)
	err = k.BridgeValset.Set(ctx, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           100,
			},
		},
	})
	require.NoError(t, err)

	err = app.EndBlock(ctx)
	require.NoError(t, err)
}
