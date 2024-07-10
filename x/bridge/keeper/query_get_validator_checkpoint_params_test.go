package keeper_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestGetValidatorCheckpointParams(t *testing.T) {
	k, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getCheckpointParamsResponse, err := keeper.NewQuerier(k).GetValidatorCheckpointParams(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getCheckpointParamsResponse)

	getCheckpointParamsResponse, err = keeper.NewQuerier(k).GetValidatorCheckpointParams(ctx, &types.QueryGetValidatorCheckpointParamsRequest{})
	require.ErrorContains(t, err, "failed to get validator checkpoint params")
	require.Nil(t, getCheckpointParamsResponse)

	err = k.ValidatorCheckpointParamsMap.Set(ctx, 0, types.ValidatorCheckpointParams{
		Checkpoint:     []byte("checkpoint"),
		ValsetHash:     []byte("valsetHash"),
		Timestamp:      0,
		PowerThreshold: 10,
	})
	require.NoError(t, err)

	getCheckpointParamsResponse, err = keeper.NewQuerier(k).GetValidatorCheckpointParams(ctx, &types.QueryGetValidatorCheckpointParamsRequest{
		Timestamp: 0,
	})
	require.NoError(t, err)
	require.Equal(t, getCheckpointParamsResponse.Checkpoint, hex.EncodeToString([]byte("checkpoint")))
	require.Equal(t, getCheckpointParamsResponse.ValsetHash, hex.EncodeToString([]byte("valsetHash")))
	require.Equal(t, getCheckpointParamsResponse.Timestamp, int64(0))
	require.Equal(t, getCheckpointParamsResponse.PowerThreshold, int64(10))
}
