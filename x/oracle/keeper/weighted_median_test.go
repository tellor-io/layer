package keeper_test

import (
	"cosmossdk.io/math"
	cosmosmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestWeightedMedian() {
	require := s.Require()
	qId := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"
	reporters := make([]sdk.AccAddress, 18)
	for i := 0; i < 18; i++ {
		reporters[i] = testutil.GenerateRandomAddress()
	}
	// normal scenario - 5 reporters various weights
	// list of addresses
	valuesInt := []int{10, 20, 30, 40, 50}
	values := testutil.IntToHex(valuesInt)
	powers := []int64{10, 4, 2, 20, 8}
	expectedIndex := 3
	expectedValue := values[expectedIndex]
	expectedReporter := reporters[expectedIndex].String()
	expectedPower := powers[expectedIndex]
	totalPowers := testutil.SumArray(powers)
	currentReporters := reporters[:5]
	reports := testutil.GenerateReports(currentReporters, values, powers, qId)
	s.stakingKeeper.On("GetLastTotalPower", mock.Anything, mock.Anything).Return(cosmosmath.NewInt(totalPowers))
	s.distrKeeper.On("GetFeePoolCommunityCoins", mock.Anything).Return(sdk.DecCoins{sdk.NewDecCoinFromDec("loya", math.LegacyNewDec(100))})
	s.distrKeeper.On("AllocateTokensToValidator", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.distrKeeper.On("GetFeePool", mock.Anything).Return(distrtypes.FeePool{CommunityPool: sdk.DecCoins{sdk.NewDecCoinFromDec("loya", math.LegacyNewDec(1000))}})
	s.distrKeeper.On("SetFeePool", mock.Anything, mock.Anything).Return(nil)

	_, err := s.oracleKeeper.WeightedMedian(s.ctx, reports)
	require.NoError(err)
	res, err := s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	require.Nil(err)
	require.Equal(res.Report.QueryId, qId, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, expectedValue, "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, expectedPower, "reporter power is not correct")
	require.Equal(res.Report.AggregateReportIndex, int64(expectedIndex), "report index is not correct")
	//  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		require.Equal(res.Report.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	weightedMean := testutil.CalculateWeightedMean(valuesInt, powers)
	require.Equal(res.Report.StandardDeviation, testutil.CalculateStandardDeviation(valuesInt, powers, weightedMean), "std deviation is not correct")

	// // special case A -- lower weighted median and upper weighted median are equal, powers are equal
	// // calculates lower median
	qId = "a6f013ee236804827b77696d350e9f0ac3e879328f2a3021d473a0b778ad78ac"
	currentReporters = reporters[5:9]
	valuesInt = []int{10, 10, 20, 20}
	values = testutil.IntToHex(valuesInt)
	powers = []int64{1, 1, 1, 1}
	totalPowers = testutil.SumArray(powers)
	expectedIndex = 1
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	expectedPower = 1
	reports = testutil.GenerateReports(currentReporters, values, powers, qId)
	s.stakingKeeper.On("GetLastTotalPower", mock.Anything, mock.Anything).Return(cosmosmath.NewInt(totalPowers))
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	require.Nil(err)
	require.Nil(err)
	require.Equal(res.Report.QueryId, qId, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, expectedValue, "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, expectedPower, "reporter power is not correct")
	require.Equal(res.Report.AggregateReportIndex, int64(expectedIndex), "report index is not correct")
	// //  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		require.Equal(res.Report.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	weightedMean = testutil.CalculateWeightedMean(valuesInt, powers)
	require.Equal(res.Report.StandardDeviation, testutil.CalculateStandardDeviation(valuesInt, powers, weightedMean), "std deviation is not correct")

	// special case B -- lower weighted median and upper weighted median are equal, powers are not all equal
	// calculates lower median
	qId = "48e9e2c732ba278de6ac88a3a57a5c5ba13d3d8370e709b3b98333a57876ca95"
	currentReporters = reporters[9:13]
	valuesInt = []int{10, 10, 20, 20}
	values = testutil.IntToHex(valuesInt)
	powers = []int64{1, 2, 1, 2}
	totalPowers = testutil.SumArray(powers)
	expectedIndex = 1
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	expectedPower = powers[expectedIndex]
	reports = testutil.GenerateReports(currentReporters, values, powers, qId)
	s.stakingKeeper.On("GetLastTotalPower", mock.Anything, mock.Anything).Return(cosmosmath.NewInt(totalPowers))
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	require.Nil(err)
	require.Nil(err)
	require.Equal(res.Report.QueryId, qId, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, expectedValue, "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, expectedPower, "reporter power is not correct")
	require.Equal(res.Report.AggregateReportIndex, int64(expectedIndex), "report index is not correct")
	// //  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		require.Equal(res.Report.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	weightedMean = testutil.CalculateWeightedMean(valuesInt, powers)
	require.Equal(res.Report.StandardDeviation, testutil.CalculateStandardDeviation(valuesInt, powers, weightedMean), "std deviation is not correct")

	// // 5 reporters with even weights, should be equal to normal median
	qId = "907154958baee4fb0ce2bbe50728141ac76eb2dc1731b3d40f0890746dd07e62"
	currentReporters = reporters[13:18]
	valuesInt = []int{10, 20, 30, 40, 50}
	values = testutil.IntToHex(valuesInt)
	powers = []int64{5, 5, 5, 5, 5}
	totalPowers = testutil.SumArray(powers)
	expectedIndex = 2
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	expectedPower = powers[expectedIndex]
	reports = testutil.GenerateReports(currentReporters, values, powers, qId)
	s.stakingKeeper.On("GetLastTotalPower", mock.Anything, mock.Anything).Return(cosmosmath.NewInt(totalPowers))
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	require.Nil(err)
	require.Nil(err)
	require.Equal(res.Report.QueryId, qId, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, expectedValue, "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, expectedPower, "reporter power is not correct")
	require.Equal(res.Report.AggregateReportIndex, int64(expectedIndex), "report index is not correct")
	// //  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		require.Equal(res.Report.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	weightedMean = testutil.CalculateWeightedMean(valuesInt, powers)
	require.Equal(res.Report.StandardDeviation, testutil.CalculateStandardDeviation(valuesInt, powers, weightedMean), "std deviation is not correct")
}
