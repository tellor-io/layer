package integration_test

import (
	"encoding/hex"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/utils"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestTRBBridgeTBRDistribution tests that TRBBridge queries receive a share of TBR
// even though they're not in the cyclelist. TRBBridge queries as a group get 1 "slot"
// (same as 1 cyclelist query), and that slot is split among all TRBBridge reporters.
func (s *IntegrationTestSuite) TestTRBBridgeTBRDistribution() {
	require := s.Require()

	// Set up consensus params for vote extensions
	ctx := s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{VoteExtensionsEnableHeight: 1},
	})

	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(1)

	// Set LivenessCycles high so distribution doesn't happen mid-test
	params, err := s.Setup.Oraclekeeper.GetParams(ctx)
	require.NoError(err)
	params.LivenessCycles = 100
	require.NoError(s.Setup.Oraclekeeper.SetParams(ctx, params))
	require.NoError(s.Setup.Oraclekeeper.CycleCount.Set(ctx, 50))

	// Reset liveness data to start fresh
	require.NoError(s.Setup.Oraclekeeper.ResetLivenessData(ctx))

	// Create 3 validators with equal power
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 100, 100})

	// Set up bridge EVM addresses (required for TRB bridge queries)
	for _, val := range valAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("evmaddr"))
		s.NoError(err)
	}

	// Create reporters (each validator is their own reporter/selector)
	for i, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(
			reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter"+string(rune('0'+i)),
		)))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
		_, err := s.Setup.Reporterkeeper.ReporterStake(ctx, rep, []byte{})
		s.NoError(err)
	}

	// Fund TBR pool with a known amount
	tipper := s.newKeysWithTokens()
	tbrAmount := math.NewInt(400_000) // Use 400,000 so it divides evenly by 4 (3 cyclelist + 1 TRBBridge)
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, tbrAmount))))

	// Verify TBR pool has the expected amount
	tbrBalance := s.Setup.Oraclekeeper.GetTimeBasedRewards(ctx)
	require.Equal(tbrAmount, tbrBalance, "TBR pool should have the funded amount")

	// Helper to get available tips
	reporterQuerier := reporterkeeper.NewQuerier(s.Setup.Reporterkeeper)
	getAvailableTips := func() []math.LegacyDec {
		tips := make([]math.LegacyDec, 3)
		for i, rep := range repAccs {
			resp, err := reporterQuerier.AvailableTips(ctx, &reportertypes.QueryAvailableTipsRequest{
				SelectorAddress: rep.String(),
			})
			s.NoError(err)
			tips[i] = resp.AvailableTips
		}
		return tips
	}

	initialTips := getAvailableTips()
	s.T().Logf("Initial tips: [%s, %s, %s]", initialTips[0], initialTips[1], initialTips[2])

	oracleMsgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	// Create TRBBridge query data
	spec := registrytypes.DataSpec{
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "tolayer", FieldType: "bool"},
			{Name: "depositId", FieldType: "uint256"},
		},
	}
	trbBridgeQueryData, err := spec.EncodeData("TRBBridge", `["true","1"]`)
	s.NoError(err)
	trbBridgeQueryId := utils.QueryIDFromData(trbBridgeQueryData)
	s.T().Logf("TRBBridge queryId: %s", hex.EncodeToString(trbBridgeQueryId[:4]))

	// Track which cyclelist queries we've submitted for
	queriesSubmitted := make(map[string]bool)
	cycleCompleted := false

	// Run blocks until cycle completes (when all 3 queries have been processed)
	for block := 1; block <= 20 && !cycleCompleted; block++ {
		ctx = ctx.WithBlockHeight(int64(block))
		ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))

		// BeginBlocker
		_, err := s.Setup.App.BeginBlocker(ctx)
		s.NoError(err)

		// Get current query in cycle
		currentQueryData, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(ctx)
		s.NoError(err)
		queryId := utils.QueryIDFromData(currentQueryData)
		queryIdStr := string(queryId)

		// Submit for cyclelist query if we haven't already
		if !queriesSubmitted[queryIdStr] {
			value := testutil.EncodeValue(100_00)

			s.T().Logf("Block %d: Submitting for cyclelist query %x", block, queryId[:4])

			// All reporters submit for cyclelist
			for _, rep := range repAccs {
				msg := types.MsgSubmitValue{
					Creator:   rep.String(),
					QueryData: currentQueryData,
					Value:     value,
				}
				_, err := oracleMsgServer.SubmitValue(ctx, &msg)
				s.NoError(err)
			}
			queriesSubmitted[queryIdStr] = true
		}

		// EndBlocker - this aggregates and rotates
		_, err = s.Setup.App.EndBlocker(ctx)
		s.NoError(err)

		// Check if cycle completed (all 3 queries submitted)
		if len(queriesSubmitted) == 3 {
			cycleCompleted = true
		}
	}

	require.True(cycleCompleted, "Cyclelist cycle should have completed")
	s.T().Logf("Cyclelist cycle completed")

	// Now submit TRBBridge queries - these are not in cyclelist but should get TBR
	// Submit 2 separate TRBBridge queries to test that all share 1 slot
	// TRBBridge query 1 - only reporter 0 and 1 submit
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(ctx)
	s.NoError(err)

	s.T().Logf("Block %d: Submitting TRBBridge query (reporters 0 and 1)", ctx.BlockHeight())
	for _, rep := range repAccs[:2] { // Only first 2 reporters
		msg := types.MsgSubmitValue{
			Creator:   rep.String(),
			QueryData: trbBridgeQueryData,
			Value:     testValue,
		}
		_, err := oracleMsgServer.SubmitValue(ctx, &msg)
		s.NoError(err)
	}

	_, err = s.Setup.App.EndBlocker(ctx)
	s.NoError(err)

	// Advance time to aggregate the TRBBridge query (needs 1 hour to pass)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 2000)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Hour + time.Second*11))
	_, err = s.Setup.App.BeginBlocker(ctx)
	s.NoError(err)
	_, err = s.Setup.App.EndBlocker(ctx)
	s.NoError(err)

	// Verify TRBBridge aggregate was created
	agg, _, err := s.Setup.Oraclekeeper.GetCurrentAggregateReport(ctx, trbBridgeQueryId)
	s.NoError(err)
	s.NotNil(agg)
	s.T().Logf("TRBBridge aggregate created with power: %d", agg.AggregatePower)

	// Check TRBBridge marker has opportunities
	trbBridgeOpportunities, err := s.Setup.Oraclekeeper.QueryOpportunities.Get(ctx, keeper.TRBBridgeMarkerQueryId)
	s.NoError(err)
	s.T().Logf("TRBBridge opportunities: %d", trbBridgeOpportunities)
	require.Equal(uint64(1), trbBridgeOpportunities, "TRBBridge should have 1 opportunity")

	// Manually trigger distribution
	s.T().Logf("Triggering TBR distribution...")
	err = s.Setup.Oraclekeeper.DistributeLivenessRewards(ctx)
	s.NoError(err)

	// After distribution, TBR should be distributed
	tbrBalanceAfter := s.Setup.Oraclekeeper.GetTimeBasedRewards(ctx)
	s.T().Logf("Final TBR pool: %s", tbrBalanceAfter)

	// Get final tips
	finalTips := getAvailableTips()

	// Calculate tip deltas
	tipDeltas := make([]math.LegacyDec, 3)
	totalDelta := math.LegacyZeroDec()
	for i := range repAccs {
		tipDeltas[i] = finalTips[i].Sub(initialTips[i])
		totalDelta = totalDelta.Add(tipDeltas[i])
	}

	s.T().Logf("Final tips: [%s, %s, %s]", finalTips[0], finalTips[1], finalTips[2])
	s.T().Logf("Tip deltas: [%s, %s, %s]", tipDeltas[0], tipDeltas[1], tipDeltas[2])
	s.T().Logf("Total distributed: %s (expected: %s)", totalDelta, tbrAmount)

	// Assertions
	// 1. TBR should have been distributed (pool should be empty)
	require.True(tbrBalanceAfter.IsZero(), "TBR pool should be empty after distribution")

	// 2. All reporters should have received some TBR (from cyclelist)
	for i, delta := range tipDeltas {
		require.True(delta.GT(math.LegacyZeroDec()),
			"Reporter %d should have received TBR, got %s", i, delta)
	}

	// 3. Reporter 0 and 1 should have received more than reporter 2
	//    (because they participated in TRBBridge as well as cyclelist)
	require.True(tipDeltas[0].GT(tipDeltas[2]),
		"Reporter 0 should have received more than reporter 2 (TRBBridge participation)")
	require.True(tipDeltas[1].GT(tipDeltas[2]),
		"Reporter 1 should have received more than reporter 2 (TRBBridge participation)")

	// 4. Reporter 0 and 1 should have received roughly equal amounts
	//    (both participated in all queries)
	ratio01 := tipDeltas[0].Quo(tipDeltas[1])
	s.T().Logf("Ratio reporter0/reporter1: %.2f (expected ~1.0)", ratio01.MustFloat64())
	require.True(ratio01.GT(math.LegacyMustNewDecFromStr("0.99")) && ratio01.LT(math.LegacyMustNewDecFromStr("1.01")),
		"Reporter 0 and 1 should have roughly equal rewards")

	// 5. Calculate expected distribution:
	//    - 4 slots total (3 cyclelist + 1 TRBBridge)
	//    - rewardPerSlot = 400,000 / 4 = 100,000 per slot
	//
	// For cyclelist (all standard, StandardOpportunities = 1):
	//    - Each reporter gets: 100,000/3 + 100,000/3 + 100,000/3 = 100,000
	//
	// For TRBBridge (1 slot, 1 opportunity, 2 reporters with equal power):
	//    - Reporter 0: (1/2) * 100,000 = 50,000
	//    - Reporter 1: (1/2) * 100,000 = 50,000
	//    - Reporter 2: 0
	//
	// Expected totals:
	//    - Reporter 0: 100,000 + 50,000 = 150,000
	//    - Reporter 1: 100,000 + 50,000 = 150,000
	//    - Reporter 2: 100,000 + 0 = 100,000
	//
	// Ratio 0:2 = 150,000 / 100,000 = 1.5
	ratio02 := tipDeltas[0].Quo(tipDeltas[2])
	s.T().Logf("Ratio reporter0/reporter2: %.2f (expected ~1.5)", ratio02.MustFloat64())
	require.True(ratio02.GT(math.LegacyMustNewDecFromStr("1.4")) && ratio02.LT(math.LegacyMustNewDecFromStr("1.6")),
		"Reporter 0 should have ~1.5x more rewards than reporter 2 due to TRBBridge participation")

	s.T().Logf("SUCCESS: TRBBridge queries received TBR distribution!")
}

// TestTRBBridgeTBRDistributionMultipleAggregates tests that multiple TRBBridge aggregates
// share the same slot proportionally
func (s *IntegrationTestSuite) TestTRBBridgeTBRDistributionMultipleAggregates() {
	require := s.Require()

	// Set up consensus params for vote extensions
	ctx := s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{VoteExtensionsEnableHeight: 1},
	})

	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(1)

	// Set LivenessCycles high so distribution doesn't happen mid-test
	params, err := s.Setup.Oraclekeeper.GetParams(ctx)
	require.NoError(err)
	params.LivenessCycles = 100
	require.NoError(s.Setup.Oraclekeeper.SetParams(ctx, params))
	require.NoError(s.Setup.Oraclekeeper.CycleCount.Set(ctx, 50))

	// Reset liveness data to start fresh
	require.NoError(s.Setup.Oraclekeeper.ResetLivenessData(ctx))

	// Create 2 validators with equal power
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 100})

	// Set up bridge EVM addresses (required for TRB bridge queries)
	for _, val := range valAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("evmaddr"))
		s.NoError(err)
	}

	// Create reporters
	for i, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(
			reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter"+string(rune('0'+i)),
		)))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
		_, err := s.Setup.Reporterkeeper.ReporterStake(ctx, rep, []byte{})
		s.NoError(err)
	}

	// Fund TBR pool
	tipper := s.newKeysWithTokens()
	tbrAmount := math.NewInt(400_000)
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, tbrAmount))))

	// Helper to get available tips
	reporterQuerier := reporterkeeper.NewQuerier(s.Setup.Reporterkeeper)
	getAvailableTips := func() []math.LegacyDec {
		tips := make([]math.LegacyDec, 2)
		for i, rep := range repAccs {
			resp, err := reporterQuerier.AvailableTips(ctx, &reportertypes.QueryAvailableTipsRequest{
				SelectorAddress: rep.String(),
			})
			s.NoError(err)
			tips[i] = resp.AvailableTips
		}
		return tips
	}

	initialTips := getAvailableTips()
	oracleMsgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	// Create two different TRBBridge queries
	spec := registrytypes.DataSpec{
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "tolayer", FieldType: "bool"},
			{Name: "depositId", FieldType: "uint256"},
		},
	}
	trbBridgeQueryData1, err := spec.EncodeData("TRBBridge", `["true","1"]`)
	s.NoError(err)
	trbBridgeQueryId1 := utils.QueryIDFromData(trbBridgeQueryData1)

	trbBridgeQueryData2, err := spec.EncodeData("TRBBridge", `["true","2"]`)
	s.NoError(err)
	trbBridgeQueryId2 := utils.QueryIDFromData(trbBridgeQueryData2)

	// Submit first TRBBridge query - both reporters
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(ctx)
	s.NoError(err)

	s.T().Logf("Submitting TRBBridge query 1 (both reporters)")
	for _, rep := range repAccs {
		msg := types.MsgSubmitValue{
			Creator:   rep.String(),
			QueryData: trbBridgeQueryData1,
			Value:     testValue,
		}
		_, err := oracleMsgServer.SubmitValue(ctx, &msg)
		s.NoError(err)
	}

	_, err = s.Setup.App.EndBlocker(ctx)
	s.NoError(err)

	// Advance time to aggregate
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 2000)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Hour + time.Second*11))
	_, err = s.Setup.App.BeginBlocker(ctx)
	s.NoError(err)
	_, err = s.Setup.App.EndBlocker(ctx)
	s.NoError(err)

	// Verify first aggregate
	agg1, _, err := s.Setup.Oraclekeeper.GetCurrentAggregateReport(ctx, trbBridgeQueryId1)
	s.NoError(err)
	s.NotNil(agg1)
	s.T().Logf("TRBBridge query 1 aggregated")

	// Submit second TRBBridge query - only reporter 0
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(ctx)
	s.NoError(err)

	s.T().Logf("Submitting TRBBridge query 2 (only reporter 0)")
	msg := types.MsgSubmitValue{
		Creator:   repAccs[0].String(),
		QueryData: trbBridgeQueryData2,
		Value:     testValue,
	}
	_, err = oracleMsgServer.SubmitValue(ctx, &msg)
	s.NoError(err)

	_, err = s.Setup.App.EndBlocker(ctx)
	s.NoError(err)

	// Advance time to aggregate
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 2000)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Hour + time.Second*11))
	_, err = s.Setup.App.BeginBlocker(ctx)
	s.NoError(err)
	_, err = s.Setup.App.EndBlocker(ctx)
	s.NoError(err)

	// Verify second aggregate
	agg2, _, err := s.Setup.Oraclekeeper.GetCurrentAggregateReport(ctx, trbBridgeQueryId2)
	s.NoError(err)
	s.NotNil(agg2)
	s.T().Logf("TRBBridge query 2 aggregated")

	// Check TRBBridge marker has 2 opportunities (one per aggregate)
	trbBridgeOpportunities, err := s.Setup.Oraclekeeper.QueryOpportunities.Get(ctx, keeper.TRBBridgeMarkerQueryId)
	s.NoError(err)
	s.T().Logf("TRBBridge opportunities: %d", trbBridgeOpportunities)
	require.Equal(uint64(2), trbBridgeOpportunities, "TRBBridge should have 2 opportunities (2 aggregates)")

	// Manually trigger distribution
	s.T().Logf("Triggering TBR distribution...")
	err = s.Setup.Oraclekeeper.DistributeLivenessRewards(ctx)
	s.NoError(err)

	// Get final tips
	finalTips := getAvailableTips()

	// Calculate tip deltas
	tipDeltas := make([]math.LegacyDec, 2)
	for i := range repAccs {
		tipDeltas[i] = finalTips[i].Sub(initialTips[i])
	}

	s.T().Logf("Tip deltas: [%s, %s]", tipDeltas[0], tipDeltas[1])

	// With 3 cyclelist queries (0 opportunities since we didn't submit) + 1 TRBBridge slot (2 opportunities):
	// Actually, we only have TRBBridge queries in this test, no cyclelist submissions
	// So total slots = 3 cyclelist (with 0 opportunities) + 1 TRBBridge (with 2 opportunities)
	//
	// Wait, the distribution formula uses opportunities per queryId.
	// For TRBBridge: 2 opportunities, reporter0 submitted to both, reporter1 submitted to 1
	// rewardPerSlot = 400,000 / 4 = 100,000 (if there are no cyclelist opportunities, this might be different)
	//
	// Actually let's think about this more carefully:
	// - numSlots = numCyclelistQueries (3) + hasTRBBridge (1 if opportunities > 0) = 4
	// - rewardPerSlot = 400,000 / 4 = 100,000
	// - For TRBBridge with 2 opportunities:
	//   - Reporter 0: share in agg1 (1/2) + share in agg2 (1/1) = 0.5 + 1.0 = 1.5 total shares
	//   - Reporter 1: share in agg1 (1/2) = 0.5 total shares
	//   - Average share for reporter 0: 1.5 / 2 opportunities = 0.75
	//   - Average share for reporter 1: 0.5 / 2 opportunities = 0.25
	//   - Reporter 0 TRBBridge reward: 0.75 * 100,000 = 75,000
	//   - Reporter 1 TRBBridge reward: 0.25 * 100,000 = 25,000

	// Reporter 0 should have received more than reporter 1
	require.True(tipDeltas[0].GT(tipDeltas[1]),
		"Reporter 0 should have received more than reporter 1 (participated in both TRBBridge)")

	// The ratio should be about 3:1 (0.75/0.25 = 3)
	ratio := tipDeltas[0].Quo(tipDeltas[1])
	s.T().Logf("Ratio reporter0/reporter1: %.2f (expected ~3.0)", ratio.MustFloat64())
	require.True(ratio.GT(math.LegacyMustNewDecFromStr("2.5")) && ratio.LT(math.LegacyMustNewDecFromStr("3.5")),
		"Reporter 0 should have ~3x more rewards than reporter 1")

	s.T().Logf("SUCCESS: Multiple TRBBridge aggregates share the slot correctly!")
}
