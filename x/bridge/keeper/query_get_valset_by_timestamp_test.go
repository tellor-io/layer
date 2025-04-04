package keeper_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestGetValsetByTimestamp(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getValsetResponse, err := keeper.NewQuerier(k).GetValsetByTimestamp(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getValsetResponse)

	getValsetResponse, err = keeper.NewQuerier(k).GetValsetByTimestamp(ctx, &types.QueryGetValsetByTimestampRequest{})
	require.ErrorContains(t, err, "failed to get eth address")
	require.Nil(t, getValsetResponse)

	// set valset
	valset := types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{
			{
				EthereumAddress: []byte("validator1"),
				Power:           1000,
			},
		},
	}
	err = k.BridgeValsetByTimestampMap.Set(ctx, 0, valset)
	require.NoError(t, err)

	getValsetResponse, err = keeper.NewQuerier(k).GetValsetByTimestamp(ctx, &types.QueryGetValsetByTimestampRequest{
		Timestamp: 0,
	})
	require.NoError(t, err)
	require.Equal(t, getValsetResponse.BridgeValidatorSet[0].Power, valset.BridgeValidatorSet[0].Power)
	require.Equal(t, common.BytesToAddress(valset.BridgeValidatorSet[0].EthereumAddress).String(), getValsetResponse.BridgeValidatorSet[0].EthereumAddress)
}
