package keeper_test

import (
	"encoding/hex"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
)

func (s *KeeperTestSuite) TestQueryGetAggregatedReport() {
	require := s.Require()
	k := s.oracleKeeper
	q := s.queryClient

	// nil request
	res, err := q.GetCurrentAggregateReport(s.ctx, nil)
	require.ErrorContains(err, "invalid request")
	require.Nil(res)

	// bad query id
	req := types.QueryGetCurrentAggregateReportRequest{
		QueryId: "badqueryid",
	}
	res, err = q.GetCurrentAggregateReport(s.ctx, &req)
	require.Error(err)
	require.Nil(res)

	// good req, no reports available
	qId, err := utils.QueryIDFromDataString(queryData)
	require.NoError(err)
	req.QueryId = hex.EncodeToString(qId)
	res, err = q.GetCurrentAggregateReport(s.ctx, &req)
	require.ErrorContains(err, "aggregate not found")
	require.Nil(res)

	// set Aggregates collection
	require.NoError(k.Aggregates.Set(s.ctx, collections.Join(qId, int64(0)), types.Aggregate{
		QueryId:           qId,
		AggregateValue:    "100",
		AggregateReporter: "reporter",
		ReporterPower:     100,
	}))
	res, err = q.GetCurrentAggregateReport(s.ctx, &req)
	require.NoError(err)
	require.NotNil(res)
	require.Equal(res.Aggregate.QueryId, qId)
	require.Equal(res.Aggregate.AggregateValue, "100")
	require.Equal(res.Aggregate.AggregateReporter, "reporter")
	require.Equal(res.Aggregate.ReporterPower, int64(100))
}
