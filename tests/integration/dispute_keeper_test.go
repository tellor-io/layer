package integration_test

import (
	"cosmossdk.io/math"

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
	require.Equal(types.Prevote, dispute.DisputeStatus)
	// these are called during begin block
	ids := k.CheckPrevoteDisputesForExpiration(ctx)
	k.StartVoting(ctx, ids)
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
	report := types.MicroReport{
		Reporter:  addrs[0].String(),
		Power:     s.stakingKeeper.Validator(ctx, valAddrs[0]).GetConsensusPower(sdk.DefaultPowerReduction),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}
	addr1Bal := s.bankKeeper.GetBalance(ctx, addrs[1], s.denom)
	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, s.disputekeeper.GetDisputeFee(ctx, addrs[1].String(), types.Warning)),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	// balance should be less than before paying fee
	addr1Balpaid := s.bankKeeper.GetBalance(ctx, addrs[1], s.denom)
	s.True(addr1Balpaid.IsLT(addr1Bal))
	// start vote
	ids := s.disputekeeper.CheckPrevoteDisputesForExpiration(ctx)
	s.disputekeeper.StartVoting(ctx, ids)

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

	_, err = msgServer.Vote(ctx, &votes[0])
	s.NoError(err)
	_, err = msgServer.Vote(ctx, &votes[1])
	s.NoError(err)
	_, err = msgServer.Vote(ctx, &votes[2])
	s.NoError(err)
	_, err = msgServer.Vote(ctx, &votes[3])
	s.NoError(err)

	//  check if validator gets tokens back for invalid vote
	//  and check if fee payers get the fee back for invalid vote
	s.disputekeeper.TallyVote(ctx, ids[0])
	reporter := s.stakingKeeper.Validator(ctx, valAddrs[0])
	valTknBeforeExecuteVote := reporter.GetBondedTokens()
	s.True(reporter.IsJailed())
	// execute vote
	s.disputekeeper.ExecuteVote(ctx, ids)

	s.True(s.stakingKeeper.Validator(ctx, valAddrs[0]).GetBondedTokens().GT(valTknBeforeExecuteVote))
	// dispute fee returned so balance should be the same as before paying fee
	addrs1Balexecuted := s.bankKeeper.GetBalance(ctx, addrs[1], s.denom)
	s.True(addrs1Balexecuted.Equal(addr1Bal))

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
	ids := s.disputekeeper.CheckPrevoteDisputesForExpiration(s.ctx)
	s.disputekeeper.StartVoting(s.ctx, ids)

	_, err = msgServer.Vote(s.ctx, &vote[0])
	s.NoError(err)
	_, err = msgServer.Vote(s.ctx, &vote[1])
	s.NoError(err)

	ctx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	s.disputekeeper.TallyVote(ctx, ids[0])

	reporter := s.stakingKeeper.Validator(ctx, valAddrs[0])
	bond := reporter.GetBondedTokens()
	// execute vote
	s.disputekeeper.ExecuteVote(ctx, ids)

	voteInfo := s.disputekeeper.GetVote(ctx, ids[0])
	s.Equal(types.VoteResult_NO_QUORUM_MAJORITY_INVALID, voteInfo.VoteResult)
	s.True(s.stakingKeeper.Validator(ctx, valAddrs[0]).GetBondedTokens().Equal(bond))

}
