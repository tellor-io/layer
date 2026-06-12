package integration_test

// Integration tests for multi-round dispute fee accounting. Round-1 fees are the only
// refundable fees; later (escalation) round fees are fully consumed as burn and voter
// rewards.

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// setupDisputedReporterWithBondPayers registers a validator-reporter for each power and
// stores a disputable report for the first one. The extra reporters have stake of their
// own, so they can pay dispute fees from bond without disputing their own report.
func (s *IntegrationTestSuite) setupDisputedReporterWithBondPayers(powers ...uint64) ([]sdk.AccAddress, oracletypes.MicroReport, math.Int) {
	repAccs, _, _ := s.createValidatorAccs(powers)
	for i, reporter := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), fmt.Sprintf("reporter_moniker_%d", i))))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter, reportertypes.NewSelection(reporter, 1)))
	}

	disputedReporter := repAccs[0]
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	stake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, disputedReporter, qId)
	s.NoError(err)
	report := oracletypes.MicroReport{
		Reporter:    disputedReporter.String(),
		Power:       stake.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Now().Add(-1 * 12 * time.Hour),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, disputedReporter.Bytes(), report.MetaId), report))
	fee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)
	return repAccs, report, fee
}

// setupDisputedReporter registers a single validator-reporter and stores a disputable report.
func (s *IntegrationTestSuite) setupDisputedReporter(power uint64) (sdk.AccAddress, oracletypes.MicroReport, math.Int) {
	repAccs, report, fee := s.setupDisputedReporterWithBondPayers(power)
	return repAccs[0], report, fee
}

// fundedDisputer returns a fresh account with enough free tokens to pay dispute fees.
func (s *IntegrationTestSuite) fundedDisputer() sdk.AccAddress {
	addr := s.newKeysWithTokens()
	s.Setup.MintTokens(addr, math.NewInt(100_000_000))
	return addr
}

// seedTipper credits addr with all tips at the report's block so the address carries
// user voting power for disputes over that report.
func (s *IntegrationTestSuite) seedTipper(addr sdk.AccAddress, report oracletypes.MicroReport) {
	s.NoError(s.Setup.Oraclekeeper.TipperTotal.Set(s.Setup.Ctx, collections.Join(addr.Bytes(), report.BlockNumber), math.NewInt(100)))
	s.NoError(s.Setup.Oraclekeeper.TotalTips.Set(s.Setup.Ctx, report.BlockNumber, math.NewInt(100)))
}

// proposeRound submits a fully funded warning-category dispute proposal; while the
// dispute is open and unresolved this opens the next escalation round.
func (s *IntegrationTestSuite) proposeRound(msgServer types.MsgServer, creator sdk.AccAddress, report oracletypes.MicroReport, fee math.Int, payFromBond bool) {
	_, err := msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          creator.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, fee),
		DisputeCategory:  types.Warning,
		PayFromBond:      payFromBond,
	})
	s.NoError(err)
}

// markRoundUnresolved gives the round a single INVALID vote that cannot reach quorum,
// waits out the voting period, and tallies, leaving the round Unresolved so the next
// escalation round can be opened.
func (s *IntegrationTestSuite) markRoundUnresolved(msgServer types.MsgServer, disputeId uint64, voter sdk.AccAddress) {
	_, err := msgServer.Vote(s.Setup.Ctx, &types.MsgVote{Voter: voter.String(), Id: disputeId, Vote: types.VoteEnum_VOTE_INVALID})
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.TWO_DAYS + 1))
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, disputeId))
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeId)
	s.NoError(err)
	s.Equal(types.Unresolved, dispute.DisputeStatus, "round must be unresolved so the next escalation round can be opened")
}

// startNoQuorumRound1 proposes a dispute fully funded by payer (optionally from bond),
// seeds the dispute's block info, and leaves round 1 Unresolved via a single no-quorum
// vote by voter.
func (s *IntegrationTestSuite) startNoQuorumRound1(msgServer types.MsgServer, report oracletypes.MicroReport, disputeFee math.Int, payer sdk.AccAddress, payFromBond bool, voter sdk.AccAddress) {
	s.proposeRound(msgServer, payer, report, disputeFee, payFromBond)
	d1, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.NoError(s.Setup.Disputekeeper.BlockInfo.Set(s.Setup.Ctx, d1.HashId, types.BlockInfo{TotalReporterPower: math.NewInt(int64(report.Power)).Mul(sdk.DefaultPowerReduction), TotalUserTips: math.NewInt(100)}))
	s.markRoundUnresolved(msgServer, 1, voter)
}

// voteAndTally casts a single vote on the round, waits out the voting period, and tallies.
func (s *IntegrationTestSuite) voteAndTally(msgServer types.MsgServer, disputeId uint64, voter sdk.AccAddress, vote types.VoteEnum) {
	_, err := msgServer.Vote(s.Setup.Ctx, &types.MsgVote{Voter: voter.String(), Id: disputeId, Vote: vote})
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, disputeId))
}

func feeTrackerContainsDelegator(tracker reportertypes.DelegationsAmounts, delegator sdk.AccAddress) bool {
	for _, origin := range tracker.TokenOrigins {
		if sdk.AccAddress(origin.DelegatorAddress).Equals(delegator) {
			return true
		}
	}
	return false
}

// TestMultiRoundSupportRefundsRoundOneStakeOnly: on a multi-round SUPPORT resolution,
// the round-1 fee payer is refunded 95% of the round-1 fee and receives the full slashed
// bond, while a later-round payer gets nothing: escalation-round fees are fully consumed
// and never refundable.
func (s *IntegrationTestSuite) TestMultiRoundSupportRefundsRoundOneStakeOnly() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, report, disputeFee := s.setupDisputedReporter(100)
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	disputer1 := s.fundedDisputer()
	s.startNoQuorumRound1(msgServer, report, disputeFee, disputer1, false, teamAddr)

	// round 2 funded by a different payer, then resolves SUPPORT
	disputer2 := s.fundedDisputer()
	s.proposeRound(msgServer, disputer2, report, disputeFee, false)
	s.voteAndTally(msgServer, 2, teamAddr, types.VoteEnum_VOTE_SUPPORT)
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))

	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)
	slashAmount := dispute.SlashAmount

	// the round-2 fee is consumed in full:
	// BurnAmount = round-1's 5% (SlashAmount/20) + the entire round-2 fee (SlashAmount/10)
	roundFee2 := slashAmount.QuoRaw(10)
	s.Equal(slashAmount.QuoRaw(20).Add(roundFee2), dispute.BurnAmount)

	// round-1 payer: refunded 95% of the round-1 fee + the full slashed bond
	expRefund, _ := keeper.CalculateRefundAmount(slashAmount, slashAmount)
	bondedBefore, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.NoError(err)
	bal1Before := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer1, s.Setup.Denom)
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer1.String(), CallerAddress: disputer1.String()})
	s.NoError(err)
	s.Equal(expRefund, s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer1, s.Setup.Denom).Amount.Sub(bal1Before.Amount))
	bondedAfter, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.NoError(err)
	s.Equal(slashAmount, bondedAfter.Sub(bondedBefore)) // entire slashed bond restaked to the round-1 payer

	// round-2 payer does not get the round-2 fee back
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer2.String(), CallerAddress: disputer2.String()})
	s.Error(err)
}

// TestMultiRoundSupportConservesWithVoterClaim: after a multi-round SUPPORT resolution,
// the round-1 payer claims the fee refund plus the slashed bond, the round-2 payer gets
// nothing, and the voter claims the voter reward (half of BurnAmount: 2.5% of the
// round-1 fee plus half of the round-2 fee). All claims together drain the dispute
// module to ~zero.
func (s *IntegrationTestSuite) TestMultiRoundSupportConservesWithVoterClaim() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, report, disputeFee := s.setupDisputedReporter(100)
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	tipper := s.fundedDisputer()
	s.seedTipper(tipper, report)

	disputer1 := s.fundedDisputer()
	s.startNoQuorumRound1(msgServer, report, disputeFee, disputer1, false, teamAddr)

	disputer2 := s.fundedDisputer()
	s.proposeRound(msgServer, disputer2, report, disputeFee, false)
	_, err = msgServer.Vote(s.Setup.Ctx, &types.MsgVote{Voter: teamAddr.String(), Id: 2, Vote: types.VoteEnum_VOTE_SUPPORT})
	s.NoError(err)
	_, err = msgServer.Vote(s.Setup.Ctx, &types.MsgVote{Voter: tipper.String(), Id: 2, Vote: types.VoteEnum_VOTE_SUPPORT})
	s.NoError(err)

	// team power plus all user power reach quorum, so the round resolves without
	// waiting out the voting period
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)
	s.True(dispute.PendingExecution)
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))
	vote, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.VoteResult_SUPPORT, vote.VoteResult)
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)

	// the voter reward is half of the consumed pool, which includes the round-2 fee
	s.Equal(dispute.BurnAmount.QuoRaw(2), dispute.VoterReward)
	s.True(dispute.VoterReward.GT(dispute.SlashAmount.QuoRaw(40))) // more than round-1's voter half alone

	disputeModAddr := authtypes.NewModuleAddress(types.ModuleName)

	// round-1 payer claims refund + bond
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer1.String(), CallerAddress: disputer1.String()})
	s.NoError(err)
	// round-2 payer is not refunded the fee
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer2.String(), CallerAddress: disputer2.String()})
	s.Error(err)
	// voter claims the voter reward
	_, err = msgServer.ClaimReward(s.Setup.Ctx, &types.MsgClaimReward{CallerAddress: tipper.String(), DisputeId: 2})
	s.NoError(err)

	modFinal := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputeModAddr, s.Setup.Denom)
	s.True(modFinal.Amount.LTE(math.NewInt(1)), "dispute module should net to ~zero after all claims, has %s", modFinal.Amount)
}

// TestMultiRoundAgainstReporterGetsRoundOneStakeOnly: when a multi-round dispute
// resolves AGAINST, the reporter is awarded their bond back plus 95% of the round-1 fee,
// while later-round fees stay consumed. Disputers get nothing.
func (s *IntegrationTestSuite) TestMultiRoundAgainstReporterGetsRoundOneStakeOnly() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, report, disputeFee := s.setupDisputedReporter(100)
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	disputer1 := s.fundedDisputer()
	s.startNoQuorumRound1(msgServer, report, disputeFee, disputer1, false, teamAddr)

	// round 2 funded by a different payer
	disputer2 := s.fundedDisputer()
	s.proposeRound(msgServer, disputer2, report, disputeFee, false)
	s.voteAndTally(msgServer, 2, teamAddr, types.VoteEnum_VOTE_AGAINST)

	disputeModAddr := authtypes.NewModuleAddress(types.ModuleName)
	bondedBefore, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.NoError(err)

	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))

	vote, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.VoteResult_NO_QUORUM_MAJORITY_AGAINST, vote.VoteResult)
	bondedAfter, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.NoError(err)
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)

	// reporter gets bond back + 95% of the round-1 fee
	expectedReporterGain := dispute.SlashAmount.Add(dispute.DisputeFee.Sub(dispute.DisputeFee.QuoRaw(20)))
	s.Equal(expectedReporterGain, bondedAfter.Sub(bondedBefore))

	// only the reserved voter reward remains in the dispute module
	s.Equal(dispute.VoterReward, s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputeModAddr, s.Setup.Denom).Amount)

	// disputers get nothing on AGAINST
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer1.String(), CallerAddress: disputer1.String()})
	s.Error(err)
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer2.String(), CallerAddress: disputer2.String()})
	s.Error(err)
}

// TestMultiRoundPayFromBondDoesNotTrackLaterRoundAsRefundableStake: the
// FeePaidFromStake tracker exists so FeeRefund can restake the refundable round-1 fee.
// With round 1 funded from an account there is no tracker, and a later round paid from
// bond must not create one: escalation-round fees are fully consumed.
func (s *IntegrationTestSuite) TestMultiRoundPayFromBondDoesNotTrackLaterRoundAsRefundableStake() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	reporters, report, disputeFee := s.setupDisputedReporterWithBondPayers(100, 100)
	roundTwoBondPayer := reporters[1]
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	roundOnePayer := s.fundedDisputer()
	s.startNoQuorumRound1(msgServer, report, disputeFee, roundOnePayer, false, teamAddr)

	firstRoundDispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	hasTrackedStakeFee, err := s.Setup.Reporterkeeper.FeePaidFromStake.Has(s.Setup.Ctx, firstRoundDispute.HashId)
	s.NoError(err)
	s.False(hasTrackedStakeFee, "round 1 was account-funded, so no stake-fee refund tracker should exist before later rounds")

	s.proposeRound(msgServer, roundTwoBondPayer, report, disputeFee, true)

	hasTrackedStakeFee, err = s.Setup.Reporterkeeper.FeePaidFromStake.Has(s.Setup.Ctx, firstRoundDispute.HashId)
	s.NoError(err)
	s.False(hasTrackedStakeFee, "later-round fees paid from bond are non-refundable and must not be tracked as refundable round-1 stake")
}

// TestMultiRoundPayFromBondDoesNotAppendToFirstRoundStakeRefundTracker: when round 1
// itself was paid from bond, FeePaidFromStake holds a snapshot of the round-1 stake
// origins. Opening a later round from bond must leave that snapshot untouched.
func (s *IntegrationTestSuite) TestMultiRoundPayFromBondDoesNotAppendToFirstRoundStakeRefundTracker() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	reporters, report, disputeFee := s.setupDisputedReporterWithBondPayers(100, 100, 100)
	roundOneBondPayer := reporters[1]
	roundTwoBondPayer := reporters[2]
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	s.startNoQuorumRound1(msgServer, report, disputeFee, roundOneBondPayer, true, teamAddr)
	firstRoundDispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	trackerBeforeRound2, err := s.Setup.Reporterkeeper.FeePaidFromStake.Get(s.Setup.Ctx, firstRoundDispute.HashId)
	s.NoError(err)
	s.Equal(disputeFee, trackerBeforeRound2.Total, "round 1 paid from bond should track exactly the refundable round-1 fee")
	s.True(feeTrackerContainsDelegator(trackerBeforeRound2, roundOneBondPayer))
	s.False(feeTrackerContainsDelegator(trackerBeforeRound2, roundTwoBondPayer))

	s.proposeRound(msgServer, roundTwoBondPayer, report, disputeFee, true)

	trackerAfterRound2, err := s.Setup.Reporterkeeper.FeePaidFromStake.Get(s.Setup.Ctx, firstRoundDispute.HashId)
	s.NoError(err)
	s.Equal(trackerBeforeRound2.Total, trackerAfterRound2.Total, "later-round fees paid from bond must not increase the refundable round-1 total")
	s.Equal(len(trackerBeforeRound2.TokenOrigins), len(trackerAfterRound2.TokenOrigins), "later-round fees paid from bond must not append token origins to the round-1 refund tracker")
	s.False(feeTrackerContainsDelegator(trackerAfterRound2, roundTwoBondPayer), "a later-round bond payer must not become a refund-tracker delegator")
}

// TestMultiRoundPayFromBondRefundDoesNotRestakeLaterRoundBondPayer: when the round-1
// payer withdraws an INVALID-resolution refund, only round-1 stake origins are restored.
// A later-round bond payer's fee was consumed by escalation and must not be restaked
// from the round-1 refund.
func (s *IntegrationTestSuite) TestMultiRoundPayFromBondRefundDoesNotRestakeLaterRoundBondPayer() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	reporters, report, disputeFee := s.setupDisputedReporterWithBondPayers(100, 100, 100)
	roundOneBondPayer := reporters[1]
	roundTwoBondPayer := reporters[2]
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	s.startNoQuorumRound1(msgServer, report, disputeFee, roundOneBondPayer, true, teamAddr)
	s.proposeRound(msgServer, roundTwoBondPayer, report, disputeFee, true)

	// resolve the final round INVALID: the round-1 fee is refunded to stake without
	// adding the disputed reporter's slashed bond, isolating the refund split across
	// FeePaidFromStake origins
	s.voteAndTally(msgServer, 2, teamAddr, types.VoteEnum_VOTE_INVALID)
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))

	roundOneStakeBefore, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, roundOneBondPayer, report.QueryId)
	s.NoError(err)
	roundTwoStakeBefore, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, roundTwoBondPayer, report.QueryId)
	s.NoError(err)
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: roundOneBondPayer.String(), CallerAddress: roundOneBondPayer.String()})
	s.NoError(err)
	roundOneStakeAfter, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, roundOneBondPayer, report.QueryId)
	s.NoError(err)
	roundTwoStakeAfter, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, roundTwoBondPayer, report.QueryId)
	s.NoError(err)

	s.True(roundOneStakeAfter.GT(roundOneStakeBefore), "the round-1 bond payer should receive a stake refund")
	s.Equal(roundTwoStakeBefore, roundTwoStakeAfter, "a later-round bond payer must not receive stake from the round-1 fee refund")
}

// TestWorstCaseMultiRoundPayFromBondMaxRoundsDoesNotDiluteFirstRoundStakeRefund: round 1
// is paid from bond, then every later round up to the round cap is opened by a different
// bond payer. The round-1 refund tracker must still contain only the original round-1
// payer and amount.
func (s *IntegrationTestSuite) TestWorstCaseMultiRoundPayFromBondMaxRoundsDoesNotDiluteFirstRoundStakeRefund() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	reporters, report, disputeFee := s.setupDisputedReporterWithBondPayers(100, 100, 100, 100, 100, 100)
	roundOneBondPayer := reporters[1]
	laterRoundBondPayers := reporters[2:]
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	s.startNoQuorumRound1(msgServer, report, disputeFee, roundOneBondPayer, true, teamAddr)
	firstRoundDispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	trackerBeforeEscalation, err := s.Setup.Reporterkeeper.FeePaidFromStake.Get(s.Setup.Ctx, firstRoundDispute.HashId)
	s.NoError(err)

	for roundId := uint64(2); roundId <= 5; roundId++ {
		s.proposeRound(msgServer, laterRoundBondPayers[roundId-2], report, disputeFee, true)
		if roundId < 5 {
			s.markRoundUnresolved(msgServer, roundId, teamAddr)
		}
	}

	trackerAfterMaxEscalation, err := s.Setup.Reporterkeeper.FeePaidFromStake.Get(s.Setup.Ctx, firstRoundDispute.HashId)
	s.NoError(err)
	s.Equal(trackerBeforeEscalation.Total, trackerAfterMaxEscalation.Total, "max escalation from bond must not dilute the refundable round-1 total")
	s.Equal(len(trackerBeforeEscalation.TokenOrigins), len(trackerAfterMaxEscalation.TokenOrigins), "max escalation from bond must not add refund recipients")
	for _, payer := range laterRoundBondPayers {
		s.False(feeTrackerContainsDelegator(trackerAfterMaxEscalation, payer), "later-round bond payer %s must not be in the round-1 refund tracker", payer.String())
	}
}

// TestMultiRoundTeamOnlyFinalRoundDoesNotCreateUnclaimableVoterReward: team votes can
// decide a dispute but cannot claim voter rewards, so a team-only resolution must burn
// the entire consumed fee pool instead of reserving an unclaimable voter half in the
// dispute module.
func (s *IntegrationTestSuite) TestMultiRoundTeamOnlyFinalRoundDoesNotCreateUnclaimableVoterReward() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, report, disputeFee := s.setupDisputedReporter(100)
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	disputer1 := s.fundedDisputer()
	s.startNoQuorumRound1(msgServer, report, disputeFee, disputer1, false, teamAddr)

	// round 2 is account-funded and resolved by a team-only vote
	disputer2 := s.fundedDisputer()
	s.proposeRound(msgServer, disputer2, report, disputeFee, false)
	s.voteAndTally(msgServer, 2, teamAddr, types.VoteEnum_VOTE_AGAINST)
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))

	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)
	s.True(dispute.VoterReward.IsZero(), "team-only resolutions have no claimable voters, so no voter reward should be reserved")

	disputeModAddr := authtypes.NewModuleAddress(types.ModuleName)
	modBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputeModAddr, s.Setup.Denom)
	s.True(modBalance.Amount.IsZero(), "later-round fees should be fully consumed rather than stranded in the dispute module")
}

// TestWorstCaseTeamOnlyMaxRoundDoesNotStrandEscalatedVoterRewards: after several
// escalation rounds the consumed fee pool is large. If the final round resolves with
// only a team vote, the whole pool must be burned; none of it may stay stranded in the
// dispute module as an unclaimable voter reward.
func (s *IntegrationTestSuite) TestWorstCaseTeamOnlyMaxRoundDoesNotStrandEscalatedVoterRewards() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, report, disputeFee := s.setupDisputedReporter(100)
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	roundOnePayer := s.fundedDisputer()
	s.startNoQuorumRound1(msgServer, report, disputeFee, roundOnePayer, false, teamAddr)

	for roundId := uint64(2); roundId <= 5; roundId++ {
		s.proposeRound(msgServer, s.fundedDisputer(), report, disputeFee, false)
		if roundId < 5 {
			s.markRoundUnresolved(msgServer, roundId, teamAddr)
		}
	}

	s.voteAndTally(msgServer, 5, teamAddr, types.VoteEnum_VOTE_AGAINST)
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 5))

	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 5)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)
	s.True(dispute.VoterReward.IsZero(), "team-only resolutions have no claimable voters, so no voter reward should be reserved")

	disputeModAddr := authtypes.NewModuleAddress(types.ModuleName)
	modBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputeModAddr, s.Setup.Denom)
	s.True(modBalance.Amount.IsZero(), "escalation fees must be fully consumed rather than stranded in the dispute module")
}

// TestMultiRoundFinalRoundNoVotesStillRewardsPreviousRoundVoters: if the final round
// expires with no votes, previous-round user/reporter voters are still claim-eligible,
// so the voter half of the consumed fee pool must be reserved and claimable.
func (s *IntegrationTestSuite) TestMultiRoundFinalRoundNoVotesStillRewardsPreviousRoundVoters() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, report, disputeFee := s.setupDisputedReporter(100)

	previousRoundVoter := s.fundedDisputer()
	s.seedTipper(previousRoundVoter, report)

	roundOnePayer := s.fundedDisputer()
	s.startNoQuorumRound1(msgServer, report, disputeFee, roundOnePayer, false, previousRoundVoter)

	roundTwoPayer := s.fundedDisputer()
	s.proposeRound(msgServer, roundTwoPayer, report, disputeFee, false)

	hasFinalRoundVoteCounts, err := s.Setup.Disputekeeper.VoteCountsByGroup.Has(s.Setup.Ctx, 2)
	s.NoError(err)
	s.False(hasFinalRoundVoteCounts, "final round must have no vote counts for this regression")

	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 2))
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))

	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)
	s.Equal(dispute.BurnAmount.QuoRaw(2), dispute.VoterReward, "previous-round voters should keep the voter reward claimable")

	expectedReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, previousRoundVoter, 2)
	s.Require().NoError(err)
	s.True(expectedReward.IsPositive(), "previous-round voter should have a claimable reward")

	balBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, previousRoundVoter, s.Setup.Denom)
	_, err = msgServer.ClaimReward(s.Setup.Ctx, &types.MsgClaimReward{CallerAddress: previousRoundVoter.String(), DisputeId: 2})
	s.NoError(err)
	balAfter := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, previousRoundVoter, s.Setup.Denom)
	s.Equal(expectedReward, balAfter.Amount.Sub(balBefore.Amount))
}

// TestClaimableDisputeRewardsUsesFirstRoundFeePayerForFinalRound: fee payer records
// live under the first-round dispute id, but wallets query the final resolved round id.
// The query must resolve the round-1 payer the same way WithdrawFeeRefund does.
func (s *IntegrationTestSuite) TestClaimableDisputeRewardsUsesFirstRoundFeePayerForFinalRound() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, report, disputeFee := s.setupDisputedReporter(100)
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	disputer1 := s.fundedDisputer()
	s.startNoQuorumRound1(msgServer, report, disputeFee, disputer1, false, teamAddr)

	disputer2 := s.fundedDisputer()
	s.proposeRound(msgServer, disputer2, report, disputeFee, false)
	s.voteAndTally(msgServer, 2, teamAddr, types.VoteEnum_VOTE_SUPPORT)
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))

	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	refund, _ := keeper.CalculateRefundAmount(dispute.SlashAmount, dispute.SlashAmount)
	expectedClaimable := refund.Add(dispute.SlashAmount)

	queryServer := keeper.NewQuerier(s.Setup.Disputekeeper)
	resp, err := queryServer.ClaimableDisputeRewards(s.Setup.Ctx, &types.QueryClaimableDisputeRewardsRequest{
		DisputeId: 2,
		Address:   disputer1.String(),
	})
	s.NoError(err)
	s.Equal(expectedClaimable, resp.ClaimableAmount.FeeRefundAmount, "final-round query should report the round-1 payer's fee refund plus reporter bond reward")
}

// TestClaimableDisputeRewardsIncludesPreviousRoundOnlyVoter: ClaimReward pays voters
// from any round by scanning PrevDisputeIds, so the query must not require a Voter
// record under the final round id before reporting the reward.
func (s *IntegrationTestSuite) TestClaimableDisputeRewardsIncludesPreviousRoundOnlyVoter() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, report, disputeFee := s.setupDisputedReporter(100)
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	previousRoundOnlyVoter := s.fundedDisputer()
	s.seedTipper(previousRoundOnlyVoter, report)

	// round 1 is voted on (and left unresolved) by the tipper, not the team
	roundOnePayer := s.fundedDisputer()
	s.startNoQuorumRound1(msgServer, report, disputeFee, roundOnePayer, false, previousRoundOnlyVoter)

	// the tipper does not vote in round 2; the team resolves it
	disputer2 := s.fundedDisputer()
	s.proposeRound(msgServer, disputer2, report, disputeFee, false)
	s.voteAndTally(msgServer, 2, teamAddr, types.VoteEnum_VOTE_SUPPORT)
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))

	expectedReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, previousRoundOnlyVoter, 2)
	s.NoError(err)
	s.True(expectedReward.IsPositive(), "the previous-round-only voter should have a real claimable reward")

	queryServer := keeper.NewQuerier(s.Setup.Disputekeeper)
	resp, err := queryServer.ClaimableDisputeRewards(s.Setup.Ctx, &types.QueryClaimableDisputeRewardsRequest{
		DisputeId: 2,
		Address:   previousRoundOnlyVoter.String(),
	})
	s.NoError(err)
	s.Equal(expectedReward, resp.ClaimableAmount.RewardAmount, "the query should report rewards for voters who only participated in previous rounds")
}

// TestWorstCaseClaimableDisputeRewardsShowsCombinedPreviousRoundClaims: one address is
// both the round-1 fee payer and a previous-round-only voter, and the user queries the
// final dispute id. The query must report both the fee refund and the voter reward, and
// both amounts must actually be withdrawable through the transaction paths.
func (s *IntegrationTestSuite) TestWorstCaseClaimableDisputeRewardsShowsCombinedPreviousRoundClaims() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, report, disputeFee := s.setupDisputedReporter(100)
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	// the claimant pays the round-1 fee and is the only round-1 voter
	claimant := s.fundedDisputer()
	s.seedTipper(claimant, report)
	s.startNoQuorumRound1(msgServer, report, disputeFee, claimant, false, claimant)

	roundTwoPayer := s.fundedDisputer()
	s.proposeRound(msgServer, roundTwoPayer, report, disputeFee, false)
	s.voteAndTally(msgServer, 2, teamAddr, types.VoteEnum_VOTE_SUPPORT)
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))

	finalDispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	expectedReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, claimant, 2)
	s.NoError(err)
	s.True(expectedReward.IsPositive(), "the claimant should have a real previous-round-only voter reward")
	expectedRefund, _ := keeper.CalculateRefundAmount(finalDispute.SlashAmount, finalDispute.SlashAmount)
	expectedFeeRefund := expectedRefund.Add(finalDispute.SlashAmount)

	queryServer := keeper.NewQuerier(s.Setup.Disputekeeper)
	resp, err := queryServer.ClaimableDisputeRewards(s.Setup.Ctx, &types.QueryClaimableDisputeRewardsRequest{
		DisputeId: 2,
		Address:   claimant.String(),
	})
	s.NoError(err)
	s.Equal(expectedFeeRefund, resp.ClaimableAmount.FeeRefundAmount, "final-id query must show the round-1 fee refund plus reporter bond reward")
	s.Equal(expectedReward, resp.ClaimableAmount.RewardAmount, "final-id query must show the previous-round-only voter reward")

	// the amounts the query reports are really withdrawable through the tx paths
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: claimant.String(), CallerAddress: claimant.String()})
	s.NoError(err)
	_, err = msgServer.ClaimReward(s.Setup.Ctx, &types.MsgClaimReward{CallerAddress: claimant.String(), DisputeId: 2})
	s.NoError(err)
}
