package integration_test

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracleKeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reporterKeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"
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
	report := oracletypes.MicroReport{
		Reporter:  repAcc.String(),
		Power:     100,
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
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
	s.Equal(uint64(2), s.disputekeeper.GetDisputeCount(s.ctx))
	open, err := s.disputekeeper.GetOpenDisputeIds(s.ctx)
	s.NoError(err)
	s.Equal(1, len(open.Ids))

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

	dispute, err := s.disputekeeper.GetDisputeById(s.ctx, 1)
	s.NoError(err)
	s.Equal(types.Voting, dispute.DisputeStatus)
	// vote on dispute
	_, err = msgServer.Vote(s.ctx, &types.MsgVote{
		Voter: disputer.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	})
	s.NoError(err)
	vtr, err := s.disputekeeper.GetVoterVote(s.ctx, disputer.String(), 1)
	s.NoError(err)
	s.Equal(types.VoteEnum_VOTE_SUPPORT, vtr.Vote)
	v, err := s.disputekeeper.GetVote(s.ctx, 1)
	s.NoError(err)
	s.Equal(v.VoteResult, types.VoteResult_NO_TALLY)
	s.Equal(v.Voters, []string{disputer.String()})
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

	report := oracletypes.MicroReport{
		Reporter:    repAcc.String(),
		Power:       100,
		QueryId:     "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
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

	report := oracletypes.MicroReport{
		Reporter:  repAcc.String(),
		Power:     100,
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.Unix(1696516597, 0),
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

	// start vote
	ids, err := s.disputekeeper.CheckPrevoteDisputesForExpiration(s.ctx)
	s.NoError(err)

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
		s.NoError(err)
	}
	// only 25 percent of the total power voted so vote should not be tallied unless it's expired
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	// // tally vote
	err = s.disputekeeper.TallyVote(s.ctx, ids[0])
	s.NoError(err)
	reporter, err := s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)

	repTknBeforeExecuteVote := reporter.TotalTokens
	disputerBalanceBeforeExecuteVote := s.bankKeeper.GetBalance(s.ctx, disputer, s.denom)
	// execute vote
	err = s.disputekeeper.ExecuteVotes(s.ctx, ids)
	s.NoError(err)
	reporter, err = s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)
	s.True(reporter.TotalTokens.GT(repTknBeforeExecuteVote))
	// // dispute fee returned so balance should be the same as before paying fee
	disputerBalanceAfterExecuteVote := s.bankKeeper.GetBalance(s.ctx, disputer, s.denom)
	v, err := s.disputekeeper.GetVote(s.ctx, 1)
	s.NoError(err)
	rewards, _ := s.disputekeeper.CalculateVoterShare(s.ctx, v.Voters, burnAmount.QuoRaw(2))
	voterReward := rewards[disputer.String()]
	// // add dispute fee returned minus burn amount plus the voter reward
	disputerBalanceBeforeExecuteVote.Amount = disputerBalanceBeforeExecuteVote.Amount.Add(disputeFee.Sub(burnAmount)).Add(voterReward)
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

	report := oracletypes.MicroReport{
		Reporter:  repAcc.String(),
		Power:     100,
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
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
	err = s.disputekeeper.TallyVote(ctx, 1)
	s.NoError(err)

	bond := reporter.Amount
	// // execute vote
	err = s.disputekeeper.ExecuteVotes(ctx, []uint64{1})
	s.NoError(err)

	voteInfo, err := s.disputekeeper.GetVote(ctx, 1)
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
	s.mintTokens(disputer, sdk.NewCoin(s.denom, math.NewInt(100_000_000)))
	oracleServer := oracleKeeper.NewMsgServerImpl(s.oraclekeeper)
	msg := oracletypes.MsgTip{
		Tipper:    disputer.String(),
		QueryData: ethQueryData,
		Amount:    sdk.NewCoin(s.denom, math.NewInt(1_000_000)),
	}
	_, err = oracleServer.Tip(s.ctx, &msg)
	s.Nil(err)
	report := oracletypes.MicroReport{
		Reporter:  repAcc.String(),
		Power:     100,
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
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
	// start vote
	ids, err := s.disputekeeper.CheckPrevoteDisputesForExpiration(s.ctx)
	s.NoError(err)

	votersBalanceBefore := []sdk.Coin{
		s.bankKeeper.GetBalance(s.ctx, repAcc, s.denom),
		s.bankKeeper.GetBalance(s.ctx, disputer, s.denom),
		s.bankKeeper.GetBalance(s.ctx, delegators[1], s.denom),
		s.bankKeeper.GetBalance(s.ctx, delegators[2], s.denom),
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
		s.NoError(err)
	}
	// tally vote
	err = s.disputekeeper.Tally(s.ctx, ids)
	s.NoError(err)
	// execute vote
	err = s.disputekeeper.ExecuteVotes(s.ctx, ids)
	s.NoError(err)
	reporterAfter, err := s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)
	s.True(reporterAfter.Jailed)
	s.True(reporterAfter.TotalTokens.LT(reporter.Amount))

	votersBalanceAfter := []sdk.Coin{
		s.bankKeeper.GetBalance(s.ctx, repAcc, s.denom),
		s.bankKeeper.GetBalance(s.ctx, disputer, s.denom),
		s.bankKeeper.GetBalance(s.ctx, delegators[1], s.denom),
		s.bankKeeper.GetBalance(s.ctx, delegators[2], s.denom),
	}
	v, err := s.disputekeeper.GetVote(s.ctx, 1)
	s.NoError(err)

	addrs := []sdk.AccAddress{repAcc, disputer, delegators[1], delegators[2]}
	votersReward, _ := s.disputekeeper.CalculateVoterShare(s.ctx, v.Voters, twoPercentBurn)
	for i := range votersBalanceBefore {
		votersBalanceBefore[i].Amount = votersBalanceBefore[i].Amount.Add(votersReward[addrs[i].String()])
		s.Equal(votersBalanceBefore[i], (votersBalanceAfter[i]))
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
	s.mintTokens(disputer, sdk.NewCoin(s.denom, math.NewInt(100_000_000)))
	oracleServer := oracleKeeper.NewMsgServerImpl(s.oraclekeeper)
	msg := oracletypes.MsgTip{
		Tipper:    disputer.String(),
		QueryData: ethQueryData,
		Amount:    sdk.NewCoin(s.denom, math.NewInt(1_000_000)),
	}
	_, err = oracleServer.Tip(s.ctx, &msg)
	s.Nil(err)

	report := oracletypes.MicroReport{
		Reporter:  repAcc.String(),
		Power:     100,
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
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

	addrs := []sdk.AccAddress{repAcc, disputer, delegators[1], delegators[2]}

	votersBalanceBefore := []sdk.Coin{
		s.bankKeeper.GetBalance(s.ctx, repAcc, s.denom),
		s.bankKeeper.GetBalance(s.ctx, disputer, s.denom),
		s.bankKeeper.GetBalance(s.ctx, delegators[1], s.denom),
		s.bankKeeper.GetBalance(s.ctx, delegators[2], s.denom),
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
		s.NoError(err)
	}
	// tally vote
	err = s.disputekeeper.TallyVote(s.ctx, 1)
	s.NoError(err)
	// execute vote
	err = s.disputekeeper.ExecuteVote(s.ctx, 1)
	s.NoError(err)
	reporterAfterDispute, err := s.reporterkeeper.Reporter(s.ctx, repAcc)
	s.NoError(err)

	s.Equal(reporterBefore.Amount.Add(disputeFeeMinusBurn), reporterAfterDispute.TotalTokens)

	votersBalanceAfter := []sdk.Coin{
		s.bankKeeper.GetBalance(s.ctx, repAcc, s.denom),
		s.bankKeeper.GetBalance(s.ctx, disputer, s.denom),
		s.bankKeeper.GetBalance(s.ctx, delegators[1], s.denom),
		s.bankKeeper.GetBalance(s.ctx, delegators[2], s.denom),
	}
	v, err := s.disputekeeper.GetVote(s.ctx, 1)
	s.NoError(err)
	votersReward, _ := s.disputekeeper.CalculateVoterShare(s.ctx, v.Voters, twoPercentBurn)
	for i := range votersBalanceBefore {
		votersBalanceBefore[i].Amount = votersBalanceBefore[i].Amount.Add(votersReward[addrs[i].String()])
		s.Equal(votersBalanceBefore[i], (votersBalanceAfter[i]))
	}
}

// func (s *IntegrationTestSuite) TestDisputeMultipleRounds() {
// 	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)
// 	addrs, valAddrs := s.createValidators([]int64{100, 200, 300})

// 	bal0, err := s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[0], s.denom))
// 	s.NoError(err)
// 	s.Equal(bal0, math.NewInt(900_000_000))

// 	bal1, err := s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[1], s.denom))
// 	s.NoError(err)
// 	s.Equal(bal1, math.NewInt(800_000_000))

// 	bal2, err := s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[2], s.denom))
// 	s.NoError(err)
// 	s.Equal(bal2, math.NewInt(700_000_000))

// 	balStaking, err := s.bankKeeper.Balances.Get(s.ctx, collections.Join(s.stakingKeeper.GetBondedPool(s.ctx).GetAddress(), s.denom))
// 	s.NoError(err)
// 	s.Equal(balStaking, math.NewInt(601_000_000))

// 	reporter, err := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	reporterStakeBefore := reporter.GetBondedTokens()
// 	report := types.MicroReport{
// 		Reporter:  addrs[0].String(),
// 		Power:     reporter.GetConsensusPower(sdk.DefaultPowerReduction),
// 		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
// 		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
// 		Timestamp: 1696516597,
// 	}
// 	disputeFee, err := s.disputekeeper.GetDisputeFee(s.ctx, report.Reporter, types.Warning)
// 	s.NoError(err)
// 	burnAmount := disputeFee.MulRaw(1).QuoRaw(20)
// 	dispute := types.MsgProposeDispute{
// 		Creator:         addrs[1].String(),
// 		Report:          &report,
// 		Fee:             sdk.NewCoin(s.denom, disputeFee),
// 		DisputeCategory: types.Warning,
// 	}
// 	balanceBefore := s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	// Propose dispute pay half of the fee from account
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.NoError(err)

// 	bal0, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[0], s.denom))
// 	s.NoError(err)
// 	s.Equal(bal0, math.NewInt(900_000_000))

// 	bal1, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[1], s.denom))
// 	s.NoError(err)
// 	s.Equal(bal1, math.NewInt(799_000_000))

// 	bal2, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[2], s.denom))
// 	s.NoError(err)
// 	s.Equal(bal2, math.NewInt(700_000_000))

// 	balStaking, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(s.stakingKeeper.GetBondedPool(s.ctx).GetAddress(), s.denom))
// 	s.NoError(err)
// 	s.Equal(balStaking, math.NewInt(600_000_000))

// 	moduleAccs := s.ModuleAccs()
// 	balDispute, err := s.bankKeeper.Balances.Get(s.ctx, collections.Join(moduleAccs.dispute.GetAddress(), s.denom))
// 	s.NoError(err)
// 	s.Equal(balDispute, disputeFee.MulRaw(2)) // disputeFee + slashAmount

// 	balanceAfter := s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	s.True(balanceBefore.Amount.Sub(disputeFee).Equal(balanceAfter.Amount))
// 	// check reporter stake
// 	reporter, err = s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	s.True(reporter.GetBondedTokens().LT(reporterStakeBefore))
// 	s.Equal(reporter.GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
// 	// begin block
// 	header := tmproto.Header{Height: s.app.LastBlockHeight() + 1, Time: s.ctx.BlockTime()}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	votes := []types.MsgVote{
// 		{
// 			Voter: addrs[0].String(),
// 			Id:    1,
// 			Vote:  types.VoteEnum_VOTE_INVALID,
// 		},
// 	}
// 	for i := range votes {
// 		_, err = msgServer.Vote(s.ctx, &votes[i])
// 		s.NoError(err)
// 	}

// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime()}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.Equal(err.Error(), "can't start a new round for this dispute 1; dispute status DISPUTE_STATUS_VOTING")
// 	// check reporter stake
// 	reporter, err = s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	s.True(reporter.GetBondedTokens().LT(reporterStakeBefore))
// 	s.Equal(reporter.GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
// 	// forward time to after vote end
// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.NoError(err)
// 	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, burnAmount.MulRaw(2))), balanceAfter)

// 	// check reporter stake
// 	reporter, err = s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	s.True(reporter.GetBondedTokens().LT(reporterStakeBefore)) // TODO: this double-check seems unnecessary
// 	s.Equal(reporter.GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	// voting that doesn't reach quorum
// 	votes = []types.MsgVote{
// 		{
// 			Voter: addrs[0].String(),
// 			Id:    2,
// 			Vote:  types.VoteEnum_VOTE_INVALID,
// 		},
// 	}
// 	for i := range votes {
// 		_, err = msgServer.Vote(s.ctx, &votes[i])
// 		s.NoError(err)
// 	}
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.NoError(err)

// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.NoError(err)
// 	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, burnAmount.MulRaw(4))), balanceAfter)

// 	// check reporter stake
// 	reporter, err = s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	s.True(reporter.GetBondedTokens().LT(reporterStakeBefore))
// 	s.Equal(reporter.GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	// voting that doesn't reach quorum
// 	votes = []types.MsgVote{
// 		{
// 			Voter: addrs[0].String(),
// 			Id:    3,
// 			Vote:  types.VoteEnum_VOTE_INVALID,
// 		},
// 	}
// 	for i := range votes {
// 		_, err = msgServer.Vote(s.ctx, &votes[i])
// 		s.NoError(err)
// 	}
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.NoError(err)

// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.NoError(err)
// 	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, burnAmount.MulRaw(8))), balanceAfter)

// 	// check reporter stake
// 	reporter, err = s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	s.True(reporter.GetBondedTokens().LT(reporterStakeBefore))
// 	s.Equal(reporter.GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	// voting that doesn't reach quorum
// 	votes = []types.MsgVote{
// 		{
// 			Voter: addrs[0].String(),
// 			Id:    4,
// 			Vote:  types.VoteEnum_VOTE_INVALID,
// 		},
// 	}
// 	for i := range votes {
// 		_, err = msgServer.Vote(s.ctx, &votes[i])
// 		s.NoError(err)
// 	}
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.Equal(err.Error(), "can't start a new round for this dispute 4; dispute status DISPUTE_STATUS_VOTING")
// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.NoError(err)
// 	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, burnAmount.MulRaw(16))), balanceAfter)

// 	// check reporter stake
// 	reporter, err = s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	s.True(reporter.GetBondedTokens().LT(reporterStakeBefore))
// 	s.Equal(reporter.GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	// voting that doesn't reach quorum
// 	votes = []types.MsgVote{
// 		{
// 			Voter: addrs[0].String(),
// 			Id:    5,
// 			Vote:  types.VoteEnum_VOTE_INVALID,
// 		},
// 	}
// 	for i := range votes {
// 		_, err = msgServer.Vote(s.ctx, &votes[i])
// 		s.NoError(err)
// 	}
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)

// 	// forward time to end vote
// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.NoError(err)
// 	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, disputeFee)), balanceAfter)

// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.NoError(err)

// 	bal0, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[0], s.denom))
// 	s.NoError(err)
// 	s.Equal(bal0, math.NewInt(900_000_000))

// 	bal1, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[1], s.denom))
// 	s.NoError(err)
// 	s.Equal(bal1, math.NewInt(795_500_000))

// 	bal2, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[2], s.denom))
// 	s.NoError(err)
// 	s.Equal(bal2, math.NewInt(700_000_000))

// 	balStaking, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(s.stakingKeeper.GetBondedPool(s.ctx).GetAddress(), s.denom))
// 	s.NoError(err)
// 	s.Equal(balStaking, math.NewInt(600_000_000))

// 	balDispute, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(moduleAccs.dispute.GetAddress(), s.denom))
// 	s.NoError(err)
// 	s.Equal(balDispute, math.NewInt(5_500_000)) // disputeFee + slashAmount + round 1(100000) + round 2(200000) + round 3(400000) + round 4(800000) + round 5(1000000) + round 6(1000000)

// 	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, disputeFee)), balanceAfter)
// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
// 	s.NoError(err)
// 	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
// 	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, disputeFee)), balanceAfter)
// 	// forward time and end dispute
// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.THREE_DAYS))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	// check reporter stake, stake should be restored due to invalid vote final result
// 	reporter, err = s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	s.Equal(reporter.GetBondedTokens(), reporterStakeBefore)

// 	bal0, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[0], s.denom)) // voter reward half the total burn Amount(the 5%)
// 	s.NoError(err)
// 	s.Equal(bal0, math.NewInt(902_275_000))

// 	bal1, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[1], s.denom))
// 	s.NoError(err)
// 	s.Equal(bal1, math.NewInt(795_450_000))

// 	bal2, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(addrs[2], s.denom))
// 	s.NoError(err)
// 	s.Equal(bal2, math.NewInt(700_000_000))

// 	balStaking, err = s.bankKeeper.Balances.Get(s.ctx, collections.Join(s.stakingKeeper.GetBondedPool(s.ctx).GetAddress(), s.denom))
// 	s.NoError(err)
// 	s.Equal(balStaking, math.NewInt(601_000_000))
// }

// func (s *IntegrationTestSuite) TestNoQorumSingleRound() {
// 	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)
// 	addrs, valAddrs := s.createValidators([]int64{100, 200, 300})
// 	reporter, err := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	reporterStakeBefore := reporter.GetBondedTokens()
// 	report := types.MicroReport{
// 		Reporter:  addrs[0].String(),
// 		Power:     reporter.GetConsensusPower(sdk.DefaultPowerReduction),
// 		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
// 		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
// 		Timestamp: 1696516597,
// 	}
// 	disputeFee, err := s.disputekeeper.GetDisputeFee(s.ctx, report.Reporter, types.Warning)
// 	s.NoError(err)
// 	// Propose dispute pay half of the fee from account
// 	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
// 		Creator:         addrs[1].String(),
// 		Report:          &report,
// 		Fee:             sdk.NewCoin(s.denom, disputeFee),
// 		DisputeCategory: types.Warning,
// 	})
// 	s.NoError(err)
// 	header := tmproto.Header{Height: s.app.LastBlockHeight() + 1, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	votes := []types.MsgVote{
// 		{
// 			Voter: addrs[0].String(),
// 			Id:    1,
// 			Vote:  types.VoteEnum_VOTE_INVALID,
// 		},
// 		{
// 			Voter: addrs[1].String(),
// 			Id:    1,
// 			Vote:  types.VoteEnum_VOTE_INVALID,
// 		},
// 	}

// 	for i := range votes {
// 		_, err = msgServer.Vote(s.ctx, &votes[i])
// 		s.NoError(err)
// 	}
// 	// forward time to expire dispute
// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*3 + 1))
// 	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)

// 	reporter, err = s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	reporterStakeAfter := reporter.GetBondedTokens()
// 	// reporter stake should be restored after dispute expires for invalid vote
// 	s.Equal(reporterStakeBefore, reporterStakeAfter)
// }

// func (s *IntegrationTestSuite) TestDisputeButNoVotes() {
// 	msgServer := keeper.NewMsgServerImpl(s.disputekeeper)
// 	addrs, valAddrs := s.createValidators([]int64{100, 200, 300})
// 	reporter, err := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	reporterStakeBefore := reporter.GetBondedTokens()
// 	report := types.MicroReport{
// 		Reporter:  addrs[0].String(),
// 		Power:     reporter.GetConsensusPower(sdk.DefaultPowerReduction),
// 		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
// 		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
// 		Timestamp: 1696516597,
// 	}
// 	disputeFee, err := s.disputekeeper.GetDisputeFee(s.ctx, report.Reporter, types.Warning)
// 	s.NoError(err)
// 	// Propose dispute pay half of the fee from account
// 	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
// 		Creator:         addrs[1].String(),
// 		Report:          &report,
// 		Fee:             sdk.NewCoin(s.denom, disputeFee),
// 		DisputeCategory: types.Warning,
// 	})
// 	s.NoError(err)

// 	reporter, err = s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	s.NotEqual(reporterStakeBefore, reporter.GetBondedTokens())
// 	// forward time to end dispute
// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.THREE_DAYS))
// 	header := tmproto.Header{Height: s.app.LastBlockHeight() + 1, Time: s.ctx.BlockTime().Add(1)}
// 	s.ctx = s.ctx.WithBlockHeader(header)
// 	_, err = s.app.BeginBlocker(s.ctx)
// 	s.NoError(err)
// 	reporter, err = s.stakingKeeper.Validator(s.ctx, valAddrs[0])
// 	s.NoError(err)
// 	s.Equal(reporterStakeBefore, reporter.GetBondedTokens())
// }
