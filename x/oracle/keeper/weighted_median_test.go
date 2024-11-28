package keeper_test

import (
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"cosmossdk.io/collections"
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
	powers := []uint64{10, 4, 2, 20, 8}
	expectedIndex := 3
	expectedValue := values[expectedIndex]
	expectedReporter := reporters[expectedIndex].String()
	var sumPowers uint64
	for _, power := range powers {
		sumPowers += power
	}
	expectedPower := sumPowers
	currentReporters := reporters[:5]
	reports := testutil.GenerateReports(currentReporters, values, powers, qId)

	aggregateReport, err := s.oracleKeeper.WeightedMedian(s.ctx, reports, 1)
	s.NoError(err)
	s.NoError(s.oracleKeeper.SetAggregate(s.ctx, aggregateReport))
	res, err := s.queryClient.GetCurrentAggregateReport(s.ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.Nil(err)
	s.Equal(res.Aggregate.QueryId, qId, "query id is not correct")
	s.Equal(res.Aggregate.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	s.Equal(res.Aggregate.AggregateValue, expectedValue, "aggregate value is not correct")
	s.Equal(res.Aggregate.AggregatePower, expectedPower, "reporter power is not correct")
	s.Equal(res.Aggregate.MetaId, uint64(1), "report meta id is not correct")
	//  check list of reporters in the aggregate report
	iter, err := s.oracleKeeper.Reports.Indexes.IdQueryId.MatchExact(s.ctx, collections.Join(res.Aggregate.MetaId, res.Aggregate.QueryId))
	s.NoError(err)
	i := 0
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		k, err := iter.PrimaryKey()
		s.NoError(err)
		microreport, err := s.oracleKeeper.Reports.Get(s.ctx, k)
		s.NoError(err)
		s.Equal(microreport.Reporter, currentReporters[i].String(), "reporter is not correct")
		i++
	}
	// weightedMean := testutil.CalculateWeightedMean(valuesInt, powers)

	// // special case A -- lower weighted median and upper weighted median are equal, powers are equal
	// // calculates lower median
	qId, _ = hex.DecodeString("a6f013ee236804827b77696d350e9f0ac3e879328f2a3021d473a0b778ad78ac")
	currentReporters = reporters[5:9]
	valuesInt = []int{10, 10, 20, 20}
	values = testutil.IntToHex(valuesInt)
	powers = []uint64{1, 1, 1, 1}
	expectedIndex = 1
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	sumPowers = uint64(0)
	for _, power := range powers {
		sumPowers += power
	}
	expectedPower = sumPowers
	reports = testutil.GenerateReports(currentReporters, values, powers, qId)
	aggregateReport, err = s.oracleKeeper.WeightedMedian(s.ctx, reports, 2)
	s.NoError(err)
	s.NoError(s.oracleKeeper.SetAggregate(s.ctx, aggregateReport))
	res, err = s.queryClient.GetCurrentAggregateReport(s.ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.Nil(err)
	s.Nil(err)
	s.Equal(res.Aggregate.QueryId, qId, "query id is not correct")
	s.Equal(res.Aggregate.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	s.Equal(res.Aggregate.AggregateValue, expectedValue, "aggregate value is not correct")
	s.Equal(res.Aggregate.AggregatePower, expectedPower, "reporter power is not correct")
	s.Equal(res.Aggregate.MetaId, uint64(2), "report meta id is not correct")
	// //  check list of reporters in the aggregate report
	iter, err = s.oracleKeeper.Reports.Indexes.IdQueryId.MatchExact(s.ctx, collections.Join(res.Aggregate.MetaId, res.Aggregate.QueryId))
	s.NoError(err)
	i = 0
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		k, err := iter.PrimaryKey()
		s.NoError(err)
		microreport, err := s.oracleKeeper.Reports.Get(s.ctx, k)
		s.NoError(err)
		s.Equal(microreport.Reporter, currentReporters[i].String(), "reporter is not correct")
		i++
	}
	// weightedMean = testutil.CalculateWeightedMean(valuesInt, powers)

	// special case B -- lower weighted median and upper weighted median are equal, powers are not all equal
	// calculates lower median
	qId, _ = hex.DecodeString("48e9e2c732ba278de6ac88a3a57a5c5ba13d3d8370e709b3b98333a57876ca95")
	currentReporters = reporters[9:13]
	valuesInt = []int{10, 10, 20, 20}
	values = testutil.IntToHex(valuesInt)
	powers = []uint64{1, 2, 1, 2}
	expectedIndex = 1
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	sumPowers = uint64(0)
	for _, power := range powers {
		sumPowers += power
	}
	expectedPower = sumPowers
	reports = testutil.GenerateReports(currentReporters, values, powers, qId)
	aggregateReport, err = s.oracleKeeper.WeightedMedian(s.ctx, reports, 3)
	s.NoError(err)
	s.NoError(s.oracleKeeper.SetAggregate(s.ctx, aggregateReport))
	res, err = s.queryClient.GetCurrentAggregateReport(s.ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.Nil(err)
	s.Nil(err)
	s.Equal(res.Aggregate.QueryId, qId, "query id is not correct")
	s.Equal(res.Aggregate.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	s.Equal(res.Aggregate.AggregateValue, expectedValue, "aggregate value is not correct")
	s.Equal(res.Aggregate.AggregatePower, expectedPower, "reporter power is not correct")
	s.Equal(res.Aggregate.MetaId, uint64(3), "report meta id is not correct")
	// //  check list of reporters in the aggregate report
	iter, err = s.oracleKeeper.Reports.Indexes.IdQueryId.MatchExact(s.ctx, collections.Join(res.Aggregate.MetaId, res.Aggregate.QueryId))
	s.NoError(err)
	i = 0
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		k, err := iter.PrimaryKey()
		s.NoError(err)
		microreport, err := s.oracleKeeper.Reports.Get(s.ctx, k)
		s.NoError(err)
		s.Equal(microreport.Reporter, currentReporters[i].String(), "reporter is not correct")
		i++
	}
	// weightedMean = testutil.CalculateWeightedMean(valuesInt, powers)

	// // 5 reporters with even weights, should be equal to normal median
	qId, _ = hex.DecodeString("907154958baee4fb0ce2bbe50728141ac76eb2dc1731b3d40f0890746dd07e62")
	currentReporters = reporters[13:18]
	valuesInt = []int{10, 20, 30, 40, 50}
	values = testutil.IntToHex(valuesInt)
	powers = []uint64{5, 5, 5, 5, 5}
	expectedIndex = 2
	expectedReporter = currentReporters[expectedIndex].String()
	expectedValue = values[expectedIndex]
	sumPowers = uint64(0)
	for _, power := range powers {
		sumPowers += power
	}
	expectedPower = sumPowers
	reports = testutil.GenerateReports(currentReporters, values, powers, qId)
	aggregateReport, err = s.oracleKeeper.WeightedMedian(s.ctx, reports, 4)
	s.NoError(err)
	s.NoError(s.oracleKeeper.SetAggregate(s.ctx, aggregateReport))
	res, err = s.queryClient.GetCurrentAggregateReport(s.ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.Nil(err)
	s.Nil(err)
	s.Equal(res.Aggregate.QueryId, qId, "query id is not correct")
	s.Equal(res.Aggregate.AggregateReporter, expectedReporter, "aggregate reporter is not correct")
	s.Equal(res.Aggregate.AggregateValue, expectedValue, "aggregate value is not correct")
	s.Equal(res.Aggregate.AggregatePower, expectedPower, "reporter power is not correct")

	s.Equal(res.Aggregate.MetaId, uint64(4), "report meta id is not correct")
	// //  check list of reporters in the aggregate report
	iter, err = s.oracleKeeper.Reports.Indexes.IdQueryId.MatchExact(s.ctx, collections.Join(res.Aggregate.MetaId, res.Aggregate.QueryId))
	s.NoError(err)
	i = 0
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		k, err := iter.PrimaryKey()
		s.NoError(err)
		microreport, err := s.oracleKeeper.Reports.Get(s.ctx, k)
		s.NoError(err)
		s.Equal(microreport.Reporter, currentReporters[i].String(), "reporter is not correct")
		i++
	}
	// weightedMean = testutil.CalculateWeightedMean(valuesInt, powers)
}

func (s *KeeperTestSuite) TestWeightedMedianBigNumbers() {
	require := s.Require()
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	s.ctx = s.ctx.WithBlockTime(time.Now())
	s.ctx = s.ctx.WithBlockHeight(1)
	reporters := make([]sdk.AccAddress, 18)
	for i := 0; i < 18; i++ {
		reporters[i] = sample.AccAddressBytes()
	}

	testCases := []struct {
		name                    string
		expectedError           bool
		numReports              int
		reporters               []sdk.AccAddress
		powers                  []uint64
		queryType               string
		queryId                 []byte
		aggregateMethod         string
		values                  []int
		timestamp               time.Time
		cyclelist               bool
		blocknumber             uint64
		expectedAggregateReport *types.Aggregate
	}{
		{
			name:            "normal cycle list report",
			numReports:      5,
			reporters:       reporters[:5],
			powers:          []uint64{10, 10, 10, 10, 10},
			queryType:       "SpotPrice",
			queryId:         qId,
			aggregateMethod: "weightedMedian",
			values:          []int{10, 11, 12, 13, 14},
			timestamp:       time.Now(),
			cyclelist:       true,
			blocknumber:     1,
			expectedAggregateReport: &types.Aggregate{
				QueryId:           qId,
				AggregateReporter: reporters[2].String(),
				AggregateValue:    testutil.IntToHex([]int{12})[0],
				AggregatePower:    50,
				MetaId:            1,
				Flagged:           false,
				Index:             1,
				Height:            1,
				MicroHeight:       1,
			},
		},
		{
			name:            "max int64 test",
			numReports:      5,
			reporters:       reporters[:5],
			powers:          []uint64{10, 10, 10, 10, 10},
			queryType:       "SpotPrice",
			queryId:         qId,
			aggregateMethod: "weightedMedian",
			values:          []int{1 * 1e18, 2 * 1e18, 3 * 1e18, 4 * 1e18, math.MaxInt64},
			timestamp:       time.Now(),
			cyclelist:       true,
			blocknumber:     1,
			expectedAggregateReport: &types.Aggregate{
				QueryId:           qId,
				AggregateReporter: reporters[2].String(),
				AggregateValue:    testutil.IntToHex([]int{3 * 1e18})[0],
				AggregatePower:    50,
				MetaId:            2,
				Flagged:           false,
				Index:             2,
				Height:            1,
				MicroHeight:       1,
			},
		},
	}
	metaId := uint64(0)
	for _, tc := range testCases {
		metaId++
		var reports []types.MicroReport
		s.Run(tc.name, func() {
			valuesInt := testutil.IntToHex(tc.values)
			for i := 0; i < tc.numReports; i++ {
				report := types.MicroReport{
					Reporter:        tc.reporters[i].String(),
					Power:           tc.powers[i],
					QueryType:       tc.queryType,
					QueryId:         tc.queryId,
					AggregateMethod: tc.aggregateMethod,
					Value:           valuesInt[i],
					Timestamp:       tc.timestamp,
					Cyclelist:       tc.cyclelist,
					BlockNumber:     tc.blocknumber,
				}
				reports = append(reports, report)
			}
			weightedMedian, err := s.oracleKeeper.WeightedMedian(s.ctx, reports, metaId)
			require.NoError(s.oracleKeeper.SetAggregate(s.ctx, weightedMedian))
			require.Equal(tc.expectedAggregateReport, weightedMedian)
			if tc.expectedError {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}

	// test max uint64, large powers
	uintCases := []struct {
		name                    string
		expectedError           bool
		numReports              int
		reporters               []sdk.AccAddress
		powers                  []uint64
		queryType               string
		queryId                 []byte
		aggregateMethod         string
		values                  []uint64
		timestamp               time.Time
		cyclelist               bool
		blocknumber             uint64
		expectedAggregateReport *types.Aggregate
	}{
		{
			name:            "max uint64 test",
			numReports:      5,
			reporters:       reporters[:5],
			powers:          []uint64{1 * 1e17, 1 * 1e17, 1 * 1e17, 1 * 1e17, 1 * 1e17},
			queryType:       "SpotPrice",
			queryId:         qId,
			aggregateMethod: "weightedMedian",
			values:          []uint64{1 * 1e18, 2 * 1e18, 3 * 1e18, 4 * 1e18, math.MaxUint64},
			timestamp:       time.Now(),
			cyclelist:       true,
			blocknumber:     1,
			expectedAggregateReport: &types.Aggregate{

				QueryId:           qId,
				AggregateReporter: reporters[2].String(),
				AggregateValue:    "29a2241af62c0000",
				AggregatePower:    5 * 1e17,
				MetaId:            3,
				Flagged:           false,
				Index:             3,
				Height:            1,
				MicroHeight:       1,
			},
		},
	}
	for _, tc := range uintCases {
		metaId++
		var reports []types.MicroReport
		s.Run(tc.name, func() {
			var hexValues []string
			for _, value := range tc.values {
				hexValues = append(hexValues, fmt.Sprintf("%x", value))
			}
			for i := 0; i < tc.numReports; i++ {
				report := types.MicroReport{
					Reporter:        tc.reporters[i].String(),
					Power:           tc.powers[i],
					QueryType:       tc.queryType,
					QueryId:         tc.queryId,
					AggregateMethod: tc.aggregateMethod,
					Value:           hexValues[i],
					Timestamp:       tc.timestamp,
					Cyclelist:       tc.cyclelist,
					BlockNumber:     tc.blocknumber,
				}
				reports = append(reports, report)
			}
			weightedMedian, err := s.oracleKeeper.WeightedMedian(s.ctx, reports, metaId)
			require.NoError(s.oracleKeeper.SetAggregate(s.ctx, weightedMedian))
			require.Equal(tc.expectedAggregateReport, weightedMedian)
			if tc.expectedError {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}
