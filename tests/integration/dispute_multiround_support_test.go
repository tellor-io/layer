package integration_test

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

// setupDisputedReporter registers a validator-reporter and stores a disputable report.
func (s *IntegrationTestSuite) setupDisputedReporter(power uint64) (sdk.AccAddress, oracletypes.MicroReport, math.Int) {
	repAccs, _, _ := s.createValidatorAccs([]uint64{power})
	reporterAcc := repAccs[0]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporterAcc, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporterAcc, reportertypes.NewSelection(reporterAcc, 1)))

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	stake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterAcc, qId)
	s.NoError(err)
	report := oracletypes.MicroReport{
		Reporter:    reporterAcc.String(),
		Power:       stake.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Now().Add(-1 * 12 * time.Hour),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, reporterAcc.Bytes(), report.MetaId), report))
	fee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)
	return reporterAcc, report, fee
}

// startNoQuorumRound1 proposes a dispute (round 1, fully funded by disputer1), gives it only the
// team vote so it ends Unresolved, and returns once a new round can be added.
func (s *IntegrationTestSuite) startNoQuorumRound1(msgServer types.MsgServer, report oracletypes.MicroReport, disputeFee math.Int, disputer1, teamAddr sdk.AccAddress) {
	disputeMsg := types.MsgProposeDispute{
		Creator:          disputer1.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
	}
	_, err := msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)
	d1, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.NoError(s.Setup.Disputekeeper.BlockInfo.Set(s.Setup.Ctx, d1.HashId, types.BlockInfo{TotalReporterPower: math.NewInt(int64(report.Power)).Mul(sdk.DefaultPowerReduction), TotalUserTips: math.NewInt(100)}))
	_, err = msgServer.Vote(s.Setup.Ctx, &types.MsgVote{Voter: teamAddr.String(), Id: 1, Vote: types.VoteEnum_VOTE_INVALID})
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.TWO_DAYS + 1))
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1))
}

// TestMultiRoundSupportRefundsRoundOneStakeOnly on a multi-round SUPPORT,
// the round-1 fee payer is refunded their round-1 stake (95%) and receives the full
// slashed bond, while a later-round fee payer gets nothing. their escalating round fee was a
// burn and not a refundable stake.
func (s *IntegrationTestSuite) TestMultiRoundSupportRefundsRoundOneStakeOnly() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, report, disputeFee := s.setupDisputedReporter(100)
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	disputer1 := s.newKeysWithTokens()
	s.Setup.MintTokens(disputer1, math.NewInt(100_000_000))
	s.startNoQuorumRound1(msgServer, report, disputeFee, disputer1, teamAddr)

	// round 2 funded by a different payer, then resolves SUPPORT
	disputer2 := s.newKeysWithTokens()
	s.Setup.MintTokens(disputer2, math.NewInt(100_000_000))
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          disputer2.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
	})
	s.NoError(err)
	_, err = msgServer.Vote(s.Setup.Ctx, &types.MsgVote{Voter: teamAddr.String(), Id: 2, Vote: types.VoteEnum_VOTE_SUPPORT})
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 2))
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))

	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)
	slashAmount := dispute.SlashAmount

	// the round-2 fee was not added to a refundable stake:
	// BurnAmount = round-1's 5% (SlashAmount/20) + the entire round-2 fee (SlashAmount/10).
	roundFee2 := slashAmount.QuoRaw(10)
	s.Equal(slashAmount.QuoRaw(20).Add(roundFee2), dispute.BurnAmount)

	// round-1 payer: refunded 95% of their round-1 stake + the full slashed bond
	expRefund, _ := keeper.CalculateRefundAmount(slashAmount, slashAmount)
	bondedBefore, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.NoError(err)
	bal1Before := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer1, s.Setup.Denom)
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer1.String(), CallerAddress: disputer1.String()})
	s.NoError(err)
	s.Equal(expRefund, s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer1, s.Setup.Denom).Amount.Sub(bal1Before.Amount))
	bondedAfter, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.NoError(err)
	s.Equal(slashAmount, bondedAfter.Sub(bondedBefore)) // entire slashed bond returned to the round-1 payer

	// round-2 payer should not receive round 2 fee back
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer2.String(), CallerAddress: disputer2.String()})
	s.Error(err)

	fmt.Printf("Slash=%s round1Refund=%s bondToRd1Payer=%s burn=%s\n", slashAmount, expRefund, slashAmount, dispute.BurnAmount)
}

// TestMultiRoundSupportConservesWithVoterClaim:
// the round-1 payer claims (fee refund + bond), the round-2 payer gets nothing, and the voters
// get the voter reward, which should be 2.5% percent of round-1's fee + half of round-2's fee.
// The dispute module should not have a balance.
func (s *IntegrationTestSuite) TestMultiRoundSupportConservesWithVoterClaim() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)

	repAccs, _, _ := s.createValidatorAccs([]uint64{100, 10})
	reporterAcc := repAccs[0]
	tipper := repAccs[1]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporterAcc, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporterAcc, reportertypes.NewSelection(reporterAcc, 1)))

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	stake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterAcc, qId)
	s.NoError(err)
	report := oracletypes.MicroReport{
		Reporter:    reporterAcc.String(),
		Power:       stake.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Now().Add(-1 * 12 * time.Hour),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, reporterAcc.Bytes(), report.MetaId), report))
	s.NoError(s.Setup.Oraclekeeper.TipperTotal.Set(s.Setup.Ctx, collections.Join(tipper.Bytes(), report.BlockNumber), math.NewInt(100)))
	s.NoError(s.Setup.Oraclekeeper.TotalTips.Set(s.Setup.Ctx, report.BlockNumber, math.NewInt(100)))

	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	disputer1 := s.newKeysWithTokens()
	s.Setup.MintTokens(disputer1, math.NewInt(100_000_000))
	s.startNoQuorumRound1(msgServer, report, disputeFee, disputer1, teamAddr)

	disputer2 := s.newKeysWithTokens()
	s.Setup.MintTokens(disputer2, math.NewInt(100_000_000))
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          disputer2.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
	})
	s.NoError(err)
	_, err = msgServer.Vote(s.Setup.Ctx, &types.MsgVote{Voter: teamAddr.String(), Id: 2, Vote: types.VoteEnum_VOTE_SUPPORT})
	s.NoError(err)
	_, err = msgServer.Vote(s.Setup.Ctx, &types.MsgVote{Voter: tipper.String(), Id: 2, Vote: types.VoteEnum_VOTE_SUPPORT})
	s.NoError(err)

	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)
	if dispute.PendingExecution {
		s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))
	}
	vote, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.VoteResult_SUPPORT, vote.VoteResult)
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)

	// the voter reward includes the round-2 fees voter half
	s.Equal(dispute.BurnAmount.QuoRaw(2), dispute.VoterReward)
	s.True(dispute.VoterReward.GT(dispute.SlashAmount.QuoRaw(40))) // more than round-1's burn alone

	disputeModAddr := authtypes.NewModuleAddress(types.ModuleName)

	// round-1 payer claims refund + bond
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer1.String(), CallerAddress: disputer1.String()})
	s.NoError(err)
	// round-2 payer is not refunded the fee.
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer2.String(), CallerAddress: disputer2.String()})
	s.Error(err)
	// voter claims the voter reward
	_, err = msgServer.ClaimReward(s.Setup.Ctx, &types.MsgClaimReward{CallerAddress: tipper.String(), DisputeId: 2})
	s.NoError(err)

	modFinal := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputeModAddr, s.Setup.Denom)
	fmt.Printf("voterReward=%s | dispute module balance after all claims: %s\n", dispute.VoterReward, modFinal.Amount)
	s.True(modFinal.Amount.LTE(math.NewInt(1)), "dispute module should net to ~zero, has %s", modFinal.Amount)
}

// TestMultiRoundAgainstReporterGetsRoundOneStakeOnly:
// when a multi-round dispute resolves AGAINST, the reporter is awarded their bond back plus 95% of
// the round-1 fee and not the later-round fees, which were burned. Disputers get nothing.
func (s *IntegrationTestSuite) TestMultiRoundAgainstReporterGetsRoundOneStakeOnly() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, report, disputeFee := s.setupDisputedReporter(100)
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)

	disputer1 := s.newKeysWithTokens()
	s.Setup.MintTokens(disputer1, math.NewInt(100_000_000))
	s.startNoQuorumRound1(msgServer, report, disputeFee, disputer1, teamAddr)

	// round 2 funded by a different payer.
	disputer2 := s.newKeysWithTokens()
	s.Setup.MintTokens(disputer2, math.NewInt(100_000_000))
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          disputer2.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
	})
	s.NoError(err)
	_, err = msgServer.Vote(s.Setup.Ctx, &types.MsgVote{Voter: teamAddr.String(), Id: 2, Vote: types.VoteEnum_VOTE_AGAINST})
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 2))

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

	// reporter gets bond back + 95% of the round-1 fee.
	expectedReporterGain := dispute.SlashAmount.Add(dispute.DisputeFee.Sub(dispute.DisputeFee.QuoRaw(20)))
	s.Equal(expectedReporterGain, bondedAfter.Sub(bondedBefore))

	s.Equal(dispute.VoterReward, s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputeModAddr, s.Setup.Denom).Amount)

	// disputers get nothing on AGAINST
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer1.String(), CallerAddress: disputer1.String()})
	s.Error(err)
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{Id: 2, PayerAddress: disputer2.String(), CallerAddress: disputer2.String()})
	s.Error(err)
}
