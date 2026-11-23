package integration_test

// Regressions for two dispute fee economics findings in the PR 1101 review.
//
// Later-round pricing: round n costs max(1, floor(slash*2^(n-1)/20)), capped by the slash, so an
// adjusted slash of 1 to 19 loya still pays a positive fee in rounds 2 to 5. The round-one burn
// allocation stays floor(slash/20), which is zero below 20 loya.
//
// Odd burn allocation: when the round-one allocation is odd and a user or reporter voted,
// ExecuteVote reserves floor(allocation/2) for voters and burns the rest, so no loya of the
// allocation stays unassigned in the dispute module.

import (
	"encoding/hex"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// econRequireLoya compares loya amounts by their decimal string. testify's Equal falls back to
// reflect.DeepEqual on the big.Int inside math.Int, which can tell apart a zero built by NewInt
// and a zero produced by arithmetic.
func (s *IntegrationTestSuite) econRequireLoya(want, got math.Int, msgAndArgs ...any) {
	s.Equal(want.String(), got.String(), msgAndArgs...)
}

func (s *IntegrationTestSuite) econBalance(addr sdk.AccAddress) math.Int {
	return s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, addr, s.Setup.Denom).Amount
}

func (s *IntegrationTestSuite) econSupply() math.Int {
	return s.Setup.Bankkeeper.GetSupply(s.Setup.Ctx, s.Setup.Denom).Amount
}

// econRequireLaterRoundFees opens a major dispute whose reporter has only slash loya left to
// escrow and escalates it through rounds 2 to 5. Each round but the last ends in a real vote and
// tally with no quorum, which leaves the dispute unresolved so the next appeal is allowed.
// wantFees[i] is the fee for round i+2, max(1, floor(slash*2^(round-1)/20)). The coins the payer
// and the dispute module actually move are compared with it, along with the totals recorded on
// the dispute.
func (s *IntegrationTestSuite) econRequireLaterRoundFees(slash int64, wantFees []int64) {
	fx := s.newReportedStake()
	s.leaveReporterSlashable(fx, math.NewInt(slash))
	quote := fx.stake
	payer := sample.AccAddressBytes()
	s.Setup.MintTokens(payer, quote)
	team, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.Require().NoError(err)
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)

	slashAmt := math.NewInt(slash)
	dispute := s.proposeDispute(fx.report, disputetypes.Major, payer, quote, false)
	s.Require().Equal(uint64(1), dispute.DisputeRound)
	s.econRequireLoya(slashAmt, dispute.SlashAmount, "round 1 escrows the whole collectible slash")
	s.econRequireLoya(slashAmt, dispute.FeeTotal, "round 1 keeps only the fee that backs the slash")
	s.econRequireLoya(math.ZeroInt(), dispute.BurnAmount, "round 1 burn allocation is 5%% of %d loya rounded down", slash)

	// The dispute EndBlocker records this snapshot for a dispute opened in the block.
	s.Require().NoError(s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, dispute.HashId))
	s.markRoundUnresolved(msgServer, dispute.DisputeId, team)

	appealFees := math.ZeroInt()
	for i, fee := range wantFees {
		round := uint64(i + 2)
		wantFee := math.NewInt(fee)
		appealFees = appealFees.Add(wantFee)
		payerBefore := s.econBalance(payer)
		moduleBefore := s.moduleBalance()
		supplyBefore := s.econSupply()

		dispute = s.proposeDispute(fx.report, disputetypes.Major, payer, quote, false)

		s.Equal(round, dispute.DisputeRound)
		s.econRequireLoya(wantFee, payerBefore.Sub(s.econBalance(payer)), "round %d fee charged to the payer on a %d loya slash", round, slash)
		s.econRequireLoya(wantFee, s.moduleBalance().Sub(moduleBefore), "round %d fee received by the dispute module", round)
		s.econRequireLoya(supplyBefore, s.econSupply(), "round %d fee is not burned before settlement", round)
		s.econRequireLoya(slashAmt, dispute.SlashAmount, "round %d keeps the slash fixed", round)
		s.econRequireLoya(appealFees, dispute.BurnAmount, "round %d burn allocation is the sum of the appeal fees", round)
		s.econRequireLoya(slashAmt.Add(appealFees), dispute.FeeTotal, "round %d fee total is the round-one fee plus the appeal fees", round)
		if round < 5 {
			s.markRoundUnresolved(msgServer, dispute.DisputeId, team)
		}
	}
}

// TestLaterRoundFeeFloorsAtOneLoyaForOneLoyaSlash: with 1 loya collectible, floor(2^(n-1)/20) is
// zero in every round, so each appeal pays the 1 loya minimum, which is also the slash cap.
func (s *IntegrationTestSuite) TestLaterRoundFeeFloorsAtOneLoyaForOneLoyaSlash() {
	s.econRequireLaterRoundFees(1, []int64{1, 1, 1, 1})
}

// TestLaterRoundFeeScalesBeforeRoundingForFifteenLoyaSlash is the reported case.
// floor(15*2^(n-1)/20) is 1, 3, 6 and 12; round 2 rounds 1.5 down to 1.
func (s *IntegrationTestSuite) TestLaterRoundFeeScalesBeforeRoundingForFifteenLoyaSlash() {
	s.econRequireLaterRoundFees(15, []int64{1, 3, 6, 12})
}

// TestLaterRoundFeeScalesBeforeRoundingForNineteenLoyaSlash: floor(19*2^(n-1)/20) is 1, 3, 7 and
// 15, so rounds 2 to 5 round a fractional price down instead of up.
func (s *IntegrationTestSuite) TestLaterRoundFeeScalesBeforeRoundingForNineteenLoyaSlash() {
	s.econRequireLaterRoundFees(19, []int64{1, 3, 7, 15})
}

func (s *IntegrationTestSuite) TestRoundedBondedAppealDoesNotCommitShortFee() {
	fx := s.newReportedStake()
	// Round two costs 81. Eighty equally-staked selectors can pay only 80.
	s.leaveReporterSlashable(fx, math.NewInt(810))
	payer := sample.AccAddressBytes()
	s.Setup.MintTokens(payer, fx.stake)
	bond := s.receiptRegNewBondPayer(fx.vals[1])
	first := s.proposeDispute(fx.report, disputetypes.Major, payer, fx.stake, false)
	team, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.Require().NoError(err)
	s.Require().NoError(s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, first.HashId))
	s.markRoundUnresolved(keeper.NewMsgServerImpl(s.Setup.Disputekeeper), first.DisputeId, team)
	moduleBefore := s.moduleBalance()
	stakeBefore := s.receiptRegStake(bond.selectors...)
	nextID := s.Setup.Disputekeeper.NextDisputeId(s.Setup.Ctx)

	msg := &disputetypes.MsgProposeDispute{
		Creator:          bond.reporter.String(),
		DisputedReporter: fx.report.Reporter,
		ReportMetaId:     fx.report.MetaId,
		ReportQueryId:    hex.EncodeToString(fx.report.QueryId),
		DisputeCategory:  disputetypes.Major,
		Fee:              sdk.NewCoin(s.Setup.Denom, fx.stake),
		PayFromBond:      true,
	}
	txConfig := s.Setup.NewTxConfig()
	builder := txConfig.NewTxBuilder()
	s.Require().NoError(builder.SetMsgs(msg))
	_, _, err = s.Setup.App.SimDeliver(txConfig.TxEncoder(), builder.GetTx())
	s.Require().ErrorContains(err, "appeal requires 81 loya, stake payment yielded 80")
	s.econRequireLoya(moduleBefore, s.moduleBalance(), "the rejected appeal must not debit any fee")
	s.econRequireLoya(stakeBefore, s.receiptRegStake(bond.selectors...), "the rejected appeal must not unbond any stake")
	s.True(s.receiptRegDispute(first.DisputeId).Open, "the prior round is still appealable")
	_, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, nextID)
	s.ErrorIs(err, collections.ErrNotFound)

	second := s.proposeDispute(fx.report, disputetypes.Major, payer, fx.stake, false)
	s.Equal(nextID, second.DisputeId)
	s.Equal(uint64(2), second.DisputeRound)
	s.econRequireLoya(moduleBefore.AddRaw(81), s.moduleBalance(), "a liquid appeal must transfer the exact fee")
}

// econOddBurn is a major dispute whose reporter had 101 loya left to escrow. The payer covers the
// whole quote, so fee adjustment keeps 101 loya, refunds the rest, and sets the round-one burn
// allocation to floor(101/20) = 5. The tipper holds all user power, so the tipper and the team
// together reach quorum.
type econOddBurn struct {
	fx         reportedStake
	dispute    disputetypes.Dispute
	payer      sdk.AccAddress
	tipper     sdk.AccAddress
	team       sdk.AccAddress
	moduleBase math.Int
}

func (s *IntegrationTestSuite) econOpenOddBurnDispute() econOddBurn {
	fx := s.newReportedStake()
	s.leaveReporterSlashable(fx, math.NewInt(101))
	payer := sample.AccAddressBytes()
	tipper := sample.AccAddressBytes()
	s.Setup.MintTokens(payer, fx.stake)
	s.seedTipper(tipper, fx.report)
	team, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.Require().NoError(err)
	moduleBase := s.moduleBalance()

	dispute := s.proposeDispute(fx.report, disputetypes.Major, payer, fx.stake, false)
	s.Require().Equal(disputetypes.Voting, dispute.DisputeStatus)
	s.econRequireLoya(math.NewInt(101), dispute.SlashAmount, "slash collected from the reporter")
	s.econRequireLoya(math.NewInt(101), dispute.DisputeFee, "fee retained after adjustment")
	s.econRequireLoya(math.NewInt(5), dispute.BurnAmount, "round-one burn allocation is floor(101/20)")
	// The module holds the retained fee and the escrowed slash, 101 loya each.
	s.econRequireLoya(moduleBase.AddRaw(202), s.moduleBalance(), "dispute module balance after funding")
	// The dispute EndBlocker records this snapshot for a dispute opened in the block.
	s.Require().NoError(s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, dispute.HashId))
	return econOddBurn{fx: fx, dispute: dispute, payer: payer, tipper: tipper, team: team, moduleBase: moduleBase}
}

// econOddBurnOutcome is the vote the team and the tipper cast and what each recipient is owed.
// The module holds the 101 loya fee and the 101 loya escrowed slash, 202 in all, of which 5 is the
// burn allocation. The fee refund is 101 less its 5 loya burn share, 96. A reporter who wins gets
// the stake back plus that 96.
type econOddBurnOutcome struct {
	choice disputetypes.VoteEnum
	result disputetypes.VoteResult
	// payerRefund is the fee refund the first-round payer withdraws. Zero means no claim.
	payerRefund int64
	// payerRestake is the slashed stake restaked for the payer when the refund is withdrawn.
	payerRestake int64
	// reporterGain is the stake returned to the disputed reporter when the vote is executed.
	reporterGain int64
}

// econRequireOddBurnFullyAssigned resolves the odd-allocation dispute through real votes and a
// real tally, executes it, and pays every recipient. The 5 loya allocation must split into a voter
// reserve of floor(5/2) = 2 and a burn of the remaining 3, so nothing stays in the module.
func (s *IntegrationTestSuite) econRequireOddBurnFullyAssigned(out econOddBurnOutcome) {
	o := s.econOpenOddBurnDispute()
	id := o.dispute.DisputeId
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)

	// Team power plus every user's tip power is quorum, so the tally resolves the round as soon
	// as both have voted.
	for _, voter := range []sdk.AccAddress{o.team, o.tipper} {
		_, err := msgServer.Vote(s.Setup.Ctx, &disputetypes.MsgVote{Voter: voter.String(), Id: id, Vote: out.choice})
		s.Require().NoError(err)
	}
	tally, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, id)
	s.Require().NoError(err)
	s.Require().Equal(out.result, tally.VoteResult)

	supplyBefore := s.econSupply()
	reporterBefore := s.stakeBalance(o.fx.reporter, o.fx.val)
	s.Require().NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, id))
	executed, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, id)
	s.Require().NoError(err)
	burned := supplyBefore.Sub(s.econSupply())

	s.econRequireLoya(math.NewInt(3), burned, "loya burned at execution: the 5 loya allocation less the voter reserve")
	s.econRequireLoya(math.NewInt(2), executed.VoterReward, "voter reserve: floor(5/2)")
	s.econRequireLoya(executed.BurnAmount, burned.Add(executed.VoterReward), "burned loya plus voter reserve must equal the allocation")
	s.econRequireLoya(math.NewInt(out.reporterGain), s.stakeBalance(o.fx.reporter, o.fx.val).Sub(reporterBefore), "stake returned to the disputed reporter")

	if out.payerRefund > 0 {
		payerBefore := s.econBalance(o.payer)
		bondedBefore, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
		s.Require().NoError(err)
		_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &disputetypes.MsgWithdrawFeeRefund{
			CallerAddress: o.payer.String(),
			PayerAddress:  o.payer.String(),
			Id:            id,
		})
		s.Require().NoError(err)
		bondedAfter, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
		s.Require().NoError(err)
		s.econRequireLoya(math.NewInt(out.payerRefund), s.econBalance(o.payer).Sub(payerBefore), "fee refund withdrawn by the first-round payer")
		s.econRequireLoya(math.NewInt(out.payerRestake), bondedAfter.Sub(bondedBefore), "slashed stake restaked for the first-round payer")
	}

	tipperBefore := s.econBalance(o.tipper)
	_, err = msgServer.ClaimReward(s.Setup.Ctx, &disputetypes.MsgClaimReward{CallerAddress: o.tipper.String(), DisputeId: id})
	s.Require().NoError(err)
	s.econRequireLoya(math.NewInt(2), s.econBalance(o.tipper).Sub(tipperBefore), "voter reward claimed by the tipper")
	s.econRequireLoya(o.moduleBase, s.moduleBalance(), "dispute module balance once every recipient is paid")
}

// TestOddBurnAllocationSupportAssignsEveryLoya settles an odd allocation on SUPPORT. The payer
// withdraws the 96 loya refund and the 101 loya slashed stake; the reporter gets nothing back.
func (s *IntegrationTestSuite) TestOddBurnAllocationSupportAssignsEveryLoya() {
	s.econRequireOddBurnFullyAssigned(econOddBurnOutcome{
		choice:       disputetypes.VoteEnum_VOTE_SUPPORT,
		result:       disputetypes.VoteResult_SUPPORT,
		payerRefund:  96,
		payerRestake: 101,
	})
}

// TestOddBurnAllocationInvalidAssignsEveryLoya settles an odd allocation on INVALID. The reporter
// gets the 101 loya stake back at execution and the payer withdraws the 96 loya refund.
func (s *IntegrationTestSuite) TestOddBurnAllocationInvalidAssignsEveryLoya() {
	s.econRequireOddBurnFullyAssigned(econOddBurnOutcome{
		choice:       disputetypes.VoteEnum_VOTE_INVALID,
		result:       disputetypes.VoteResult_INVALID,
		payerRefund:  96,
		reporterGain: 101,
	})
}

// TestOddBurnAllocationAgainstAssignsEveryLoya settles an odd allocation on AGAINST. The reporter
// gets the 101 loya stake plus 96 loya of the fee at execution and the payer has no claim.
func (s *IntegrationTestSuite) TestOddBurnAllocationAgainstAssignsEveryLoya() {
	s.econRequireOddBurnFullyAssigned(econOddBurnOutcome{
		choice:       disputetypes.VoteEnum_VOTE_AGAINST,
		result:       disputetypes.VoteResult_AGAINST,
		reporterGain: 197,
	})
}
