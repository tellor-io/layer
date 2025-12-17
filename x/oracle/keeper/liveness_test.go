package keeper_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (s *KeeperTestSuite) TestIncrementQueryOpportunities() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	queryId := []byte("test_query_id")

	// First increment - should set to 1
	err := k.IncrementQueryOpportunities(ctx, queryId)
	require.NoError(err)

	opp, err := k.QueryOpportunities.Get(ctx, queryId)
	require.NoError(err)
	require.Equal(uint64(1), opp)

	// Second increment - should set to 2
	err = k.IncrementQueryOpportunities(ctx, queryId)
	require.NoError(err)

	opp, err = k.QueryOpportunities.Get(ctx, queryId)
	require.NoError(err)
	require.Equal(uint64(2), opp)

	// Third increment - should set to 3
	err = k.IncrementQueryOpportunities(ctx, queryId)
	require.NoError(err)

	opp, err = k.QueryOpportunities.Get(ctx, queryId)
	require.NoError(err)
	require.Equal(uint64(3), opp)
}

func (s *KeeperTestSuite) TestIncrementTotalQueriesInPeriod() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// First increment - should set to 1
	err := k.IncrementTotalQueriesInPeriod(ctx)
	require.NoError(err)

	total, err := k.TotalQueriesInPeriod.Get(ctx)
	require.NoError(err)
	require.Equal(uint64(1), total)

	// Second increment - should set to 2
	err = k.IncrementTotalQueriesInPeriod(ctx)
	require.NoError(err)

	total, err = k.TotalQueriesInPeriod.Get(ctx)
	require.NoError(err)
	require.Equal(uint64(2), total)
}

func (s *KeeperTestSuite) TestTrackReporterQuery() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	reporter := sample.AccAddressBytes()
	queryId := []byte("test_query_id")

	// Track reporter query
	err := k.TrackReporterQuery(ctx, reporter, queryId)
	require.NoError(err)

	// Verify it was recorded with count 1
	reported, err := k.ReporterQueriesInPeriod.Get(ctx, collections.Join([]byte(reporter), queryId))
	require.NoError(err)
	require.Equal(uint64(1), reported)
}

func (s *KeeperTestSuite) TestUpdateReporterLiveness() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	reporter := sample.AccAddressBytes()
	queryId1 := []byte("test_query_id_1")
	queryId2 := []byte("test_query_id_2")
	power := uint64(100)

	// First report
	err := k.UpdateReporterLiveness(ctx, reporter, queryId1, power)
	require.NoError(err)

	// Verify liveness record
	record, err := k.LivenessRecords.Get(ctx, reporter)
	require.NoError(err)
	require.Equal(uint64(1), record.QueriesReported)
	require.Equal(power, record.AccumulatedPower)

	// Verify reporter query was tracked with count 1
	reported, err := k.ReporterQueriesInPeriod.Get(ctx, collections.Join([]byte(reporter), queryId1))
	require.NoError(err)
	require.Equal(uint64(1), reported)

	// Second report for different query
	err = k.UpdateReporterLiveness(ctx, reporter, queryId2, power)
	require.NoError(err)

	// Verify updated liveness record
	record, err = k.LivenessRecords.Get(ctx, reporter)
	require.NoError(err)
	require.Equal(uint64(2), record.QueriesReported)
	require.Equal(power*2, record.AccumulatedPower)
}

func (s *KeeperTestSuite) TestCheckAndDistributeLivenessRewards_NotYetTime() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// Set LivenessCycles to 3 (distribute every 3 cycles)
	params := types.DefaultParams()
	params.LivenessCycles = 3
	require.NoError(k.SetParams(ctx, params))

	// Setup mocks for TBR (returns zero, so distribution is skipped)
	add := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(add)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)
	s.accountKeeper.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(testModuleAccount))
	s.bankKeeper.On("GetBalance", ctx, testModuleAccount.GetAddress(), "loya").Return(sdk.Coin{Amount: math.ZeroInt(), Denom: "loya"})

	// First cycle check - cycle 0, 0 % 3 == 0, so should distribute (but TBR is 0)
	err := k.CheckAndDistributeLivenessRewards(ctx)
	require.NoError(err)

	// Second cycle check - cycle 1, 1 % 3 != 0, so should NOT distribute
	err = k.CheckAndDistributeLivenessRewards(ctx)
	require.NoError(err)

	// Third cycle check - cycle 2, 2 % 3 != 0, so should NOT distribute
	err = k.CheckAndDistributeLivenessRewards(ctx)
	require.NoError(err)

	// Fourth cycle check - cycle 3, 3 % 3 == 0, so should distribute (but TBR is 0)
	err = k.CheckAndDistributeLivenessRewards(ctx)
	require.NoError(err)
}

func (s *KeeperTestSuite) TestDistributeLivenessRewards_ZeroTBR() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// Setup mocks for TBR - return zero balance
	add := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(add)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)
	s.accountKeeper.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(testModuleAccount))
	s.bankKeeper.On("GetBalance", ctx, testModuleAccount.GetAddress(), "loya").Return(sdk.Coin{Amount: math.ZeroInt(), Denom: "loya"}).Once()

	// Add some liveness data
	reporter := sample.AccAddressBytes()
	require.NoError(k.LivenessRecords.Set(ctx, reporter, types.LivenessRecord{
		QueriesReported:  3,
		AccumulatedPower: 100,
	}))
	require.NoError(k.TotalQueriesInPeriod.Set(ctx, 3))

	// Distribute - should reset data since TBR is 0
	err := k.DistributeLivenessRewards(ctx)
	require.NoError(err)

	// Verify data was reset
	_, err = k.LivenessRecords.Get(ctx, reporter)
	require.Error(err) // Should be cleared

	total, err := k.TotalQueriesInPeriod.Get(ctx)
	require.NoError(err)
	require.Equal(uint64(0), total)
}

func (s *KeeperTestSuite) TestDistributeLivenessRewards_NoReporters() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// Setup mocks for TBR - return some balance
	add := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(add)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)
	s.accountKeeper.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(testModuleAccount))
	s.bankKeeper.On("GetBalance", ctx, testModuleAccount.GetAddress(), "loya").Return(sdk.Coin{Amount: math.NewInt(1000), Denom: "loya"}).Once()

	// No reporters in period

	// Distribute - should reset data since no reporters
	err := k.DistributeLivenessRewards(ctx)
	require.NoError(err)

	// Verify data was reset
	total, err := k.TotalQueriesInPeriod.Get(ctx)
	require.NoError(err)
	require.Equal(uint64(0), total)
}

func (s *KeeperTestSuite) TestResetLivenessData() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	reporter := sample.AccAddressBytes()
	queryId := []byte("test_query_id")

	// Setup some data
	require.NoError(k.TotalQueriesInPeriod.Set(ctx, 5))
	require.NoError(k.LivenessRecords.Set(ctx, reporter, types.LivenessRecord{
		QueriesReported:  3,
		AccumulatedPower: 100,
	}))
	require.NoError(k.QueryOpportunities.Set(ctx, queryId, 2))
	require.NoError(k.ReporterQueriesInPeriod.Set(ctx, collections.Join([]byte(reporter), queryId), uint64(1)))
	require.NoError(k.Dust.Set(ctx, math.NewInt(50)))

	// Reset
	err := k.ResetLivenessData(ctx)
	require.NoError(err)

	// Verify TotalQueriesInPeriod is reset
	total, err := k.TotalQueriesInPeriod.Get(ctx)
	require.NoError(err)
	require.Equal(uint64(0), total)

	// Verify LivenessRecords is cleared
	_, err = k.LivenessRecords.Get(ctx, reporter)
	require.Error(err)

	// Verify QueryOpportunities is cleared
	_, err = k.QueryOpportunities.Get(ctx, queryId)
	require.Error(err)

	// Verify ReporterQueriesInPeriod is cleared
	_, err = k.ReporterQueriesInPeriod.Get(ctx, collections.Join([]byte(reporter), queryId))
	require.Error(err)

	// Verify Dust is preserved (not reset)
	dust, err := k.Dust.Get(ctx)
	require.NoError(err)
	require.Equal(math.NewInt(50), dust)
}

func (s *KeeperTestSuite) TestLivenessWeightCalculation() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// Scenario: 3 queries in cyclelist, query 1 has 2 opportunities (split weight)
	queryId1 := []byte("query_1")
	queryId2 := []byte("query_2")
	queryId3 := []byte("query_3")

	// Set up opportunities: query1 = 2 (split), query2 = 1, query3 = 1
	require.NoError(k.QueryOpportunities.Set(ctx, queryId1, 2))
	require.NoError(k.QueryOpportunities.Set(ctx, queryId2, 1))
	require.NoError(k.QueryOpportunities.Set(ctx, queryId3, 1))

	// Reporter A: reports on all 4 opportunities (query1 twice, query2 once, query3 once)
	reporterA := sample.AccAddressBytes()
	require.NoError(k.ReporterQueriesInPeriod.Set(ctx, collections.Join([]byte(reporterA), queryId1), uint64(2))) // reported twice
	require.NoError(k.ReporterQueriesInPeriod.Set(ctx, collections.Join([]byte(reporterA), queryId2), uint64(1)))
	require.NoError(k.ReporterQueriesInPeriod.Set(ctx, collections.Join([]byte(reporterA), queryId3), uint64(1)))
	require.NoError(k.LivenessRecords.Set(ctx, reporterA, types.LivenessRecord{
		QueriesReported:  4, // reported on query1 twice (in rotation + out-of-turn)
		AccumulatedPower: 100,
	}))

	// Reporter B: reports only on query2 and query3 (misses query1 entirely)
	reporterB := sample.AccAddressBytes()
	require.NoError(k.ReporterQueriesInPeriod.Set(ctx, collections.Join([]byte(reporterB), queryId2), uint64(1)))
	require.NoError(k.ReporterQueriesInPeriod.Set(ctx, collections.Join([]byte(reporterB), queryId3), uint64(1)))
	require.NoError(k.LivenessRecords.Set(ctx, reporterB, types.LivenessRecord{
		QueriesReported:  2,
		AccumulatedPower: 100,
	}))

	// Verify weighted liveness calculation:
	// Reporter A: (2/2 + 1/1 + 1/1) / 3 = (1 + 1 + 1) / 3 = 3/3 = 1.0 (100%)
	// Reporter B: (1/1 + 1/1) / 3 = 2/3 = 0.666... (66.7%)

	// This test verifies the data structure is set up correctly
	// The actual calculation is done in DistributeLivenessRewards
	oppA, err := k.QueryOpportunities.Get(ctx, queryId1)
	require.NoError(err)
	require.Equal(uint64(2), oppA)

	oppB, err := k.QueryOpportunities.Get(ctx, queryId2)
	require.NoError(err)
	require.Equal(uint64(1), oppB)
}
