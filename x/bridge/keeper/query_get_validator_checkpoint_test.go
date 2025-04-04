package keeper_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestGetValidatorCheckpoint(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getCheckpointResponse, err := keeper.NewQuerier(k).GetValidatorCheckpoint(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getCheckpointResponse)

	getCheckpointResponse, err = keeper.NewQuerier(k).GetValidatorCheckpoint(ctx, &types.QueryGetValidatorCheckpointRequest{})
	require.ErrorContains(t, err, "failed to get validator checkpoint")
	require.Nil(t, getCheckpointResponse)

	err = k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{
		Checkpoint: []byte("checkpoint"),
	})
	require.NoError(t, err)

	getCheckpointResponse, err = keeper.NewQuerier(k).GetValidatorCheckpoint(ctx, &types.QueryGetValidatorCheckpointRequest{})
	require.NoError(t, err)
	require.Equal(t, getCheckpointResponse.ValidatorCheckpoint, hex.EncodeToString([]byte("checkpoint")))
}
