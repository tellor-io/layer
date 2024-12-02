package keeper_test

// import (
// 	"encoding/hex"

// 	"github.com/tellor-io/layer/testutil/sample"
// 	"github.com/tellor-io/layer/x/oracle/types"

// 	sdk "github.com/cosmos/cosmos-sdk/types"
// )

// func (s *KeeperTestSuite) TestWeightedMode() {
// 	reporters := make([]sdk.AccAddress, 18)
// 	for i := 0; i < 10; i++ {
// 		reporters[i] = sample.AccAddressBytes()
// 	}
// 	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
// 	// normal scenario
// 	// list of reports
// 	expectedReporter := reporters[3].String()
// 	reports := []types.MicroReport{
// 		{
// 			Reporter: reporters[0].String(),
// 			Value:    "aaa",
// 			Power:    10,
// 			QueryId:  qId,
// 		},
// 		{
// 			Reporter: reporters[1].String(),
// 			Value:    "aaa",
// 			Power:    4,
// 			QueryId:  qId,
// 		},
// 		{
// 			Reporter: reporters[2].String(),
// 			Value:    "aaa",
// 			Power:    2,
// 			QueryId:  qId,
// 		},
// 		{
// 			Reporter: expectedReporter,
// 			Value:    "aaa",
// 			Power:    20,
// 			QueryId:  qId,
// 		},
// 		{
// 			Reporter: reporters[4].String(),
// 			Value:    "bbb",
// 			Power:    8,
// 			QueryId:  qId,
// 		},
// 	}
// 	aggregates, err := s.oracleKeeper.WeightedMode(s.ctx, reports, 1)
// 	s.Nil(err)
// 	s.NotNil(aggregates)
// 	_, _, err = s.oracleKeeper.AggregateReport(s.ctx, 1)
// 	s.NoError(err)
// 	res, err := s.queryClient.GetCurrentAggregateReport(s.ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
// 	s.Nil(err)
// 	s.Equal(res.Aggregate.QueryId, qId, "query id is not correct")
// 	s.Equal(res.Aggregate.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
// 	s.Equal(res.Aggregate.AggregateValue, "aaa", "aggregate value is not correct")
// 	s.Equal(res.Aggregate.ReporterPower, uint64(44), "aggregate reporter power is not correct")
// 	s.Equal(res.Aggregate.MetaId, uint64(1), "report meta id is not correct")
// 	//  check list of reporters in the aggregate report
// 	s.Equal(res.Aggregate.Reporters[0].Reporter, reporters[0].String(), "reporter is not correct")
// 	s.Equal(res.Aggregate.Reporters[1].Reporter, reporters[1].String(), "reporter is not correct")
// 	s.Equal(res.Aggregate.Reporters[2].Reporter, reporters[2].String(), "reporter is not correct")
// 	s.Equal(res.Aggregate.Reporters[3].Reporter, expectedReporter, "reporter is not correct")
// 	s.Equal(res.Aggregate.Reporters[4].Reporter, reporters[4].String(), "reporter is not correct")
// 	s.Equal(res.Aggregate.AggregateReportIndex, uint64(3), "report index is not correct")

// 	// scenario where mode is not decided by most powerful reporter
// 	qId2, _ := hex.DecodeString("a6f013ee236804827b77696d350e9f0ac3e879328f2a3021d473a0b778ad78ac")
// 	expectedReporter = reporters[6].String()
// 	reports = []types.MicroReport{
// 		{
// 			Reporter: reporters[5].String(),
// 			Value:    "ccc",
// 			Power:    1,
// 			QueryId:  qId2,
// 		},
// 		{
// 			Reporter: expectedReporter,
// 			Value:    "ccc",
// 			Power:    2,
// 			QueryId:  qId2,
// 		},
// 		{
// 			Reporter: reporters[7].String(),
// 			Value:    "ccc",
// 			Power:    2,
// 			QueryId:  qId2,
// 		},
// 		{
// 			Reporter: reporters[8].String(),
// 			Value:    "ddd",
// 			Power:    5,
// 			QueryId:  qId2,
// 		},
// 		{
// 			Reporter: reporters[9].String(),
// 			Value:    "ccc",
// 			Power:    1,
// 			QueryId:  qId2,
// 		},
// 	}

// 	aggregates, err = s.oracleKeeper.WeightedMode(s.ctx, reports, 2)
// 	s.NoError(err)
// 	_, _, err = s.oracleKeeper.AggregateReport(s.ctx, 1)
// 	s.NoError(err)
// 	res, err = s.queryClient.GetCurrentAggregateReport(s.ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId2)})
// 	s.Nil(err)
// 	s.Equal(res.Aggregate.QueryId, qId2, "query id is not correct")
// 	s.Equal(res.Aggregate.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
// 	s.Equal(res.Aggregate.AggregateValue, "ccc", "aggregate value is not correct")
// 	s.Equal(res.Aggregate.ReporterPower, uint64(11), "aggregate reporter power is not correct")
// 	s.Equal(res.Aggregate.MetaId, uint64(2), "report meta id is not correct")
// 	//  check list of reporters in the aggregate report
// 	s.Equal(res.Aggregate.Reporters[0].Reporter, reporters[5].String(), "reporter is not correct")
// 	s.Equal(res.Aggregate.Reporters[1].Reporter, expectedReporter, "reporter is not correct")
// 	s.Equal(res.Aggregate.Reporters[2].Reporter, reporters[7].String(), "reporter is not correct")
// 	s.Equal(res.Aggregate.Reporters[3].Reporter, reporters[8].String(), "reporter is not correct")
// 	s.Equal(res.Aggregate.Reporters[4].Reporter, reporters[9].String(), "reporter is not correct")
// 	s.Equal(res.Aggregate.AggregateReportIndex, uint64(1), "report index is not correct")
// }
