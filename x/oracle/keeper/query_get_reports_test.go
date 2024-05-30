package keeper_test

import (
	"encoding/hex"

	"github.com/tellor-io/layer/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestGetReportsByQueryId() {
	stakedReporter, queryIdStr := s.TestSubmitValue()

	req := &types.QueryGetReportsbyQidRequest{QueryId: hex.EncodeToString(queryIdStr)}

	report, err := s.queryClient.GetReportsbyQid(s.ctx, req)
	s.Nil(err)

	MicroReport := &types.MicroReport{
		Reporter:        sdk.AccAddress(stakedReporter.GetReporter()).String(),
		Power:           stakedReporter.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryType:       "SpotPrice",
		QueryId:         queryIdStr,
		AggregateMethod: "weighted-median",
		Value:           value,
		Timestamp:       s.ctx.BlockTime(),
		Cyclelist:       true,
		BlockNumber:     s.ctx.BlockHeight(),
	}
	expectedReports := types.Reports{
		MicroReports: []*types.MicroReport{MicroReport},
	}

	s.Equal(expectedReports, report.Reports)

	report2, err := s.queryClient.GetReportsbyReporter(s.ctx, &types.QueryGetReportsbyReporterRequest{Reporter: sdk.AccAddress(stakedReporter.GetReporter()).String()})
	s.NoError(err)
	s.Equal(*expectedReports.MicroReports[0], report2.MicroReports[0])

	report3, err := s.queryClient.GetReportsbyReporterQid(s.ctx, &types.QueryGetReportsbyReporterQidRequest{Reporter: sdk.AccAddress(stakedReporter.GetReporter()).String(), QueryId: hex.EncodeToString(queryIdStr)})
	s.NoError(err)
	s.EqualValues(expectedReports.MicroReports, report3.Reports.MicroReports)

	report, err = s.queryClient.GetReportsbyQid(s.ctx, &types.QueryGetReportsbyQidRequest{QueryId: hex.EncodeToString(queryIdStr)})
	s.NoError(err)
	s.Equal(expectedReports, report.Reports)
}
