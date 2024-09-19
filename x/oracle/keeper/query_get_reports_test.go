package keeper_test

import (
	"encoding/hex"

	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestGetReportsByQueryId() {
	addr, queryIdStr := s.TestSubmitValue()

	req := &types.QueryGetReportsbyQidRequest{QueryId: hex.EncodeToString(queryIdStr)}

	report, err := s.queryClient.GetReportsbyQid(s.ctx, req)
	s.Nil(err)

	MicroReport := types.MicroReport{
		Reporter:        addr.String(),
		Power:           1,
		QueryType:       "SpotPrice",
		QueryId:         queryIdStr,
		AggregateMethod: "weighted-median",
		Value:           value,
		Timestamp:       s.ctx.BlockTime(),
		Cyclelist:       true,
		BlockNumber:     uint64(s.ctx.BlockHeight()),
	}
	expectedReports := []types.MicroReport{MicroReport}

	s.Equal(expectedReports, report.MicroReports)

	report2, err := s.queryClient.GetReportsbyReporter(s.ctx, &types.QueryGetReportsbyReporterRequest{Reporter: addr.String()})
	s.NoError(err)
	s.Equal(expectedReports[0], report2.MicroReports[0])

	report3, err := s.queryClient.GetReportsbyReporterQid(s.ctx, &types.QueryGetReportsbyReporterQidRequest{Reporter: addr.String(), QueryId: hex.EncodeToString(queryIdStr)})
	s.NoError(err)
	s.EqualValues(expectedReports, report3.MicroReports)

	report, err = s.queryClient.GetReportsbyQid(s.ctx, &types.QueryGetReportsbyQidRequest{QueryId: hex.EncodeToString(queryIdStr)})
	s.NoError(err)
	s.Equal(expectedReports, report.MicroReports)
}
