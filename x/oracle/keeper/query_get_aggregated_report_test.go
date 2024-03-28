package keeper_test

import (
	"encoding/hex"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestQueryGetAggregatedReport() {
	// require := s.Require()

	// queryId, err := utils.QueryIDFromDataString(ethQueryData)
	// s.NoError(err)
	// aggregate, err := s.oracleKeeper.GetCurrentValueForQueryId(s.ctx, queryId)
	// require.NoError(err)
	// fmt.Println(aggregate)

	// s.TestSubmitValue()
	// aggregate, err = s.oracleKeeper.GetCurrentValueForQueryId(s.ctx, queryId)
	// require.NoError(err)
	// fmt.Println(aggregate)

	// aggReportRequest := &types.QueryGetCurrentAggregatedReportRequest{
	// 	QueryId: hex.EncodeToString(queryId),
	// }

	// response, err := s.oracleKeeper.GetAggregatedReport(s.ctx, aggReportRequest)
	// require.NoError(err)
	// require.NotNil(response)
	// // get validator info for expectedAggregate
	// validatorData, err := s.stakingKeeper.Validator(s.ctx, sdk.ValAddress(Addr.String()))
	// require.Nil(err)
	// valTokens := validatorData.GetBondedTokens()
	// // get queryID
	// queryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	// queryDataBytes, err := hex.DecodeString(queryData[2:])
	// require.Nil(err)
	// queryIdBytes := crypto.Keccak256(queryDataBytes)
	// queryId := hex.EncodeToString(queryIdBytes)
	// // submit and set aggregated report
	// s.TestSubmitValue()
	// // todo: use proper variables instead of mock.Anything
	// s.accountKeeper.On("GetAccount", mock.Anything, mock.Anything).Return(Addr)
	// s.accountKeeper.On("GetModuleAccount", mock.Anything, mock.Anything).Return(authtypes.NewEmptyModuleAccount("oracle"))
	// s.bankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: "loya", Amount: valTokens})
	// s.distrKeeper.On("AllocateTokensToValidator", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// s.bankKeeper.On("SendCoinsFromModuleToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// s.oracleKeeper.SetAggregatedReport(s.ctx)

	// expectedAggregate := types.Aggregate{
	// 	QueryId:           queryId,
	// 	AggregateValue:    "000000000000000000000000000000000000000000000058528649cf80ee0000",
	// 	AggregateReporter: Addr.String(),
	// 	ReporterPower:     validatorData.GetConsensusPower(sdk.DefaultPowerReduction),
	// 	StandardDeviation: 0,
	// 	Reporters: []*types.AggregateReporter{
	// 		{
	// 			Reporter: Addr.String(),
	// 			Power:    validatorData.GetConsensusPower(sdk.DefaultPowerReduction),
	// 		},
	// 	},
	// 	Flagged:              false,
	// 	Nonce:                1,
	// 	AggregateReportIndex: 0,
	// }

	// aggregate, err := s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: queryId})
	// require.Nil(err)
	// require.Equal(expectedAggregate.AggregateReporter, aggregate.Report.AggregateReporter)
	// require.Equal(expectedAggregate.AggregateValue, aggregate.Report.AggregateValue)
	// require.Equal(expectedAggregate.QueryId, aggregate.Report.QueryId)
	// require.Equal(expectedAggregate.Reporters, aggregate.Report.Reporters)
	// require.Equal(expectedAggregate.StandardDeviation, aggregate.Report.StandardDeviation)
	// require.Equal(expectedAggregate.Flagged, aggregate.Report.Flagged)
	// require.Equal(expectedAggregate.Nonce, aggregate.Report.Nonce)
	// require.Equal(expectedAggregate.AggregateReportIndex, aggregate.Report.AggregateReportIndex)

}

func (s *KeeperTestSuite) TestQueryGetAggregatedReportNilRequest() {
	require := s.Require()

	_, err := s.oracleKeeper.GetAggregatedReport(s.ctx, nil)
	require.ErrorContains(err, "invalid request")
}

func (s *KeeperTestSuite) TestQueryGetAggregatedReportInvalidQueryId() {
	require := s.Require()
	require.Panics(func() {
		s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: "invalidQueryID"})
	}, "invalid queryID")
}

func (s *KeeperTestSuite) TestQueryGetAggregatedReportNoAvailableTimestamps() {
	require := s.Require()

	// get queryID
	queryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	queryDataBytes, err := hex.DecodeString(queryData[2:])
	require.Nil(err)
	queryIdBytes := crypto.Keccak256(queryDataBytes)
	queryId := hex.EncodeToString(queryIdBytes)
	// submit without setting aggregate report
	s.TestSubmitValue()

	_, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: queryId})
	require.ErrorContains(err, "no available reports")

}
