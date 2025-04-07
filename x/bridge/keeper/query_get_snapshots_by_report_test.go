package keeper_test

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestGetSnapshotsByReport(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getSnapshotResponse, err := keeper.NewQuerier(k).GetSnapshotsByReport(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getSnapshotResponse)

	getSnapshotResponse, err = keeper.NewQuerier(k).GetSnapshotsByReport(ctx, &types.QueryGetSnapshotsByReportRequest{})
	require.ErrorContains(t, err, "invalid")
	require.Nil(t, getSnapshotResponse)

	queryIdBytes, err := hex.DecodeString("abcd1234")
	require.NoError(t, err)
	timestamp := time.Unix(0, 0)
	key := crypto.Keccak256([]byte(hex.EncodeToString(queryIdBytes) + fmt.Sprint(timestamp.Unix())))

	err = k.AttestSnapshotsByReportMap.Set(ctx, key, types.AttestationSnapshots{
		Snapshots: [][]byte{[]byte("snapshot")},
	})
	require.NoError(t, err)

	getSnapshotResponse, err = keeper.NewQuerier(k).GetSnapshotsByReport(ctx, &types.QueryGetSnapshotsByReportRequest{
		QueryId:   "abcd1234",
		Timestamp: "0",
	})
	require.NoError(t, err)
	require.Equal(t, getSnapshotResponse.Snapshots, []string{hex.EncodeToString([]byte("snapshot"))})
}
