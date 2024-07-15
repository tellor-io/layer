package keeper_test

import (
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
)

func (s *KeeperTestSuite) TestQueryGetDataBefore() {
	require := s.Require()
	k := s.oracleKeeper
	q := s.queryClient
	ctx := s.ctx

	// request is nil
	res, err := q.GetDataBefore(ctx, nil)
	require.ErrorContains(err, "invalid request")
	require.Nil(res)

	// bad querybytes
	res, err = q.GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId:   "bad",
		Timestamp: ctx.BlockTime().Unix(),
	})
	require.Error(err)
	require.Nil(res)

	// getDataBefore 1 sec after aggregate
	qId, err := utils.QueryIDFromDataString(queryData)
	require.NoError(err)
	require.NoError(k.Aggregates.Set(ctx, collections.Join(qId, ctx.BlockTime().UnixMilli()), types.Aggregate{
		QueryId:           qId,
		AggregateValue:    "100",
		AggregateReporter: "reporter",
		ReporterPower:     100,
		Height:            0,
		Flagged:           false,
	}))
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(1 * time.Second))
	ctx = ctx.WithBlockHeight(1)
	res, err = q.GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId:   hex.EncodeToString(qId),
		Timestamp: ctx.BlockTime().Unix(),
	})
	require.NoError(err)
	require.Equal(res.Report.AggregateValue, "100")
	require.Equal(res.Report.AggregateReporter, "reporter")
	require.Equal(res.Report.ReporterPower, int64(100))
	require.Equal(res.Report.Height, int64(0))
	require.Equal(res.Report.Flagged, false)
	require.Equal(res.Report.QueryId, qId)

	// getDataBefore at same timestamp as aggregate
	require.NoError(k.Aggregates.Set(ctx, collections.Join(qId, ctx.BlockTime().UnixMilli()), types.Aggregate{
		QueryId:           qId,
		AggregateValue:    "200",
		AggregateReporter: "reporter2",
		ReporterPower:     500,
		Height:            1,
		Flagged:           false,
	}))
	res, err = q.GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId:   hex.EncodeToString(qId),
		Timestamp: ctx.BlockTime().Unix(),
	})
	require.NoError(err)
	require.Equal(res.Report.AggregateValue, "200")
	require.Equal(res.Report.AggregateReporter, "reporter2")
	require.Equal(res.Report.ReporterPower, int64(500))
	require.Equal(res.Report.Height, int64(1))
	require.Equal(res.Report.Flagged, false)
	require.Equal(res.Report.QueryId, qId)
}
