package integration_test

import (
	"encoding/hex"
	"fmt"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/utils"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestTBRDistributionNaturalFlow tests that TBR is distributed evenly when all reporters
// submit for the same cyclelist query
func (s *IntegrationTestSuite) TestTBRfullDistribution() {
	require := s.Require()

	// Set up consensus params for vote extensions
	ctx := s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{VoteExtensionsEnableHeight: 1},
	})

	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(1)

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
			reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter"+string(int32('0'+i)),
		)))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
		_, err := s.Setup.Reporterkeeper.ReporterStake(ctx, rep, []byte{})
		s.NoError(err)
	}

	// Fund TBR pool
	tipper := s.newKeysWithTokens()
	tbrAmount := math.NewInt(90_000) // Use 90,000 so it divides evenly by 3
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

	// Track which queries we've submitted for to detect cycle completion
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
		queryIdStr := fmt.Sprintf("%x", queryId)

		// Submit for this query if we haven't already
		if !queriesSubmitted[queryIdStr] {
			// Determine the value format based on query type
			value := testutil.EncodeValue(100_00)

			s.T().Logf("Block %d: Submitting for query %x", block, queryId[:4])

			// All reporters submit
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

		// Check if cycle completed (all 3 queries submitted and we're back to first)
		if len(queriesSubmitted) == 3 {
			cycleCompleted = true
		}

		tbrNow := s.Setup.Oraclekeeper.GetTimeBasedRewards(ctx)
		s.T().Logf("Block %d: TBR = %s, queries submitted = %d", block, tbrNow, len(queriesSubmitted))
	}

	require.True(cycleCompleted, "Cycle should have completed")

	// Manually trigger distribution since we set LivenessCycles high
	s.NoError(s.Setup.Oraclekeeper.DistributeLivenessRewards(ctx))

	// After completing the full cycle, TBR should be distributed
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
	s.T().Logf("Total tips received: %s", totalDelta)

	tbrDistributed := tbrAmount.Sub(tbrBalanceAfter)
	s.T().Logf("TBR distributed: %s (was %s, now %s)", tbrDistributed, tbrAmount, tbrBalanceAfter)

	// 1. TBR should have been distributed (pool should be empty)
	require.True(tbrBalanceAfter.IsZero(), "TBR pool should be empty after distribution")

	// 2. Total tips received should match what was distributed
	require.True(totalDelta.GT(math.LegacyZeroDec()),
		"Re wards should have been distributed")

	// 3. All reporters should have equal tips (since they all have equal power and participation)
	require.True(tipDeltas[0].Sub(tipDeltas[1]).Abs().LTE(math.LegacyNewDec(1)),
		"Reporter 0 and 1 should have equal tips: %s vs %s", tipDeltas[0], tipDeltas[1])
	require.True(tipDeltas[0].Sub(tipDeltas[2]).Abs().LTE(math.LegacyNewDec(1)),
		"Reporter 0 and 2 should have equal tips: %s vs %s", tipDeltas[0], tipDeltas[2])

	// 4. Each reporter should have ~33.3% of total distributed (equal share among 3 reporters)
	expectedShare := totalDelta.Quo(math.LegacyNewDec(3))
	for i, delta := range tipDeltas {
		require.True(delta.Sub(expectedShare).Abs().LTE(math.LegacyNewDec(1)),
			"Reporter %d should have ~33.3%% of total: got %s, expected %s", i, delta, expectedShare)
	}

	s.T().Log("SUCCESS: TBR was distributed evenly to all 3 reporters!")

	// Withdraw tips to verify they're actually claimable
	reporterMsgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	for i, rep := range repAccs {
		bondBefore, err := s.Setup.Stakingkeeper.GetDelegatorBonded(ctx, rep)
		s.NoError(err)

		_, err = reporterMsgServer.WithdrawTip(ctx, &reportertypes.MsgWithdrawTip{
			SelectorAddress:  rep.String(),
			ValidatorAddress: valAddrs[i].String(),
		})
		s.NoError(err)

		bondAfter, err := s.Setup.Stakingkeeper.GetDelegatorBonded(ctx, rep)
		s.NoError(err)

		withdrawn := bondAfter.Sub(bondBefore)
		s.T().Logf("Reporter %d withdrew: %s", i, withdrawn)
		require.True(withdrawn.GT(math.ZeroInt()), "Reporter %d should have withdrawn some tips", i)
	}
}

// TestTBRDistributionPartialParticipation tests that when only 2 of 3 reporters submit,
// those 2 get 50% each and the non-participating reporter gets nothing
func (s *IntegrationTestSuite) TestTBRDistributionPartialParticipation() {
	require := s.Require()

	ctx := s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{VoteExtensionsEnableHeight: 1},
	})

	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(1)

	// Create 3 validators with equal power
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 100, 100})

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
	tbrAmount := math.NewInt(100_000)
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, tbrAmount))))

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
	oracleMsgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	queriesSubmitted := make(map[string]bool)
	cycleCompleted := false

	// Only reporters 0 and 1 will submit (reporter 2 does not participate)
	activeReporters := repAccs[:2]

	for block := 1; block <= 20 && !cycleCompleted; block++ {
		ctx = ctx.WithBlockHeight(int64(block))
		ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))

		_, err := s.Setup.App.BeginBlocker(ctx)
		s.NoError(err)

		currentQueryData, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(ctx)
		s.NoError(err)
		queryId := utils.QueryIDFromData(currentQueryData)
		queryIdStr := string(queryId)

		if !queriesSubmitted[queryIdStr] {
			value := testutil.EncodeValue(100_00)
			s.T().Logf("Block %d: Only reporters 0 and 1 submitting for query %x", block, queryId[:4])

			// Only active reporters submit
			for _, rep := range activeReporters {
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

		_, err = s.Setup.App.EndBlocker(ctx)
		s.NoError(err)

		if len(queriesSubmitted) == 3 {
			cycleCompleted = true
		}
	}

	require.True(cycleCompleted, "Cycle should have completed")

	finalTips := getAvailableTips()
	tipDeltas := make([]math.LegacyDec, 3)
	for i := range repAccs {
		tipDeltas[i] = finalTips[i].Sub(initialTips[i])
	}

	s.T().Logf("Tip deltas: Reporter0=%s, Reporter1=%s, Reporter2=%s", tipDeltas[0], tipDeltas[1], tipDeltas[2])

	// Reporter 2 should have 0 tips (did not participate)
	require.True(tipDeltas[2].IsZero(), "Reporter 2 should have 0 tips (did not participate)")

	// Reporter 0 and 1 should have equal tips (50% each of participating reporters)
	require.True(tipDeltas[0].Sub(tipDeltas[1]).Abs().LTE(math.LegacyNewDec(1)),
		"Reporter 0 and 1 should have equal tips: %s vs %s", tipDeltas[0], tipDeltas[1])

	// Reporters 0 and 1 should have positive tips
	require.True(tipDeltas[0].GT(math.LegacyZeroDec()), "Reporter 0 should have positive tips")
	require.True(tipDeltas[1].GT(math.LegacyZeroDec()), "Reporter 1 should have positive tips")
}

// TestTBRDistributionDifferentPower tests that TBR is distributed proportionally
// to reporter power when all have 100% liveness
func (s *IntegrationTestSuite) TestTBRDistributionDifferentPower() {
	require := s.Require()

	ctx := s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{VoteExtensionsEnableHeight: 1},
	})

	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(1)

	// Create 3 validators with DIFFERENT power: 100, 200, 300 (total 600)
	// Expected distribution: 16.67%, 33.33%, 50%
	powers := []uint64{100, 200, 300}
	repAccs, valAddrs, _ := s.createValidatorAccs(powers)

	for _, val := range valAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("evmaddr"))
		s.NoError(err)
	}

	for i, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(
			reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter"+string(rune('0'+i)),
		)))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
		_, err := s.Setup.Reporterkeeper.ReporterStake(ctx, rep, []byte{})
		s.NoError(err)
	}

	// Fund TBR pool with amount divisible by 6 for clean math
	tipper := s.newKeysWithTokens()
	tbrAmount := math.NewInt(120_000)
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, tbrAmount))))

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
	oracleMsgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	queriesSubmitted := make(map[string]bool)
	cycleCompleted := false

	for block := 1; block <= 20 && !cycleCompleted; block++ {
		ctx = ctx.WithBlockHeight(int64(block))
		ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))

		_, err := s.Setup.App.BeginBlocker(ctx)
		s.NoError(err)

		currentQueryData, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(ctx)
		s.NoError(err)
		queryId := utils.QueryIDFromData(currentQueryData)
		queryIdStr := fmt.Sprintf("%x", queryId)

		if !queriesSubmitted[queryIdStr] {
			value := testutil.EncodeValue(100_00)

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

		_, err = s.Setup.App.EndBlocker(ctx)
		s.NoError(err)

		if len(queriesSubmitted) == 3 {
			cycleCompleted = true
		}
	}

	require.True(cycleCompleted, "Cycle should have completed")

	finalTips := getAvailableTips()
	tipDeltas := make([]math.LegacyDec, 3)
	totalDelta := math.LegacyZeroDec()
	for i := range repAccs {
		tipDeltas[i] = finalTips[i].Sub(initialTips[i])
		totalDelta = totalDelta.Add(tipDeltas[i])
	}

	s.T().Logf("Power: [%d, %d, %d]", powers[0], powers[1], powers[2])
	s.T().Logf("Tip deltas: [%s, %s, %s]", tipDeltas[0], tipDeltas[1], tipDeltas[2])

	// Calculate expected percentages based on power
	// Power 100 = 1/6 = 16.67%, Power 200 = 2/6 = 33.33%, Power 300 = 3/6 = 50%
	totalPower := uint64(600)
	for i, power := range powers {
		expectedPct := float64(power) / float64(totalPower) * 100
		actualPct := tipDeltas[i].Quo(totalDelta).MulInt64(100).MustFloat64()
		s.T().Logf("Reporter %d: power=%d, expected=%.1f%%, actual=%.1f%%", i, power, expectedPct, actualPct)

		// Allow 1% tolerance
		require.InDelta(expectedPct, actualPct, 1.0,
			"Reporter %d should have %.1f%% of total", i, expectedPct)
	}
	fmt.Println(tipDeltas)
	// Verify ratios: reporter1 should have 2x reporter0, reporter2 should have 3x reporter0
	ratio10 := tipDeltas[1].Quo(tipDeltas[0])
	ratio20 := tipDeltas[2].Quo(tipDeltas[0])
	s.T().Logf("Ratio reporter1/reporter0: %s (expected 2.0)", ratio10)
	s.T().Logf("Ratio reporter2/reporter0: %s (expected 3.0)", ratio20)

	require.InDelta(2.0, ratio10.MustFloat64(), 0.1, "Reporter 1 should have 2x Reporter 0")
	require.InDelta(3.0, ratio20.MustFloat64(), 0.1, "Reporter 2 should have 3x Reporter 0")
}

// TestTBRDistributionPartialLiveness tests that reporters who submit for only
// some queries get proportionally less rewards.
func (s *IntegrationTestSuite) TestTBRDistributionPartialLiveness() {
	require := s.Require()

	ctx := s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Block:     &cmtproto.BlockParams{MaxBytes: 200000, MaxGas: 100_000_000},
		Evidence:  &cmtproto.EvidenceParams{MaxAgeNumBlocks: 302400, MaxAgeDuration: 504 * time.Hour, MaxBytes: 10000},
		Validator: &cmtproto.ValidatorParams{PubKeyTypes: []string{cmttypes.ABCIPubKeyTypeEd25519}},
		Abci:      &cmtproto.ABCIParams{VoteExtensionsEnableHeight: 1},
	})

	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(1)

	// Create 3 validators with equal power
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 100, 100})

	for _, val := range valAddrs {
		s.NoError(s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("evmaddr")))
	}

	for i, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(
			reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter"+string(rune('0'+i)),
		)))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
		_, err := s.Setup.Reporterkeeper.ReporterStake(ctx, rep, []byte{})
		s.NoError(err)
	}

	tipper := s.newKeysWithTokens()
	tbrAmount := math.NewInt(90_000)
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, tbrAmount))))

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
	oracleMsgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	// - Reporter 0: submits for ALL queries (will have 2/3 liveness at distribution)
	// - Reporter 1: submits for queries 1 and 2 (same as Reporter 0 at distribution)
	// - Reporter 2: submits for query 1 only (1/3 liveness at distribution)
	queryCount := 0
	queriesSubmitted := make(map[string]bool)
	cycleCompleted := false

	for block := 1; block <= 20 && !cycleCompleted; block++ {
		ctx = ctx.WithBlockHeight(int64(block))
		ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))

		_, err := s.Setup.App.BeginBlocker(ctx)
		s.NoError(err)

		currentQueryData, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(ctx)
		s.NoError(err)
		queryId := utils.QueryIDFromData(currentQueryData)
		queryIdStr := fmt.Sprintf("%x", queryId)

		if !queriesSubmitted[queryIdStr] {
			queryCount++
			value := testutil.EncodeValue(100_00)
			s.T().Logf("Block %d: Query #%d (%x)", block, queryCount, queryId[:4])

			// Reporter 0 always submits
			msg := types.MsgSubmitValue{
				Creator:   repAccs[0].String(),
				QueryData: currentQueryData,
				Value:     value,
			}
			_, err := oracleMsgServer.SubmitValue(ctx, &msg)
			s.NoError(err)

			// Reporter 1 submits for queries 1 and 2
			if queryCount <= 2 {
				msg := types.MsgSubmitValue{
					Creator:   repAccs[1].String(),
					QueryData: currentQueryData,
					Value:     value,
				}
				_, err := oracleMsgServer.SubmitValue(ctx, &msg)
				s.NoError(err)
			}

			// Reporter 2 submits for query 1 only
			if queryCount == 1 {
				msg := types.MsgSubmitValue{
					Creator:   repAccs[2].String(),
					QueryData: currentQueryData,
					Value:     value,
				}
				_, err := oracleMsgServer.SubmitValue(ctx, &msg)
				s.NoError(err)
			}

			queriesSubmitted[queryIdStr] = true
		}

		_, err = s.Setup.App.EndBlocker(ctx)
		s.NoError(err)

		if len(queriesSubmitted) == 3 {
			cycleCompleted = true
		}
	}

	require.True(cycleCompleted, "Cycle should have completed")

	finalTips := getAvailableTips()
	tipDeltas := make([]math.LegacyDec, 3)
	totalDelta := math.LegacyZeroDec()
	for i := range repAccs {
		tipDeltas[i] = finalTips[i].Sub(initialTips[i])
		totalDelta = totalDelta.Add(tipDeltas[i])
	}

	s.T().Logf("Tip deltas: [%s, %s, %s]", tipDeltas[0], tipDeltas[1], tipDeltas[2])
	s.T().Logf("Total distributed: %s (expected: %d)", totalDelta, tbrAmount.Int64())

	// With per-aggregate power share:
	// - Q1: 3 reporters, each gets 1/3 share, reward = 30,000/3 = 10,000 each
	// - Q2: 2 reporters (0,1), each gets 1/2 share, reward = 30,000/2 = 15,000 each
	// - Q3: 1 reporter (0), gets 1 share, reward = 30,000
	//
	// Expected:
	// - Reporter 0: Q1 (10,000) + Q2 (15,000) = 25,000
	// - Reporter 1: Q1 (10,000) + Q2 (15,000) = 25,000
	// - Reporter 2: Q1 (10,000) = 10,000
	// Ratio is 2.5:2.5:1

	// Reporter 0 and 1 should be equal (both submitted for same queries before distribution)
	ratio01 := tipDeltas[0].Quo(tipDeltas[1]).MustFloat64()
	s.T().Logf("Ratio reporter0/reporter1: %.2f (expected ~1.0)", ratio01)
	require.InDelta(1.0, ratio01, 0.1, "Reporter 0 and 1 should have equal rewards")

	// Reporter 2 should have less than Reporter 0 and 1
	require.True(tipDeltas[0].GT(tipDeltas[2]), "Reporter 0 should have more than Reporter 2")
	require.True(tipDeltas[1].GT(tipDeltas[2]), "Reporter 1 should have more than Reporter 2")

	// Check approximate ratio (2.5:1 with per-aggregate power share)
	if tipDeltas[2].IsPositive() {
		ratio02 := tipDeltas[0].Quo(tipDeltas[2]).MustFloat64()
		s.T().Logf("Ratio reporter0/reporter2: %.2f (expected ~2.5)", ratio02)
		require.InDelta(2.5, ratio02, 1.0, "Reporter 0 should have ~2.5x Reporter 2")
	}
}

// TestPerAggregatePowerShareNaturalFlow tests the per-aggregate power share TBR distribution
//
// Scenario:
// - 3 cyclelist queries: Q1, Q2, Q3
// - 3 reporters: alice (power 100), bob (power 200), charlie (power 300)
// - TBR = 1000
//
// Participation:
// - Q1: alice(100), bob(200), charlie(300) → total 600
// - Q2: alice(100), bob(200), charlie(300) → total 600
// - Q3: alice(100), bob(200) → total 300 (charlie skips)
//
// Expected rewards with StandardOpportunities=1, rewardPerSlot=333.33:
// - alice: (100/600)*333.33 + (100/600)*333.33 + (100/300)*333.33 = 55.56 + 55.56 + 111.11 = 222.22
// - bob:   (200/600)*333.33 + (200/600)*333.33 + (200/300)*333.33 = 111.11 + 111.11 + 222.22 = 444.44
// - charlie: (300/600)*333.33 + (300/600)*333.33 + 0 = 166.67 + 166.67 + 0 = 333.33
// Total: 1000 (full distribution)
func (s *IntegrationTestSuite) TestPerAggregatePowerShareNaturalFlow() {
	require := s.Require()

	// Set up consensus params
	ctx := s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{VoteExtensionsEnableHeight: 1},
	})

	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(1)

	// Confirm params are set correctly
	params, err := s.Setup.Oraclekeeper.GetParams(ctx)
	require.NoError(err)
	require.Equal(uint64(1), params.LivenessCycles)

	// Create 3 validators with different powers: alice=100, bob=200, charlie=300
	powers := []uint64{100, 200, 300}
	repAccs, valAddrs, _ := s.createValidatorAccs(powers)
	alice, bob, _ := repAccs[0], repAccs[1], repAccs[2]

	// Set up bridge EVM addresses
	for _, val := range valAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("evmaddr"))
		s.NoError(err)
	}

	// Create reporters
	reporterNames := []string{"alice", "bob", "charlie"}
	for i, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(
			reportertypes.DefaultMinCommissionRate, math.OneInt(), reporterNames[i],
		)))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
	}

	// Initialize reporter stakes
	for _, rep := range repAccs {
		_, err := s.Setup.Reporterkeeper.ReporterStake(ctx, rep, []byte{})
		s.NoError(err)
	}

	// Advance block height so any old queries are clearly expired
	ctx = ctx.WithBlockHeight(5)

	// Reset cycle to start fresh: set sequencer to 2, run EndBlocker which wraps to 0
	s.NoError(s.Setup.Oraclekeeper.CyclelistSequencer.Set(ctx, 2))
	// Run EndBlocker to properly rotate and set up Q1 with fresh expiration
	_, err = s.Setup.App.EndBlocker(ctx)
	s.NoError(err)

	// Fund TBR pool with exactly 1000
	tipper := s.newKeysWithTokens()
	tbrAmount := math.NewInt(1000)
	err = s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, tbrAmount)))
	s.NoError(err)

	// Get available tips helper
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
	s.T().Logf("Initial tips: alice=%s, bob=%s, charlie=%s", initialTips[0], initialTips[1], initialTips[2])

	oracleMsgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	// Get cyclelist for comparison
	cyclelist, err := s.Setup.Oraclekeeper.GetCyclelist(ctx)
	s.NoError(err)
	require.Equal(len(cyclelist), 3, "Need at least 3 queries in cyclelist")

	// Track queries
	queryCount := 0
	queriesSubmitted := make(map[string]bool)
	cycleCompleted := false

	// Run through the cycle with specific participation pattern
	// We need to continue until all queries are aggregated (not just submitted)
	for block := 2; block <= 11; block++ {
		ctx = ctx.WithBlockHeight(int64(block))
		ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))

		_, err := s.Setup.App.BeginBlocker(ctx)
		s.NoError(err)

		currentQueryData, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(ctx)
		s.NoError(err)
		queryId := utils.QueryIDFromData(currentQueryData)
		queryIdStr := hex.EncodeToString(queryId)

		// Check if this is a new query in the cycle that we haven't submitted for yet
		if !queriesSubmitted[queryIdStr] && queryCount < 3 {
			queryCount++
			value := testutil.EncodeValue(100_00)

			s.T().Logf("Block %d: Query #%d (%x)", block, queryCount, queryId[:4])

			// Submit based on query number
			switch queryCount {
			case 1: // Q1: all three report
				for _, rep := range repAccs {
					msg := types.MsgSubmitValue{Creator: rep.String(), QueryData: currentQueryData, Value: value}
					_, err := oracleMsgServer.SubmitValue(ctx, &msg)
					s.NoError(err)
				}
				s.T().Logf("  Q1: alice, bob, charlie all submitted")

			case 2: // Q2: all three report
				for _, rep := range repAccs {
					msg := types.MsgSubmitValue{Creator: rep.String(), QueryData: currentQueryData, Value: value}
					_, err := oracleMsgServer.SubmitValue(ctx, &msg)
					s.NoError(err)
				}
				s.T().Logf("  Q2: alice, bob, charlie all submitted")

			case 3: // Q3: only alice and bob report (charlie skips)
				msg := types.MsgSubmitValue{Creator: alice.String(), QueryData: currentQueryData, Value: value}
				_, err := oracleMsgServer.SubmitValue(ctx, &msg)
				s.NoError(err)

				msg = types.MsgSubmitValue{Creator: bob.String(), QueryData: currentQueryData, Value: value}
				_, err = oracleMsgServer.SubmitValue(ctx, &msg)
				s.NoError(err)
				s.T().Logf("  Q3: alice, bob submitted (charlie skipped)")
			}
			queriesSubmitted[queryIdStr] = true
		}
		fmt.Println("Queries Submitted So Far:", len(queriesSubmitted))
		if len(queriesSubmitted) == 3 {
			tbrBefore := s.Setup.Oraclekeeper.GetTimeBasedRewards(ctx)
			s.T().Logf("TBR before distribution: %s", tbrBefore)
		}
		_, err = s.Setup.App.EndBlocker(ctx)
		s.NoError(err)

		// Check if we've completed one full cycle (all 3 queries submitted and Q3 aggregated)
		if len(queriesSubmitted) == 3 {
			// Once Q3 is submitted, we need 2 more blocks for aggregation
			currentQuery, _ := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(ctx)
			currentQueryId := utils.QueryIDFromData(currentQuery)

			// If current query is no longer Q3, Q3 was aggregated
			q3Id := utils.QueryIDFromData(cyclelist[2]) // Third query in cyclelist
			if string(currentQueryId) != string(q3Id) && block > 10 {
				cycleCompleted = true
				break
			}
		}
	}

	require.True(cycleCompleted, "Cycle should have completed")

	// Get final tips
	finalTips := getAvailableTips()

	// Calculate deltas
	tipDeltas := make([]math.LegacyDec, 3)
	totalDelta := math.LegacyZeroDec()
	for i := range repAccs {
		tipDeltas[i] = finalTips[i].Sub(initialTips[i])
		totalDelta = totalDelta.Add(tipDeltas[i])
	}

	s.T().Logf("Final tips: alice=%s, bob=%s, charlie=%s", finalTips[0], finalTips[1], finalTips[2])
	s.T().Logf("Tip deltas: alice=%s, bob=%s, charlie=%s", tipDeltas[0], tipDeltas[1], tipDeltas[2])
	s.T().Logf("Total delta: %s (expected: 1000)", totalDelta)

	// Verify basic properties:
	// 1. TBR was distributed
	tbrBalance := s.Setup.Oraclekeeper.GetTimeBasedRewards(ctx)
	s.T().Logf("Remaining TBR: %s", tbrBalance)
	require.True(tbrBalance.IsZero(), "TBR should be fully distributed")

	// 2. Bob should have more than alice (bob has more power)
	require.True(tipDeltas[1].GT(tipDeltas[0]),
		"Bob should have more than Alice (bob has 2x power): bob=%s, alice=%s",
		tipDeltas[1], tipDeltas[0])

	// 3. Bob should have more than charlie (charlie skipped Q3)
	require.True(tipDeltas[1].GT(tipDeltas[2]),
		"Bob should have more than Charlie (charlie skipped Q3): bob=%s, charlie=%s",
		tipDeltas[1], tipDeltas[2])

	// 4. Total distributed should be ~1000 (full distribution)
	// All queries are treated as standard with StandardOpportunities = 1
	require.True(totalDelta.GT(math.LegacyNewDec(990)),
		"Total distribution should be >999, got %s", totalDelta)

	// 5. Verify approximate expected ratios
	// With all queries standard (StandardOpp = 1):
	// Q1: alice(100/600), bob(200/600), charlie(300/600)
	// Q2: alice(100/600), bob(200/600), charlie(300/600)
	// Q3: alice(100/300), bob(200/300), charlie skipped
	// rewardPerSlot = 1000/3 = 333.33
	// Alice: (100/600 + 100/600 + 100/300) * 333.33 = 222.22
	// Bob: (200/600 + 200/600 + 200/300) * 333.33 = 444.44
	// Charlie: (300/600 + 300/600 + 0) * 333.33 = 333.33
	expectedAlice := math.LegacyMustNewDecFromStr("222")
	expectedBob := math.LegacyMustNewDecFromStr("444")
	expectedCharlie := math.LegacyMustNewDecFromStr("333")
	tolerance := math.LegacyNewDec(5)

	require.True(tipDeltas[0].Sub(expectedAlice).Abs().LTE(tolerance),
		"Alice reward should be ~222, got %s", tipDeltas[0])
	require.True(tipDeltas[1].Sub(expectedBob).Abs().LTE(tolerance),
		"Bob reward should be ~444, got %s", tipDeltas[1])
	require.True(tipDeltas[2].Sub(expectedCharlie).Abs().LTE(tolerance),
		"Charlie reward should be ~333, got %s", tipDeltas[2])
}

// TestPerAggregatePowerShareExactNumbers tests the exact calculation from the spreadsheet example
func (s *IntegrationTestSuite) TestPerAggregatePowerShareExactNumbers() {
	require := s.Require()

	ctx := s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{VoteExtensionsEnableHeight: 1},
	})

	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(1)

	// Create 3 validators with powers matching spreadsheet
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 200, 300})
	alice, bob, charlie := repAccs[0], repAccs[1], repAccs[2]

	for _, val := range valAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("evmaddr"))
		s.NoError(err)
	}

	for i, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(
			reportertypes.DefaultMinCommissionRate, math.OneInt(), []string{"alice", "bob", "charlie"}[i],
		)))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
	}

	for _, rep := range repAccs {
		_, err := s.Setup.Reporterkeeper.ReporterStake(ctx, rep, []byte{})
		s.NoError(err)
	}

	// Fund TBR pool with exactly 1000
	tipper := s.newKeysWithTokens()
	tbrAmount := math.NewInt(1000)
	err := s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, tbrAmount)))
	s.NoError(err)

	// Get cyclelist queries
	cyclelist, err := s.Setup.Oraclekeeper.GetCyclelist(ctx)
	s.NoError(err)
	require.GreaterOrEqual(len(cyclelist), 3, "Need at least 3 queries in cyclelist")

	q1 := utils.QueryIDFromData(cyclelist[0])
	q2 := utils.QueryIDFromData(cyclelist[1])
	q3 := utils.QueryIDFromData(cyclelist[2])

	// Reset any liveness data that might have been set during test setup
	// This ensures we have a clean slate for our manual scenario
	s.NoError(s.Setup.Oraclekeeper.ResetLivenessData(ctx))

	// Manually set up the exact scenario from the spreadsheet:
	// Q1: 2 opportunities (non-standard, out-of-turn tipped), Q2: 1 opportunity, Q3: 1 opportunity
	// Standard opportunities = 1 (q2 and q3 are standard)
	s.NoError(s.Setup.Oraclekeeper.StandardOpportunities.Set(ctx, 1))
	s.NoError(s.Setup.Oraclekeeper.QueryOpportunities.Set(ctx, q1, 2)) // Non-standard query opportunities
	s.NoError(s.Setup.Oraclekeeper.QueryOpportunities.Set(ctx, q2, 1)) // Not used for standard queries
	s.NoError(s.Setup.Oraclekeeper.QueryOpportunities.Set(ctx, q3, 1)) // Not used for standard queries

	// Set up reporter shares using UpdateReporterLiveness
	// Matching the spreadsheet exactly:
	//
	// Aggregate 1 (aaa): alice(100/600), bob(200/600), charlie(300/600)
	// Note: q1 has 2 opportunities (was tipped out-of-turn), so treat as non-standard
	s.NoError(s.Setup.Oraclekeeper.NonStandardQueries.Set(ctx, q1, true)) // Mark as non-standard since it has 2 opportunities
	s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, alice, q1, 100, 600, true))
	s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, bob, q1, 200, 600, true))
	s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, charlie, q1, 300, 600, true))

	// Aggregate 2 (bbb): alice(100/600), bob(200/600), charlie(300/600)
	s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, alice, q2, 100, 600, false))
	s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, bob, q2, 200, 600, false))
	s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, charlie, q2, 300, 600, false))

	// Aggregate 3 (ccc): alice(150/350), bob(200/350) - charlie skips
	// NOTE: Alice's power changed to 150 (stake increase)
	s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, alice, q3, 150, 350, false))
	s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, bob, q3, 200, 350, false))

	// Aggregate 4 (aaa, out-of-turn tipped): bob(200/500), charlie(300/500) - alice skips
	s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, bob, q1, 200, 500, true))
	s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, charlie, q1, 300, 500, true))

	// Get initial tips
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

	// Distribute rewards
	err = s.Setup.Oraclekeeper.DistributeLivenessRewards(ctx)
	s.NoError(err)

	// Get final tips
	finalTips := getAvailableTips()

	// Calculate deltas
	aliceDelta := finalTips[0].Sub(initialTips[0])
	bobDelta := finalTips[1].Sub(initialTips[1])
	charlieDelta := finalTips[2].Sub(initialTips[2])
	totalDelta := aliceDelta.Add(bobDelta).Add(charlieDelta)

	s.T().Logf("Tip deltas: alice=%s, bob=%s, charlie=%s", aliceDelta, bobDelta, charlieDelta)
	s.T().Logf("Total delta: %s (expected: 1000)", totalDelta)

	// Expected values from spreadsheet (alice has stake change: 100->150 for ccc):
	// rewardPerQuery = 1000/3 = 333.333...
	//
	// Alice (skipped aaa agg4):
	//   aaa: (100/600) / 2 * 333.33 = 0.1667 / 2 * 333.33 = 27.78
	//   bbb: (100/600) / 1 * 333.33 = 0.1667 * 333.33 = 55.56
	//   ccc: (150/350) / 1 * 333.33 = 0.4286 * 333.33 = 142.86
	//   Total: 226.19
	//
	// Bob (participated in all):
	//   aaa: (200/600 + 200/500) / 2 * 333.33 = (0.3333 + 0.4) / 2 * 333.33 = 0.3667 * 333.33 = 122.22
	//   bbb: (200/600) / 1 * 333.33 = 0.3333 * 333.33 = 111.11
	//   ccc: (200/350) / 1 * 333.33 = 0.5714 * 333.33 = 190.48
	//   Total: 423.81
	//
	// Charlie (skipped ccc):
	//   aaa: (300/600 + 300/500) / 2 * 333.33 = (0.5 + 0.6) / 2 * 333.33 = 0.55 * 333.33 = 183.33
	//   bbb: (300/600) / 1 * 333.33 = 0.5 * 333.33 = 166.67
	//   ccc: 0
	//   Total: 350.00

	// Allow small tolerance for rounding
	tolerance := math.LegacyNewDec(2)

	expectedAlice := math.LegacyMustNewDecFromStr("226.190476190476190476")
	expectedBob := math.LegacyMustNewDecFromStr("423.809523809523809523")
	expectedCharlie := math.LegacyMustNewDecFromStr("350")

	require.True(aliceDelta.Sub(expectedAlice).Abs().LTE(tolerance),
		"Alice reward should be ~226.19, got %s", aliceDelta)
	require.True(bobDelta.Sub(expectedBob).Abs().LTE(tolerance),
		"Bob reward should be ~423.81, got %s", bobDelta)
	require.True(charlieDelta.Sub(expectedCharlie).Abs().LTE(tolerance),
		"Charlie reward should be ~350.00, got %s", charlieDelta)

	// Verify total is close to 1000
	require.True(totalDelta.Sub(math.LegacyNewDec(1000)).Abs().LTE(math.LegacyNewDec(1)),
		"Total distribution should be ~1000, got %s", totalDelta)
}
