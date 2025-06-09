package bridge

import (
	"testing"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/mocks"
	"github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
	dk := new(mocks.DisputeKeeper)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		sk,
		ok,
		bk,
		rk,
		dk,
		"gov",
	)

	app := NewAppModule(
		cdc,
		k,
		ak,
		bk,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	require.NoError(t, k.Params.Set(ctx, types.DefaultParams()))

	require.NoError(t, k.SnapshotLimit.Set(ctx, types.SnapshotLimit{
		Limit: types.DefaultSnapshotLimit,
	}))
	return app, k, ctx, sk, ok
}

func SetupBridgeAppBenchmark(b *testing.B) (AppModule, keeper.Keeper, sdk.Context, *mocks.StakingKeeper, *mocks.OracleKeeper) {
	b.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := cosmosdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(b, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	ak := new(mocks.AccountKeeper)
	bk := new(mocks.BankKeeper)
	ok := new(mocks.OracleKeeper)
	rk := new(mocks.ReporterKeeper)
	sk := new(mocks.StakingKeeper)
	dk := new(mocks.DisputeKeeper)
	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		sk,
		ok,
		bk,
		rk,
		dk,
		"gov",
	)

	app := NewAppModule(
		cdc,
		k,
		ak,
		bk,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	require.NoError(b, k.Params.Set(ctx, types.DefaultParams()))

	require.NoError(b, k.SnapshotLimit.Set(ctx, types.SnapshotLimit{
		Limit: types.DefaultSnapshotLimit,
	}))
	return app, k, ctx, sk, ok
}

func TestEndBlock(t *testing.T) {
	app, k, ctx, sk, ok := SetupBridgeApp(t)
	require.NotNil(t, app)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	require.NotNil(t, sk)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithConsensusParams(tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{
			MaxBytes: 200000,
			MaxGas:   100_000_000,
		},
		Evidence: &tmproto.EvidenceParams{
			MaxAgeNumBlocks: 302400,
			MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
			MaxBytes:        10000,
		},
		Validator: &tmproto.ValidatorParams{
			PubKeyTypes: []string{
				cmttypes.ABCIPubKeyTypeEd25519,
			},
		},
		Abci: &tmproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})
	ctx = ctx.WithConsensusParams(sdkCtx.ConsensusParams())

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
		err := k.SetEVMAddressByOperator(sdkCtx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)

		evmAddresses[i], err = k.OperatorToEVMAddressMap.Get(sdkCtx, val.GetOperator())
		require.NoError(t, err)
		require.Equal(t, evmAddresses[i].EVMAddress, []byte(val.Description.Moniker))
	}
	sk.On("GetAllValidators", sdkCtx).Return(validators, nil)

	ok.On("GetAggregatedReportsByHeight", sdkCtx, uint64(0)).Return([]oracletypes.Aggregate{
		{
			Height:         1,
			QueryId:        []byte("queryId"),
			AggregateValue: "5000",
			AggregatePower: uint64(100),
		},
	}, nil)
	queryId := []byte("queryId")
	timestamp := sdkCtx.BlockTime()
	timestampPlus1 := timestamp.Add(time.Second)

	ok.On("GetTimestampBefore", sdkCtx, queryId, timestampPlus1).Return(timestamp, nil).Once()
	ok.On("GetTimestampBefore", sdkCtx, queryId, timestamp).Return(timestamp, nil)
	ok.On("GetAggregateByTimestamp", sdkCtx, queryId, uint64(timestamp.UnixMilli())).Return(oracletypes.Aggregate{
		QueryId:        queryId,
		AggregateValue: "5000",
		AggregatePower: uint64(100),
	}, nil)

	err := k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{
		Checkpoint: []byte("checkpoint"),
	})
	require.NoError(t, err)
	// save checkpoint params
	checkpointParams := types.ValidatorCheckpointParams{
		Checkpoint:     []byte("checkpoint"),
		ValsetHash:     []byte("validatorSetHash"),
		Timestamp:      uint64(timestamp.UnixMilli()),
		PowerThreshold: uint64(100),
	}
	err = k.ValidatorCheckpointParamsMap.Set(ctx, uint64(timestamp.UnixMilli()), checkpointParams)
	require.NoError(t, err)
	ok.On("GetTimestampAfter", sdkCtx, queryId, timestamp).Return(timestampPlus1, nil)
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

func BenchmarkBridgeEndBlock(b *testing.B) {
	app, k, ctx, sk, ok := SetupBridgeAppBenchmark(b)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// set up validators
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

	// set up EVM addresses
	for _, val := range validators {
		require.NoError(b, k.SetEVMAddressByOperator(sdkCtx, val.OperatorAddress, []byte(val.Description.Moniker)))
	}

	// set up mocks
	sk.On("GetAllValidators", sdkCtx).Return(validators, nil)

	queryId := []byte("queryId")
	timestamp := sdkCtx.BlockTime()
	timestampPlus1 := timestamp.Add(time.Second)

	ok.On("GetAggregatedReportsByHeight", sdkCtx, uint64(0)).Return([]oracletypes.Aggregate{
		{
			Height:         1,
			QueryId:        queryId,
			AggregateValue: "5000",
			AggregatePower: uint64(100),
		},
	}, nil)

	ok.On("GetTimestampBefore", sdkCtx, queryId, timestampPlus1).Return(timestamp, nil)
	ok.On("GetTimestampBefore", sdkCtx, queryId, timestamp).Return(timestamp, nil)
	ok.On("GetTimestampAfter", sdkCtx, queryId, timestamp).Return(timestampPlus1, nil)
	ok.On("GetAggregateByTimestamp", sdkCtx, queryId, uint64(timestamp.UnixMilli())).Return(oracletypes.Aggregate{
		QueryId:        queryId,
		AggregateValue: "5000",
		AggregatePower: uint64(100),
	}, nil)

	// set up checkpoint data
	require.NoError(b, k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{
		Checkpoint: []byte("checkpoint"),
	}))

	require.NoError(b, k.ValidatorCheckpointParamsMap.Set(ctx, uint64(timestamp.UnixMilli()), types.ValidatorCheckpointParams{
		Checkpoint:     []byte("checkpoint"),
		ValsetHash:     []byte("validatorSetHash"),
		Timestamp:      uint64(timestamp.UnixMilli()),
		PowerThreshold: uint64(100),
	}))

	require.NoError(b, k.BridgeValset.Set(ctx, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           100,
			},
		},
	}))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := app.EndBlock(ctx)
		require.NoError(b, err)
	}
}
