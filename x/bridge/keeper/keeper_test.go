package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/mocks"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
)

func setupKeeper(t testing.TB) (keeper.Keeper, *mocks.AccountKeeper, *mocks.BankKeeper, *mocks.OracleKeeper, *mocks.ReporterKeeper, *mocks.StakingKeeper, context.Context) {
	k, ak, bk, ok, rk, sk, ctx := keepertest.BridgeKeeper(t)
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

	sk.On("GetAllValidators", ctx).Return(validators, nil)

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
	validators = append(validators, stakingtypes.Validator{
		Jailed:          false,
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(5000),
		DelegatorShares: math.LegacyNewDec(5000),
		Description:     stakingtypes.Description{Moniker: "validator3"},
		OperatorAddress: "operatorAddr3",
	})

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

	// Check if the third validator is seen
	require.Equal(t, len(currentValidatorSetEVMCompatible.BridgeValidatorSet), 3) // Should now see 3 validators

	err = k.BridgeValset.Set(ctx, *currentValidatorSetEVMCompatible)
	require.NoError(t, err)

	// should return true since valset has changed
	res, err = k.CompareBridgeValidators(ctx)
	require.NoError(t, err)
	fmt.Println("res: ", res)

}
