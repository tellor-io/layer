package keeper_test

import (
	"encoding/hex"

	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestWeightedMedian() {
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	reporters := make([]sdk.AccAddress, 18)
	for i := 0; i < 18; i++ {
		reporters[i] = sample.AccAddressBytes()
	}
	// normal scenario - 5 reporters various weights
	// list of addresses
	valuesInt := []int{10, 20, 30, 40, 50}
	values := testutil.IntToHex(valuesInt)
	powers := []int64{10, 4, 2, 20, 8}
	expectedIndex := 3
	expectedValue := values[expectedIndex]
	expectedReporter := reporters[expectedIndex].String()
	var sumPowers int64
	for _, power := range powers {
		sumPowers += power
	}
	expectedPower := sumPowers
	currentReporters := reporters[:5]
	reports := testutil.GenerateReports(currentReporters, values, powers, qId)

	_, err := s.oracleKeeper.WeightedMedian(s.ctx, reports, 1)
	s.NoError(err)
	res, err := s.queryClient.GetCurrentAggregateReport(s.ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.Nil(err)
	s.Equal(res.Aggregate.QueryId, qId, "query id is not correct")
	s.Equal(res.Aggregate.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	s.Equal(res.Aggregate.AggregateValue, expectedValue, "aggregate value is not correct")
	s.Equal(res.Aggregate.ReporterPower, expectedPower, "reporter power is not correct")
	s.Equal(res.Aggregate.AggregateReportIndex, int64(expectedIndex), "report index is not correct")
	s.Equal(res.Aggregate.MetaId, uint64(1), "report meta id is not correct")
	//  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		s.Equal(res.Aggregate.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	// weightedMean := testutil.CalculateWeightedMean(valuesInt, powers)
	s.Equal(res.Aggregate.StandardDeviation, "0", "std deviation is not correct")

	// // special case A -- lower weighted median and upper weighted median are equal, powers are equal
	// // calculates lower median
	qId, _ = hex.DecodeString("a6f013ee236804827b77696d350e9f0ac3e879328f2a3021d473a0b778ad78ac")
	currentReporters = reporters[5:9]
	valuesInt = []int{10, 10, 20, 20}
	values = testutil.IntToHex(valuesInt)
	powers = []int64{1, 1, 1, 1}
	expectedIndex = 1
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	sumPowers = int64(0)
	for _, power := range powers {
		sumPowers += power
	}
	expectedPower = sumPowers
	reports = testutil.GenerateReports(currentReporters, values, powers, qId)
	_, err = s.oracleKeeper.WeightedMedian(s.ctx, reports, 2)
	s.NoError(err)
	res, err = s.queryClient.GetCurrentAggregateReport(s.ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.Nil(err)
	s.Nil(err)
	s.Equal(res.Aggregate.QueryId, qId, "query id is not correct")
	s.Equal(res.Aggregate.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	s.Equal(res.Aggregate.AggregateValue, expectedValue, "aggregate value is not correct")
	s.Equal(res.Aggregate.ReporterPower, expectedPower, "reporter power is not correct")
	s.Equal(res.Aggregate.AggregateReportIndex, int64(expectedIndex), "report index is not correct")
	s.Equal(res.Aggregate.MetaId, uint64(2), "report meta id is not correct")
	// //  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		s.Equal(res.Aggregate.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	// weightedMean = testutil.CalculateWeightedMean(valuesInt, powers)
	s.Equal(res.Aggregate.StandardDeviation, "0", "std deviation is not correct")

	// special case B -- lower weighted median and upper weighted median are equal, powers are not all equal
	// calculates lower median
	qId, _ = hex.DecodeString("48e9e2c732ba278de6ac88a3a57a5c5ba13d3d8370e709b3b98333a57876ca95")
	currentReporters = reporters[9:13]
	valuesInt = []int{10, 10, 20, 20}
	values = testutil.IntToHex(valuesInt)
	powers = []int64{1, 2, 1, 2}
	expectedIndex = 1
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	sumPowers = int64(0)
	for _, power := range powers {
		sumPowers += power
	}
	expectedPower = sumPowers
	reports = testutil.GenerateReports(currentReporters, values, powers, qId)
	_, err = s.oracleKeeper.WeightedMedian(s.ctx, reports, 3)
	s.NoError(err)
	res, err = s.queryClient.GetCurrentAggregateReport(s.ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.Nil(err)
	s.Nil(err)
	s.Equal(res.Aggregate.QueryId, qId, "query id is not correct")
	s.Equal(res.Aggregate.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	s.Equal(res.Aggregate.AggregateValue, expectedValue, "aggregate value is not correct")
	s.Equal(res.Aggregate.ReporterPower, expectedPower, "reporter power is not correct")
	s.Equal(res.Aggregate.AggregateReportIndex, int64(expectedIndex), "report index is not correct")
	s.Equal(res.Aggregate.MetaId, uint64(3), "report meta id is not correct")
	// //  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		s.Equal(res.Aggregate.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	// weightedMean = testutil.CalculateWeightedMean(valuesInt, powers)
	s.Equal(res.Aggregate.StandardDeviation, "0", "std deviation is not correct")

	// // 5 reporters with even weights, should be equal to normal median
	qId, _ = hex.DecodeString("907154958baee4fb0ce2bbe50728141ac76eb2dc1731b3d40f0890746dd07e62")
	currentReporters = reporters[13:18]
	valuesInt = []int{10, 20, 30, 40, 50}
	values = testutil.IntToHex(valuesInt)
	powers = []int64{5, 5, 5, 5, 5}
	expectedIndex = 2
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	sumPowers = int64(0)
	for _, power := range powers {
		sumPowers += power
	}
	expectedPower = sumPowers
	reports = testutil.GenerateReports(currentReporters, values, powers, qId)
	_, err = s.oracleKeeper.WeightedMedian(s.ctx, reports, 4)
	s.NoError(err)
	res, err = s.queryClient.GetCurrentAggregateReport(s.ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.Nil(err)
	s.Nil(err)
	s.Equal(res.Aggregate.QueryId, qId, "query id is not correct")
	s.Equal(res.Aggregate.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	s.Equal(res.Aggregate.AggregateValue, expectedValue, "aggregate value is not correct")
	s.Equal(res.Aggregate.ReporterPower, expectedPower, "reporter power is not correct")
	s.Equal(res.Aggregate.AggregateReportIndex, int64(expectedIndex), "report index is not correct")
	s.Equal(res.Aggregate.MetaId, uint64(4), "report meta id is not correct")
	// //  check list of reporters in the aggregate report
	for i, reporter := range currentReporters {
		s.Equal(res.Aggregate.Reporters[i].Reporter, reporter.String(), "reporter is not correct")
	}
	// weightedMean = testutil.CalculateWeightedMean(valuesInt, powers)
	s.Equal(res.Aggregate.StandardDeviation, "0", "std deviation is not correct")
}
