package keeper_test

import (
	"crypto/rand"
	"fmt"
	"math"

	cosmosmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/x/oracle/types"
)

func GenerateRandomAddress() sdk.AccAddress {
	randBytes := make([]byte, 20)
	rand.Read(randBytes)
	return sdk.AccAddress(randBytes)
}

func GenerateReports(reporters []sdk.AccAddress, values []string, powers []int64, qId string) []types.MicroReport {
	var reports []types.MicroReport
	for i, reporter := range reporters {
		reports = append(reports, types.MicroReport{
			Reporter: reporter.String(),
			Value:    values[i],
			Power:    powers[i],
			QueryId:  qId,
		})
	}
	return reports
}

func SumArray(arr []int64) int64 {
	sum := int64(0)
	for _, value := range arr {
		sum += value
	}
	return sum
}

func CalculateWeightedMean(values []int, powers []int64) float64 {
	var totalWeight, weightedSum float64
	for i, value := range values {
		weightedSum += float64(value) * float64(powers[i])
		totalWeight += float64(powers[i])
	}
	return weightedSum / totalWeight
}

func CalculateStandardDeviation(values []int, powers []int64, mean float64) float64 {
	var sum float64
	totalWeight := float64(SumArray(powers))

	for i, value := range values {
		deviation := float64(value) - mean
		sum += float64(powers[i]) * deviation * deviation
	}

	return math.Sqrt(sum / totalWeight)
}

func IntToHex(values []int) []string {
	var hexValues []string
	for _, value := range values {
		hexValues = append(hexValues, fmt.Sprintf("%x", value))
	}
	return hexValues
}
func (s *KeeperTestSuite) TestWeightedMedian() {
	require := s.Require()
	qId := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"
	reporters := make([]sdk.AccAddress, 18)
	for i := 0; i < 18; i++ {
		reporters[i] = GenerateRandomAddress()
	}
	// normal scenario - 5 reporters various weights
	// list of addresses
	valuesInt := []int{10, 20, 30, 40, 50}
	values := IntToHex(valuesInt)
	powers := []int64{10, 4, 2, 20, 8}
	expectedIndex := 3
	expectedValue := values[expectedIndex]
	expectedReporter := reporters[expectedIndex].String()
	expectedPower := powers[expectedIndex]
	totalPowers := SumArray(powers)
	currentReporters := reporters[:5]
	reports := GenerateReports(currentReporters, values, powers, qId)
	s.stakingKeeper.On("GetLastTotalPower", mock.Anything, mock.Anything).Return(cosmosmath.NewInt(totalPowers))
	s.distrKeeper.On("GetFeePoolCommunityCoins", mock.Anything).Return(sdk.DecCoins{sdk.NewDecCoinFromDec("loya", sdk.NewDec(100))})
	s.distrKeeper.On("AllocateTokensToValidator", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.distrKeeper.On("GetFeePool", mock.Anything).Return(distrtypes.FeePool{CommunityPool: sdk.DecCoins{sdk.NewDecCoinFromDec("loya", sdk.NewDec(1000))}})
	s.distrKeeper.On("SetFeePool", mock.Anything, mock.Anything).Return(nil)

	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err := s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	require.Nil(err)
	require.Equal(res.Report.QueryId, qId, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, expectedValue, "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, expectedPower, "reporter power is not correct")
	//  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		require.Equal(res.Report.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	weightedMean := CalculateWeightedMean(valuesInt, powers)
	require.Equal(res.Report.StandardDeviation, CalculateStandardDeviation(valuesInt, powers, weightedMean), "std deviation is not correct")

	// // special case A -- lower weighted median and upper weighted median are equal, powers are equal
	// // calculates lower median
	qId = "a6f013ee236804827b77696d350e9f0ac3e879328f2a3021d473a0b778ad78ac"
	currentReporters = reporters[5:9]
	valuesInt = []int{10, 10, 20, 20}
	values = IntToHex(valuesInt)
	powers = []int64{1, 1, 1, 1}
	totalPowers = SumArray(powers)
	expectedIndex = 1
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	expectedPower = 1
	reports = GenerateReports(currentReporters, values, powers, qId)
	s.stakingKeeper.On("GetLastTotalPower", mock.Anything, mock.Anything).Return(cosmosmath.NewInt(totalPowers))
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	require.Nil(err)
	require.Nil(err)
	require.Equal(res.Report.QueryId, qId, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, expectedValue, "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, expectedPower, "reporter power is not correct")
	// //  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		require.Equal(res.Report.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	weightedMean = CalculateWeightedMean(valuesInt, powers)
	require.Equal(res.Report.StandardDeviation, CalculateStandardDeviation(valuesInt, powers, weightedMean), "std deviation is not correct")

	// special case B -- lower weighted median and upper weighted median are equal, powers are not all equal
	// calculates lower median
	qId = "48e9e2c732ba278de6ac88a3a57a5c5ba13d3d8370e709b3b98333a57876ca95"
	currentReporters = reporters[9:13]
	valuesInt = []int{10, 10, 20, 20}
	values = IntToHex(valuesInt)
	powers = []int64{1, 2, 1, 2}
	totalPowers = SumArray(powers)
	expectedIndex = 1
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	expectedPower = powers[expectedIndex]
	reports = GenerateReports(currentReporters, values, powers, qId)
	s.stakingKeeper.On("GetLastTotalPower", mock.Anything, mock.Anything).Return(cosmosmath.NewInt(totalPowers))
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	require.Nil(err)
	require.Nil(err)
	require.Equal(res.Report.QueryId, qId, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, expectedValue, "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, expectedPower, "reporter power is not correct")
	// //  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		require.Equal(res.Report.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	weightedMean = CalculateWeightedMean(valuesInt, powers)
	require.Equal(res.Report.StandardDeviation, CalculateStandardDeviation(valuesInt, powers, weightedMean), "std deviation is not correct")

	// // 5 reporters with even weights, should be equal to normal median
	qId = "907154958baee4fb0ce2bbe50728141ac76eb2dc1731b3d40f0890746dd07e62"
	currentReporters = reporters[13:18]
	valuesInt = []int{10, 20, 30, 40, 50}
	values = IntToHex(valuesInt)
	powers = []int64{5, 5, 5, 5, 5}
	totalPowers = SumArray(powers)
	expectedIndex = 2
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	expectedPower = powers[expectedIndex]
	reports = GenerateReports(currentReporters, values, powers, qId)
	s.stakingKeeper.On("GetLastTotalPower", mock.Anything, mock.Anything).Return(cosmosmath.NewInt(totalPowers))
	s.oracleKeeper.WeightedMedian(s.ctx, reports)
	res, err = s.oracleKeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	require.Nil(err)
	require.Nil(err)
	require.Equal(res.Report.QueryId, qId, "query id is not correct")
	require.Equal(res.Report.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	require.Equal(res.Report.AggregateValue, expectedValue, "aggregate value is not correct")
	require.Equal(res.Report.ReporterPower, expectedPower, "reporter power is not correct")
	// //  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		require.Equal(res.Report.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	weightedMean = CalculateWeightedMean(valuesInt, powers)
	require.Equal(res.Report.StandardDeviation, CalculateStandardDeviation(valuesInt, powers, weightedMean), "std deviation is not correct")
}
