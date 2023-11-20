package keeper_test

import (
	"math"

	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestWeightedMode() {
	require := s.Require()

	// list of reports
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
	require.Equal(res.Report.QueryId, "query1")
	require.Equal(res.Report.AggregateReporter, "reporter4")
	require.Equal(res.Report.AggregateValue, "28")
	require.Equal(res.Report.ReporterPower, int64(20))
	//  check list of reporters in the aggregate report
	require.Equal(res.Report.Reporters[0].Reporter, "reporter1")
	require.Equal(res.Report.Reporters[1].Reporter, "reporter2")

	weightedMean := float64((10*10)+(20*4)+(30*2)+(40*20)+(50*8)) / (10 + 4 + 2 + 20 + 8)
	sum := ((10 * math.Pow(10-weightedMean, 2)) + (4 * math.Pow(20-weightedMean, 2)) + (2 * math.Pow(30-weightedMean, 2)) + (20 * math.Pow(40-weightedMean, 2)) + (8 * math.Pow(50-weightedMean, 2))) / (10 + 4 + 2 + 20 + 8)
	require.Equal(res.Report.StandardDeviation, math.Sqrt(sum))
}
