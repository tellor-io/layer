package keeper_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestGetEvmAddressByValidatorAddress(t *testing.T) {
	k, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getEvmAddrResponse, err := keeper.NewQuerier(k).GetEvmAddressByValidatorAddress(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getEvmAddrResponse)

	getEvmAddrResponse, err = keeper.NewQuerier(k).GetEvmAddressByValidatorAddress(ctx, &types.QueryGetEvmAddressByValidatorAddressRequest{})
	require.ErrorContains(t, err, "failed to get eth address")
	require.Nil(t, getEvmAddrResponse)

	err = k.SetEVMAddressByOperator(ctx, "operatorAddr1", []byte("validator1"))
	require.NoError(t, err)

	getEvmAddrResponse, err = keeper.NewQuerier(k).GetEvmAddressByValidatorAddress(ctx, &types.QueryGetEvmAddressByValidatorAddressRequest{
		ValidatorAddress: "operatorAddr1",
	})
	require.NoError(t, err)
	require.Equal(t, getEvmAddrResponse.EvmAddress, hex.EncodeToString([]byte("validator1")))
}
