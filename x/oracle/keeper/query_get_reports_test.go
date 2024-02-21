package keeper_test

import (
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestGetReportsByQueryId() {
	require := s.Require()
	s.TestCommitValue()
	queryIdStr := s.TestSubmitValue()
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	report, err := s.oracleKeeper.GetReportsbyQid(s.ctx, &types.QueryGetReportsbyQidRequest{QueryId: queryIdStr})

	require.Nil(err)
	MicroReport := &types.MicroReport{
		Reporter:        Addr.String(),
		Power:           1000000000000,
		QueryType:       "SpotPrice",
		QueryId:         queryIdStr,
		AggregateMethod: "weighted-median",
		Value:           value,
		BlockNumber:     s.ctx.BlockHeight(),
		Timestamp:       s.ctx.BlockTime(),
	}
	expectedReports := types.Reports{
		MicroReports: []*types.MicroReport{MicroReport},
	}

	require.Equal(expectedReports, report.Reports)

	report2, err := s.oracleKeeper.GetReportsbyReporter(s.ctx, &types.QueryGetReportsbyReporterRequest{Reporter: Addr.String()})
	require.NoError(err)
	require.Equal(*expectedReports.MicroReports[0], report2.MicroReports[0])

	report3, err := s.oracleKeeper.GetReportsbyReporterQid(s.ctx, &types.QueryGetReportsbyReporterQidRequest{Reporter: Addr.String(), QueryId: queryIdStr})
	require.NoError(err)
	require.EqualValues(expectedReports.MicroReports, report3.Reports.MicroReports)

	report, err = s.oracleKeeper.GetReportsbyQid(s.ctx, &types.QueryGetReportsbyQidRequest{QueryId: queryIdStr})
	require.NoError(err)
	require.Equal(expectedReports, report.Reports)
}
