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

func (s *KeeperTestSuite) TestAddReporterQueryShareSum() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	reporter := sample.AccAddressBytes()
	queryId := []byte("test_query_id")

	// Add share: 100 / 600 = 0.1666...
	err := k.AddReporterQueryShareSum(ctx, reporter, queryId, 100, 600)
	require.NoError(err)

	// Verify it was recorded
	shareSum, err := k.ReporterQueryShareSum.Get(ctx, collections.Join([]byte(reporter), queryId))
	require.NoError(err)

	// Expected: 100 / 600 = 0.1666...
	expectedShare := math.LegacyNewDec(100).Quo(math.LegacyNewDec(600))
	require.Equal(expectedShare, shareSum)

	// Add another share for same query: 200 / 500 = 0.4
	err = k.AddReporterQueryShareSum(ctx, reporter, queryId, 200, 500)
	require.NoError(err)

	// Verify sum is updated
	shareSum, err = k.ReporterQueryShareSum.Get(ctx, collections.Join([]byte(reporter), queryId))
	require.NoError(err)

	// Expected: 0.1666... + 0.4 = 0.5666...
	secondShare := math.LegacyNewDec(200).Quo(math.LegacyNewDec(500))
	require.Equal(expectedShare.Add(secondShare), shareSum)
}

func (s *KeeperTestSuite) TestUpdateReporterLiveness() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	reporter := sample.AccAddressBytes()
	queryId1 := []byte("test_query_id_1")
	queryId2 := []byte("test_query_id_2")
	reporterPower := uint64(100)
	aggregateTotalPower := uint64(600)

	// First report (standard query)
	err := k.UpdateReporterLiveness(ctx, reporter, queryId1, reporterPower, aggregateTotalPower, false)
	require.NoError(err)

	// Verify reporter query share sum was tracked
	shareSum, err := k.ReporterQueryShareSum.Get(ctx, collections.Join([]byte(reporter), queryId1))
	require.NoError(err)
	expectedShare := math.LegacyNewDec(int64(reporterPower)).Quo(math.LegacyNewDec(int64(aggregateTotalPower)))
	require.Equal(expectedShare, shareSum)

	// Verify standard share sum was also tracked (since isNonStandard=false)
	standardShare, err := k.ReporterStandardShareSum.Get(ctx, reporter)
	require.NoError(err)
	require.Equal(expectedShare, standardShare)

	// Second report for different query (standard query)
	err = k.UpdateReporterLiveness(ctx, reporter, queryId2, reporterPower, aggregateTotalPower, false)
	require.NoError(err)

	// Verify standard share sum is updated (sum of both queries)
	standardShare, err = k.ReporterStandardShareSum.Get(ctx, reporter)
	require.NoError(err)
	require.Equal(expectedShare.MulInt64(2), standardShare)
}

func (s *KeeperTestSuite) TestUpdateReporterLiveness_NonStandard() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	reporter := sample.AccAddressBytes()
	queryId := []byte("test_query_id")
	reporterPower := uint64(100)
	aggregateTotalPower := uint64(600)

	// Report for non-standard query
	err := k.UpdateReporterLiveness(ctx, reporter, queryId, reporterPower, aggregateTotalPower, true)
	require.NoError(err)

	// Verify reporter query share sum was tracked
	shareSum, err := k.ReporterQueryShareSum.Get(ctx, collections.Join([]byte(reporter), queryId))
	require.NoError(err)
	expectedShare := math.LegacyNewDec(int64(reporterPower)).Quo(math.LegacyNewDec(int64(aggregateTotalPower)))
	require.Equal(expectedShare, shareSum)

	// Verify standard share sum was NOT tracked (since isNonStandard=true)
	_, err = k.ReporterStandardShareSum.Get(ctx, reporter)
	require.Error(err) // Should not exist
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

	// Distribute - should reset data since TBR is 0
	err := k.DistributeLivenessRewards(ctx)
	require.NoError(err)
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

	// Mock the SendCoinsFromModuleToModule call (TBR is still transferred even with no reporters)
	s.bankKeeper.On("SendCoinsFromModuleToModule", ctx, minttypes.TimeBasedRewards, "tips_escrow_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(1000)))).Return(nil).Once()

	// Distribute
	err := k.DistributeLivenessRewards(ctx)
	require.NoError(err)

	// Since no reporters, all TBR should become dust
	dust, err := k.Dust.Get(ctx)
	require.NoError(err)
	require.Equal(math.NewInt(1000), dust)
}

func (s *KeeperTestSuite) TestResetLivenessData() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	reporter := sample.AccAddressBytes()
	queryId := []byte("test_query_id")

	// Setup some data
	require.NoError(k.QueryOpportunities.Set(ctx, queryId, 2))
	require.NoError(k.ReporterQueryShareSum.Set(ctx, collections.Join([]byte(reporter), queryId), math.LegacyNewDec(1000)))
	require.NoError(k.Dust.Set(ctx, math.NewInt(50)))
	require.NoError(k.ReporterStandardShareSum.Set(ctx, reporter, math.LegacyNewDec(500)))
	require.NoError(k.NonStandardQueries.Set(ctx, queryId, true))
	require.NoError(k.StandardOpportunities.Set(ctx, 3))

	// Reset
	err := k.ResetLivenessData(ctx)
	require.NoError(err)

	// Verify QueryOpportunities is cleared
	_, err = k.QueryOpportunities.Get(ctx, queryId)
	require.Error(err)

	// Verify ReporterQueryShareSum is cleared
	_, err = k.ReporterQueryShareSum.Get(ctx, collections.Join([]byte(reporter), queryId))
	require.Error(err)

	// Verify Dust is preserved (not reset)
	dust, err := k.Dust.Get(ctx)
	require.NoError(err)
	require.Equal(math.NewInt(50), dust)

	// Verify ReporterStandardShareSum is cleared
	_, err = k.ReporterStandardShareSum.Get(ctx, reporter)
	require.Error(err)

	// Verify NonStandardQueries is cleared
	has, err := k.NonStandardQueries.Has(ctx, queryId)
	require.NoError(err)
	require.False(has)

	// Verify StandardOpportunities is reset to 0
	stdOpp, err := k.StandardOpportunities.Get(ctx)
	require.NoError(err)
	require.Equal(uint64(0), stdOpp)
}

func (s *KeeperTestSuite) TestPowerShareCalculation() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// Scenario from spreadsheet:
	// Query aaa has 2 aggregates, query bbb has 1, query ccc has 1
	// Total TBR = 1000, cyclelist = 3 queries
	// reward_per_query = 1000/3 = 333.33

	queryAaa := []byte("aaa")
	queryBbb := []byte("bbb")
	queryCcc := []byte("ccc")

	// Set up opportunities
	require.NoError(k.QueryOpportunities.Set(ctx, queryAaa, 2))
	require.NoError(k.QueryOpportunities.Set(ctx, queryBbb, 1))
	require.NoError(k.QueryOpportunities.Set(ctx, queryCcc, 1))

	// Reporter Alice: power 100 in agg1 for aaa, power 100 for bbb, power 150 for ccc
	// Aggregate powers: aaa_agg1=600, bbb=600, ccc=350
	alice := sample.AccAddressBytes()

	// Alice's shares:
	// aaa: 100/600 = 0.1667 (only reported on 1 of 2 aggregates)
	// bbb: 100/600 = 0.1667
	// ccc: 150/350 = 0.4286
	require.NoError(k.AddReporterQueryShareSum(ctx, alice, queryAaa, 100, 600))
	require.NoError(k.AddReporterQueryShareSum(ctx, alice, queryBbb, 100, 600))
	require.NoError(k.AddReporterQueryShareSum(ctx, alice, queryCcc, 150, 350))

	// Verify the shares are stored correctly
	aliceAaaShare, err := k.ReporterQueryShareSum.Get(ctx, collections.Join([]byte(alice), queryAaa))
	require.NoError(err)

	// Expected: 100 / 600 = 0.1666...
	expectedAliceAaa := math.LegacyNewDec(100).Quo(math.LegacyNewDec(600))
	require.Equal(expectedAliceAaa, aliceAaaShare)

	// At distribution time with TBR=1000 and 3 queries:
	// rewardPerQuery = 1000/3 = 333.33
	// Alice reward for aaa = (0.1667 / 2) * 333.33 = 27.78
	// Alice reward for bbb = (0.1667 / 1) * 333.33 = 55.56
	// Alice reward for ccc = (0.4286 / 1) * 333.33 = 142.86
	// Total Alice = 226.20

	// This test verifies the share accumulation is correct
	// The actual distribution is tested in integration tests
}

func (s *KeeperTestSuite) TestIncrementStandardOpportunities() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// First increment - should set to 1
	err := k.IncrementStandardOpportunities(ctx)
	require.NoError(err)

	opp, err := k.StandardOpportunities.Get(ctx)
	require.NoError(err)
	require.Equal(uint64(1), opp)

	// Second increment - should set to 2
	err = k.IncrementStandardOpportunities(ctx)
	require.NoError(err)

	opp, err = k.StandardOpportunities.Get(ctx)
	require.NoError(err)
	require.Equal(uint64(2), opp)
}

func (s *KeeperTestSuite) TestDemoteQueryToNonStandard() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	reporter1 := sample.AccAddressBytes()
	reporter2 := sample.AccAddressBytes()
	queryId := []byte("test_query_id")
	otherQueryId := []byte("other_query_id")

	// Setup: reporters have standard shares for queryId
	share1 := math.LegacyNewDec(100).Quo(math.LegacyNewDec(600))
	share2 := math.LegacyNewDec(200).Quo(math.LegacyNewDec(600))

	// Simulate UpdateReporterLiveness for standard queries
	require.NoError(k.ReporterQueryShareSum.Set(ctx, collections.Join([]byte(reporter1), queryId), share1))
	require.NoError(k.ReporterQueryShareSum.Set(ctx, collections.Join([]byte(reporter2), queryId), share2))
	require.NoError(k.ReporterQueryShareSum.Set(ctx, collections.Join([]byte(reporter1), otherQueryId), share1))

	require.NoError(k.ReporterStandardShareSum.Set(ctx, reporter1, share1.Add(share1))) // queryId + otherQueryId
	require.NoError(k.ReporterStandardShareSum.Set(ctx, reporter2, share2))             // only queryId

	// Set standard opportunities
	require.NoError(k.StandardOpportunities.Set(ctx, 2))

	// Demote queryId to non-standard
	err := k.DemoteQueryToNonStandard(ctx, queryId)
	require.NoError(err)

	// Verify queryId is now non-standard
	isNonStandard, err := k.NonStandardQueries.Has(ctx, queryId)
	require.NoError(err)
	require.True(isNonStandard)

	// Verify query opportunities is set
	opp, err := k.QueryOpportunities.Get(ctx, queryId)
	require.NoError(err)
	require.Equal(uint64(2), opp) // Should be set to current standard opportunities

	// Verify reporter1's standard share sum is reduced (only otherQueryId remains)
	r1Standard, err := k.ReporterStandardShareSum.Get(ctx, reporter1)
	require.NoError(err)
	require.Equal(share1, r1Standard) // Only otherQueryId's share remains

	// Verify reporter2's standard share sum is reduced to zero
	r2Standard, err := k.ReporterStandardShareSum.Get(ctx, reporter2)
	require.NoError(err)
	require.True(r2Standard.IsZero())

	// Verify per-query shares are still there (for non-standard calculation)
	r1QueryShare, err := k.ReporterQueryShareSum.Get(ctx, collections.Join([]byte(reporter1), queryId))
	require.NoError(err)
	require.Equal(share1, r1QueryShare)

	// Second demotion should be a no-op
	err = k.DemoteQueryToNonStandard(ctx, queryId)
	require.NoError(err)
}
