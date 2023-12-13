package keeper_test

import (
	"math"

	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestWeightedMedian() {
	require := s.Require()
	qId := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"
	// normal scenario - 5 reporters various weights
	reports := []types.MicroReport{
		{
			Reporter: "reporter1",
			Value:    "a", // Hex value for 10
			Power:    10,
			QueryId:  qId,
		},
		{
			Reporter: "reporter2",
			Value:    "14", // Hex value for 20
			Power:    4,
			QueryId:  qId,
		},
		{
			Reporter: "reporter3",
			Value:    "1e", // Hex value for 30
			Power:    2,
			QueryId:  qId,
		},
		{
			Reporter: "reporter4",
			Value:    "28", // Hex value for 40
			Power:    20,
			QueryId:  qId,
		},
		{
			Reporter: "reporter5",
			Value:    "32", // Hex value for 50
			Power:    8,
			QueryId:  qId,
		},
	}

	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err := s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetAggregatedReportRequest{QueryId: qId})
	require.Nil(err)
	//fmt.Println("REPORT 1: ", res.Report)
	require.Equal(res.Report.QueryId, qId, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, "reporter4", "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, "28", "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, int64(20), "reporter power is not correct")
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter1", "reporter is not correct")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter2", "reporter is not correct")
	require.Equal(res.Report.Reporters[2].Reporter, "reporter3", "reporter is not correct")
	require.Equal(res.Report.Reporters[3].Reporter, "reporter4", "reporter is not correct")
	require.Equal(res.Report.Reporters[4].Reporter, "reporter5", "reporter is not correct")

	weightedMean := float64((10*10)+(20*4)+(30*2)+(40*20)+(50*8)) / (10 + 4 + 2 + 20 + 8)
	sum := ((10 * math.Pow(10-weightedMean, 2)) + (4 * math.Pow(20-weightedMean, 2)) + (2 * math.Pow(30-weightedMean, 2)) + (20 * math.Pow(40-weightedMean, 2)) + (8 * math.Pow(50-weightedMean, 2))) / (10 + 4 + 2 + 20 + 8)
	require.Equal(res.Report.StandardDeviation, math.Sqrt(sum), "std deviation is not correct")

	// special case A -- lower weighted median and upper weighted median are equal, powers are equal
	// calculates lower median
	qId2 := "a6f013ee236804827b77696d350e9f0ac3e879328f2a3021d473a0b778ad78ac"
	reports = []types.MicroReport{
		{
			Reporter: "reporter6",
			Value:    "a", // Hex value for 10
			Power:    1,
			QueryId:  qId2,
		},
		{
			Reporter: "reporter7",
			Value:    "a", // Hex value for 10
			Power:    1,
			QueryId:  qId2,
		},
		{
			Reporter: "reporter8",
			Value:    "14", // Hex value for 20
			Power:    1,
			QueryId:  qId2,
		},
		{
			Reporter: "reporter9",
			Value:    "14", // Hex value for 20
			Power:    1,
			QueryId:  qId2,
		},
	}
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetAggregatedReportRequest{QueryId: qId2})
	require.Nil(err)
	//fmt.Println("REPORT 2: ", res.Report)
	require.Equal(res.Report.QueryId, qId2, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, "reporter7", "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, "a", "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, int64(1), "reporter power is not correct")
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter6", "reporter is not correct")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter7", "reporter is not correct")
	require.Equal(res.Report.Reporters[2].Reporter, "reporter8", "reporter is not correct")
	require.Equal(res.Report.Reporters[3].Reporter, "reporter9", "reporter is not correct")

	weightedMean = float64((10*1)+(10*1)+(20*1)+(20*1)) / (1 + 1 + 1 + 1)
	sum = ((1 * math.Pow(10-weightedMean, 2)) + (1 * math.Pow(10-weightedMean, 2)) + (1 * math.Pow(20-weightedMean, 2)) + (1 * math.Pow(20-weightedMean, 2))) / (1 + 1 + 1 + 1)
	require.Equal(res.Report.StandardDeviation, math.Sqrt(sum), "std deviation is not correct")

	// special case B -- lower weighted median and upper weighted median are equal, powers are not all equal
	// calculates lower median
	qId3 := "48e9e2c732ba278de6ac88a3a57a5c5ba13d3d8370e709b3b98333a57876ca95"
	reports = []types.MicroReport{
		{
			Reporter: "reporter10",
			Value:    "a", // Hex value for 10
			Power:    1,
			QueryId:  qId3,
		},
		{
			Reporter: "reporter11",
			Value:    "a", // Hex value for 10
			Power:    2,
			QueryId:  qId3,
		},
		{
			Reporter: "reporter12",
			Value:    "14", // Hex value for 20
			Power:    1,
			QueryId:  qId3,
		},
		{
			Reporter: "reporter13",
			Value:    "14", // Hex value for 20
			Power:    2,
			QueryId:  qId3,
		},
	}
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetAggregatedReportRequest{QueryId: qId3})
	require.Nil(err)
	//fmt.Println("REPORT 3: ", res.Report)
	require.Equal(res.Report.QueryId, qId3, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, "reporter11", "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, "a", "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, int64(2), "reporter power is not correct")
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter10", "reporter is not correct")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter11", "reporter is not correct")
	require.Equal(res.Report.Reporters[2].Reporter, "reporter12", "reporter is not correct")
	require.Equal(res.Report.Reporters[3].Reporter, "reporter13", "reporter is not correct")

	weightedMean = float64((10*1)+(10*2)+(20*1)+(20*2)) / (1 + 2 + 1 + 2)
	sum = ((1 * math.Pow(10-weightedMean, 2)) + (2 * math.Pow(10-weightedMean, 2)) + (1 * math.Pow(20-weightedMean, 2)) + (2 * math.Pow(20-weightedMean, 2))) / (1 + 2 + 1 + 2)
	require.Equal(res.Report.StandardDeviation, math.Sqrt(sum), "std deviation is not correct")

	// 5 reporters with even weights, should be equal to normal median
	qId4 := "907154958baee4fb0ce2bbe50728141ac76eb2dc1731b3d40f0890746dd07e62"
	reports = []types.MicroReport{
		{
			Reporter: "reporter14",
			Value:    "a", // Hex value for 10
			Power:    5,
			QueryId:  qId4,
		},
		{
			Reporter: "reporter15",
			Value:    "14", // Hex value for 20
			Power:    5,
			QueryId:  qId4,
		},
		{
			Reporter: "reporter16",
			Value:    "1e", // Hex value for 30
			Power:    5,
			QueryId:  qId4,
		},
		{
			Reporter: "reporter17",
			Value:    "28", // Hex value for 40
			Power:    5,
			QueryId:  qId4,
		},
		{
			Reporter: "reporter18",
			Value:    "32", // Hex value for 50
			Power:    5,
			QueryId:  qId4,
		},
	}
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetAggregatedReportRequest{QueryId: qId4})
	require.Nil(err)
	//fmt.Println("REPORT 4: ", res.Report)
	require.Equal(res.Report.QueryId, qId4, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, "reporter16", "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, "1e", "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, int64(5), "reporter power is not correct")
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter14", "reporter is not correct")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter15", "reporter is not correct")
	require.Equal(res.Report.Reporters[2].Reporter, "reporter16", "reporter is not correct")
	require.Equal(res.Report.Reporters[3].Reporter, "reporter17", "reporter is not correct")
	require.Equal(res.Report.Reporters[4].Reporter, "reporter18", "reporter is not correct")

	weightedMean = float64((10*5)+(20*5)+(30*5)+(40*5)+(50*5)) / (5 + 5 + 5 + 5 + 5)
	sum = ((5 * math.Pow(10-weightedMean, 2)) + (5 * math.Pow(20-weightedMean, 2)) + (5 * math.Pow(30-weightedMean, 2)) + (5 * math.Pow(40-weightedMean, 2)) + (5 * math.Pow(50-weightedMean, 2))) / (5 + 5 + 5 + 5 + 5)
	require.Equal(res.Report.StandardDeviation, math.Sqrt(sum), "std deviation is not correct")

}
