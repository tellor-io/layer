package keeper_test

import (
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestSubmitValue() {
	require := s.Require()
	queryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	// Commit value transaction first
	s.TestCommitValue()
	var submitreq types.MsgSubmitValue
	var submitres types.MsgSubmitValueResponse
	// forward block by 1 and reveal value
	height := s.ctx.BlockHeight() + 1
	s.ctx = s.ctx.WithBlockHeight(height)
	// Submit value transaction with value revealed, this checks if the value is correctly signed
	submitreq.Creator = Addr.String()
	submitreq.QueryData = queryData
	submitreq.Value = value
	res, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	require.Equal(&submitres, res)
	require.Nil(err)
	report, err := s.oracleKeeper.GetReportsbyQid(s.ctx, &types.QueryGetReportsbyQidRequest{QueryId: "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"})
	require.Nil(err)
	microReport := types.MicroReport{
		Reporter:        Addr.String(),
		Power:           1000000000000,
		QueryType:       "SpotPrice",
		QueryId:         "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		AggregateMethod: "weighted-median",
		Value:           value,
		BlockNumber:     s.ctx.BlockHeight(),
		Timestamp:       s.ctx.BlockTime(),
	}
	expectedReport := types.QueryGetReportsbyQidResponse{
		Reports: types.Reports{
			MicroReports: []*types.MicroReport{&microReport},
		},
	}
	require.Equal(&expectedReport, report)
}
