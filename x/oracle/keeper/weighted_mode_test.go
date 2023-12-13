package keeper_test

import (
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestWeightedMode() {
	require := s.Require()
	qId := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"
	// normal scenario
	// list of reports
	reports := []types.MicroReport{
		{
			Reporter: "reporter1",
			Value:    "aaa",
			Power:    10,
			QueryId:  qId,
		},
		{
			Reporter: "reporter2",
			Value:    "aaa",
			Power:    4,
			QueryId:  qId,
		},
		{
			Reporter: "reporter3",
			Value:    "aaa",
			Power:    2,
			QueryId:  qId,
		},
		{
			Reporter: "reporter4",
			Value:    "aaa",
			Power:    20,
			QueryId:  qId,
		},
		{
			Reporter: "reporter5",
			Value:    "bbb",
			Power:    8,
			QueryId:  qId,
		},
	}

	s.oracleKeeper.WeightedMode(s.ctx, reports)
	res, err := s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	require.Nil(err)
	require.Equal(res.Report.QueryId, qId, "query id is not correct")
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
	qId2 := "a6f013ee236804827b77696d350e9f0ac3e879328f2a3021d473a0b778ad78ac"
	reports = []types.MicroReport{
		{
			Reporter: "reporter6",
			Value:    "ccc",
			Power:    1,
			QueryId:  qId2,
		},
		{
			Reporter: "reporter7",
			Value:    "ccc",
			Power:    2,
			QueryId:  qId2,
		},
		{
			Reporter: "reporter8",
			Value:    "ccc",
			Power:    2,
			QueryId:  qId2,
		},
		{
			Reporter: "reporter9",
			Value:    "ddd",
			Power:    5,
			QueryId:  qId2,
		},
		{
			Reporter: "reporter10",
			Value:    "ccc",
			Power:    1,
			QueryId:  qId2,
		},
	}

	s.oracleKeeper.WeightedMode(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId2})
	require.Nil(err)
	require.Equal(res.Report.QueryId, qId2, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, "reporter7", "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, "ccc", "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, int64(2), "aggregate reporter power is not correct")
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter6", "reporter is not correct")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter7", "reporter is not correct")
	require.Equal(res.Report.Reporters[2].Reporter, "reporter8", "reporter is not correct")
	require.Equal(res.Report.Reporters[3].Reporter, "reporter9", "reporter is not correct")
	require.Equal(res.Report.Reporters[4].Reporter, "reporter10", "reporter is not correct")

}
