package integration_test

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracleKeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reporterKeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *IntegrationTestSuite) TestVotingOnDispute() {
	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)

	_, valAddrs, _ := s.createValidatorAccs([]int64{1000})

	repAccs := s.CreateAccountsWithTokens(2, 100*1e6)
	stakeAmount := math.NewInt(100 * 1e6)

	repAcc := repAccs[0]
	valAddr := valAddrs[0]
	delegators := repAccs
	commission := reportertypes.NewCommissionWithTime(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec(), s.ctx.BlockTime())
	_, err := createReporterStakedWithValidator(s.ctx, s.reporterkeeper, s.stakingKeeper, valAddr, delegators, commission, stakeAmount)
	s.NoError(err)

	// assemble report with reporter to dispute
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:  repAcc.String(),
		Power:     100,
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.Unix(1696516597, 0),
	}
	// disputer with tokens to pay fee
	disputer := s.newKeysWithTokens()

	// Propose dispute pay half of the fee from account
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, math.NewInt(500_000)),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)

	// 2 here because dispute count starts from 1 and dispute count gives the next dispute id
	s.Equal(uint64(2), s.disputekeeper.NextDisputeId(s.ctx))
	open, err := s.disputekeeper.GetOpenDisputes(s.ctx)
	s.NoError(err)
	s.Equal(1, len(open))

	// check validator wasn't slashed/jailed
	rep, err := s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)
	// reporter tokens should be the same as the stake amount since fee wasn't fully paid
	s.Equal(rep.TotalTokens, stakeAmount)
	s.False(rep.Jailed)
	// Add dispute fee to complete the fee and jail/slash validator
	_, err = msgServer.AddFeeToDispute(s.ctx, &types.MsgAddFeeToDispute{
		Creator:   disputer.String(),
		DisputeId: 1,
		Amount:    sdk.NewCoin(s.denom, math.NewInt(500_000)),
	})
	s.NoError(err)
	// check reporter was slashed/jailed after fee was added
	rep, err = s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)
	s.Equal(rep.TotalTokens, stakeAmount.Sub(math.NewInt(1_000_000)))
	s.True(rep.Jailed)

	dispute, err := s.disputekeeper.Disputes.Get(s.ctx, 1)
	s.NoError(err)
	s.Equal(types.Voting, dispute.DisputeStatus)
	// vote on dispute
	// mint more tokens to disputer to give voting power
	s.mintTokens(disputer, math.NewInt(1_000_000))
	_, err = msgServer.Vote(s.ctx, &types.MsgVote{
		Voter: disputer.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	})
	s.NoError(err)
	vtr, err := s.disputekeeper.Voter.Get(s.ctx, collections.Join(uint64(1), disputer.Bytes()))
	s.NoError(err)
	s.Equal(types.VoteEnum_VOTE_SUPPORT, vtr.Vote)
	v, err := s.disputekeeper.Votes.Get(s.ctx, 1)
	s.NoError(err)
	s.Equal(v.VoteResult, types.VoteResult_NO_TALLY)
	iter, err := s.disputekeeper.Voter.Indexes.VotersById.MatchExact(s.ctx, uint64(1))
	s.NoError(err)
	voters, err := iter.PrimaryKeys()
	s.NoError(err)
	s.Equal(voters[0].K2(), disputer.Bytes())
}

func (s *IntegrationTestSuite) TestProposeDisputeFromBond() {
	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)

	_, valAddrs, _ := s.createValidatorAccs([]int64{1000})

	repAccs := s.CreateAccountsWithTokens(2, 100*1e6)
	stakeAmount := math.NewInt(100 * 1e6)

	repAcc := repAccs[0]
	valAddr := valAddrs[0]
	delegators := repAccs
	commission := reportertypes.NewCommissionWithTime(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec(), s.ctx.BlockTime())
	_, err := createReporterStakedWithValidator(s.ctx, s.reporterkeeper, s.stakingKeeper, valAddr, delegators, commission, stakeAmount)
	s.NoError(err)

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    repAcc.String(),
		Power:       100,
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: s.ctx.BlockHeight(),
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         repAcc.String(),
		Report:          &report,
		DisputeCategory: types.Warning,
		Fee:             sdk.NewCoin(s.denom, math.NewInt(1_000_000)), // one percent dispute fee
		PayFromBond:     true,
	})
	s.NoError(err)

	// check reporter was slashed/jailed after fee was added
	rep, err := s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)
	s.Equal(rep.TotalTokens, stakeAmount.Sub(math.NewInt(2_000_000))) // two because fee was paid from bond (self dispute) and reporter was slashed
	s.True(rep.Jailed)

	reporterServer := reporterKeeper.NewMsgServerImpl(s.reporterkeeper)
	req := &reportertypes.MsgUnjailReporter{
		ReporterAddress: repAcc.String(),
	}
	_, err = reporterServer.UnjailReporter(s.ctx, req)
	s.NoError(err)
	rep, err = s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)
	s.False(rep.Jailed)
}

func (s *IntegrationTestSuite) TestExecuteVoteInvalid() {
	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)

	_, valAddrs, _ := s.createValidatorAccs([]int64{1000})

	repAccs := s.CreateAccountsWithTokens(3, 100*1e6)
	stakeAmount := math.NewInt(100 * 1e6)
	disputer := s.newKeysWithTokens()

	repAcc := repAccs[0]
	valAddr := valAddrs[0]
	delegators := repAccs
	commission := reportertypes.NewCommissionWithTime(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec(), s.ctx.BlockTime())
	_, err := createReporterStakedWithValidator(s.ctx, s.reporterkeeper, s.stakingKeeper, valAddr, delegators, commission, stakeAmount)
	s.NoError(err)

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    repAcc.String(),
		Power:       100,
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: s.ctx.BlockHeight(),
	}
	disputeFee, err := s.disputekeeper.GetDisputeFee(s.ctx, repAcc.String(), types.Warning)
	s.NoError(err)
	burnAmount := disputeFee.MulRaw(1).QuoRaw(20)
	disputerBalanceBefore := s.bankKeeper.GetBalance(s.ctx, disputer, s.denom)
	// Propose dispute pay half of the fee from account
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	s.True(s.bankKeeper.GetBalance(s.ctx, disputer, s.denom).IsLT(disputerBalanceBefore))

	s.NoError(dispute.CheckPrevoteDisputesForExpiration(s.ctx, s.disputekeeper))

	votes := []types.MsgVote{
		{
			Voter: report.Reporter,
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: disputer.String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: delegators[1].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: delegators[2].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		if err != nil {
			s.Error(err, "voter power is zero")
		}

	}
	// only 25 percent of the total power voted so vote should not be tallied unless it's expired
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	// // tally vote
	err = s.disputekeeper.Tallyvote(s.ctx, 1)
	s.NoError(err)
	reporter, err := s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)

	repTknBeforeExecuteVote := reporter.TotalTokens
	disputerBalanceBeforeExecuteVote := s.bankKeeper.GetBalance(s.ctx, disputer, s.denom)
	// execute vote
	err = s.disputekeeper.ExecuteVote(s.ctx, 1)
	s.NoError(err)
	reporter, err = s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)
	s.True(reporter.TotalTokens.GT(repTknBeforeExecuteVote))
	// // dispute fee returned so balance should be the same as before paying fee
	disputerBalanceAfterExecuteVote := s.bankKeeper.GetBalance(s.ctx, disputer, s.denom)
	iter, err := s.disputekeeper.Voter.Indexes.VotersById.MatchExact(s.ctx, uint64(1))
	s.NoError(err)
	keys, err := iter.PrimaryKeys()
	s.NoError(err)
	voters := make([]keeper.VoterInfo, len(keys))
	disputerInfo := keeper.VoterInfo{Share: math.ZeroInt()}
	totalVoterPower := math.ZeroInt()
	for i := range keys {
		v, err := s.disputekeeper.Voter.Get(s.ctx, keys[i])
		s.NoError(err)
		voters[i] = keeper.VoterInfo{Voter: keys[i].K2(), Power: v.VoterPower}
		totalVoterPower = totalVoterPower.Add(v.VoterPower)
	}
	rewards, _ := s.disputekeeper.CalculateVoterShare(s.ctx, voters, burnAmount.QuoRaw(2), totalVoterPower)
	for i := range rewards {
		if rewards[i].Voter.String() == disputer.String() {
			disputerInfo = rewards[i]
		}
	}
	// // add dispute fee returned minus burn amount plus the voter reward
	disputerBalanceBeforeExecuteVote.Amount = disputerBalanceBeforeExecuteVote.Amount.Add(disputeFee.Sub(burnAmount)).Add(disputerInfo.Share)
	s.Equal(disputerBalanceBeforeExecuteVote, disputerBalanceAfterExecuteVote)
}

func (s *IntegrationTestSuite) TestExecuteVoteNoQuorumInvalid() {
	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)

	_, valAddrs, _ := s.createValidatorAccs([]int64{1000})

	repAccs := s.CreateAccountsWithTokens(3, 100*1e6)
	stakeAmount := math.NewInt(100 * 1e6)
	disputer := s.newKeysWithTokens()

	repAcc := repAccs[0]
	valAddr := valAddrs[0]
	delegators := repAccs
	commission := reportertypes.NewCommissionWithTime(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec(), s.ctx.BlockTime())
	reporter, err := createReporterStakedWithValidator(s.ctx, s.reporterkeeper, s.stakingKeeper, valAddr, delegators, commission, stakeAmount)
	s.NoError(err)

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:  repAcc.String(),
		Power:     100,
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.Unix(1696516597, 0),
	}

	disputeFee, err := s.disputekeeper.GetDisputeFee(s.ctx, report.Reporter, types.Warning)
	s.NoError(err)

	// Propose dispute pay half of the fee from account
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)

	vote := []types.MsgVote{
		{
			Voter: repAcc.String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	// start vote
	_, err = msgServer.Vote(s.ctx, &vote[0])
	s.NoError(err)

	ctx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	err = s.disputekeeper.Tallyvote(ctx, 1)
	s.NoError(err)

	bond := reporter.Amount
	// execute vote
	s.NoError(s.disputekeeper.ExecuteVote(ctx, 1))

	voteInfo, err := s.disputekeeper.Votes.Get(s.ctx, 1)
	s.NoError(err)
	s.Equal(types.VoteResult_NO_QUORUM_MAJORITY_INVALID, voteInfo.VoteResult)
	rep, err := s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)
	s.True(rep.TotalTokens.Equal(bond))
}

func (s *IntegrationTestSuite) TestExecuteVoteSupport() {
	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)

	_, valAddrs, _ := s.createValidatorAccs([]int64{1000})

	repAccs := s.CreateAccountsWithTokens(3, 100*1e6)
	stakeAmount := math.NewInt(100 * 1e6)
	disputer := s.newKeysWithTokens()

	repAcc := repAccs[0]
	valAddr := valAddrs[0]
	delegators := repAccs
	commission := reportertypes.NewCommissionWithTime(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec(), s.ctx.BlockTime())
	reporter, err := createReporterStakedWithValidator(s.ctx, s.reporterkeeper, s.stakingKeeper, valAddr, delegators, commission, stakeAmount)
	s.NoError(err)
	disputerBefore, err := s.stakingKeeper.GetAllDelegatorDelegations(s.ctx, disputer)
	s.NoError(err)
	s.True(len(disputerBefore) == 0)

	// mint tokens to voters
	s.mintTokens(disputer, math.NewInt(100_000_000))
	oracleServer := oracleKeeper.NewMsgServerImpl(s.oraclekeeper)
	msg := oracletypes.MsgTip{
		Tipper:    disputer.String(),
		QueryData: ethQueryData,
		Amount:    sdk.NewCoin(s.denom, math.NewInt(1_000_000)),
	}
	_, err = oracleServer.Tip(s.ctx, &msg)
	s.Nil(err)

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:  repAcc.String(),
		Power:     100,
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.Unix(1696516597, 0),
	}
	disputeFee, err := s.disputekeeper.GetDisputeFee(s.ctx, repAcc.String(), types.Warning)
	s.NoError(err)
	fivePercentBurn := disputeFee.MulRaw(1).QuoRaw(20)
	twoPercentBurn := fivePercentBurn.QuoRaw(2)
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	s.NoError(dispute.CheckPrevoteDisputesForExpiration(s.ctx, s.disputekeeper))

	votersBalanceBefore := map[string]sdk.Coin{
		repAcc.String():        s.bankKeeper.GetBalance(s.ctx, repAcc, s.denom),
		disputer.String():      s.bankKeeper.GetBalance(s.ctx, disputer, s.denom),
		delegators[1].String(): s.bankKeeper.GetBalance(s.ctx, delegators[1], s.denom),
		delegators[2].String(): s.bankKeeper.GetBalance(s.ctx, delegators[2], s.denom),
	}
	votes := []types.MsgVote{
		{
			Voter: repAcc.String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
		{
			Voter: disputer.String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
		{
			Voter: delegators[1].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
		{
			Voter: delegators[2].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		if err != nil {
			s.Error(err, "voter power is zero")
		}
	}
	err = s.disputekeeper.Tallyvote(s.ctx, 1)
	s.NoError(err)
	// execute vote
	s.NoError(s.disputekeeper.ExecuteVote(s.ctx, 1))
	reporterAfter, err := s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)
	s.True(reporterAfter.Jailed)
	s.True(reporterAfter.TotalTokens.LT(reporter.Amount))

	votersBalanceAfter := map[string]sdk.Coin{
		repAcc.String():        s.bankKeeper.GetBalance(s.ctx, repAcc, s.denom),
		disputer.String():      s.bankKeeper.GetBalance(s.ctx, disputer, s.denom),
		delegators[1].String(): s.bankKeeper.GetBalance(s.ctx, delegators[1], s.denom),
		delegators[2].String(): s.bankKeeper.GetBalance(s.ctx, delegators[2], s.denom),
	}

	iter, err := s.disputekeeper.Voter.Indexes.VotersById.MatchExact(s.ctx, uint64(1))
	s.NoError(err)
	keys, err := iter.PrimaryKeys()
	s.NoError(err)
	voters := make([]keeper.VoterInfo, len(keys))
	totalVoterPower := math.ZeroInt()
	for i := range keys {
		v, err := s.disputekeeper.Voter.Get(s.ctx, keys[i])
		s.NoError(err)
		voters[i] = keeper.VoterInfo{Voter: keys[i].K2(), Power: v.VoterPower}
		totalVoterPower = totalVoterPower.Add(v.VoterPower)
	}
	votersReward, _ := s.disputekeeper.CalculateVoterShare(s.ctx, voters, twoPercentBurn, totalVoterPower)
	for i, v := range votersReward {
		// voterBal := votersBalanceBefore[i].Amount.Add(votersReward[addrs[i].String()])
		voterBal := votersBalanceBefore[v.Voter.String()].AddAmount(votersReward[i].Share)
		if bytes.Equal(disputer, votersReward[i].Voter) {
			// disputer gets the dispute fee they paid minus the 5% burn for a one rounder dispute
			voterBal = voterBal.AddAmount(disputeFee.Sub(fivePercentBurn))
		}
		s.Equal(voterBal, votersBalanceAfter[v.Voter.String()])
	}
	disputerDelgation, err := s.stakingKeeper.GetDelegatorBonded(s.ctx, disputer)
	s.NoError(err)
	s.True(disputerDelgation.Equal(math.NewInt(1_000_000)))
}

func (s *IntegrationTestSuite) TestExecuteVoteAgainst() {
	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)

	_, valAddrs, _ := s.createValidatorAccs([]int64{1000})

	repAccs := s.CreateAccountsWithTokens(3, 100*1e6)
	stakeAmount := math.NewInt(100 * 1e6)
	disputer := s.newKeysWithTokens()

	repAcc := repAccs[0]
	valAddr := valAddrs[0]
	delegators := repAccs
	commission := reportertypes.NewCommissionWithTime(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec(), s.ctx.BlockTime())
	reporterBefore, err := createReporterStakedWithValidator(s.ctx, s.reporterkeeper, s.stakingKeeper, valAddr, delegators, commission, stakeAmount)
	s.NoError(err)

	// tip to capture other group of voters 25% of the total power
	s.mintTokens(disputer, math.NewInt(100_000_000))
	oracleServer := oracleKeeper.NewMsgServerImpl(s.oraclekeeper)
	msg := oracletypes.MsgTip{
		Tipper:    disputer.String(),
		QueryData: ethQueryData,
		Amount:    sdk.NewCoin(s.denom, math.NewInt(1_000_000)),
	}
	_, err = oracleServer.Tip(s.ctx, &msg)
	s.Nil(err)

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:  repAcc.String(),
		Power:     100,
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.Unix(1696516597, 0),
	}
	disputeFee, err := s.disputekeeper.GetDisputeFee(s.ctx, repAcc.String(), types.Warning)
	s.NoError(err)

	fivePercentBurn := disputeFee.MulRaw(1).QuoRaw(20)
	twoPercentBurn := fivePercentBurn.QuoRaw(2)
	disputeFeeMinusBurn := disputeFee.Sub(disputeFee.MulRaw(1).QuoRaw(20))

	// Propose dispute pay half of the fee from account
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	votersBalanceBefore := map[string]sdk.Coin{
		repAcc.String():        s.bankKeeper.GetBalance(s.ctx, repAcc, s.denom),
		disputer.String():      s.bankKeeper.GetBalance(s.ctx, disputer, s.denom),
		delegators[1].String(): s.bankKeeper.GetBalance(s.ctx, delegators[1], s.denom),
		delegators[2].String(): s.bankKeeper.GetBalance(s.ctx, delegators[2], s.denom),
	}
	votes := []types.MsgVote{
		{
			Voter: repAcc.String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
		{
			Voter: disputer.String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
		{
			Voter: delegators[1].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
		{
			Voter: delegators[2].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		if err != nil {
			s.Error(err, "voter power is zero")
		}
	}
	// tally vote
	err = s.disputekeeper.Tallyvote(s.ctx, 1)
	s.NoError(err)
	// execute vote
	err = s.disputekeeper.ExecuteVote(s.ctx, 1)
	s.NoError(err)
	reporterAfterDispute, err := s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)

	s.Equal(reporterBefore.Amount.Add(disputeFeeMinusBurn), reporterAfterDispute.TotalTokens)
	votersBalanceAfter := map[string]sdk.Coin{
		repAcc.String():        s.bankKeeper.GetBalance(s.ctx, repAcc, s.denom),
		disputer.String():      s.bankKeeper.GetBalance(s.ctx, disputer, s.denom),
		delegators[1].String(): s.bankKeeper.GetBalance(s.ctx, delegators[1], s.denom),
		delegators[2].String(): s.bankKeeper.GetBalance(s.ctx, delegators[2], s.denom),
	}

	iter, err := s.disputekeeper.Voter.Indexes.VotersById.MatchExact(s.ctx, uint64(1))
	s.NoError(err)
	keys, err := iter.PrimaryKeys()
	s.NoError(err)
	voters := make([]keeper.VoterInfo, len(keys))
	totalVoterPower := math.ZeroInt()
	for i := range keys {
		v, err := s.disputekeeper.Voter.Get(s.ctx, keys[i])
		s.NoError(err)
		voters[i] = keeper.VoterInfo{Voter: keys[i].K2(), Power: v.VoterPower, Share: math.ZeroInt()}
		totalVoterPower = totalVoterPower.Add(v.VoterPower)
	}
	votersReward, _ := s.disputekeeper.CalculateVoterShare(s.ctx, voters, twoPercentBurn, totalVoterPower)

	for _, v := range votersReward {
		newBal := votersBalanceBefore[v.Voter.String()].Amount.Add(v.Share)
		// votersBalanceBefore[votersReward[i].Voter.String()].Amount = votersBalanceBefore[i].Amount.Add(votersReward[i].Share)
		s.Equal(newBal, votersBalanceAfter[v.Voter.String()].Amount)
	}
}

func (s *IntegrationTestSuite) TestDisputeMultipleRounds() {
	reporter1Acc, reporter2Acc := s.Reporters()
	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)
	reporter1, err := s.reporterkeeper.Reporter(s.ctx, reporter1Acc)
	s.NoError(err)
	reporter1StakeBefore := reporter1.TotalTokens
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    sdk.AccAddress(reporter1.Reporter).String(),
		Power:       reporter1.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: s.ctx.BlockHeight(),
	}
	disputeFee, err := s.disputekeeper.GetDisputeFee(s.ctx, report.Reporter, types.Warning)
	s.NoError(err)
	burnAmount := disputeFee.MulRaw(1).QuoRaw(20)
	disputer := sample.AccAddressBytes()
	// mint disputer tokens
	s.mintTokens(disputer, math.NewInt(100_000_000))
	// disputer balance before proposing dispute
	disputerBalanceBefore := s.bankKeeper.GetBalance(s.ctx, disputer, s.denom)
	disputeMsg := types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.ctx, &disputeMsg)
	s.NoError(err)
	// check disputer balance after proposing dispute
	disputerBalanceAfter1stRound := s.bankKeeper.GetBalance(s.ctx, disputer, s.denom)
	s.True(disputerBalanceBefore.Amount.GT(disputerBalanceAfter1stRound.Amount))
	// assert reporter tokens slashed and reporter jailed
	reporter1, err = s.reporterkeeper.Reporter(s.ctx, reporter1Acc)
	s.NoError(err)
	reporter1TokensAfterDispute1stround := reporter1.TotalTokens
	s.True(reporter1.Jailed)
	s.True(reporter1.TotalTokens.LT(reporter1StakeBefore))
	s.Equal(reporter1.TotalTokens, reporter1StakeBefore.Sub(disputeFee))

	voteMsg := types.MsgVote{
		Voter: reporter2Acc.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}
	_, err = msgServer.Vote(s.ctx, &voteMsg)
	s.NoError(err)

	_, err = msgServer.ProposeDispute(s.ctx, &disputeMsg)
	s.Error(err, "can't start a new round for this dispute 1; dispute status DISPUTE_STATUS_VOTING")
	// forward time to end voting period pre execute vote
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS + 1))
	s.NoError(s.disputekeeper.Tallyvote(s.ctx, 1))
	s.ErrorContains(s.disputekeeper.Tallyvote(s.ctx, 1), "vote already tallied")
	s.Error(s.disputekeeper.ExecuteVote(s.ctx, 1), "dispute is not resolved yet")
	// start another dispute round
	_, err = msgServer.ProposeDispute(s.ctx, &disputeMsg)
	s.NoError(err)
	disputerBalanceAfter2ndRound := s.bankKeeper.GetBalance(s.ctx, disputer, s.denom)
	s.Equal(disputerBalanceAfter1stRound.Amount.Sub(burnAmount.MulRaw(2)), disputerBalanceAfter2ndRound.Amount)
	reporter1, err = s.reporterkeeper.Reporter(s.ctx, reporter1Acc)
	s.NoError(err)
	s.Equal(reporter1.TotalTokens, reporter1TokensAfterDispute1stround)
	s.Error(s.disputekeeper.Tallyvote(s.ctx, 2), "vote period not ended and quorum not reached")

	// voting that doesn't reach quorum
	voteMsg = types.MsgVote{
		Voter: reporter2Acc.String(),
		Id:    2,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}

	_, err = msgServer.Vote(s.ctx, &voteMsg)
	s.NoError(err)

	// expire vote period
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	s.NoError(s.disputekeeper.Tallyvote(s.ctx, 2))
	s.NoError(s.disputekeeper.ExecuteVote(s.ctx, 2))
	// attempt to start another round
	_, err = msgServer.ProposeDispute(s.ctx, &disputeMsg)
	s.Error(err, "can't start a new round for this dispute 2; dispute status DISPUTE_STATUS_RESOLVED")
	dispute, err := s.disputekeeper.Disputes.Get(s.ctx, 2)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)
	vote, err := s.disputekeeper.Votes.Get(s.ctx, 2)
	s.NoError(err)
	s.True(vote.Executed)
}

func (s *IntegrationTestSuite) TestNoQorumSingleRound() {
	reporter1Acc, reporter2Acc := s.Reporters()
	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)
	reporter1, err := s.reporterkeeper.Reporter(s.ctx, reporter1Acc)
	s.NoError(err)

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    sdk.AccAddress(reporter1.Reporter).String(),
		Power:       reporter1.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: s.ctx.BlockHeight(),
	}
	disputeFee, err := s.disputekeeper.GetDisputeFee(s.ctx, report.Reporter, types.Warning)
	s.NoError(err)

	disputer := sample.AccAddressBytes()
	// mint disputer tokens
	s.mintTokens(disputer, math.NewInt(100_000_000))

	disputeMsg := types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.ctx, &disputeMsg)
	s.NoError(err)

	voteMsg := types.MsgVote{
		Voter: reporter2Acc.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}

	_, err = msgServer.Vote(s.ctx, &voteMsg)
	s.NoError(err)
	// forward time to expire dispute
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	s.NoError(s.disputekeeper.Tallyvote(s.ctx, 1))
	s.NoError(s.disputekeeper.ExecuteVote(s.ctx, 1))
}

func (s *IntegrationTestSuite) TestDisputeButNoVotes() {
	reporter1Acc, _ := s.Reporters()
	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)
	reporter1, err := s.reporterkeeper.Reporter(s.ctx, reporter1Acc)
	s.NoError(err)

	reporterStakeBefore := reporter1.TotalTokens

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    sdk.AccAddress(reporter1.Reporter).String(),
		Power:       reporter1.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: s.ctx.BlockHeight(),
	}

	disputeFee, err := s.disputekeeper.GetDisputeFee(s.ctx, report.Reporter, types.Warning)
	s.NoError(err)

	disputer := sample.AccAddressBytes()
	// mint disputer tokens
	s.mintTokens(disputer, math.NewInt(100_000_000))

	disputeMsg := types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.ctx, &disputeMsg)
	s.NoError(err)

	reporter1, err = s.reporterkeeper.Reporter(s.ctx, reporter1Acc)
	s.NoError(err)
	s.NotEqual(reporterStakeBefore, reporter1.TotalTokens)

	// forward time to expire dispute
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.THREE_DAYS + 1))

	s.NoError(s.disputekeeper.Tallyvote(s.ctx, 1))
	s.NoError(s.disputekeeper.ExecuteVote(s.ctx, 1))

	reporter1, err = s.reporterkeeper.Reporter(s.ctx, reporter1Acc)
	s.NoError(err)
	s.Equal(reporterStakeBefore, reporter1.TotalTokens)
}
