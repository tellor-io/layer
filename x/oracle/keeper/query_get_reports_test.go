package keeper_test

import (
	// "github.com/stretchr/testify/require"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestGetReportsByQueryId() {

	stakedReporter, queryIdStr := s.TestSubmitValue()

	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	req := &types.QueryGetReportsbyQidRequest{QueryId: queryIdStr}

	report, err := s.queryClient.GetReportsbyQid(s.ctx, req)
	s.Nil(err)

	MicroReport := &types.MicroReport{
		Reporter:        stakedReporter.GetReporter(),
		Power:           stakedReporter.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryType:       "SpotPrice",
		QueryId:         queryIdStr,
		AggregateMethod: "weighted-median",
		Value:           value,
		Timestamp:       s.ctx.BlockTime(),
		Cyclelist:       true,
	}
	expectedReports := types.Reports{
		MicroReports: []*types.MicroReport{MicroReport},
	}

	s.Equal(expectedReports, report.Reports)

	report2, err := s.queryClient.GetReportsbyReporter(s.ctx, &types.QueryGetReportsbyReporterRequest{Reporter: stakedReporter.GetReporter()})
	s.NoError(err)
	s.Equal(*expectedReports.MicroReports[0], report2.MicroReports[0])

	report3, err := s.queryClient.GetReportsbyReporterQid(s.ctx, &types.QueryGetReportsbyReporterQidRequest{Reporter: stakedReporter.GetReporter(), QueryId: queryIdStr})
	s.NoError(err)
	s.EqualValues(expectedReports.MicroReports, report3.Reports.MicroReports)

	report, err = s.queryClient.GetReportsbyQid(s.ctx, &types.QueryGetReportsbyQidRequest{QueryId: queryIdStr})
	s.NoError(err)
	s.Equal(expectedReports, report.Reports)
}
