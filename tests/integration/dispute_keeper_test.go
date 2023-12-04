package integration_test

import (
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (s *IntegrationTestSuite) disputeKeeper() (queryClient types.QueryClient, msgServer types.MsgServer) {
	types.RegisterQueryServer(s.queryHelper, s.disputekeeper)
	types.RegisterInterfaces(s.interfaceRegistry)
	queryClient = types.NewQueryClient(s.queryHelper)
	msgServer = keeper.NewMsgServerImpl(s.disputekeeper)

	return
}

func (s *IntegrationTestSuite) TestVotingOnDispute() {
	_, msgServer := s.disputeKeeper()
	require := s.Require()
	ctx := s.ctx
	k := s.disputekeeper
	Addr := s.newKeysWithTokens()

	report, valAddr := s.microReport()
	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(ctx, &types.MsgProposeDispute{
		Creator:         Addr.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, sdk.NewInt(5000)),
		DisputeCategory: types.Warning,
	})
	require.Equal(uint64(1), k.GetDisputeCount(ctx))
	require.Equal(1, len(k.GetOpenDisputeIds(ctx).Ids))
	require.NoError(err)
	// check validator wasn't slashed/jailed
	val, found := s.stakingKeeper.GetValidator(ctx, valAddr)
	bondedTokensBefore := val.GetBondedTokens()
	require.True(found)
	require.False(val.IsJailed())
	require.Equal(bondedTokensBefore, sdk.NewInt(1000000))
	// Add dispute fee to complete the fee and jail/slash validator
	_, err = msgServer.AddFeeToDispute(ctx, &types.MsgAddFeeToDispute{
		Creator:   Addr.String(),
		DisputeId: 0,
		Amount:    sdk.NewCoin(s.denom, sdk.NewInt(5000)),
	})
	require.NoError(err)
	// check validator was slashed/jailed
	val, found = s.stakingKeeper.GetValidator(ctx, valAddr)
	require.True(found)
	require.True(val.IsJailed())
	// check validator was slashed 1% of tokens
	require.Equal(val.GetBondedTokens(), bondedTokensBefore.Sub(bondedTokensBefore.Mul(math.NewInt(1)).Quo(math.NewInt(100))))
	dispute := k.GetDisputeById(ctx, 0)
	require.Equal(types.Voting, dispute.DisputeStatus)
	dispute = k.GetDisputeById(ctx, 0)
	require.Equal(types.Voting, dispute.DisputeStatus)
	// vote on dispute
	_, err = msgServer.Vote(ctx, &types.MsgVote{
		Voter: Addr.String(),
		Id:    0,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	})
	require.NoError(err)
	voterV := k.GetVoterVote(ctx, Addr.String(), 0)
	require.Equal(types.VoteEnum_VOTE_SUPPORT, voterV.Vote)
	v := k.GetVote(ctx, 0)
	require.Equal(v.VoteResult, types.VoteResult_NO_TALLY)
	require.Equal(v.Voters, []string{Addr.String()})
}

func (s *IntegrationTestSuite) TestProposeDisputeFromBond() {
	_, msgServer := s.disputeKeeper()
	require := s.Require()
	ctx := s.ctx
	// k := suite.disputekeeper
	report, valAddr := s.microReport()
	val, found := s.stakingKeeper.GetValidator(ctx, valAddr)
	require.True(found)
	bondedTokensBefore := val.GetBondedTokens()
	onePercent := bondedTokensBefore.Mul(math.NewInt(1)).Quo(math.NewInt(100))
	disputeFee := sdk.NewCoin("stake", onePercent)
	slashAmount := disputeFee.Amount
	_, err := msgServer.ProposeDispute(ctx, &types.MsgProposeDispute{
		Creator:         sdk.AccAddress(valAddr).String(),
		Report:          &report,
		DisputeCategory: types.Warning,
		Fee:             disputeFee,
		PayFromBond:     true,
	})
	require.NoError(err)

	val, _ = s.stakingKeeper.GetValidator(ctx, valAddr)
	require.Equal(val.GetBondedTokens(), bondedTokensBefore.Sub(slashAmount).Sub(disputeFee.Amount))
	require.True(val.IsJailed())
	// jail time for a warning is zero seconds so unjailing should be immediate
	// TODO: have to unjail through the staking keeper, if no self delegation then validator can't unjail
	s.mintTokens(sdk.AccAddress(valAddr), sdk.NewCoin(s.denom, sdk.NewInt(100)))
	_, err = s.stakingKeeper.Delegate(ctx, sdk.AccAddress(valAddr), sdk.NewInt(10), stakingtypes.Unbonded, val, true)
	require.NoError(err)
	err = s.slashingKeeper.Unjail(ctx, valAddr)
	require.NoError(err)
	val, _ = s.stakingKeeper.GetValidator(ctx, valAddr)
	require.False(val.IsJailed())
}

func (s *IntegrationTestSuite) TestExecuteVoteInvalid() {
	ctx := s.ctx
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{2, 3, 4, 5})
	reporterAddr := addrs[0].String()
	disputerAcc := addrs[1]
	disputerAddr := disputerAcc.String()
	report := types.MicroReport{
		Reporter:  reporterAddr,
		Power:     s.stakingKeeper.Validator(ctx, valAddrs[0]).GetConsensusPower(sdk.DefaultPowerReduction),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}
	disputeFee := s.disputekeeper.GetDisputeFee(ctx, reporterAddr, types.Warning)
	burnAmount := disputeFee.MulRaw(1).QuoRaw(20)
	// voter Reward is half of the burn amount divided by number of voters
	voterReward := burnAmount.QuoRaw(2).QuoRaw(4)

	disputerBalanceBefore := s.bankKeeper.GetBalance(ctx, disputerAcc, s.denom)
	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(ctx, &types.MsgProposeDispute{
		Creator:         disputerAddr,
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	// balance should be less than before paying fee
	disputerBalanceAfter := s.bankKeeper.GetBalance(ctx, disputerAcc, s.denom)
	s.True(disputerBalanceAfter.IsLT(disputerBalanceBefore))

	// start vote
	ids := s.disputekeeper.CheckPrevoteDisputesForExpiration(ctx)

	votes := []types.MsgVote{
		{
			Voter: reporterAddr,
			Id:    0,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: disputerAddr,
			Id:    0,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: addrs[2].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: addrs[3].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}

	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	// tally vote
	s.disputekeeper.TallyVote(ctx, ids[0])
	reporter := s.stakingKeeper.Validator(ctx, valAddrs[0])
	valTknBeforeExecuteVote := reporter.GetBondedTokens()
	disputerBalanceBeforeExecuteVote := s.bankKeeper.GetBalance(ctx, disputerAcc, s.denom)
	// execute vote
	s.disputekeeper.ExecuteVotes(ctx, ids)
	s.True(s.stakingKeeper.Validator(ctx, valAddrs[0]).GetBondedTokens().GT(valTknBeforeExecuteVote))
	// dispute fee returned so balance should be the same as before paying fee
	disputerBalanceAfterExecuteVote := s.bankKeeper.GetBalance(ctx, disputerAcc, s.denom)
	// add dispute fee returned minus burn amount plus the voter reward
	disputerBalanceBeforeExecuteVote.Amount = disputerBalanceBeforeExecuteVote.Amount.Add(disputeFee.Sub(burnAmount)).Add(voterReward)
	s.Equal(disputerBalanceBeforeExecuteVote, disputerBalanceAfterExecuteVote)
}

func (s *IntegrationTestSuite) TestExecuteVoteNoQuorumInvalid() {
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{1, 2, 3})
	report := types.MicroReport{
		Reporter:  addrs[0].String(),
		Power:     s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetConsensusPower(sdk.DefaultPowerReduction),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}

	disputeFee := s.disputekeeper.GetDisputeFee(s.ctx, report.Reporter, types.Warning)
	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)

	vote := []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: addrs[1].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	// start vote
	_, err = msgServer.Vote(s.ctx, &vote[0])
	s.NoError(err)
	_, err = msgServer.Vote(s.ctx, &vote[1])
	s.NoError(err)

	ctx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	s.disputekeeper.TallyVote(ctx, 0)

	reporter := s.stakingKeeper.Validator(ctx, valAddrs[0])
	bond := reporter.GetBondedTokens()
	// execute vote
	s.disputekeeper.ExecuteVotes(ctx, []uint64{0})

	voteInfo := s.disputekeeper.GetVote(ctx, 0)
	s.Equal(types.VoteResult_NO_QUORUM_MAJORITY_INVALID, voteInfo.VoteResult)
	s.True(s.stakingKeeper.Validator(ctx, valAddrs[0]).GetBondedTokens().Equal(bond))
}

func (s *IntegrationTestSuite) TestExecuteVoteSupport() {
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{2, 3, 4, 5})
	reporter := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
	disputerBefore := s.stakingKeeper.Validator(s.ctx, valAddrs[1])
	reporterAddr := sdk.AccAddress(valAddrs[0]).String()
	disputerAddr := sdk.AccAddress(valAddrs[1]).String()
	report := types.MicroReport{
		Reporter:  reporterAddr,
		Power:     reporter.GetConsensusPower(sdk.DefaultPowerReduction),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}
	disputeFee := s.disputekeeper.GetDisputeFee(s.ctx, reporterAddr, types.Warning)
	fivePercentBurn := disputeFee.MulRaw(1).QuoRaw(20)
	twoPercentBurn := fivePercentBurn.QuoRaw(2)
	_, err := msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         disputerAddr,
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	// start vote
	ids := s.disputekeeper.CheckPrevoteDisputesForExpiration(s.ctx)

	votersBalanceBefore := []sdk.Coin{
		s.bankKeeper.GetBalance(s.ctx, addrs[0], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[2], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[3], s.denom),
	}
	votes := []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
		{
			Voter: addrs[1].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
		{
			Voter: addrs[2].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
		{
			Voter: addrs[3].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	// tally vote
	s.disputekeeper.Tally(s.ctx, ids)
	// execute vote
	s.disputekeeper.ExecuteVotes(s.ctx, ids)
	reporterAfter := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
	s.True(reporterAfter.IsJailed())
	s.True(reporterAfter.GetBondedTokens().LT(reporter.GetBondedTokens()))

	votersBalanceAfter := []sdk.Coin{
		s.bankKeeper.GetBalance(s.ctx, addrs[0], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[2], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[3], s.denom),
	}
	voterReward := twoPercentBurn.QuoRaw(4)
	for i := range votersBalanceBefore {
		votersBalanceBefore[i].Amount = votersBalanceBefore[i].Amount.Add(voterReward)
		s.Equal(votersBalanceBefore[i], (votersBalanceAfter[i]))
	}
	s.True(disputerBefore.GetBondedTokens().Add(disputeFee).Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[1]).GetBondedTokens()))
}

func (s *IntegrationTestSuite) TestExecuteVoteAgainst() {
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{2, 3, 4, 5})
	reporterBefore := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
	reporterAddr := sdk.AccAddress(valAddrs[0]).String()
	disputerAddr := sdk.AccAddress(valAddrs[1]).String()
	report := types.MicroReport{
		Reporter:  reporterAddr,
		Power:     reporterBefore.GetConsensusPower(sdk.DefaultPowerReduction),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}
	disputeFee := s.disputekeeper.GetDisputeFee(s.ctx, reporterAddr, types.Warning)
	fivePercentBurn := disputeFee.MulRaw(1).QuoRaw(20)
	twoPercentBurn := fivePercentBurn.QuoRaw(2)
	disputeFeeMinusBurn := disputeFee.Sub(disputeFee.MulRaw(1).QuoRaw(20))
	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         disputerAddr,
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)

	votersBalanceBefore := []sdk.Coin{
		s.bankKeeper.GetBalance(s.ctx, addrs[0], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[2], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[3], s.denom),
	}
	votes := []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
		{
			Voter: addrs[1].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
		{
			Voter: addrs[2].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
		{
			Voter: addrs[3].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	// tally vote
	s.disputekeeper.TallyVote(s.ctx, 0)
	// execute vote
	s.disputekeeper.ExecuteVote(s.ctx, 0)
	reporterAfterDispute := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
	s.Equal(reporterBefore.GetBondedTokens().Add(disputeFeeMinusBurn), reporterAfterDispute.GetBondedTokens())

	votersBalanceAfter := []sdk.Coin{
		s.bankKeeper.GetBalance(s.ctx, addrs[0], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[2], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[3], s.denom),
	}
	voterReward := twoPercentBurn.QuoRaw(4)
	for i := range votersBalanceBefore {
		votersBalanceBefore[i].Amount = votersBalanceBefore[i].Amount.Add(voterReward)
		s.Equal(votersBalanceBefore[i], (votersBalanceAfter[i]))
	}
}

func (s *IntegrationTestSuite) TestDisputeMultipleRounds() {
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{1, 2, 3})
	reporter := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
	reporterStakeBefore := reporter.GetBondedTokens()
	report := types.MicroReport{
		Reporter:  addrs[0].String(),
		Power:     reporter.GetConsensusPower(sdk.DefaultPowerReduction),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}
	disputeFee := s.disputekeeper.GetDisputeFee(s.ctx, report.Reporter, types.Warning)
	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	// check reporter stake
	s.True(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens().LT(reporterStakeBefore))
	s.Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
	header := tmproto.Header{Height: s.app.LastBlockHeight() + 1, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})

	votes := []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: addrs[1].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}

	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}

	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.Error(err)
	// check reporter stake
	s.True(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens().LT(reporterStakeBefore))
	s.Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
	// forward time to after vote end
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	// check reporter stake
	s.True(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens().LT(reporterStakeBefore))
	s.Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	// voting that doesn't reach quorum
	votes = []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: addrs[1].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.Error(err) //fails since hasn't been tallied and executed
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	// check reporter stake
	s.True(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens().LT(reporterStakeBefore))
	s.Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	// voting that doesn't reach quorum
	votes = []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    2,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: addrs[1].String(),
			Id:    2,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.Error(err) //fails since hasn't been tallied and executed
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	// check reporter stake
	s.True(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens().LT(reporterStakeBefore))
	s.Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	// voting that doesn't reach quorum
	votes = []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    3,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: addrs[1].String(),
			Id:    3,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.Error(err) //fails since hasn't been tallied and executed
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	// check reporter stake
	s.True(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens().LT(reporterStakeBefore))
	s.Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	// voting that doesn't reach quorum
	votes = []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    4,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: addrs[1].String(),
			Id:    4,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.Equal(err.Error(), "can't start a new round for this dispute 4; dispute status DISPUTE_STATUS_VOTING") //fails since hasn't been tallied and executed
	// forward time to end vote
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})

	_, err = msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.Equal(err.Error(), "can't start a new round for this dispute 4; dispute status DISPUTE_STATUS_RESOLVED") //max rounds reached
	// check reporter stake, stake should be restored due to invalid vote final result
	s.Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens(), reporterStakeBefore)
}

func (s *IntegrationTestSuite) TestNoQorumSingleRound() {
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{1, 2, 3})
	reporter := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
	reporterStakeBefore := reporter.GetBondedTokens()
	report := types.MicroReport{
		Reporter:  addrs[0].String(),
		Power:     reporter.GetConsensusPower(sdk.DefaultPowerReduction),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}
	disputeFee := s.disputekeeper.GetDisputeFee(s.ctx, report.Reporter, types.Warning)
	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	header := tmproto.Header{Height: s.app.LastBlockHeight() + 1, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})

	votes := []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: addrs[1].String(),
			Id:    0,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}

	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	// forward time to expire dispute
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*3 + 1))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	reporter = s.stakingKeeper.Validator(s.ctx, valAddrs[0])
	reporterStakeAfter := reporter.GetBondedTokens()
	// reporter stake should be restored after dispute expires for invalid vote
	s.Equal(reporterStakeBefore, reporterStakeAfter)
}

func (s *IntegrationTestSuite) TestDisputeButNoVotes() {
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{1, 2, 3})
	reporter := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
	reporterStakeBefore := reporter.GetBondedTokens()
	report := types.MicroReport{
		Reporter:  addrs[0].String(),
		Power:     reporter.GetConsensusPower(sdk.DefaultPowerReduction),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}
	disputeFee := s.disputekeeper.GetDisputeFee(s.ctx, report.Reporter, types.Warning)
	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	s.NotEqual(reporterStakeBefore, s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens())
	// forward time to end vote
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*3 + 1))
	header := tmproto.Header{Height: s.app.LastBlockHeight() + 1, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	s.Equal(reporterStakeBefore, s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens())
}
