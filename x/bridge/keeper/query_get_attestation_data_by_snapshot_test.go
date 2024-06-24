package keeper_test

import (
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func TestGetAttestationDataBySnapshot(t *testing.T) {
	k, _, _, ok, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getAttDataBySnapResponse, err := keeper.NewQuerier(k).GetAttestationDataBySnapshot(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getAttDataBySnapResponse)

	getAttDataBySnapResponse, err = keeper.NewQuerier(k).GetAttestationDataBySnapshot(ctx, &types.QueryGetAttestationDataBySnapshotRequest{
		Snapshot: "a",
	})
	require.ErrorContains(t, err, "failed to decode snapshot")
	require.Nil(t, getAttDataBySnapResponse)

	getAttDataBySnapResponse, err = keeper.NewQuerier(k).GetAttestationDataBySnapshot(ctx, &types.QueryGetAttestationDataBySnapshotRequest{
		Snapshot: "abcd1234",
	})
	require.ErrorContains(t, err, "snapshot not found for snapshot")
	require.Nil(t, getAttDataBySnapResponse)

	queryId := []byte("queryId")
	timestampTime := time.Unix(100, 0)
	aggReport := oracletypes.Aggregate{
		QueryId:        queryId,
		AggregateValue: "1",
		ReporterPower:  int64(10),
	}
	ok.On("GetAggregateByTimestamp", ctx, queryId, timestampTime).Return(&aggReport, nil).Once()
	snapshot, err := utils.QueryBytesFromString("abcd1234")
	require.NoError(t, err)
	err = k.AttestSnapshotDataMap.Set(ctx, snapshot, types.AttestationSnapshotData{
		ValidatorCheckpoint:  []byte("checkpoint"),
		AttestationTimestamp: timestampTime.Unix() - 1,
		PrevReportTimestamp:  timestampTime.Unix() - 2,
		NextReportTimestamp:  timestampTime.Unix() + 1,
		QueryId:              queryId,
		Timestamp:            timestampTime.Unix(),
	})
	require.NoError(t, err)

	getAttDataBySnapResponse, err = keeper.NewQuerier(k).GetAttestationDataBySnapshot(ctx, &types.QueryGetAttestationDataBySnapshotRequest{
		Snapshot: "abcd1234",
	})
	require.NoError(t, err)
	require.Equal(t, getAttDataBySnapResponse.QueryId, hex.EncodeToString(aggReport.QueryId))
	require.Equal(t, getAttDataBySnapResponse.Timestamp, strconv.FormatInt(timestampTime.Unix(), 10))
	require.Equal(t, getAttDataBySnapResponse.AggregateValue, aggReport.AggregateValue)
	require.Equal(t, getAttDataBySnapResponse.AggregatePower, strconv.FormatInt(aggReport.ReporterPower, 10))
	require.Equal(t, getAttDataBySnapResponse.Checkpoint, hex.EncodeToString([]byte("checkpoint")))
	require.Equal(t, getAttDataBySnapResponse.PreviousReportTimestamp, strconv.FormatInt(timestampTime.Unix()-2, 10))
	require.Equal(t, getAttDataBySnapResponse.NextReportTimestamp, strconv.FormatInt(timestampTime.Unix()+1, 10))

}
