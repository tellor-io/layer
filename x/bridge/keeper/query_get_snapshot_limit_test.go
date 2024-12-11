package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestGetSnapshotLimit(t *testing.T) {
	k, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	// Test error case: nil request
	response, err := keeper.NewQuerier(k).GetSnapshotLimit(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, response)

	// Test error case: snapshot limit not set
	err = k.SnapshotLimit.Remove(ctx)
	require.NoError(t, err)
	response, err = keeper.NewQuerier(k).GetSnapshotLimit(ctx, &types.QueryGetSnapshotLimitRequest{})
	require.ErrorContains(t, err, "snapshot limit not found")
	require.Nil(t, response)

	// Set snapshot limit
	snapshotLimit := types.SnapshotLimit{Limit: 10}
	err = k.SnapshotLimit.Set(ctx, snapshotLimit)
	require.NoError(t, err)

	// Test valid case
	response, err = keeper.NewQuerier(k).GetSnapshotLimit(ctx, &types.QueryGetSnapshotLimitRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryGetSnapshotLimitResponse{Limit: 10}, response)
}
