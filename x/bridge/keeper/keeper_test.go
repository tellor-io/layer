package keeper_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/mocks"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
)

func setupKeeper(tb testing.TB) (keeper.Keeper, *mocks.AccountKeeper, *mocks.BankKeeper, *mocks.OracleKeeper, *mocks.ReporterKeeper, *mocks.StakingKeeper, context.Context) {
	tb.Helper()
	k, ak, bk, ok, rk, sk, ctx := keepertest.BridgeKeeper(tb)
	return k, ak, bk, ok, rk, sk, ctx
}

func TestKeeper(t *testing.T) {
	k, ak, bk, ok, rk, sk, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ak)
	require.NotNil(t, bk)
	require.NotNil(t, ok)
	require.NotNil(t, rk)
	require.NotNil(t, sk)
	require.NotNil(t, ctx)
}

func TestGetCurrentValidatorsEVMCompatible(t *testing.T) {
	k, _, _, _, _, sk, ctx := setupKeeper(t)

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(3000),
			DelegatorShares: math.LegacyNewDec(3000),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: "operatorAddr1",
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: "operatorAddr2",
		},
	}

	evmAddresses := make([]bridgetypes.EVMAddress, len(validators))
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
		require.Equal(t, bridgeValSet[i].Power, uint64(validators[i].GetConsensusPower(math.NewInt(10))))
		require.Equal(t, bridgeValSet[i].EthereumAddress, evmAddresses[i].EVMAddress)
	}
}

func TestGetCurrentValidatorsEVMCompatibleNoValidators(t *testing.T) {
	k, _, _, _, _, sk, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	validators := []stakingtypes.Validator{}

	sk.On("GetAllValidators", ctx).Return(validators, nil)
	bridgeValSet, err := k.GetCurrentValidatorsEVMCompatible(ctx)
	require.ErrorContains(t, err, "no validators found")
	require.Nil(t, bridgeValSet)
}

func TestGetCurrentValidatorsEVMCompatibleEqualPowers(t *testing.T) {
	k, _, _, _, _, sk, ctx := setupKeeper(t)

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: "operatorAddr1",
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: "operatorAddr2",
		},
	}

	evmAddresses := make([]bridgetypes.EVMAddress, len(validators))
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
		require.Equal(t, bridgeValSet[i].Power, uint64(validators[i].GetConsensusPower(math.NewInt(10))))
		require.Equal(t, bridgeValSet[i].EthereumAddress, evmAddresses[i].EVMAddress)
		require.LessOrEqual(t, string(bridgeValSet[i].EthereumAddress), string(bridgeValSet[i+1].EthereumAddress))
	}
}

func TestGetCurrentValidatorSetEVMCompatible(t *testing.T) {
	k, _, _, _, _, sk, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(2000),
			DelegatorShares: math.LegacyNewDec(2000),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: "operatorAddr1",
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: "operatorAddr2",
		},
	}

	evmAddresses := make([]bridgetypes.EVMAddress, len(validators))
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
		require.Equal(t, bridgeValidatorSet.BridgeValidatorSet[i].Power, uint64(validators[i].GetConsensusPower(math.NewInt(10))))
	}
}

func TestCompareBridgeValidators(t *testing.T) {
	k, _, _, _, _, sk, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(2000),
			DelegatorShares: math.LegacyNewDec(2000),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: "operatorAddr1",
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: "operatorAddr2",
		},
	}

	evmAddresses := make([]bridgetypes.EVMAddress, len(validators))
	for i, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)

		evmAddresses[i], err = k.OperatorToEVMAddressMap.Get(ctx, val.GetOperator())
		require.NoError(t, err)
		require.Equal(t, evmAddresses[i].EVMAddress, []byte(val.Description.Moniker))
	}

	sk.On("GetAllValidators", ctx).Return(validators, nil).Times(2)

	// since BridgeValSet has not been set, should error
	lastSavedBridgeValidators, err := k.BridgeValset.Get(ctx)
	require.Error(t, err)
	require.Nil(t, lastSavedBridgeValidators.BridgeValidatorSet)

	// set BridgeValSet, should hit false because no BridgeValSet exists yet
	res, err := k.CompareBridgeValidators(ctx)
	require.NoError(t, err)
	require.False(t, res)

	// should return 2 validators
	lastSavedBridgeValidators, err = k.BridgeValset.Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, lastSavedBridgeValidators)
	require.Equal(t, len(lastSavedBridgeValidators.BridgeValidatorSet), 2)

	// should return false since valset has not changed
	res, err = k.CompareBridgeValidators(ctx)
	require.NoError(t, err)
	require.False(t, res)

	// Append the third validator
	validators = []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(2000),
			DelegatorShares: math.LegacyNewDec(2000),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: "operatorAddr1",
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: "operatorAddr2",
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(5000),
			DelegatorShares: math.LegacyNewDec(5000),
			Description:     stakingtypes.Description{Moniker: "validator3"},
			OperatorAddress: "operatorAddr3",
		},
	}

	// Update EVM addresses for all validators including the new one
	evmAddresses = make([]bridgetypes.EVMAddress, len(validators))
	for i, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)

		evmAddresses[i], err = k.OperatorToEVMAddressMap.Get(ctx, val.GetOperator())
		require.NoError(t, err)
		require.Equal(t, evmAddresses[i].EVMAddress, []byte(val.Description.Moniker))
	}

	sk.On("GetAllValidators", ctx).Return(validators, nil)

	currentValidatorSetEVMCompatible, err := k.GetCurrentValidatorSetEVMCompatible(ctx)
	require.NoError(t, err)
	require.NotNil(t, currentValidatorSetEVMCompatible)

	// Check for third val
	require.Equal(t, len(currentValidatorSetEVMCompatible.BridgeValidatorSet), 3)

	err = k.BridgeValset.Set(ctx, *currentValidatorSetEVMCompatible)
	require.NoError(t, err)

	// should return true since valset has changed
	res, err = k.CompareBridgeValidators(ctx)
	require.NoError(t, err)
	fmt.Println("res: ", res)

}

func TestSetBridgeValidatorParams(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	// err := k.SetBridgeValidatorParams(ctx, &bridgetypes.BridgeValidatorSet{})
	// require.Error(t, err)

	bridgeValSet := bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: []*bridgetypes.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           1000,
			},
		},
	}

	err := k.SetBridgeValidatorParams(ctx, &bridgeValSet)
	require.NoError(t, err)

	// todo: check stores

	bridgeValSet = bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: []*bridgetypes.BridgeValidator{
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

	// todo: recheck stores
}

func TestCalculateValidatorSetCheckpoint(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
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
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	res, err := k.GetValidatorCheckpointFromStorage(ctx)
	require.Error(t, err)
	require.Nil(t, res)

	bridgeValSet := bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: []*bridgetypes.BridgeValidator{
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
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	res, err := k.GetValidatorTimestampByIdxFromStorage(ctx, 0)
	require.Error(t, err)
	require.Equal(t, res.Timestamp, uint64(0))

	bridgeValSet := bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: []*bridgetypes.BridgeValidator{
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
	validatorTimestamp := uint64(sdkCtx.BlockTime().Unix())
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

}

func TestGetValidatorSetSignaturesFromStorage(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	res, err := k.GetValidatorSetSignaturesFromStorage(ctx, 0)
	require.Error(t, err)
	require.Nil(t, res)

	bridgeValSet := bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: []*bridgetypes.BridgeValidator{
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
	fmt.Println("res: ", res)
}

// needs finished
func TestEncodeAndHashValidatorSet(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	bridgeValSet := bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: []*bridgetypes.BridgeValidator{
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

	// verify encoding
}

// needs finished
func TestPowerDiff(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	bridgeValSetA := bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: []*bridgetypes.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           1000,
			},
		},
	}

	bridgeValSetB := bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: []*bridgetypes.BridgeValidator{
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

	relativeDiff := k.PowerDiff(ctx, bridgeValSetA, bridgeValSetB)
	fmt.Println("relativeDiffAtoB: ", relativeDiff)
	require.Equal(t, relativeDiff, float64(2))

	bridgeValSetC := bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: []*bridgetypes.BridgeValidator{
			{
				EthereumAddress: []byte("validator3"),
				Power:           0,
			},
		},
	}

	relativeDiff = k.PowerDiff(ctx, bridgeValSetB, bridgeValSetC)
	fmt.Println("relativeDiffBtoC: ", relativeDiff)

	relativeDiff = k.PowerDiff(ctx, bridgeValSetC, bridgeValSetB)
	fmt.Println("relativeDiffCtoB: ", relativeDiff)

	relativeDiff = k.PowerDiff(ctx, bridgeValSetB, bridgeValSetB)
	fmt.Println("relativeDiffBtoB: ", relativeDiff)
}

func TestEVMAddressFromSignatures(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

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

	evmAddress, err := k.EVMAddressFromSignatures(ctx, sigA, sigB)
	require.NoError(t, err)
	require.NotNil(t, evmAddress)

	require.Equal(t, addressExpected, evmAddress.Hex())

}

// needs finished
func TestTryRecoverAddressWithBothIDs(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

}

func TestSetEVMAddressByOperator(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(2000),
			DelegatorShares: math.LegacyNewDec(2000),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: "operatorAddr1",
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(1000),
			DelegatorShares: math.LegacyNewDec(1000),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: "operatorAddr2",
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(5000),
			DelegatorShares: math.LegacyNewDec(5000),
			Description:     stakingtypes.Description{Moniker: "validator3"},
			OperatorAddress: "operatorAddr3",
		},
	}

	// Update EVM addresses for all validators including the new one
	evmAddresses := make([]bridgetypes.EVMAddress, len(validators))
	for i, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)

		evmAddresses[i], err = k.OperatorToEVMAddressMap.Get(ctx, val.GetOperator())
		require.NoError(t, err)
		require.Equal(t, evmAddresses[i].EVMAddress, []byte(val.Description.Moniker))
	}

}

// needs finished
func TestSetBridgeValsetSignature(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

}
