package keeper_test

import (
	"math"

	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestWeightedMedian() {
	require := s.Require()

	// normal scenario - 5 reporters various weights
	reports := []types.MicroReport{
		{
			Reporter: "reporter1",
			Value:    "a", // Hex value for 10
			Power:    10,
			QueryId:  "query1",
		},
		{
			Reporter: "reporter2",
			Value:    "14", // Hex value for 20
			Power:    4,
			QueryId:  "query1",
		},
		{
			Reporter: "reporter3",
			Value:    "1e", // Hex value for 30
			Power:    2,
			QueryId:  "query1",
		},
		{
			Reporter: "reporter4",
			Value:    "28", // Hex value for 40
			Power:    20,
			QueryId:  "query1",
		},
		{
			Reporter: "reporter5",
			Value:    "32", // Hex value for 50
			Power:    8,
			QueryId:  "query1",
		},
	}

	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err := s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetAggregatedReportRequest{QueryId: "query1"})
	require.Nil(err)
	//fmt.Println("REPORT 1: ", res.Report)
	require.Equal(res.Report.QueryId, "query1")
	require.Equal(res.Report.AggregateReporter, "reporter4")
	require.Equal(res.Report.AggregateValue, "28")
	require.Equal(res.Report.ReporterPower, int64(20))
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter1")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter2")
	require.Equal(res.Report.Reporters[2].Reporter, "reporter3")
	require.Equal(res.Report.Reporters[3].Reporter, "reporter4")
	require.Equal(res.Report.Reporters[4].Reporter, "reporter5")

	weightedMean := float64((10*10)+(20*4)+(30*2)+(40*20)+(50*8)) / (10 + 4 + 2 + 20 + 8)
	sum := ((10 * math.Pow(10-weightedMean, 2)) + (4 * math.Pow(20-weightedMean, 2)) + (2 * math.Pow(30-weightedMean, 2)) + (20 * math.Pow(40-weightedMean, 2)) + (8 * math.Pow(50-weightedMean, 2))) / (10 + 4 + 2 + 20 + 8)
	require.Equal(res.Report.StandardDeviation, math.Sqrt(sum))

	// special case A -- lower weighted median and upper weighted median are equal, powers are equal
	// calculates lower median
	reports = []types.MicroReport{
		{
			Reporter: "reporter6",
			Value:    "a", // Hex value for 10
			Power:    1,
			QueryId:  "query2",
		},
		{
			Reporter: "reporter7",
			Value:    "a", // Hex value for 10
			Power:    1,
			QueryId:  "query2",
		},
		{
			Reporter: "reporter8",
			Value:    "14", // Hex value for 20
			Power:    1,
			QueryId:  "query2",
		},
		{
			Reporter: "reporter9",
			Value:    "14", // Hex value for 20
			Power:    1,
			QueryId:  "query2",
		},
	}
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetAggregatedReportRequest{QueryId: "query2"})
	require.Nil(err)
	//fmt.Println("REPORT 2: ", res.Report)
	require.Equal(res.Report.QueryId, "query2")
	require.Equal(res.Report.AggregateReporter, "reporter7")
	require.Equal(res.Report.AggregateValue, "a")
	require.Equal(res.Report.ReporterPower, int64(1))
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter6")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter7")
	require.Equal(res.Report.Reporters[2].Reporter, "reporter8")
	require.Equal(res.Report.Reporters[3].Reporter, "reporter9")

	weightedMean = float64((10*1)+(10*1)+(20*1)+(20*1)) / (1 + 1 + 1 + 1)
	sum = ((1 * math.Pow(10-weightedMean, 2)) + (1 * math.Pow(10-weightedMean, 2)) + (1 * math.Pow(20-weightedMean, 2)) + (1 * math.Pow(20-weightedMean, 2))) / (1 + 1 + 1 + 1)
	require.Equal(res.Report.StandardDeviation, math.Sqrt(sum))

	// special case B -- lower weighted median and upper weighted median are equal, powers are not all equal
	// calculates lower median
	reports = []types.MicroReport{
		{
			Reporter: "reporter10",
			Value:    "a", // Hex value for 10
			Power:    1,
			QueryId:  "query3",
		},
		{
			Reporter: "reporter11",
			Value:    "a", // Hex value for 10
			Power:    2,
			QueryId:  "query3",
		},
		{
			Reporter: "reporter12",
			Value:    "14", // Hex value for 20
			Power:    1,
			QueryId:  "query3",
		},
		{
			Reporter: "reporter13",
			Value:    "14", // Hex value for 20
			Power:    2,
			QueryId:  "query3",
		},
	}
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetAggregatedReportRequest{QueryId: "query3"})
	require.Nil(err)
	//fmt.Println("REPORT 3: ", res.Report)
	require.Equal(res.Report.QueryId, "query3")
	require.Equal(res.Report.AggregateReporter, "reporter11")
	require.Equal(res.Report.AggregateValue, "a")
	require.Equal(res.Report.ReporterPower, int64(2))
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter10")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter11")
	require.Equal(res.Report.Reporters[2].Reporter, "reporter12")
	require.Equal(res.Report.Reporters[3].Reporter, "reporter13")

	weightedMean = float64((10*1)+(10*2)+(20*1)+(20*2)) / (1 + 2 + 1 + 2)
	sum = ((1 * math.Pow(10-weightedMean, 2)) + (2 * math.Pow(10-weightedMean, 2)) + (1 * math.Pow(20-weightedMean, 2)) + (2 * math.Pow(20-weightedMean, 2))) / (1 + 2 + 1 + 2)
	require.Equal(res.Report.StandardDeviation, math.Sqrt(sum))

	// 5 reporters with even weights, should be equal to normal median
	reports = []types.MicroReport{
		{
			Reporter: "reporter14",
			Value:    "a", // Hex value for 10
			Power:    5,
			QueryId:  "query4",
		},
		{
			Reporter: "reporter15",
			Value:    "14", // Hex value for 20
			Power:    5,
			QueryId:  "query4",
		},
		{
			Reporter: "reporter16",
			Value:    "1e", // Hex value for 30
			Power:    5,
			QueryId:  "query4",
		},
		{
			Reporter: "reporter17",
			Value:    "28", // Hex value for 40
			Power:    5,
			QueryId:  "query4",
		},
		{
			Reporter: "reporter18",
			Value:    "32", // Hex value for 50
			Power:    5,
			QueryId:  "query4",
		},
	}
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetAggregatedReportRequest{QueryId: "query4"})
	require.Nil(err)
	//fmt.Println("REPORT 4: ", res.Report)
	require.Equal(res.Report.QueryId, "query4")
	require.Equal(res.Report.AggregateReporter, "reporter16")
	require.Equal(res.Report.AggregateValue, "1e")
	require.Equal(res.Report.ReporterPower, int64(5))
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter14")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter15")
	require.Equal(res.Report.Reporters[2].Reporter, "reporter16")
	require.Equal(res.Report.Reporters[3].Reporter, "reporter17")
	require.Equal(res.Report.Reporters[4].Reporter, "reporter18")

	weightedMean = float64((10*5)+(20*5)+(30*5)+(40*5)+(50*5)) / (5 + 5 + 5 + 5 + 5)
	sum = ((5 * math.Pow(10-weightedMean, 2)) + (5 * math.Pow(20-weightedMean, 2)) + (5 * math.Pow(30-weightedMean, 2)) + (5 * math.Pow(40-weightedMean, 2)) + (5 * math.Pow(50-weightedMean, 2))) / (5 + 5 + 5 + 5 + 5)
	require.Equal(res.Report.StandardDeviation, math.Sqrt(sum))

}
