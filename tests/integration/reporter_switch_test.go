package integration_test

import (
	"bytes"
	"encoding/hex"
	"errors"
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

func disputeTallyTotals(t disputetypes.StakeholderVoteCounts) (reporters, users, team uint64) {
	reporters = t.Reporters.Support + t.Reporters.Against + t.Reporters.Invalid
	users = t.Users.Support + t.Users.Against + t.Users.Invalid
	team = t.Team.Support + t.Team.Against + t.Team.Invalid
	return
}

// disputeTallyOrZero treats a missing VoteCountsByGroup entry as an empty tally (pre-first-vote state).
func (s *IntegrationTestSuite) disputeTallyOrZero(disputeID uint64) disputetypes.StakeholderVoteCounts {
	tally, err := s.Setup.Disputekeeper.VoteCountsByGroup.Get(s.Setup.Ctx, disputeID)
	if errors.Is(err, collections.ErrNotFound) {
		return disputetypes.StakeholderVoteCounts{}
	}
	s.NoError(err)
	return tally
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

// TestReporterSwitchDisputeOnFormerReporter verifies that after a selector switches from
// reporter A to B (pending over a long bridge open-commitment window) and A's pre-switch
// report is disputed, the selector stays dispute-locked through switch finalization, B's
// oracle power excludes them until jail expires, then includes them again.
func (s *IntegrationTestSuite) TestReporterSwitchDisputeOnFormerReporter() {
	const minorJailDuration = 600 * time.Second

	reporterA, reporterB, _, selector, bridgeQueryData, queryID := s.reporterSwitchFixture()
	msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	bridgeHeight := int64(10)
	report, reportBlock := s.submitBridgeReport(reporterA, bridgeQueryData, bridgeHeight)
	s.True(s.snapshotAtReportBlockHasSelector(reporterA, reportBlock, selector))

	stakeBBase, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)

	maxCommit, err := s.Setup.Oraclekeeper.GetMaxOpenCommitmentForReporter(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)
	s.Equal(uint64(bridgeHeight)+2000, maxCommit, "bridge report must defer switch over a long block window")

	qMeta, err := s.Setup.Oraclekeeper.CurrentQuery(s.Setup.Ctx, queryID)
	s.NoError(err)

	// Switch A → B while the bridge commitment is still open (pending until maxCommit).
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(bridgeHeight + 1).WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
		SelectorAddress: selector.String(),
		ReporterAddress: reporterB.String(),
	})
	s.NoError(err)

	selPending, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	s.True(bytes.Equal(selPending.Reporter, reporterA.Bytes()), "selection must stay on A until finalize")

	disputer := s.newKeysWithTokens()
	disputeTime := s.Setup.Ctx.BlockTime()
	s.proposeFullMinorDispute(disputer, report)

	selLocked, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	s.True(bytes.Equal(selLocked.Reporter, reporterA.Bytes()), "still on A while switch pending")
	s.True(selLocked.DisputeLockedUntil.After(disputeTime))
	s.True(reportertypes.SelectorStakeLocked(selLocked, s.Setup.Ctx.BlockTime()),
		"selector must be dispute-locked from A's report snapshot")

	stakeBPendingLocked, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.Equal(stakeBBase, stakeBPendingLocked, "B must not gain selector stake while pending and dispute-locked")

	// B reports while switch is still pending and selector is dispute-locked.
	pendingReportHeight := bridgeHeight + 2
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(pendingReportHeight).WithBlockTime(s.Setup.Ctx.BlockTime().Add(10 * time.Second))
	_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &oracletypes.MsgSubmitValue{
		Creator:   reporterB.String(),
		QueryData: bridgeQueryData,
		Value:     bridgeTestValue,
	})
	s.NoError(err)
	repBPending, err := s.Setup.Oraclekeeper.Reports.Get(s.Setup.Ctx, collections.Join3(queryID, reporterB.Bytes(), qMeta.Id))
	s.NoError(err)
	s.Equal(stakeBPendingLocked.Quo(layertypes.PowerReduction).Uint64(), repBPending.Power,
		"B report power must exclude selector while switch pending and dispute-locked")
	s.True(repBPending.Power < report.Power)

	// Finalize after the long open-commitment height, but stay within minor jail (600s wall clock)
	// so the selector is still dispute-locked when the handoff completes.
	finalizeHeight := int64(maxCommit) + 1
	s.True(finalizeHeight > pendingReportHeight, "finalize must wait for the long reporting window")
	elapsedSinceDispute := 30 * time.Second
	s.True(elapsedSinceDispute < minorJailDuration)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(finalizeHeight).WithBlockTime(disputeTime.Add(elapsedSinceDispute))
	_, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)

	selFinal, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	s.True(bytes.Equal(selFinal.Reporter, reporterB.Bytes()), "switch must finalize onto B after unlock height")
	s.True(reportertypes.SelectorStakeLocked(selFinal, s.Setup.Ctx.BlockTime()),
		"selector must still be dispute-locked when switch finalizes after long reporting window")
	s.True(s.Setup.Ctx.BlockTime().Before(selFinal.DisputeLockedUntil),
		"dispute lock must outlive switch finalization in this scenario")

	stakeBAfterFinalizeLocked, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.Equal(stakeBBase, stakeBAfterFinalizeLocked, "dispute-locked selector must not count toward B after finalize")

	lockedReportHeight := finalizeHeight + 1
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(lockedReportHeight).WithBlockTime(disputeTime.Add(elapsedSinceDispute + time.Second))
	_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &oracletypes.MsgSubmitValue{
		Creator:   reporterB.String(),
		QueryData: bridgeQueryData,
		Value:     bridgeTestValue,
	})
	s.NoError(err)
	qMetaAfterFinalize, err := s.Setup.Oraclekeeper.CurrentQuery(s.Setup.Ctx, queryID)
	s.NoError(err)
	repBWhileLocked, err := s.Setup.Oraclekeeper.Reports.Get(s.Setup.Ctx, collections.Join3(queryID, reporterB.Bytes(), qMetaAfterFinalize.Id))
	s.NoError(err)
	expectedLockedPower := stakeBAfterFinalizeLocked.Quo(layertypes.PowerReduction).Uint64()
	s.Equal(expectedLockedPower, repBWhileLocked.Power,
		"B report power after finalize must still exclude selector while dispute-locked")
	s.True(repBWhileLocked.Power < report.Power)

	// Advance past minor jail wall-clock duration (from dispute time, not finalize time).
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(disputeTime.Add(minorJailDuration + time.Second))

	selUnlocked, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	s.False(reportertypes.SelectorStakeLocked(selUnlocked, s.Setup.Ctx.BlockTime()),
		"dispute lock must expire after jail duration")

	stakeBAfterUnlock, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.True(stakeBAfterUnlock.GT(stakeBBase), "B must regain full selector stake after lock expiry")

	spec := registrytypes.DataSpec{
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "tolayer", FieldType: "bool"},
			{Name: "depositId", FieldType: "uint256"},
		},
	}
	bridgeQueryDataUnlocked, err := spec.EncodeData("TRBBridgeV2", `["true","9002"]`)
	s.NoError(err)
	queryIDUnlocked := utils.QueryIDFromData(bridgeQueryDataUnlocked)

	unlockedReportHeight := lockedReportHeight + 1
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(unlockedReportHeight)
	_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &oracletypes.MsgSubmitValue{
		Creator:   reporterB.String(),
		QueryData: bridgeQueryDataUnlocked,
		Value:     bridgeTestValue,
	})
	s.NoError(err)

	qMetaUnlocked, err := s.Setup.Oraclekeeper.CurrentQuery(s.Setup.Ctx, queryIDUnlocked)
	s.NoError(err)
	repBAfterUnlock, err := s.Setup.Oraclekeeper.Reports.Get(s.Setup.Ctx, collections.Join3(queryIDUnlocked, reporterB.Bytes(), qMetaUnlocked.Id))
	s.NoError(err)
	expectedUnlockedPower := stakeBAfterUnlock.Quo(layertypes.PowerReduction).Uint64()
	s.Equal(expectedUnlockedPower, repBAfterUnlock.Power,
		"B report power after dispute lock expiry must include selector stake")
	s.True(repBAfterUnlock.Power > repBWhileLocked.Power,
		"B must gain oracle power from selector after dispute lock expires")
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

// reporterSelfSwitchFixture wires two validator-reporters (A self-reporter, B) with no external selectors.
func (s *IntegrationTestSuite) reporterSelfSwitchFixture() (
	reporterA, reporterB sdk.AccAddress,
	bridgeQueryData, queryID []byte,
) {
	msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)

	valAccs, _, _ := s.createValidatorAccs([]uint64{100, 200})
	reporterA, reporterB = valAccs[0], valAccs[1]

	for i, rep := range []sdk.AccAddress{reporterA, reporterB} {
		_, err := msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
			ReporterAddress:   rep.String(),
			CommissionRate:    reportertypes.DefaultMinCommissionRate,
			MinTokensRequired: math.NewIntWithDecimal(1, 6),
			Moniker:           fmt.Sprintf("self_switch_rep_%d", i),
		})
		s.NoError(err)
	}

	spec := registrytypes.DataSpec{
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "tolayer", FieldType: "bool"},
			{Name: "depositId", FieldType: "uint256"},
		},
	}
	var err error
	bridgeQueryData, err = spec.EncodeData("TRBBridgeV2", `["true","9001"]`)
	s.NoError(err)
	queryID = utils.QueryIDFromData(bridgeQueryData)
	return reporterA, reporterB, bridgeQueryData, queryID
}

// executeMinorDisputeAgainst drives a minor dispute to an executed reporter-wins (AGAINST) outcome.
// Quorum needs ~51% weighted participation across team, users, and reporters (see dispute keeper
// TallyVote); team alone is only ~33%, so reporter validators must vote with non-zero stake.
// Match TestReporterSwitchMinorDisputeOutcomes: advance one block after propose before
// SetBlockInfo/voting, and include every fixture reporter so reporter-group weight counts.
func (s *IntegrationTestSuite) executeMinorDisputeAgainst(
	disputeID uint64,
	reporterVoters ...sdk.AccAddress,
) disputetypes.VoteResult {
	disputeMsgServer := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)

	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1).
		WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	d, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
	s.NoError(err)
	s.NoError(s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, d.HashId))

	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)
	voters := append([]sdk.AccAddress{teamAddr}, reporterVoters...)
	for _, voter := range voters {
		d, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
		s.NoError(err)
		if d.DisputeStatus != disputetypes.Voting {
			break
		}
		_, err = disputeMsgServer.Vote(s.Setup.Ctx, &disputetypes.MsgVote{
			Voter: voter.String(),
			Id:    disputeID,
			Vote:  disputetypes.VoteEnum_VOTE_AGAINST,
		})
		s.NoError(err)
	}

	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(disputekeeper.THREE_DAYS + time.Hour))
	s.NoError(dispute.CheckOpenDisputesForExpiration(s.Setup.Ctx, s.Setup.Disputekeeper))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	s.NoError(err)

	voteInfo, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, disputeID)
	s.NoError(err)
	s.True(voteInfo.Executed)
	s.Equal(disputetypes.VoteResult_AGAINST, voteInfo.VoteResult)
	return voteInfo.VoteResult
}

// TestReporterSwitchFailedDisputePreservesLegacyLock verifies AGAINST clears dispute_locked_until only
// while LockedUntilTime remains; selector stays excluded from B after switch finalize.
//
// 1. Fixture + legacy LockedUntilTime on selector
// 2. Bridge report, minor dispute, vote AGAINST, execute
// 3. Assert dispute lock cleared, legacy lock remains
// 4. Switch to B, finalize at maxCommit+1
// 5. B stake and report power exclude selector
func (s *IntegrationTestSuite) TestReporterSwitchFailedDisputePreservesLegacyLock() {
	reporterA, reporterB, reporterC, selector, bridgeQueryData, queryID := s.reporterSwitchFixture()
	msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	legacyLock := s.Setup.Ctx.BlockTime().Add(21 * 24 * time.Hour)
	sel, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	sel.LockedUntilTime = legacyLock
	sel.DisputeLockedUntil = time.Time{}
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, selector.Bytes(), sel))
	s.True(reportertypes.SelectorStakeLocked(sel, s.Setup.Ctx.BlockTime()))

	bridgeHeight := int64(10)
	report, _ := s.submitBridgeReport(reporterA, bridgeQueryData, bridgeHeight)
	stakeBBase, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)

	disputeID := s.proposeFullMinorDispute(s.newKeysWithTokens(), report)
	// Snapshot reporter C so their vote carries reporter-group weight toward quorum.
	_, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterC, queryID)
	s.NoError(err)
	s.executeMinorDisputeAgainst(disputeID, reporterA, reporterB, reporterC)

	selAfter, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	s.True(selAfter.DisputeLockedUntil.Before(s.Setup.Ctx.BlockTime()) || selAfter.DisputeLockedUntil.IsZero())
	s.True(selAfter.LockedUntilTime.Equal(legacyLock))
	s.True(reportertypes.SelectorStakeLocked(selAfter, s.Setup.Ctx.BlockTime()))

	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(bridgeHeight + 1)
	_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
		SelectorAddress: selector.String(),
		ReporterAddress: reporterB.String(),
	})
	s.NoError(err)

	maxCommit, err := s.Setup.Oraclekeeper.GetMaxOpenCommitmentForReporter(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)
	finalizeHeight := int64(maxCommit) + 1
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(finalizeHeight).WithBlockTime(s.Setup.Ctx.BlockTime().Add(2 * time.Hour))
	stakeBAfter, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.Equal(stakeBBase, stakeBAfter, "legacy LockedUntilTime must keep selector excluded from B after finalize")

	reportHeight := finalizeHeight + 1
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(reportHeight).WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &oracletypes.MsgSubmitValue{
		Creator:   reporterB.String(),
		QueryData: bridgeQueryData,
		Value:     bridgeTestValue,
	})
	s.NoError(err)
	qMetaAfter, err := s.Setup.Oraclekeeper.CurrentQuery(s.Setup.Ctx, queryID)
	s.NoError(err)
	repB, err := s.Setup.Oraclekeeper.Reports.Get(s.Setup.Ctx, collections.Join3(queryID, reporterB.Bytes(), qMetaAfter.Id))
	s.NoError(err)
	s.Equal(stakeBBase.Quo(layertypes.PowerReduction).Uint64(), repB.Power,
		"B report power must exclude selector while legacy LockedUntilTime is active")
}

// TestReporterSwitchSelfDemotionWhileJailed verifies jailed self-reporter demotion copies dispute jail
// to selection, removes the reporter row, and B excludes their stake until unlock.
//
// 1. Self-reporter A and reporter B; A bridge report + minor dispute
// 2. SwitchReporter A→B; Reporters.Has(A) false; selection jail copied
// 3. Pending switch; B stake = solo base through finalize
// 4. After disputeTime+601s, B stake and report power include self delegation
func (s *IntegrationTestSuite) TestReporterSwitchSelfDemotionWhileJailed() {
	const minorJailDuration = 600 * time.Second

	reporterA, reporterB, bridgeQueryData, queryID := s.reporterSelfSwitchFixture()
	msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	valB, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, sdk.ValAddress(reporterB))
	s.NoError(err)
	stakeBBase, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.Equal(valB.Tokens, stakeBBase)

	bridgeHeight := int64(10)
	report, _ := s.submitBridgeReport(reporterA, bridgeQueryData, bridgeHeight)

	disputeTime := s.Setup.Ctx.BlockTime()
	s.proposeFullMinorDispute(s.newKeysWithTokens(), report)

	repA, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)
	s.True(repA.Jailed)

	maxCommit, err := s.Setup.Oraclekeeper.GetMaxOpenCommitmentForReporter(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(int64(maxCommit) + 1)
	_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
		SelectorAddress: reporterA.String(),
		ReporterAddress: reporterB.String(),
	})
	s.NoError(err)

	hasA, err := s.Setup.Reporterkeeper.Reporters.Has(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)
	s.False(hasA)

	sel, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)
	s.True(sel.DisputeLockedUntil.After(disputeTime))
	s.True(sel.LockedUntilTime.IsZero())
	s.True(reportertypes.SelectorStakeLocked(sel, s.Setup.Ctx.BlockTime()))
	s.True(bytes.Equal(sel.Reporter, reporterA.Bytes()), "selection stays on A until finalize")

	stakeBPending, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.Equal(stakeBBase, stakeBPending)

	maxCommit, err = s.Setup.Oraclekeeper.GetMaxOpenCommitmentForReporter(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)
	finalizeHeight := int64(maxCommit) + 1
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(finalizeHeight).WithBlockTime(disputeTime.Add(30 * time.Second))
	stakeBAfterFinalize, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.Equal(stakeBBase, stakeBAfterFinalize)

	selFinal, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)
	s.True(bytes.Equal(selFinal.Reporter, reporterB.Bytes()))

	lockedReportHeight := finalizeHeight + 1
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(lockedReportHeight).WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &oracletypes.MsgSubmitValue{
		Creator:   reporterB.String(),
		QueryData: bridgeQueryData,
		Value:     bridgeTestValue,
	})
	s.NoError(err)
	qMetaAfter, err := s.Setup.Oraclekeeper.CurrentQuery(s.Setup.Ctx, queryID)
	s.NoError(err)
	repBLocked, err := s.Setup.Oraclekeeper.Reports.Get(s.Setup.Ctx, collections.Join3(queryID, reporterB.Bytes(), qMetaAfter.Id))
	s.NoError(err)
	s.Equal(stakeBBase.Quo(layertypes.PowerReduction).Uint64(), repBLocked.Power)

	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(disputeTime.Add(minorJailDuration + time.Second))
	selUnlocked, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)
	s.False(reportertypes.SelectorStakeLocked(selUnlocked, s.Setup.Ctx.BlockTime()))

	stakeBUnlocked, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.True(stakeBUnlocked.GT(stakeBBase))
}

// TestReporterSwitchDisputeLockExpiredBeforeFinalize verifies Option A: when dispute lock expires
// before block-height unlock, selector stake counts toward B immediately at finalize.
//
// 1. Fixture + bridge report; switch A→B pending; dispute A's report
// 2. While pending+locked: B stake = base
// 3. Advance wall clock past minor jail only; then maxCommit+1 finalize
// 4. B stake and report power include selector
func (s *IntegrationTestSuite) TestReporterSwitchDisputeLockExpiredBeforeFinalize() {
	const minorJailDuration = 600 * time.Second

	reporterA, reporterB, _, selector, bridgeQueryData, queryID := s.reporterSwitchFixture()
	msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	bridgeHeight := int64(10)
	report, _ := s.submitBridgeReport(reporterA, bridgeQueryData, bridgeHeight)
	stakeBBase, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)

	maxCommit, err := s.Setup.Oraclekeeper.GetMaxOpenCommitmentForReporter(s.Setup.Ctx, reporterA.Bytes())
	s.NoError(err)

	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(bridgeHeight + 1)
	_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
		SelectorAddress: selector.String(),
		ReporterAddress: reporterB.String(),
	})
	s.NoError(err)

	disputeTime := s.Setup.Ctx.BlockTime()
	s.proposeFullMinorDispute(s.newKeysWithTokens(), report)

	stakeBPending, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.Equal(stakeBBase, stakeBPending)

	// Simulate jail ending before the long bridge window closes (accelerated block time in tests).
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(disputeTime.Add(minorJailDuration + time.Second))
	sel, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	s.False(reportertypes.SelectorStakeLocked(sel, s.Setup.Ctx.BlockTime()))

	finalizeHeight := int64(maxCommit) + 1
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(finalizeHeight)
	stakeBAfter, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterB, queryID)
	s.NoError(err)
	s.True(stakeBAfter.GT(stakeBBase), "selector stake must count toward B at finalize when dispute lock already expired")

	selFinal, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
	s.NoError(err)
	s.True(bytes.Equal(selFinal.Reporter, reporterB.Bytes()))

	reportHeight := finalizeHeight + 1
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(reportHeight).WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &oracletypes.MsgSubmitValue{
		Creator:   reporterB.String(),
		QueryData: bridgeQueryData,
		Value:     bridgeTestValue,
	})
	s.NoError(err)
	qMetaAfter, err := s.Setup.Oraclekeeper.CurrentQuery(s.Setup.Ctx, queryID)
	s.NoError(err)
	repB, err := s.Setup.Oraclekeeper.Reports.Get(s.Setup.Ctx, collections.Join3(queryID, reporterB.Bytes(), qMetaAfter.Id))
	s.NoError(err)
	s.Equal(stakeBAfter.Quo(layertypes.PowerReduction).Uint64(), repB.Power)
}

// TestDisputeVoteSkipsLockedSelectorPendingSwitch verifies dispute-locked selectors with a pending
// switch do not contribute reporter-group stake (SetVoterReporterStake returns 0). MsgVote may still
// fail when team + tips + reporter power sum to zero; it must not fail solely because repP is zero.
func (s *IntegrationTestSuite) TestDisputeVoteSkipsLockedSelectorPendingSwitch() {
	s.Run("locked_selector_no_total_power", func() {
		s.SetupTest()
		reporterA, reporterB, _, selector, bridgeQueryData, _ := s.reporterSwitchFixture()
		msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
		disputeMsgServer := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)

		report, _ := s.submitBridgeReport(reporterA, bridgeQueryData, 10)

		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(11)
		_, err := msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
			SelectorAddress: selector.String(),
			ReporterAddress: reporterB.String(),
		})
		s.NoError(err)

		disputeID := s.proposeFullMinorDispute(s.newKeysWithTokens(), report)
		selAfter, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
		s.NoError(err)
		s.True(reportertypes.SelectorStakeLocked(selAfter, s.Setup.Ctx.BlockTime()))

		d, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
		s.NoError(err)
		s.NoError(s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, d.HashId))

		repP, err := s.Setup.Disputekeeper.SetVoterReporterStake(
			s.Setup.Ctx, disputeID, selector, d.BlockNumber,
			disputetypes.VoteEnum_VOTE_SUPPORT, nil,
		)
		s.NoError(err)
		s.True(repP.IsZero(), "locked selector must not contribute reporter-group stake")

		tallyBefore := disputetypes.StakeholderVoteCounts{}

		_, err = disputeMsgServer.Vote(s.Setup.Ctx, &disputetypes.MsgVote{
			Voter: selector.String(),
			Id:    disputeID,
			Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
		})
		s.ErrorContains(err, "voter power is zero",
			"vote fails when total power is zero, not because reporter power alone is zero")

		tallyAfter := s.disputeTallyOrZero(disputeID)
		repBefore, userBefore, teamBefore := disputeTallyTotals(tallyBefore)
		repAfter, userAfter, teamAfter := disputeTallyTotals(tallyAfter)
		s.Equal(repBefore, repAfter)
		s.Equal(userBefore, userAfter)
		s.Equal(teamBefore, teamAfter, "failed vote must not change group tallies")

		tallyBefore = tallyAfter
		teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
		s.NoError(err)
		_, err = disputeMsgServer.Vote(s.Setup.Ctx, &disputetypes.MsgVote{
			Voter: teamAddr.String(),
			Id:    disputeID,
			Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
		})
		s.NoError(err)

		tallyAfter = s.disputeTallyOrZero(disputeID)
		_, _, teamBefore = disputeTallyTotals(tallyBefore)
		_, _, teamAfter = disputeTallyTotals(tallyAfter)
		s.Greater(teamAfter, teamBefore, "team vote should increase tallies after failed selector vote")

		hasSelVote, err := s.Setup.Disputekeeper.Voter.Has(s.Setup.Ctx, collections.Join(disputeID, selector.Bytes()))
		s.NoError(err)
		s.False(hasSelVote)

		hasBVote, err := s.Setup.Disputekeeper.Voter.Has(s.Setup.Ctx, collections.Join(disputeID, reporterB.Bytes()))
		s.NoError(err)
		s.False(hasBVote, "selector stake must not count toward B reporter vote bucket")
	})

	s.Run("locked_selector_with_tips_succeeds_without_reporter_power", func() {
		s.SetupTest()
		reporterA, reporterB, _, selector, bridgeQueryData, _ := s.reporterSwitchFixture()
		msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
		disputeMsgServer := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)
		oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

		report, _ := s.submitBridgeReport(reporterA, bridgeQueryData, 10)

		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(11)
		_, err := msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{
			SelectorAddress: selector.String(),
			ReporterAddress: reporterB.String(),
		})
		s.NoError(err)

		// Tip at the dispute block so user-group power is non-zero while reporter stake stays excluded.
		s.Setup.MintTokens(selector, math.NewInt(10_000_000))
		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(12)
		_, err = oracleMsgServer.Tip(s.Setup.Ctx, &oracletypes.MsgTip{
			Tipper:    selector.String(),
			QueryData: bridgeQueryData,
			Amount:    sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000)),
		})
		s.NoError(err)

		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(12)
		disputeID := s.proposeFullMinorDispute(s.newKeysWithTokens(), report)
		selAfter, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
		s.NoError(err)
		s.True(reportertypes.SelectorStakeLocked(selAfter, s.Setup.Ctx.BlockTime()))

		d, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
		s.NoError(err)
		s.NoError(s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, d.HashId))

		tallyBefore := disputetypes.StakeholderVoteCounts{}

		_, err = disputeMsgServer.Vote(s.Setup.Ctx, &disputetypes.MsgVote{
			Voter: selector.String(),
			Id:    disputeID,
			Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
		})
		s.NoError(err, "vote succeeds when tips supply non-zero total power despite zero reporter stake")

		tallyAfter := s.disputeTallyOrZero(disputeID)
		repBefore, userBefore, _ := disputeTallyTotals(tallyBefore)
		repAfter, userAfter, _ := disputeTallyTotals(tallyAfter)
		s.Equal(repBefore, repAfter, "locked selector must not contribute reporter-group tallies")
		s.Greater(userAfter, userBefore, "selector tips should count toward user-group tallies")

		tallyBefore = tallyAfter
		teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
		s.NoError(err)
		_, err = disputeMsgServer.Vote(s.Setup.Ctx, &disputetypes.MsgVote{
			Voter: teamAddr.String(),
			Id:    disputeID,
			Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
		})
		s.NoError(err)

		tallyAfter = s.disputeTallyOrZero(disputeID)
		repBefore, userBefore, teamBefore := disputeTallyTotals(tallyBefore)
		repAfter, userAfter, teamAfter := disputeTallyTotals(tallyAfter)
		s.Equal(repBefore, repAfter, "team vote must not change reporter tallies")
		s.Equal(userBefore, userAfter, "team vote must not change user tallies from selector vote")
		s.Greater(teamAfter, teamBefore, "team vote should increase tallies after selector vote")

		selVote, err := s.Setup.Disputekeeper.Voter.Get(s.Setup.Ctx, collections.Join(disputeID, selector.Bytes()))
		s.NoError(err)
		s.True(selVote.ReporterPower.IsZero())
		s.True(selVote.VoterPower.IsPositive())

		hasBVote, err := s.Setup.Disputekeeper.Voter.Has(s.Setup.Ctx, collections.Join(disputeID, reporterB.Bytes()))
		s.NoError(err)
		s.False(hasBVote)
		_ = reporterB
	})

	s.Run("unlocked_selector_contributes_reporter_power", func() {
		s.SetupTest()
		reporterA, reporterB, reporterC, selector, bridgeQueryData, queryID := s.reporterSwitchFixture()

		report, _ := s.submitBridgeReport(reporterA, bridgeQueryData, 10)
		disputeID := s.proposeFullMinorDispute(s.newKeysWithTokens(), report)
		_, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterC, queryID)
		s.NoError(err)
		s.executeMinorDisputeAgainst(disputeID, reporterA, reporterB, reporterC)

		sel, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, selector.Bytes())
		s.NoError(err)
		s.False(reportertypes.SelectorStakeLocked(sel, s.Setup.Ctx.BlockTime()))

		d, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
		s.NoError(err)

		repP, err := s.Setup.Disputekeeper.SetVoterReporterStake(
			s.Setup.Ctx, disputeID, selector, d.BlockNumber,
			disputetypes.VoteEnum_VOTE_SUPPORT, nil,
		)
		s.NoError(err)
		s.True(repP.IsPositive(), "unlocked selector must contribute reporter-group stake")
	})
}
