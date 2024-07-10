package keeper_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestGetAttestationsBySnapshot(t *testing.T) {
	k, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getAttBySnapResponse, err := keeper.NewQuerier(k).GetAttestationsBySnapshot(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getAttBySnapResponse)

	getAttBySnapResponse, err = keeper.NewQuerier(k).GetAttestationsBySnapshot(ctx, &types.QueryGetAttestationsBySnapshotRequest{
		Snapshot: "a",
	})
	require.ErrorContains(t, err, "failed to decode snapshot")
	require.Nil(t, getAttBySnapResponse)

	getAttBySnapResponse, err = keeper.NewQuerier(k).GetAttestationsBySnapshot(ctx, &types.QueryGetAttestationsBySnapshotRequest{
		Snapshot: "abcd1234",
	})
	require.ErrorContains(t, err, "attestations not found for snapshot")
	require.Nil(t, getAttBySnapResponse)

	snapshot, err := utils.QueryBytesFromString("abcd1234")
	require.NoError(t, err)
	err = k.SnapshotToAttestationsMap.Set(ctx, snapshot, types.OracleAttestations{
		Attestations: [][]byte{[]byte("attestation")},
	})
	require.NoError(t, err)

	getAttBySnapResponse, err = keeper.NewQuerier(k).GetAttestationsBySnapshot(ctx, &types.QueryGetAttestationsBySnapshotRequest{
		Snapshot: "abcd1234",
	})
	require.NoError(t, err)
	require.Equal(t, getAttBySnapResponse.Attestations[0], hex.EncodeToString([]byte("attestation")))
}
