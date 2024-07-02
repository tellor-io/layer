package keeper_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"

	math "cosmossdk.io/math"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestGetEvmValidators(t *testing.T) {
	k, _, _, _, _, sk, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getEvmValsResponse, err := keeper.NewQuerier(k).GetEvmValidators(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getEvmValsResponse)

	sk.On("GetAllValidators", ctx).Return([]stakingtypes.Validator{}, nil).Once()

	getEvmValsResponse, err = keeper.NewQuerier(k).GetEvmValidators(ctx, &types.QueryGetEvmValidatorsRequest{})
	require.ErrorContains(t, err, "failed to get current validators")
	require.Nil(t, getEvmValsResponse)

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(200 * 1e6),
			DelegatorShares: math.LegacyNewDec(200 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: "operatorAddr1",
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(100 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
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

	getEvmValsResponse, err = keeper.NewQuerier(k).GetEvmValidators(ctx, &types.QueryGetEvmValidatorsRequest{})
	require.NoError(t, err)
	require.Equal(t, getEvmValsResponse.BridgeValidatorSet[0].EthereumAddress, hex.EncodeToString([]byte("validator1")))
	require.Equal(t, getEvmValsResponse.BridgeValidatorSet[0].Power, uint64(200))
	require.Equal(t, getEvmValsResponse.BridgeValidatorSet[1].EthereumAddress, hex.EncodeToString([]byte("validator2")))
	require.Equal(t, getEvmValsResponse.BridgeValidatorSet[1].Power, uint64(100))
}
