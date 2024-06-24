package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestGetValidatorTimestampByIndex(t *testing.T) {
	k, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getValResponse, err := keeper.NewQuerier(k).GetValidatorTimestampByIndex(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getValResponse)

	getValResponse, err = keeper.NewQuerier(k).GetValidatorTimestampByIndex(ctx, &types.QueryGetValidatorTimestampByIndexRequest{})
	require.ErrorContains(t, err, "failed to get validator timestamp by index")
	require.Nil(t, getValResponse)

	err = k.ValidatorCheckpointIdxMap.Set(ctx, 0, types.CheckpointTimestamp{
		Timestamp: 10,
	})
	require.NoError(t, err)

	getValResponse, err = keeper.NewQuerier(k).GetValidatorTimestampByIndex(ctx, &types.QueryGetValidatorTimestampByIndexRequest{
		Index: 0,
	})
	require.NoError(t, err)
	require.Equal(t, getValResponse.Timestamp, int64(10))
}
