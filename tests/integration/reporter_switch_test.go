package integration_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/dispute"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// reporterSwitchFixture wires three validator-reporters (A,B,C) and one external selector.
func (s *IntegrationTestSuite) reporterSwitchFixture() (
	reporterA, reporterB, reporterC, selector sdk.AccAddress,
	bridgeQueryData, queryID []byte,
) {
	msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)

	valAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 200, 300})
	reporterA, reporterB, reporterC = valAccs[0], valAccs[1], valAccs[2]

	selector = sample.AccAddressBytes()
	s.Setup.MintTokens(selector, math.NewInt(1000*1e6))
	msgDelegate := stakingtypes.NewMsgDelegate(
		selector.String(),
		valAddrs[0].String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	_, err := stakingMsgServer.Delegate(s.Setup.Ctx, msgDelegate)
	s.NoError(err)

	for i, rep := range []sdk.AccAddress{reporterA, reporterB, reporterC} {
		_, err = msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
			ReporterAddress:   rep.String(),
			CommissionRate:    reportertypes.DefaultMinCommissionRate,
			MinTokensRequired: math.NewIntWithDecimal(1, 6),
			Moniker:           fmt.Sprintf("switch_rep_%d", i),
		})
		s.NoError(err)
	}

	// MsgSelectReporter sets the Selection and FlagStakeRecalc(reporterA); prefer it over
	// assignSelectorToReporter unless you need to bypass msg validation.
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(2)
	_, err = msgServer.SelectReporter(s.Setup.Ctx, &reportertypes.MsgSelectReporter{
		SelectorAddress: selector.String(),
		ReporterAddress: reporterA.String(),
	})
	s.NoError(err)

	spec := registrytypes.DataSpec{
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "tolayer", FieldType: "bool"},
			{Name: "depositId", FieldType: "uint256"},
		},
	}
	bridgeQueryData, err = spec.EncodeData("TRBBridgeV2", `["true","9001"]`)
	s.NoError(err)
	queryID = utils.QueryIDFromData(bridgeQueryData)
	return reporterA, reporterB, reporterC, selector, bridgeQueryData, queryID
}

func (s *IntegrationTestSuite) submitBridgeReport(
	reporter sdk.AccAddress,
	bridgeQueryData []byte,
	height int64,
) (oracletypes.MicroReport, uint64) {
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(height).WithBlockTime(time.Now())
	_, err := oracleMsgServer.SubmitValue(s.Setup.Ctx, &oracletypes.MsgSubmitValue{
		Creator:   reporter.String(),
		QueryData: bridgeQueryData,
		Value:     bridgeTestValue,
	})
	s.NoError(err)

	queryID := utils.QueryIDFromData(bridgeQueryData)
	qMeta, err := s.Setup.Oraclekeeper.CurrentQuery(s.Setup.Ctx, queryID)
	s.NoError(err)
	report, err := s.Setup.Oraclekeeper.Reports.Get(s.Setup.Ctx, collections.Join3(queryID, reporter.Bytes(), qMeta.Id))
	s.NoError(err)
	return report, uint64(height)
}

// proposeFullMinorDispute pays the full minor fee in one shot (jail + voting) and returns the new dispute id.
func (s *IntegrationTestSuite) proposeFullMinorDispute(disputer sdk.AccAddress, report oracletypes.MicroReport) uint64 {
	s.Setup.MintTokens(disputer, math.NewInt(20_000_000_000))
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, disputetypes.Minor)
	s.NoError(err)

	disputeID := s.Setup.Disputekeeper.NextDisputeId(s.Setup.Ctx)
	disputeMsgServer := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, err = disputeMsgServer.ProposeDispute(s.Setup.Ctx, &disputetypes.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  disputetypes.Minor,
	})
	s.NoError(err)
	return disputeID
}

// snapshotAtReportBlockHasSelector matches dispute jail: nearest delegation snapshot at or before reportBlock.
func (s *IntegrationTestSuite) snapshotAtReportBlockHasSelector(
	reporter sdk.AccAddress, reportBlock uint64, selector sdk.AccAddress,
) bool {
	snap, err := s.Setup.Reporterkeeper.GetDelegationsAmount(s.Setup.Ctx, reporter.Bytes(), reportBlock)
	if err != nil {
		return false
	}
	for _, o := range snap.TokenOrigins {
		if bytes.Equal(o.DelegatorAddress, selector.Bytes()) {
			return true
		}
	}
	return false
}

// TestReporterSwitchStakeExclusionAndDisputeSnapshot verifies stake handoff rules and that
// a minor dispute on reporter A's pre-switch report jails the selector from the block snapshot.
func (s *IntegrationTestSuite) TestReporterSwitchStakeExclusionAndDisputeSnapshot() {
	reporterA, reporterB, reporterC, selector, bridgeQueryData, queryID := s.reporterSwitchFixture()
	msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	valA, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, sdk.ValAddress(reporterA))
	s.NoError(err)

	bridgeHeight := int64(10)
	report, reportBlock := s.submitBridgeReport(reporterA, bridgeQueryData, bridgeHeight)

	stakeAWithSelector, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterA, queryID)
	s.NoError(err)
	stakeBBase, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.True(stakeAWithSelector.GT(stakeBBase))

	maxCommit, err := s.Setup.Oraclekeeper.GetMaxOpenCommitmentForReporter(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)
	s.Equal(uint64(bridgeHeight)+2000, maxCommit)

	qMeta, err := s.Setup.Oraclekeeper.CurrentQuery(s.Setup.Ctx, queryID)
	s.NoError(err)
	expectedAPower := stakeAWithSelector.Quo(layertypes.PowerReduction).Uint64()
	s.Equal(expectedAPower, report.Power, "reporter A's report must include selector stake")

	s.True(s.snapshotAtReportBlockHasSelector(reporterA, reportBlock, selector),
		"delegation snapshot at report block must include selector before switch")

	// Dispute before switch so jail uses the report-block snapshot that still lists the selector.
	disputer := s.newKeysWithTokens()
	s.proposeFullMinorDispute(disputer, report)

	repA, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)
	s.True(repA.Jailed)

	selAfterDispute, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	s.True(reportertypes.SelectorStakeLocked(selAfterDispute, s.Setup.Ctx.BlockTime()),
		"selector from report snapshot must be dispute-locked")

	// Deferred switch A → B while the bridge window is still open.
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(bridgeHeight + 1).WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
		SelectorAddress: selector.String(),
		ReporterAddress: reporterB.String(),
	})
	s.NoError(err)

	sel, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	s.True(bytes.Equal(sel.Reporter, reporterA.Bytes()), "selection must stay on A until finalize")

	outPK := collections.Join(reporterA.Bytes(), selector.Bytes())
	hasPending, err := s.Setup.Reporterkeeper.OutgoingPendingSwitches.Has(s.Setup.Ctx, outPK)
	s.NoError(err)
	s.True(hasPending, "pending switch must be recorded on outgoing reporter A")

	// Reporter A is jailed after the dispute; verify exclusion via B and the pending row.
	stakeBPending, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.Equal(stakeBBase, stakeBPending, "B must not gain selector stake before finalize")

	_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &oracletypes.MsgSubmitValue{
		Creator:   reporterB.String(),
		QueryData: bridgeQueryData,
		Value:     bridgeTestValue,
	})
	s.NoError(err)
	repBReport, err := s.Setup.Oraclekeeper.Reports.Get(s.Setup.Ctx, collections.Join3(queryID, reporterB.Bytes(), qMeta.Id))
	s.NoError(err)
	s.Equal(stakeBPending.Quo(layertypes.PowerReduction).Uint64(), repBReport.Power)
	s.True(repBReport.Power < report.Power, "B reporting same query must not include switched selector")

	// Finalize switch after open commitment expires. Stay within minor jail (600s) so the selector
	// remains dispute-locked and must not count toward B.
	finalizeHeight := int64(maxCommit) + 1
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(finalizeHeight).WithBlockTime(s.Setup.Ctx.BlockTime().Add(30 * time.Second))
	_, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)

	selFinal, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	s.True(bytes.Equal(selFinal.Reporter, reporterB.Bytes()), "switch must finalize onto B after unlock height")
	s.True(reportertypes.SelectorStakeLocked(selFinal, s.Setup.Ctx.BlockTime()),
		"selector must still be dispute-locked within minor jail window")

	stakeBAfterFinalize, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.Equal(stakeBBase, stakeBAfterFinalize, "dispute-locked selector must not count toward B after finalize")

	// Reporter A is jailed; do not call ReporterStake on A (it errors). Stake was already excluded while pending.
	_ = reporterC
	_ = valA
}

// TestReporterSwitchDisputeAfterSwitch verifies that a minor dispute on reporter A's report still
// jails and locks selectors from the report-block snapshot when the switch happened first.
func (s *IntegrationTestSuite) TestReporterSwitchDisputeAfterSwitch() {
	s.Run("pending_switch_then_dispute", func() {
		s.SetupTest()
		reporterA, reporterB, _, selector, bridgeQueryData, queryID := s.reporterSwitchFixture()
		msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
		oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

		bridgeHeight := int64(10)
		report, reportBlock := s.submitBridgeReport(reporterA, bridgeQueryData, bridgeHeight)
		stakeBBase, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
		s.NoError(err)
		s.True(s.snapshotAtReportBlockHasSelector(reporterA, reportBlock, selector))

		// Switch before dispute; selection stays on A until finalize.
		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(bridgeHeight + 1).WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
		_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
			SelectorAddress: selector.String(),
			ReporterAddress: reporterB.String(),
		})
		s.NoError(err)

		sel, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
		s.NoError(err)
		s.True(bytes.Equal(sel.Reporter, reporterA.Bytes()), "selection must stay on A until finalize")
		s.False(reportertypes.SelectorStakeLocked(sel, s.Setup.Ctx.BlockTime()),
			"selector must not be dispute-locked before dispute")

		disputer := s.newKeysWithTokens()
		s.proposeFullMinorDispute(disputer, report)

		repA, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterA.Bytes())
		s.NoError(err)
		s.True(repA.Jailed)

		selAfter, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
		s.NoError(err)
		s.True(reportertypes.SelectorStakeLocked(selAfter, s.Setup.Ctx.BlockTime()),
			"selector must be dispute-locked from report snapshot despite pending switch to B")
		s.True(bytes.Equal(selAfter.Reporter, reporterA.Bytes()))

		stakeBPending, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
		s.NoError(err)
		s.Equal(stakeBBase, stakeBPending, "B must not gain selector stake while locked and pending")

		qMeta, err := s.Setup.Oraclekeeper.CurrentQuery(s.Setup.Ctx, queryID)
		s.NoError(err)
		_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &oracletypes.MsgSubmitValue{
			Creator:   reporterB.String(),
			QueryData: bridgeQueryData,
			Value:     bridgeTestValue,
		})
		s.NoError(err)
		repBReport, err := s.Setup.Oraclekeeper.Reports.Get(s.Setup.Ctx, collections.Join3(queryID, reporterB.Bytes(), qMeta.Id))
		s.NoError(err)
		s.Equal(stakeBPending.Quo(layertypes.PowerReduction).Uint64(), repBReport.Power)
		s.True(repBReport.Power < report.Power)
	})

	s.Run("finalized_switch_then_dispute", func() {
		s.SetupTest()
		reporterA, reporterB, _, selector, bridgeQueryData, queryID := s.reporterSwitchFixture()
		msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)

		bridgeHeight := int64(10)
		report, reportBlock := s.submitBridgeReport(reporterA, bridgeQueryData, bridgeHeight)
		stakeBBase, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
		s.NoError(err)
		s.True(s.snapshotAtReportBlockHasSelector(reporterA, reportBlock, selector))

		maxCommit, err := s.Setup.Oraclekeeper.GetMaxOpenCommitmentForReporter(s.Setup.Ctx, reporterA.Bytes())
		s.NoError(err)

		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(bridgeHeight + 1).WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
		_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
			SelectorAddress: selector.String(),
			ReporterAddress: reporterB.String(),
		})
		s.NoError(err)

		finalizeHeight := int64(maxCommit) + 1
		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(finalizeHeight).WithBlockTime(s.Setup.Ctx.BlockTime().Add(2 * time.Hour))
		stakeBWithSelector, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
		s.NoError(err)
		s.True(stakeBWithSelector.GT(stakeBBase), "B must include selector stake after finalize before dispute")

		selFinal, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
		s.NoError(err)
		s.True(bytes.Equal(selFinal.Reporter, reporterB.Bytes()), "switch must finalize onto B before dispute")

		disputer := s.newKeysWithTokens()
		s.proposeFullMinorDispute(disputer, report)

		repA, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterA.Bytes())
		s.NoError(err)
		s.True(repA.Jailed)

		selAfter, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
		s.NoError(err)
		s.True(bytes.Equal(selAfter.Reporter, reporterB.Bytes()),
			"selection must remain on B after dispute; lock is on selector row not live index")
		s.True(reportertypes.SelectorStakeLocked(selAfter, s.Setup.Ctx.BlockTime()),
			"selector must be dispute-locked from A's report snapshot even though no longer on A")

		stakeBAfterDispute, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
		s.NoError(err)
		s.Equal(stakeBBase, stakeBAfterDispute,
			"dispute-locked selector must not count toward B after switch finalized")
	})
}

// TestReporterSwitchPendingEdgeCases covers replace/idempotent pending switches and lock gating.
func (s *IntegrationTestSuite) TestReporterSwitchPendingEdgeCases() {
	s.Run("idempotent_switch_to_same_pending_target", func() {
		s.SetupTest()
		reporterA, reporterB, _, selector, _, _ := s.reporterSwitchFixture()
		msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)

		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(5)
		_, err := msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
			SelectorAddress: selector.String(),
			ReporterAddress: reporterB.String(),
		})
		s.NoError(err)

		_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
			SelectorAddress: selector.String(),
			ReporterAddress: reporterB.String(),
		})
		s.NoError(err, "second switch to same pending target must be a no-op success")

		outPK := collections.Join(reporterA.Bytes(), selector.Bytes())
		ent, err := s.Setup.Reporterkeeper.OutgoingPendingSwitches.Get(s.Setup.Ctx, outPK)
		s.NoError(err)
		s.True(bytes.Equal(ent.ToReporter, reporterB.Bytes()))
	})

	s.Run("replace_pending_target_while_lock_active", func() {
		s.SetupTest()
		reporterA, reporterB, reporterC, selector, bridgeQueryData, _ := s.reporterSwitchFixture()
		msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)

		_, _ = s.submitBridgeReport(reporterA, bridgeQueryData, 10)

		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(11)
		_, err := msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
			SelectorAddress: selector.String(),
			ReporterAddress: reporterB.String(),
		})
		s.NoError(err)

		sel, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
		s.NoError(err)
		s.Greater(sel.SwitchOutLockedUntilBlock, uint64(s.Setup.Ctx.BlockHeight()))

		// Replace B with C while the outgoing lock is still active (hasPending bypasses lock rejection).
		_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
			SelectorAddress: selector.String(),
			ReporterAddress: reporterC.String(),
		})
		s.NoError(err)

		outPK := collections.Join(reporterA.Bytes(), selector.Bytes())
		ent, err := s.Setup.Reporterkeeper.OutgoingPendingSwitches.Get(s.Setup.Ctx, outPK)
		s.NoError(err)
		s.True(bytes.Equal(ent.ToReporter, reporterC.Bytes()))

		hasB, err := s.Setup.Reporterkeeper.IncomingPendingSwitchIdx.Has(s.Setup.Ctx, collections.Join(reporterB.Bytes(), selector.Bytes()))
		s.NoError(err)
		s.False(hasB)
	})

	s.Run("rejects_new_switch_when_outgoing_lock_active_without_pending_row", func() {
		s.SetupTest()
		reporterA, reporterB, reporterC, selector, _, _ := s.reporterSwitchFixture()
		msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)

		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(20)
		sel, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
		s.NoError(err)
		sel.SwitchOutLockedUntilBlock = 100
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, selector.Bytes(), sel))

		_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
			SelectorAddress: selector.String(),
			ReporterAddress: reporterB.String(),
		})
		s.ErrorContains(err, "selector is locked until the current reporter switch completes")

		// Different target also blocked.
		_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
			SelectorAddress: selector.String(),
			ReporterAddress: reporterC.String(),
		})
		s.ErrorContains(err, "selector is locked until the current reporter switch completes")
		_ = reporterA
	})
}

// TestReporterSwitchSkipIntermediateReporter verifies A→B→C replace finalizes on C with selector stake
// once unlock_block < current height, without B ever reporting.
func (s *IntegrationTestSuite) TestReporterSwitchSkipIntermediateReporter() {
	reporterA, reporterB, reporterC, selector, bridgeQueryData, queryID := s.reporterSwitchFixture()
	msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)

	valC, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, sdk.ValAddress(reporterC))
	s.NoError(err)
	stakeCBefore, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterC, queryID)
	s.NoError(err)
	s.Equal(valC.Tokens, stakeCBefore)

	bridgeHeight := int64(10)
	_, _ = s.submitBridgeReport(reporterA, bridgeQueryData, bridgeHeight)
	maxCommit, err := s.Setup.Oraclekeeper.GetMaxOpenCommitmentForReporter(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)

	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(bridgeHeight + 1)
	_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
		SelectorAddress: selector.String(),
		ReporterAddress: reporterB.String(),
	})
	s.NoError(err)

	_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
		SelectorAddress: selector.String(),
		ReporterAddress: reporterC.String(),
	})
	s.NoError(err)

	outPK := collections.Join(reporterA.Bytes(), selector.Bytes())
	ent, err := s.Setup.Reporterkeeper.OutgoingPendingSwitches.Get(s.Setup.Ctx, outPK)
	s.NoError(err)
	s.True(bytes.Equal(ent.ToReporter, reporterC.Bytes()))

	// Past open commitment: finalize via ReporterStake on incoming reporter C only.
	finalizeHeight := int64(maxCommit) + 1
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(finalizeHeight).WithBlockTime(s.Setup.Ctx.BlockTime().Add(2 * time.Hour))
	stakeCAfter, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterC, queryID)
	s.NoError(err)
	s.True(stakeCAfter.GT(stakeCBefore), "C must include selector stake immediately after unlock when B never reported")

	sel, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	s.True(bytes.Equal(sel.Reporter, reporterC.Bytes()))

	hasPending, err := s.Setup.Reporterkeeper.OutgoingPendingSwitches.Has(s.Setup.Ctx, outPK)
	s.NoError(err)
	s.False(hasPending)
}

// disputeOutcomeCase drives tally/execute for a jailed minor dispute after a reporter switch.
type disputeOutcomeCase struct {
	name           string
	votes          []disputetypes.MsgVote
	expectResult   disputetypes.VoteResult
	reporterJailed bool
	selectorLocked bool
}

// TestReporterSwitchMinorDisputeOutcomes exercises vote results on a report that was made before
// the selector switched away; dispute is opened before the switch so the snapshot includes the selector.
func (s *IntegrationTestSuite) TestReporterSwitchMinorDisputeOutcomes() {
	cases := []disputeOutcomeCase{
		{
			name: "support",
			votes: []disputetypes.MsgVote{
				{Vote: disputetypes.VoteEnum_VOTE_SUPPORT},
				{Vote: disputetypes.VoteEnum_VOTE_SUPPORT},
				{Vote: disputetypes.VoteEnum_VOTE_SUPPORT},
			},
			expectResult:   disputetypes.VoteResult_SUPPORT,
			reporterJailed: true,
			selectorLocked: true,
		},
		{
			name: "against",
			votes: []disputetypes.MsgVote{
				{Vote: disputetypes.VoteEnum_VOTE_AGAINST},
				{Vote: disputetypes.VoteEnum_VOTE_AGAINST},
				{Vote: disputetypes.VoteEnum_VOTE_AGAINST},
			},
			expectResult:   disputetypes.VoteResult_AGAINST,
			reporterJailed: false,
			selectorLocked: false,
		},
		{
			name: "invalid",
			votes: []disputetypes.MsgVote{
				{Vote: disputetypes.VoteEnum_VOTE_INVALID},
				{Vote: disputetypes.VoteEnum_VOTE_INVALID},
				{Vote: disputetypes.VoteEnum_VOTE_INVALID},
			},
			expectResult:   disputetypes.VoteResult_INVALID,
			reporterJailed: false,
			selectorLocked: false,
		},
		{
			name: "no_quorum_majority_support",
			votes: []disputetypes.MsgVote{
				{Vote: disputetypes.VoteEnum_VOTE_SUPPORT},
			},
			expectResult:   disputetypes.VoteResult_NO_QUORUM_MAJORITY_SUPPORT,
			reporterJailed: true,
			selectorLocked: true,
		},
		{
			name: "no_quorum_majority_against",
			votes: []disputetypes.MsgVote{
				{Vote: disputetypes.VoteEnum_VOTE_AGAINST},
			},
			expectResult:   disputetypes.VoteResult_NO_QUORUM_MAJORITY_AGAINST,
			reporterJailed: false,
			selectorLocked: false,
		},
		{
			name: "no_quorum_majority_invalid",
			votes: []disputetypes.MsgVote{
				{Vote: disputetypes.VoteEnum_VOTE_INVALID},
			},
			expectResult:   disputetypes.VoteResult_NO_QUORUM_MAJORITY_INVALID,
			reporterJailed: false,
			selectorLocked: false,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			reporterA, reporterB, _, selector, bridgeQueryData, queryID := s.reporterSwitchFixture()
			msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
			disputeMsgServer := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)

			report, reportBlock := s.submitBridgeReport(reporterA, bridgeQueryData, 10)
			s.True(s.snapshotAtReportBlockHasSelector(reporterA, reportBlock, selector))

			disputer := s.newKeysWithTokens()
			disputeID := s.proposeFullMinorDispute(disputer, report)

			// Switch after dispute so we test outcomes while the selector is pending on B.
			s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(11).WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
			_, err := msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
				SelectorAddress: selector.String(),
				ReporterAddress: reporterB.String(),
			})
			s.NoError(err)

			d, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
			s.NoError(err)
			s.NoError(s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, d.HashId))

			teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
			s.NoError(err)
			var voterAddrs []sdk.AccAddress
			if len(tc.votes) == 1 {
				voterAddrs = []sdk.AccAddress{teamAddr}
			} else {
				voterAddrs = []sdk.AccAddress{teamAddr, reporterA, reporterB}
			}
			for i := range tc.votes {
				if i >= len(voterAddrs) {
					break
				}
				d, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
				s.NoError(err)
				if d.DisputeStatus != disputetypes.Voting {
					break // quorum may resolve the dispute before all votes are cast
				}
				vote := tc.votes[i]
				vote.Voter = voterAddrs[i].String()
				vote.Id = disputeID
				_, err = disputeMsgServer.Vote(s.Setup.Ctx, &vote)
				s.NoError(err, "vote %d", i)
			}

			// Full minor fee jails on propose; upheld outcomes keep the lock until we check post-execute.
			if tc.selectorLocked {
				sel, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
				s.NoError(err)
				s.True(reportertypes.SelectorStakeLocked(sel, s.Setup.Ctx.BlockTime()), tc.name)
				s.True(s.snapshotAtReportBlockHasSelector(reporterA, reportBlock, selector))
				stakeB, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
				s.NoError(err)
				valB, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, sdk.ValAddress(reporterB))
				s.NoError(err)
				s.True(stakeB.Equal(valB.Tokens), "locked selector must not add stake to B during pending switch")
			}

			s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(disputekeeper.THREE_DAYS + time.Hour))
			s.NoError(dispute.CheckOpenDisputesForExpiration(s.Setup.Ctx, s.Setup.Disputekeeper))
			_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
			s.NoError(err)

			voteInfo, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, disputeID)
			s.NoError(err)
			s.True(voteInfo.Executed, tc.name)
			s.Equal(tc.expectResult, voteInfo.VoteResult, tc.name)

			repA, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterA.Bytes())
			s.NoError(err)
			effectivelyJailed := repA.Jailed && s.Setup.Ctx.BlockTime().Before(repA.JailedUntil)
			if tc.reporterJailed {
				s.True(repA.Jailed, tc.name)
			} else {
				s.False(effectivelyJailed, tc.name)
			}

			if !tc.selectorLocked {
				sel, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
				s.NoError(err)
				s.False(reportertypes.SelectorStakeLocked(sel, s.Setup.Ctx.BlockTime()), tc.name)
			}
		})
	}
}
