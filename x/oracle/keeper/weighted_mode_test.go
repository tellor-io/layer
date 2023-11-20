package keeper_test

import (
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestWeightedMode() {
	require := s.Require()

	// normal scenario
	// list of reports
	reports := []types.MicroReport{
		{
			Reporter: "reporter1",
			Value:    "aaa",
			Power:    10,
			QueryId:  "query1",
		},
		{
			Reporter: "reporter2",
			Value:    "aaa",
			Power:    4,
			QueryId:  "query1",
		},
		{
			Reporter: "reporter3",
			Value:    "aaa",
			Power:    2,
			QueryId:  "query1",
		},
		{
			Reporter: "reporter4",
			Value:    "aaa",
			Power:    20,
			QueryId:  "query1",
		},
		{
			Reporter: "reporter5",
			Value:    "bbb",
			Power:    8,
			QueryId:  "query1",
		},
	}

	s.oracleKeeper.WeightedMode(s.ctx, reports)
	res, err := s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetAggregatedReportRequest{QueryId: "query1"})
	require.Nil(err)
	require.Equal(res.Report.QueryId, "query1", "query id is not correct")
	require.Equal(res.Report.AggregateReporter, "reporter4", "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, "aaa", "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, int64(20), "aggregate reporter power is not correct")
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter1", "reporter is not correct")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter2", "reporter is not correct")
	require.Equal(res.Report.Reporters[2].Reporter, "reporter3", "reporter is not correct")
	require.Equal(res.Report.Reporters[3].Reporter, "reporter4", "reporter is not correct")
	require.Equal(res.Report.Reporters[4].Reporter, "reporter5", "reporter is not correct")

	// scenario where mode is not decided by most powerful reporter
	reports = []types.MicroReport{
		{
			Reporter: "reporter6",
			Value:    "ccc",
			Power:    1,
			QueryId:  "query2",
		},
		{
			Reporter: "reporter7",
			Value:    "ccc",
			Power:    2,
			QueryId:  "query2",
		},
		{
			Reporter: "reporter8",
			Value:    "ccc",
			Power:    2,
			QueryId:  "query2",
		},
		{
			Reporter: "reporter9",
			Value:    "ddd",
			Power:    5,
			QueryId:  "query2",
		},
		{
			Reporter: "reporter10",
			Value:    "ccc",
			Power:    1,
			QueryId:  "query2",
		},
	}

	s.oracleKeeper.WeightedMode(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetAggregatedReportRequest{QueryId: "query1"})
	require.Nil(err)
	//require.Equal(res.Report.QueryId, "query1", "query id is not correct")
	//require.Equal(res.Report.AggregateReporter, "reporter4", "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, "ccc", "aggregate value is not correct")
	//require.Equal(res.Report.ReporterPower, int64(20), "aggregate reporter power is not correct")
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter1", "reporter is not correct")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter2", "reporter is not correct")
	require.Equal(res.Report.Reporters[2].Reporter, "reporter3", "reporter is not correct")
	require.Equal(res.Report.Reporters[3].Reporter, "reporter4", "reporter is not correct")
	require.Equal(res.Report.Reporters[4].Reporter, "reporter5", "reporter is not correct")

}
