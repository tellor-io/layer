package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
)

func TestNewQuerier(t *testing.T) {
	k, _, _, _, _, _ := testkeeper.OracleKeeper(t)
	q := keeper.NewQuerier(k)
	require.NotNil(t, q)
}

func TestGetCurrentAggregateReport(t *testing.T) {
	k, _, _, _, _, ctx := testkeeper.OracleKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getCurrentAggResponse, err := keeper.NewQuerier(k).GetCurrentAggregateReport(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getCurrentAggResponse)

	getCurrentAggResponse, err = keeper.NewQuerier(k).GetCurrentAggregateReport(ctx, &types.QueryGetCurrentAggregateReportRequest{
		QueryId: "z",
	})
	require.ErrorContains(t, err, "invalid query id")
	require.Nil(t, getCurrentAggResponse)

	agg := (*types.Aggregate)(nil)
	timestamp := time.Unix(int64(1), 0)
	queryId := "1234abcd"
	qIdBz, err := utils.QueryBytesFromString(queryId)
	require.NoError(t, err)
	// ok.On("GetCurrentAggregateReport", ctx, qIdBz).Return(agg, timestamp).Once()

	getCurrentAggResponse, err = keeper.NewQuerier(k).GetCurrentAggregateReport(ctx, &types.QueryGetCurrentAggregateReportRequest{
		QueryId: queryId,
	})
	require.ErrorContains(t, err, "aggregate not found")
	require.Nil(t, getCurrentAggResponse)

	agg = &types.Aggregate{
		QueryId:              []byte(queryId),
		AggregateValue:       "10_000",
		AggregateReporter:    "reporter1",
		ReporterPower:        int64(100),
		StandardDeviation:    float64(0),
		Reporters:            []*types.AggregateReporter{{}},
		Flagged:              false,
		Index:                uint64(0),
		AggregateReportIndex: int64(0),
		Height:               int64(0),
		MicroHeight:          int64(0),
	}

	require.NoError(t, k.Aggregates.Set(ctx, collections.Join(qIdBz, timestamp.UnixMilli()), *agg))
	getCurrentAggResponse, err = keeper.NewQuerier(k).GetCurrentAggregateReport(ctx, &types.QueryGetCurrentAggregateReportRequest{
		QueryId: queryId,
	})
	require.NoError(t, err)
	require.Equal(t, getCurrentAggResponse.Timestamp, uint64(timestamp.UnixMilli()))
	require.Equal(t, getCurrentAggResponse.Aggregate.QueryId, agg.QueryId)
	require.Equal(t, getCurrentAggResponse.Aggregate.AggregateValue, agg.AggregateValue)
	require.Equal(t, getCurrentAggResponse.Aggregate.AggregateReporter, agg.AggregateReporter)
	require.Equal(t, getCurrentAggResponse.Aggregate.ReporterPower, agg.ReporterPower)
	require.Equal(t, getCurrentAggResponse.Aggregate.StandardDeviation, agg.StandardDeviation)
	require.Equal(t, getCurrentAggResponse.Aggregate.Flagged, agg.Flagged)
	require.Equal(t, getCurrentAggResponse.Aggregate.Index, agg.Index)
	require.Equal(t, getCurrentAggResponse.Aggregate.AggregateReportIndex, agg.AggregateReportIndex)
	require.Equal(t, getCurrentAggResponse.Aggregate.Height, agg.Height)
}
