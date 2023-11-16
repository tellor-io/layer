package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestGetReportsByQueryId() {
	require := s.Require()
	s.TestCommitValue()
	s.TestSubmitValue()
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	report, err := s.oracleKeeper.GetReportsbyQid(sdk.WrapSDKContext(s.ctx), &types.QueryGetReportsbyQidRequest{QueryId: "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"})

	require.Nil(err)
	MicroReport := &types.MicroReport{
		Reporter:        Addr.String(),
		Power:           1,
		QueryType:       "SpotPrice",
		QueryId:         "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		AggregateMethod: "weighted-median",
		Value:           value,
		BlockNumber:     s.ctx.BlockHeight(),
		Timestamp:       s.ctx.BlockTime(),
	}
	expectedReports := types.Reports{
		MicroReports: []*types.MicroReport{MicroReport},
	}
	expectedByQidResponse := &types.QueryGetReportsbyQidResponse{
		Reports: expectedReports,
	}
	require.Equal(expectedByQidResponse, report)
	report2, err := s.oracleKeeper.GetReportsbyReporter(sdk.WrapSDKContext(s.ctx), &types.QueryGetReportsbyReporterRequest{Reporter: Addr.String()})
	expectedByReporterResponse := &types.QueryGetReportsbyReporterResponse{
		MicroReports: []types.MicroReport{*MicroReport},
	}
	require.Equal(expectedByReporterResponse, report2)
	report3, err := s.oracleKeeper.GetReportsbyReporterQid(sdk.WrapSDKContext(s.ctx), &types.QueryGetReportsbyReporterQidRequest{Reporter: Addr.String(), QueryId: "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"})
	require.Equal(expectedByQidResponse, report3)
}
