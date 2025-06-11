package keeper_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/mocks"
	"github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const (
	testOperatorAddr1 = "cosmosvaloper1alcefjzkk37qmfrnel8q4eruyll0pc8axy6gsg"
	testOperatorAddr2 = "cosmosvaloper18lllgejqwydmnakd8mfvfxhw5lqd6kqkftg48v"
)

func setupKeeper(tb testing.TB) (keeper.Keeper, *mocks.AccountKeeper, *mocks.BankKeeper, *mocks.OracleKeeper, *mocks.ReporterKeeper, *mocks.StakingKeeper, *mocks.DisputeKeeper, context.Context) {
	tb.Helper()
	k, ak, bk, ok, rk, sk, dk, ctx := keepertest.BridgeKeeper(tb)

	// Initialize genesis state with default snapshot limit
	err := k.SnapshotLimit.Set(ctx, types.SnapshotLimit{
		Limit: types.DefaultSnapshotLimit,
	})
	require.NoError(tb, err)

	// Initialize default params
	err = k.Params.Set(ctx, types.DefaultParams())
	require.NoError(tb, err)

	return k, ak, bk, ok, rk, sk, dk, ctx
}

func TestKeeper(t *testing.T) {
	k, ak, bk, ok, rk, sk, dk, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ak)
	require.NotNil(t, bk)
	require.NotNil(t, ok)
	require.NotNil(t, rk)
	require.NotNil(t, sk)
	require.NotNil(t, dk)
	require.NotNil(t, ctx)
}

func TestGetCurrentValidatorsEVMCompatible(t *testing.T) {
	k, _, _, _, _, sk, _, ctx := setupKeeper(t)

	operatorAddr1 := testOperatorAddr1
	operatorAddr2 := testOperatorAddr2

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(3000000000),
			DelegatorShares: math.LegacyNewDec(3000),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: operatorAddr1,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000000000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: operatorAddr2,
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
	bridgeValSet, err := k.GetCurrentValidatorsEVMCompatible(ctx)
	require.NoError(t, err)
	require.NotNil(t, bridgeValSet)

	for i := 0; i < len(bridgeValSet)-1; i++ {
		require.GreaterOrEqual(t, bridgeValSet[i].Power, bridgeValSet[i+1].Power)
		require.Equal(t, bridgeValSet[i].Power, uint64(validators[i].GetConsensusPower(layertypes.PowerReduction)))
		require.Equal(t, bridgeValSet[i].EthereumAddress, evmAddresses[i].EVMAddress)
	}
}

func TestGetCurrentValidatorsEVMCompatibleNoValidators(t *testing.T) {
	k, _, _, _, _, sk, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	validators := []stakingtypes.Validator{}

	sk.On("GetAllValidators", ctx).Return(validators, nil)
	bridgeValSet, err := k.GetCurrentValidatorsEVMCompatible(ctx)
	require.ErrorContains(t, err, "no validators found")
	require.Nil(t, bridgeValSet)
}

func TestGetCurrentValidatorsEVMCompatibleEqualPowers(t *testing.T) {
	k, _, _, _, _, sk, _, ctx := setupKeeper(t)

	operatorAddr1 := testOperatorAddr1
	operatorAddr2 := testOperatorAddr2

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000000000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: operatorAddr1,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000000000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: operatorAddr2,
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
	bridgeValSet, err := k.GetCurrentValidatorsEVMCompatible(ctx)
	require.NoError(t, err)
	require.NotNil(t, bridgeValSet)

	for i := 0; i < len(bridgeValSet)-1; i++ {
		require.GreaterOrEqual(t, bridgeValSet[i].Power, bridgeValSet[i+1].Power)
		require.Equal(t, bridgeValSet[i].Power, uint64(validators[i].GetConsensusPower(layertypes.PowerReduction)))
		require.Equal(t, bridgeValSet[i].EthereumAddress, evmAddresses[i].EVMAddress)
		require.LessOrEqual(t, string(bridgeValSet[i].EthereumAddress), string(bridgeValSet[i+1].EthereumAddress))
	}
}

func TestGetCurrentValidatorSetEVMCompatible(t *testing.T) {
	k, _, _, _, _, sk, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	operatorAddr1 := testOperatorAddr1
	operatorAddr2 := testOperatorAddr2

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(2000000000),
			DelegatorShares: math.LegacyNewDec(2000),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: operatorAddr1,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000000000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: operatorAddr2,
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
	bridgeValidatorSet, err := k.GetCurrentValidatorSetEVMCompatible(ctx)
	require.NoError(t, err)
	require.NotNil(t, bridgeValidatorSet)

	for i := 0; i < len(bridgeValidatorSet.BridgeValidatorSet)-1; i++ {
		require.GreaterOrEqual(t, bridgeValidatorSet.BridgeValidatorSet[i].Power, bridgeValidatorSet.BridgeValidatorSet[i+1].Power)
		require.Equal(t, bridgeValidatorSet.BridgeValidatorSet[i].Power, uint64(validators[i].GetConsensusPower(layertypes.PowerReduction)))
	}
}

func TestCompareAndSetBridgeValidators(t *testing.T) {
	k, _, _, _, _, sk, _, ctx := setupKeeper(t)
	logger := k.Logger(ctx)

	// call without setting validator set
	sk.On("GetAllValidators", ctx).Return(nil, nil).Once()
	res, err := k.CompareAndSetBridgeValidators(ctx)
	require.ErrorContains(t, err, "no validators found")
	require.False(t, res)
	logger.Error("err: ", err)

	// call with empty valset
	sk.On("GetAllValidators", ctx).Return([]stakingtypes.Validator{}, nil).Once()
	res, err = k.CompareAndSetBridgeValidators(ctx)
	require.ErrorContains(t, err, "no validators found")
	require.False(t, res)
	logger.Error("err: ", err)

	operatorAddr1 := testOperatorAddr1
	operatorAddr2 := testOperatorAddr2

	// call for initial val set, should hit false since no valset exists yet
	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(100 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: operatorAddr1,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(100 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: operatorAddr2,
		},
	}
	for _, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)
	}

	sk.On("GetAllValidators", ctx).Return(validators, nil).Once()
	res, err = k.CompareAndSetBridgeValidators(ctx)
	require.NoError(t, err)
	require.False(t, res)

	// change validator1 power by more than 5% of total power, should return true since bridgevalset needs updated
	validators = []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(111 * 1e6),
			DelegatorShares: math.LegacyNewDec(111 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: operatorAddr1,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(100 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: operatorAddr2,
		},
	}
	for _, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)
	}

	sk.On("GetAllValidators", ctx).Return(validators, nil).Twice()
	res, err = k.CompareAndSetBridgeValidators(ctx)
	require.NoError(t, err)
	require.True(t, res)

	// call without changing anything, should hit false since valset bytes are equal
	res, err = k.CompareAndSetBridgeValidators(ctx)
	require.NoError(t, err)
	require.False(t, res)

	// change by less than 5%, should hit false
	validators = []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(112 * 1e6),
			DelegatorShares: math.LegacyNewDec(112 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: operatorAddr1,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(100 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: operatorAddr2,
		},
	}
	for _, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)
	}

	sk.On("GetAllValidators", ctx).Return(validators, nil).Once()
	res, err = k.CompareAndSetBridgeValidators(ctx)
	require.NoError(t, err)
	require.False(t, res)
}

func TestSetBridgeValidatorParams(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	bridgeValSet := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           1000,
			},
		},
	}

	err := k.SetBridgeValidatorParams(ctx, &bridgeValSet)
	require.NoError(t, err)

	params, err := k.Params.Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, params)

	bridgeValSet = types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           1000,
			},
			{
				EthereumAddress: []byte("validator2"),
				Power:           2000,
			},
		},
	}

	err = k.SetBridgeValidatorParams(ctx, &bridgeValSet)
	require.NoError(t, err)

	params2, err := k.Params.Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, params2)
}

// todo: check all stores
func TestCalculateValidatorSetCheckpoint(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	powerThreshold := uint64(5000)
	validatorTimestamp := uint64(100_000)
	valSetHash := []byte("valSetHash")

	checkpoint, err := k.CalculateValidatorSetCheckpoint(ctx, powerThreshold, validatorTimestamp, valSetHash)
	require.NoError(t, err)
	require.NotNil(t, checkpoint)
	lastCheckpointIdx, err := k.LatestCheckpointIdx.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, lastCheckpointIdx.Index, uint64(0))

	powerThreshold = 0
	validatorTimestamp = 0
	valSetHash = []byte{}

	checkpoint, err = k.CalculateValidatorSetCheckpoint(ctx, powerThreshold, validatorTimestamp, valSetHash)
	require.NoError(t, err)
	require.NotNil(t, checkpoint)
	lastCheckpointIdx, err = k.LatestCheckpointIdx.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, lastCheckpointIdx.Index, uint64(1))

	powerThreshold = ^uint64(0)
	validatorTimestamp = ^uint64(0)
	valSetHash = []byte("hash0123456789")

	checkpoint, err = k.CalculateValidatorSetCheckpoint(ctx, powerThreshold, validatorTimestamp, valSetHash)
	require.NoError(t, err)
	require.NotNil(t, checkpoint)
	lastCheckpointIdx, err = k.LatestCheckpointIdx.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, lastCheckpointIdx.Index, uint64(2))
}

func TestGetValidatorCheckpointFromStorage(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	res, err := k.GetValidatorCheckpointFromStorage(ctx)
	require.Error(t, err)
	require.Nil(t, res)

	bridgeValSet := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           1000,
			},
		},
	}

	err = k.SetBridgeValidatorParams(ctx, &bridgeValSet)
	require.NoError(t, err)

	res, err = k.GetValidatorCheckpointFromStorage(ctx)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestGetValidatorTimestampByIdxFromStorage(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	res, err := k.GetValidatorTimestampByIdxFromStorage(ctx, 0)
	require.Error(t, err)
	require.Equal(t, res.Timestamp, uint64(0))

	bridgeValSet := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           1000,
			},
		},
	}

	err = k.SetBridgeValidatorParams(ctx, &bridgeValSet)
	require.NoError(t, err)

	res, err = k.GetValidatorTimestampByIdxFromStorage(ctx, 0)
	require.NoError(t, err)
	require.NotNil(t, res)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	validatorTimestamp := uint64(sdkCtx.BlockTime().UnixMilli())
	require.Equal(t, res.Timestamp, validatorTimestamp)

	prevBlockTime := sdkCtx.BlockTime()
	sdkCtx = sdkCtx.WithBlockTime(prevBlockTime.Add(20 * time.Second))
	validatorTimestamp = uint64(sdkCtx.BlockTime().Unix())

	// create new checkpoint
	powerThreshold := uint64(5000)
	valSetHash := []byte("valSetHash")

	checkpoint, err := k.CalculateValidatorSetCheckpoint(sdkCtx, powerThreshold, validatorTimestamp, valSetHash)
	require.NoError(t, err)
	require.NotNil(t, checkpoint)
	lastCheckpointIdx, err := k.LatestCheckpointIdx.Get(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, lastCheckpointIdx.Index, uint64(1))

	res, err = k.GetValidatorTimestampByIdxFromStorage(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, res.Timestamp, validatorTimestamp)

	// test max
	maxUint := ^uint64(0)
	require.NoError(t, k.ValidatorCheckpointIdxMap.Set(ctx, maxUint, types.CheckpointTimestamp{
		Timestamp: maxUint,
	}))
	res, err = k.GetValidatorTimestampByIdxFromStorage(ctx, maxUint)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, res.Timestamp, maxUint)
}

func TestGetValidatorSetSignaturesFromStorage(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	res, err := k.GetValidatorSetSignaturesFromStorage(ctx, 0)
	require.Error(t, err)
	require.Nil(t, res)

	bridgeValSet := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           1000,
			},
		},
	}

	err = k.SetBridgeValidatorParams(ctx, &bridgeValSet)
	require.NoError(t, err)

	timestamp, err := k.GetValidatorTimestampByIdxFromStorage(ctx, 0)
	require.NoError(t, err)
	require.NotNil(t, timestamp)

	res, err = k.GetValidatorSetSignaturesFromStorage(ctx, timestamp.Timestamp)
	require.NoError(t, err)
	require.NotNil(t, res)

	// test max
	maxUint := ^uint64(0)
	require.NoError(t, k.BridgeValsetSignaturesMap.Set(ctx, maxUint, types.BridgeValsetSignatures{
		Signatures: [][]byte{
			[]byte("signature1"),
			[]byte("signature2"),
		},
	}))
	res, err = k.GetValidatorSetSignaturesFromStorage(ctx, maxUint)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, len(res.Signatures), 2)
}

func TestEncodeAndHashValidatorSet(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	bridgeValSet := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           1000,
			},
		},
	}

	encodedBridgeValSet, bridgeValSetHash, err := k.EncodeAndHashValidatorSet(ctx, &bridgeValSet)
	require.NoError(t, err)
	require.NotNil(t, encodedBridgeValSet)
	require.NotNil(t, bridgeValSetHash)

	bridgeValSet2 := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator10"),
				Power:           10000000,
			},
			{
				EthereumAddress: []byte("validator100"),
				Power:           20000,
			},
		},
	}
	encodedBridgeValSet2, bridgeValSetHash2, err := k.EncodeAndHashValidatorSet(ctx, &bridgeValSet2)
	require.NoError(t, err)
	require.NotNil(t, encodedBridgeValSet2)
	require.NotNil(t, bridgeValSetHash2)
	require.NotEqual(t, bridgeValSetHash, bridgeValSetHash2)
	require.NotEqual(t, encodedBridgeValSet, encodedBridgeValSet2)
}

func TestPowerDiff(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	// empty to 0
	bridgeValSetEmpty := types.BridgeValidatorSet{}
	bridgeValSet0 := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           0,
			},
		},
	}

	relativeDiff := k.PowerDiff(ctx, bridgeValSetEmpty, bridgeValSet0)
	require.Equal(t, relativeDiff, int64(0))

	// 0 to 100, returns 0 if valset b is 0
	bridgeValSet100 := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           100,
			},
		},
	}
	relativeDiff = k.PowerDiff(ctx, bridgeValSet0, bridgeValSet100)
	require.Equal(t, relativeDiff, int64(0))

	// 100 to 104 (increase just under 5%)
	bridgeValSet104 := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           100,
			},
			{
				EthereumAddress: []byte("validator2"),
				Power:           4,
			},
		},
	}
	relativeDiff = k.PowerDiff(ctx, bridgeValSet100, bridgeValSet104)
	require.Equal(t, relativeDiff, int64(4e4))

	// 104 to 110 (increase just over 5%)
	bridgeValSet110 := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           100,
			},
			{
				EthereumAddress: []byte("validator2"),
				Power:           10,
			},
		},
	}
	relativeDiff = k.PowerDiff(ctx, bridgeValSet104, bridgeValSet110)
	require.Greater(t, relativeDiff, int64(5e4))
	require.Less(t, relativeDiff, int64(6e4))

	// 110 to 104 (decrease just over 5%)
	relativeDiff = k.PowerDiff(ctx, bridgeValSet110, bridgeValSet104)
	require.Greater(t, relativeDiff, int64(5e4))
	require.Less(t, relativeDiff, int64(6e4))

	// 104 to 100 (decrease just under 5%)
	relativeDiff = k.PowerDiff(ctx, bridgeValSet104, bridgeValSet100)
	require.Less(t, relativeDiff, int64(5e4))
	require.Greater(t, relativeDiff, int64(3e4))

	// 100 to 100,000 (big increase)
	bridgeValSet100_000 := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           100_000,
			},
		},
	}
	relativeDiff = k.PowerDiff(ctx, bridgeValSet100, bridgeValSet100_000)
	require.Equal(t, relativeDiff, int64(999e6))

	// 100,000 to 100 (big decrease)
	relativeDiff = k.PowerDiff(ctx, bridgeValSet100_000, bridgeValSet100)
	require.Equal(t, relativeDiff, int64(999e3))
}

func TestEVMAddressFromSignatures(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	// https://goethereumbook.org/signature-generate/
	privateKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	require.NotNil(t, privateKey)
	require.NoError(t, err)

	pkCoord := &ecdsa.PublicKey{
		X: privateKey.X,
		Y: privateKey.Y,
	}
	addressExpected := crypto.PubkeyToAddress(*pkCoord).Hex()
	operatorAddr := "operatorAddr1"

	msgA := fmt.Sprintf("TellorLayer: Initial bridge signature A for operator %s", operatorAddr)
	msgB := fmt.Sprintf("TellorLayer: Initial bridge signature B for operator %s", operatorAddr)
	msgBytesA := []byte(msgA)
	msgBytesB := []byte(msgB)

	// hash messages
	msgHashBytes32A := sha256.Sum256(msgBytesA)
	msgHashBytesA := msgHashBytes32A[:]

	msgHashBytes32B := sha256.Sum256(msgBytesB)
	msgHashBytesB := msgHashBytes32B[:]

	// hash the hash, since the keyring signer automatically hashes the message
	msgDoubleHashBytes32A := sha256.Sum256(msgHashBytesA)
	msgDoubleHashBytesA := msgDoubleHashBytes32A[:]

	msgDoubleHashBytes32B := sha256.Sum256(msgHashBytesB)
	msgDoubleHashBytesB := msgDoubleHashBytes32B[:]

	sigA, err := crypto.Sign(msgDoubleHashBytesA, privateKey)
	require.NoError(t, err)
	require.NotNil(t, sigA)

	sigB, err := crypto.Sign(msgDoubleHashBytesB, privateKey)
	require.NoError(t, err)
	require.NotNil(t, sigB)

	evmAddress, err := k.EVMAddressFromSignatures(ctx, sigA[:64], sigB[:64], operatorAddr)
	require.NoError(t, err)
	require.NotNil(t, evmAddress)

	require.Equal(t, addressExpected, evmAddress.Hex())

	badSigA := []byte("badSigA")
	badSigB := []byte("badSigB")

	_, err = k.EVMAddressFromSignatures(ctx, badSigA, sigB[:64], operatorAddr)
	require.Error(t, err)
	_, err = k.EVMAddressFromSignatures(ctx, sigA[:64], badSigB, operatorAddr)
	require.Error(t, err)
	_, err = k.EVMAddressFromSignatures(ctx, badSigA, badSigB, operatorAddr)
	require.Error(t, err)
}

func TestTryRecoverAddressWithBothIDs(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	// https://goethereumbook.org/signature-generate/
	privateKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	require.NotNil(t, privateKey)
	require.NoError(t, err)

	pkCoord := &ecdsa.PublicKey{
		X: privateKey.X,
		Y: privateKey.Y,
	}
	addressExpected := crypto.PubkeyToAddress(*pkCoord).Hex()

	msgA := "TellorLayer: Initial bridge signature A"
	msgB := "TellorLayer: Initial bridge signature B"
	msgBytesA := []byte(msgA)
	msgBytesB := []byte(msgB)

	// hash messages
	msgHashBytes32A := sha256.Sum256(msgBytesA)
	msgHashBytesA := msgHashBytes32A[:]

	msgHashBytes32B := sha256.Sum256(msgBytesB)
	msgHashBytesB := msgHashBytes32B[:]

	// hash the hash, since the keyring signer automatically hashes the message
	msgDoubleHashBytes32A := sha256.Sum256(msgHashBytesA)
	msgDoubleHashBytesA := msgDoubleHashBytes32A[:]

	msgDoubleHashBytes32B := sha256.Sum256(msgHashBytesB)
	msgDoubleHashBytesB := msgDoubleHashBytes32B[:]

	sigA, err := crypto.Sign(msgDoubleHashBytesA, privateKey)
	require.NoError(t, err)
	require.NotNil(t, sigA)

	sigB, err := crypto.Sign(msgDoubleHashBytesB, privateKey)
	require.NoError(t, err)
	require.NotNil(t, sigB)

	address, err := k.TryRecoverAddressWithBothIDs(sigA[:64], msgDoubleHashBytesA)
	require.NoError(t, err)
	require.NotNil(t, address)
	require.Equal(t, address[0].String(), addressExpected)

	// try with bad msg
	badMsg := []byte("badMsg")
	_, err = k.TryRecoverAddressWithBothIDs(sigA, badMsg)
	require.Error(t, err)

	// try with bad sig
	badSig := []byte("badSig")
	_, err = k.TryRecoverAddressWithBothIDs(badSig, msgDoubleHashBytesA)
	require.Error(t, err)
}

func TestSetEVMAddressByOperator(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	operatorAddr1 := testOperatorAddr1
	operatorAddr2 := testOperatorAddr2
	operatorAddr3 := "cosmosvaloper1zkue5gwhm5xyv4v5fa9lmcym7cwzaxtnpcl7jl"

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(2000),
			DelegatorShares: math.LegacyNewDec(2000),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: operatorAddr1,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: operatorAddr2,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(5000),
			DelegatorShares: math.LegacyNewDec(5000),
			Description:     stakingtypes.Description{Moniker: "validator3"},
			OperatorAddress: operatorAddr3,
		},
	}

	// Update EVM addresses for all validators including the new one
	evmAddresses := make([]types.EVMAddress, len(validators))
	for i, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)

		evmAddresses[i], err = k.OperatorToEVMAddressMap.Get(ctx, val.GetOperator())
		require.NoError(t, err)
		require.Equal(t, evmAddresses[i].EVMAddress, []byte(val.Description.Moniker))
	}
}

func TestSetBridgeValsetSignature(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	res, err := k.GetValidatorSetSignaturesFromStorage(ctx, 1)
	require.Error(t, err)
	require.Nil(t, res)

	err = k.SetBridgeValsetSignature(ctx, "operatorAddr1", 1, "abcd1234")
	require.Error(t, err)
	require.Nil(t, res)

	timestamp := uint64(100)
	err = k.BridgeValsetSignaturesMap.Set(ctx, timestamp, types.BridgeValsetSignatures{
		Signatures: [][]byte{
			[]byte("abcd1234"),
		},
	})
	require.NoError(t, err)
	err = k.SetBridgeValsetSignature(ctx, "operatorAddr1", timestamp, "abcd1234")
	require.Error(t, err)

	err = k.OperatorToEVMAddressMap.Set(ctx, "operatorAddr1", types.EVMAddress{
		EVMAddress: []byte("evmAddress1"),
	})
	require.NoError(t, err)
	err = k.SetBridgeValsetSignature(ctx, "operatorAddr1", timestamp, "abcd1234")
	require.Error(t, err)

	err = k.ValsetTimestampToIdxMap.Set(ctx, timestamp, types.CheckpointIdx{
		Index: 1,
	})
	require.NoError(t, err)
	err = k.SetBridgeValsetSignature(ctx, "operatorAddr1", timestamp, "abcd1234")
	require.Error(t, err)

	err = k.ValidatorCheckpointIdxMap.Set(ctx, 0, types.CheckpointTimestamp{
		Timestamp: timestamp,
	})
	require.NoError(t, err)
	err = k.SetBridgeValsetSignature(ctx, "operatorAddr1", timestamp, "abcd1234")
	require.Error(t, err)

	err = k.BridgeValsetByTimestampMap.Set(ctx, timestamp, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           100,
			},
		},
	})
	require.NoError(t, err)

	err = k.SetBridgeValsetSignature(ctx, "operatorAddr1", timestamp, "abcd1234")
	require.NoError(t, err)

	sigMap, err := k.BridgeValsetSignaturesMap.Get(ctx, timestamp)
	require.NoError(t, err)
	require.NotNil(t, sigMap)
	require.Equal(t, sigMap.Signatures, [][]byte{
		[]byte("abcd1234"),
	})
}

func TestGetEVMAddressByOperator(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	operatorAddr1 := testOperatorAddr1
	operatorAddr2 := testOperatorAddr2

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(2000),
			DelegatorShares: math.LegacyNewDec(2000),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: operatorAddr1,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: operatorAddr2,
		},
	}

	for _, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)

		evmAddress, err := k.GetEVMAddressByOperator(ctx, val.OperatorAddress)
		require.NoError(t, err)
		require.Equal(t, evmAddress, []byte(val.Description.Moniker))
	}

	addr, err := k.GetEVMAddressByOperator(ctx, "badAddress")
	require.Error(t, err)
	require.Nil(t, addr)
}

func TestGetValidatorCheckpointParamsFromStorage(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	_, err := k.GetValidatorCheckpointParamsFromStorage(ctx, 0)
	require.Error(t, err)

	timestamp := uint64(100)
	err = k.ValidatorCheckpointParamsMap.Set(ctx, timestamp, types.ValidatorCheckpointParams{
		Checkpoint:     []byte("checkpoint"),
		ValsetHash:     []byte("valsetHash"),
		Timestamp:      timestamp,
		PowerThreshold: uint64(100),
	})
	require.NoError(t, err)

	res, err := k.GetValidatorCheckpointParamsFromStorage(ctx, timestamp)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, res.Checkpoint, []byte("checkpoint"))
	require.Equal(t, res.ValsetHash, []byte("valsetHash"))
	require.Equal(t, res.Timestamp, (timestamp))
	require.Equal(t, res.PowerThreshold, uint64(100))
}

func TestSetOracleAttestation(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	err := k.SetOracleAttestation(ctx, "operatorAddr1", []byte("1"), []byte("abcd1234"))
	require.Error(t, err)

	err = k.OperatorToEVMAddressMap.Set(ctx, "operatorAddr1", types.EVMAddress{
		EVMAddress: []byte("evmAddress1"),
	})
	require.NoError(t, err)
	err = k.SetOracleAttestation(ctx, "operatorAddr1", []byte("1"), []byte("abcd1234"))
	require.Error(t, err)

	err = k.BridgeValset.Set(ctx, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("evmAddress1"),
				Power:           100,
			},
		},
	})
	require.NoError(t, err)
	err = k.SnapshotToAttestationsMap.Set(ctx, []byte("1"), types.OracleAttestations{
		Attestations: [][]byte{
			[]byte("abcd1234"),
		},
	})
	require.NoError(t, err)
	err = k.SetOracleAttestation(ctx, "operatorAddr1", []byte("1"), []byte("abcd1234"))
	require.NoError(t, err)
}

func TestGetAttestationRequestsByHeight(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	res, err := k.GetAttestationRequestsByHeight(ctx, 1)
	require.Error(t, err)
	require.Nil(t, res)

	err = k.AttestRequestsByHeightMap.Set(ctx, 1, types.AttestationRequests{
		Requests: []*types.AttestationRequest{
			{
				Snapshot: []byte("snapshot"),
			},
		},
	})
	require.NoError(t, err)

	res, err = k.GetAttestationRequestsByHeight(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, res.Requests, []*types.AttestationRequest{
		{
			Snapshot: []byte("snapshot"),
		},
	})
}

func TestGetLatestCheckpointIndex(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	// if error getting latest index, return 0
	index, err := k.GetLatestCheckpointIndex(ctx)
	require.Error(t, err)
	require.Equal(t, index, uint64(0))

	err = k.LatestCheckpointIdx.Set(ctx, types.CheckpointIdx{
		Index: 1,
	})
	require.NoError(t, err)
	index, err = k.GetLatestCheckpointIndex(ctx)
	require.NoError(t, err)
	require.Equal(t, index, uint64(1))
}

func TestGetValidatorDidSignCheckpoint(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	// if err getting valset index, return false, -1, err
	didSign, prevIndex, err := k.GetValidatorDidSignCheckpoint(ctx, "operatorAddr1", 1)
	require.Error(t, err)
	require.False(t, didSign)
	require.Equal(t, prevIndex, int64(-1))

	// if valset index is 0, return no err, false, -1
	timestamp := uint64(100)
	err = k.ValsetTimestampToIdxMap.Set(ctx, timestamp, types.CheckpointIdx{
		Index: 0,
	})
	require.NoError(t, err)
	didSign, prevIndex, err = k.GetValidatorDidSignCheckpoint(ctx, "operatorAddr1", timestamp)
	require.NoError(t, err)
	require.False(t, didSign)
	require.Equal(t, prevIndex, int64(-1))

	// set prev checkpoint maps
	err = k.ValidatorCheckpointIdxMap.Set(ctx, 0, types.CheckpointTimestamp{
		Timestamp: timestamp,
	})
	require.NoError(t, err)
	err = k.BridgeValsetByTimestampMap.Set(ctx, timestamp, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("evmAddress1"),
				Power:           100,
			},
		},
	})
	require.NoError(t, err)
	err = k.ValsetTimestampToIdxMap.Set(ctx, timestamp, types.CheckpointIdx{
		Index: 1,
	})
	require.NoError(t, err)

	// set index 1 maps
	timestamp2 := timestamp + 1
	err = k.ValsetTimestampToIdxMap.Set(ctx, timestamp2, types.CheckpointIdx{
		Index: 1,
	})
	require.NoError(t, err)
	err = k.OperatorToEVMAddressMap.Set(ctx, "operatorAddr1", types.EVMAddress{
		EVMAddress: []byte("evmAddress1"),
	})
	require.NoError(t, err)
	err = k.BridgeValsetSignaturesMap.Set(ctx, timestamp2, types.BridgeValsetSignatures{
		Signatures: [][]byte{
			[]byte("abcd1234"),
		},
	})
	require.NoError(t, err)
	didSign, prevValsetIndex, err := k.GetValidatorDidSignCheckpoint(ctx, "operatorAddr1", timestamp2)
	require.NoError(t, err)
	require.True(t, didSign)
	require.Equal(t, prevValsetIndex, int64(0))
}

func TestCreateSnapshot(t *testing.T) {
	k, _, _, ok, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	timestamp := time.Now()
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockTime(timestamp)

	qId, err := hex.DecodeString("efa84ae5ea9eb0545e159f78f0a44911ac5a81ecb6ff0c4e32107bcfc66c4baa")
	require.NoError(t, err)
	ok.On("GetAggregateByTimestamp", sdkCtx, qId, uint64(timestamp.UnixMilli())).Return(oracletypes.Aggregate{
		QueryId:        qId,
		AggregateValue: "5000",
		AggregatePower: uint64(100),
	}, nil)

	err = k.ValidatorCheckpoint.Set(sdkCtx, types.ValidatorCheckpoint{
		Checkpoint: []byte("checkpoint"),
	})
	require.NoError(t, err)

	ok.On("GetTimestampBefore", sdkCtx, qId, timestamp).Return(timestamp.Add(-1*time.Hour), nil)
	ok.On("GetTimestampAfter", sdkCtx, qId, timestamp).Return(timestamp.Add(1*time.Hour), nil)

	err = k.BridgeValset.Set(sdkCtx, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           100,
			},
		},
	})
	require.NoError(t, err)

	// set last consensus timestamp
	err = k.AttestSnapshotDataMap.Set(sdkCtx, qId, types.AttestationSnapshotData{
		LastConsensusTimestamp: 100,
	})
	require.NoError(t, err)

	// set GetCurrentValidatorSetTimestamp
	err = k.LatestCheckpointIdx.Set(sdkCtx, types.CheckpointIdx{
		Index: 1,
	})
	require.NoError(t, err)
	err = k.ValidatorCheckpointIdxMap.Set(sdkCtx, 1, types.CheckpointTimestamp{
		Timestamp: 100,
	})
	require.NoError(t, err)

	// ValidatorCheckpointParamsMap
	err = k.ValidatorCheckpointParamsMap.Set(sdkCtx, 100, types.ValidatorCheckpointParams{
		Checkpoint:     []byte("checkpoint"),
		ValsetHash:     []byte("valsetHash"),
		Timestamp:      100,
		PowerThreshold: 100,
	})
	require.NoError(t, err)

	err = k.CreateSnapshot(sdkCtx, qId, timestamp, false)
	require.NoError(t, err)

	// check if snapshot is created
	attReq, err := k.AttestRequestsByHeightMap.Get(sdkCtx, 0)
	require.NoError(t, err)
	require.NotNil(t, attReq)

	// get snapshot by  report
	querier := keeper.NewQuerier(k)
	req := &types.QueryGetSnapshotsByReportRequest{
		QueryId:   hex.EncodeToString(qId),
		Timestamp: strconv.FormatUint(uint64(timestamp.UnixMilli()), 10),
	}
	snapshots, err := querier.GetSnapshotsByReport(sdkCtx, req)
	require.NoError(t, err)
	require.NotNil(t, snapshots)

	// get attestations by snapshot
	snapshotBytes, err := hex.DecodeString(snapshots.Snapshots[0])
	require.NoError(t, err)
	attestations, err := k.SnapshotToAttestationsMap.Get(sdkCtx, snapshotBytes)
	require.NoError(t, err)
	require.NotNil(t, attestations)
	require.Equal(t, len(attestations.Attestations), 1)
}

func TestCreateNewReportSnapshots(t *testing.T) {
	k, _, _, ok, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	timestamp := sdkCtx.BlockTime()
	timestampPlus1 := timestamp.Add(time.Second)

	queryId := []byte("queryId")
	ok.On("GetAggregatedReportsByHeight", ctx, uint64(0)).Return([]oracletypes.Aggregate{
		{
			Height:         0,
			QueryId:        queryId,
			AggregateValue: "5000",
			AggregatePower: uint64(100),
		},
	}, nil)
	ok.On("GetTimestampBefore", sdkCtx, queryId, timestampPlus1).Return(timestamp, nil).Once()
	ok.On("GetTimestampBefore", sdkCtx, queryId, timestamp).Return(timestamp, nil)
	ok.On("GetAggregateByTimestamp", ctx, queryId, uint64(timestamp.UnixMilli())).Return(oracletypes.Aggregate{
		QueryId:        queryId,
		AggregateValue: "5000",
		AggregatePower: uint64(100),
	}, nil)

	err := k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{
		Checkpoint: []byte("checkpoint"),
	})
	require.NoError(t, err)
	ok.On("GetTimestampAfter", ctx, queryId, timestamp).Return(timestamp.Add(1*time.Hour), nil)
	err = k.BridgeValset.Set(ctx, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           100,
			},
		},
	})
	require.NoError(t, err)

	// set last consensus timestamp
	err = k.AttestSnapshotDataMap.Set(sdkCtx, queryId, types.AttestationSnapshotData{
		LastConsensusTimestamp: 100,
	})
	require.NoError(t, err)

	// set GetCurrentValidatorSetTimestamp
	err = k.LatestCheckpointIdx.Set(sdkCtx, types.CheckpointIdx{
		Index: 1,
	})
	require.NoError(t, err)
	err = k.ValidatorCheckpointIdxMap.Set(sdkCtx, 1, types.CheckpointTimestamp{
		Timestamp: 100,
	})
	require.NoError(t, err)

	// ValidatorCheckpointParamsMap
	err = k.ValidatorCheckpointParamsMap.Set(sdkCtx, 100, types.ValidatorCheckpointParams{
		Checkpoint:     []byte("checkpoint"),
		ValsetHash:     []byte("valsetHash"),
		Timestamp:      100,
		PowerThreshold: 100,
	})
	require.NoError(t, err)

	err = k.CreateNewReportSnapshots(ctx)
	require.NoError(t, err)
}

func TestCreateSnapshotDisputedReport(t *testing.T) {
	k, _, _, ok, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	timestamp := time.Now()
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockTime(timestamp)

	qId, err := hex.DecodeString("efa84ae5ea9eb0545e159f78f0a44911ac5a81ecb6ff0c4e32107bcfc66c4baa")
	require.NoError(t, err)
	ok.On("GetAggregateByTimestamp", sdkCtx, qId, uint64(timestamp.UnixMilli())).Return(oracletypes.Aggregate{
		QueryId:        qId,
		AggregateValue: "5000",
		AggregatePower: uint64(100),
		Flagged:        true,
	}, nil)

	err = k.CreateSnapshot(sdkCtx, qId, timestamp, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "report is flagged as dispute evidence")

	// make sure no snapshot is created
	snapshotExists, err := k.AttestRequestsByHeightMap.Has(sdkCtx, 0)
	require.NoError(t, err)
	require.False(t, snapshotExists)
}

func TestEncodeOracleAttestationData(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	queryId := []byte("queryId")
	value := "1000"
	timestamp := uint64(100)
	power := uint64(1000)
	tsBefore := uint64(90)
	tsAfter := uint64(110)
	checkpoint := []byte("checkpoint")
	attestationTimestamp := uint64(100)
	lastConsensusTimestamp := uint64(100)
	res, err := k.EncodeOracleAttestationData(queryId, value, timestamp, power, tsBefore, tsAfter, checkpoint, attestationTimestamp, lastConsensusTimestamp)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestGetCurrentValidatorSetTimestamp(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	testCases := []struct {
		name              string
		setup             func()
		expectedTimestamp uint64
		err               bool
	}{
		{
			name: "LatestCheckpointIdx not set",
			err:  true,
		},
		{
			name: "ValidatorCheckpointIdxMap not set",
			setup: func() {
				err := k.LatestCheckpointIdx.Set(ctx, types.CheckpointIdx{
					Index: 1,
				})
				require.NoError(t, err)
			},
			err: true,
		},
		{
			name: "all good",
			setup: func() {
				err := k.LatestCheckpointIdx.Set(ctx, types.CheckpointIdx{
					Index: 1,
				})
				require.NoError(t, err)
				err = k.ValidatorCheckpointIdxMap.Set(ctx, 1, types.CheckpointTimestamp{
					Timestamp: 100,
				})
				require.NoError(t, err)
			},
			expectedTimestamp: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			timestamp, err := k.GetCurrentValidatorSetTimestamp(ctx)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, timestamp, tc.expectedTimestamp)
			}
		})
	}
}

func TestGetValidatorSetIndexByTimestamp(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	testCases := []struct {
		name          string
		setup         func()
		timestamp     uint64
		expectedIndex uint64
		err           bool
	}{
		{
			name: "ValsetTimestampToIdxMap not set",
			err:  true,
		},
		{
			name:      "all good",
			timestamp: 100,
			setup: func() {
				err := k.ValsetTimestampToIdxMap.Set(ctx, 100, types.CheckpointIdx{
					Index: 1,
				})
				require.NoError(t, err)
			},
			expectedIndex: 1,
		},
		{
			name:      "max uint64",
			timestamp: ^uint64(0),
			setup: func() {
				maxUint := ^uint64(0)
				err := k.ValsetTimestampToIdxMap.Set(ctx, maxUint, types.CheckpointIdx{
					Index: 2,
				})
				require.NoError(t, err)
			},
			expectedIndex: 2,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			index, err := k.GetValidatorSetIndexByTimestamp(ctx, tc.timestamp)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, index, tc.expectedIndex)
			}
		})
	}
}
