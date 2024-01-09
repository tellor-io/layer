package integration_test

import (
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
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
	addrs, valAddrs := s.createValidators([]int64{1000, 20})
	report := types.MicroReport{
		Reporter:  addrs[0].String(),
		Power:     s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetConsensusPower(sdk.DefaultPowerReduction),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}
	Addr := s.newKeysWithTokens()
	valAddr := valAddrs[0]

	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, sdk.NewInt(5_000_000)),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	s.Equal(uint64(2), s.disputekeeper.GetDisputeCount(s.ctx))
	s.Equal(1, len(s.disputekeeper.GetOpenDisputeIds(s.ctx).Ids))

	// check validator wasn't slashed/jailed
	val, found := s.stakingKeeper.GetValidator(s.ctx, valAddr)
	bondedTokensBefore := val.GetBondedTokens()
	s.True(found)
	s.False(val.IsJailed())
	s.Equal(bondedTokensBefore, sdk.NewInt(1000_000_000))
	// Add dispute fee to complete the fee and jail/slash validator
	_, err = msgServer.AddFeeToDispute(s.ctx, &types.MsgAddFeeToDispute{
		Creator:   addrs[1].String(),
		DisputeId: 1,
		Amount:    sdk.NewCoin(s.denom, sdk.NewInt(5_000_000)),
	})
	s.NoError(err)
	// check validator was slashed/jailed
	val, found = s.stakingKeeper.GetValidator(s.ctx, valAddr)
	s.True(found)
	s.True(val.IsJailed())
	// check validator was slashed 1% of tokens
	s.Equal(val.GetBondedTokens(), bondedTokensBefore.Sub(bondedTokensBefore.Mul(math.NewInt(1)).Quo(math.NewInt(100))))
	dispute := s.disputekeeper.GetDisputeById(s.ctx, 1)
	s.Equal(types.Voting, dispute.DisputeStatus)
	// vote on dispute
	_, err = msgServer.Vote(s.ctx, &types.MsgVote{
		Voter: Addr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	})
	s.NoError(err)
	voterV := s.disputekeeper.GetVoterVote(s.ctx, Addr.String(), 1)
	s.Equal(types.VoteEnum_VOTE_SUPPORT, voterV.Vote)
	v := s.disputekeeper.GetVote(s.ctx, 1)
	s.Equal(v.VoteResult, types.VoteResult_NO_TALLY)
	s.Equal(v.Voters, []string{Addr.String()})
}

func (s *IntegrationTestSuite) TestProposeDisputeFromBond() {
	_, msgServer := s.disputeKeeper()
	require := s.Require()
	ctx := s.ctx
	addrs, valAddrs := s.createValidators([]int64{100})
	report := types.MicroReport{
		Reporter:  addrs[0].String(),
		Power:     s.stakingKeeper.Validator(ctx, valAddrs[0]).GetConsensusPower(sdk.DefaultPowerReduction),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}
	valAddr := valAddrs[0]
	val, found := s.stakingKeeper.GetValidator(ctx, valAddr)
	require.True(found)

	bondedTokensBefore := val.GetBondedTokens()
	onePercent := bondedTokensBefore.Mul(math.NewInt(1)).Quo(math.NewInt(100))
	disputeFee := sdk.NewCoin(s.denom, onePercent)
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
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{200, 300, 400, 500})
	reporterAddr := addrs[0].String()
	disputerAcc := addrs[1]
	disputerAddr := disputerAcc.String()
	report := types.MicroReport{
		Reporter:  reporterAddr,
		Power:     s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetConsensusPower(sdk.DefaultPowerReduction),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}
	disputeFee := s.disputekeeper.GetDisputeFee(s.ctx, reporterAddr, types.Warning)
	burnAmount := disputeFee.MulRaw(1).QuoRaw(20)
	disputerBalanceBefore := s.bankKeeper.GetBalance(s.ctx, disputerAcc, s.denom)
	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(s.ctx, &types.MsgProposeDispute{
		Creator:         disputerAddr,
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	s.True(s.bankKeeper.GetBalance(s.ctx, disputerAcc, s.denom).IsLT(disputerBalanceBefore))

	// start vote
	ids := s.disputekeeper.CheckPrevoteDisputesForExpiration(s.ctx)
	votes := []types.MsgVote{
		{
			Voter: reporterAddr,
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: disputerAddr,
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: addrs[2].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
		{
			Voter: addrs[3].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	// tally vote
	s.disputekeeper.TallyVote(s.ctx, ids[0])
	reporter := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
	valTknBeforeExecuteVote := reporter.GetBondedTokens()
	disputerBalanceBeforeExecuteVote := s.bankKeeper.GetBalance(s.ctx, disputerAcc, s.denom)
	// execute vote
	s.disputekeeper.ExecuteVotes(s.ctx, ids)
	s.True(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens().GT(valTknBeforeExecuteVote))
	// dispute fee returned so balance should be the same as before paying fee
	disputerBalanceAfterExecuteVote := s.bankKeeper.GetBalance(s.ctx, disputerAcc, s.denom)
	voters := s.disputekeeper.GetVote(s.ctx, 1).Voters
	rewards, _ := s.disputekeeper.CalculateVoterShare(s.ctx, voters, burnAmount.QuoRaw(2))
	voterReward := rewards[disputerAddr]
	// add dispute fee returned minus burn amount plus the voter reward
	disputerBalanceBeforeExecuteVote.Amount = disputerBalanceBeforeExecuteVote.Amount.Add(disputeFee.Sub(burnAmount)).Add(voterReward)
	s.Equal(disputerBalanceBeforeExecuteVote, disputerBalanceAfterExecuteVote)
}

func (suite *IntegrationTestSuite) getTestMetadata() banktypes.Metadata {
	return banktypes.Metadata{
		Name:        "Tellor Layer Tributes",
		Symbol:      "TRB",
		Description: "The native staking token of the TellorLayer.",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "loya", Exponent: uint32(0), Aliases: nil},
			{Denom: "mloya", Exponent: uint32(3), Aliases: []string{"milliloya"}},
			{Denom: "trb", Exponent: uint32(6), Aliases: nil},
		},
		Base:    "loya",
		Display: "trb",
	}
}

func (s *IntegrationTestSuite) TestExecuteVoteNoQuorumInvalid() {
	_, msgServer := s.disputeKeeper()

	addrs, valAddrs := s.createValidators([]int64{100, 200, 300})
	reporter := s.stakingKeeper.Validator(s.ctx, valAddrs[0])

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

	vote := []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	// start vote
	_, err = msgServer.Vote(s.ctx, &vote[0])
	s.NoError(err)

	ctx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS + 1))
	s.disputekeeper.TallyVote(ctx, 1)

	reporter = s.stakingKeeper.Validator(ctx, valAddrs[0])
	bond := reporter.GetBondedTokens()
	// execute vote
	s.disputekeeper.ExecuteVotes(ctx, []uint64{0})

	voteInfo := s.disputekeeper.GetVote(ctx, 1)
	s.Equal(types.VoteResult_NO_QUORUM_MAJORITY_INVALID, voteInfo.VoteResult)
	s.True(s.stakingKeeper.Validator(ctx, valAddrs[0]).GetBondedTokens().Equal(bond))
}

func (s *IntegrationTestSuite) TestExecuteVoteSupport() {
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{200, 300, 400, 500})
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
			Id:    1,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
		{
			Voter: addrs[1].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
		{
			Voter: addrs[2].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
		{
			Voter: addrs[3].String(),
			Id:    1,
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
	voters := s.disputekeeper.GetVote(s.ctx, 1).Voters
	votersReward, _ := s.disputekeeper.CalculateVoterShare(s.ctx, voters, twoPercentBurn)
	for i := range votersBalanceBefore {
		votersBalanceBefore[i].Amount = votersBalanceBefore[i].Amount.Add(votersReward[addrs[i].String()])
		s.Equal(votersBalanceBefore[i], (votersBalanceAfter[i]))
	}
	s.True(disputerBefore.GetBondedTokens().Add(disputeFee).Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[1]).GetBondedTokens()))
}

func (s *IntegrationTestSuite) TestExecuteVoteAgainst() {
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{200, 300, 400, 500})
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
			Id:    1,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
		{
			Voter: addrs[1].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
		{
			Voter: addrs[2].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
		{
			Voter: addrs[3].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	// tally vote
	s.disputekeeper.TallyVote(s.ctx, 1)
	// execute vote
	s.disputekeeper.ExecuteVote(s.ctx, 1)
	reporterAfterDispute := s.stakingKeeper.Validator(s.ctx, valAddrs[0])
	s.Equal(reporterBefore.GetBondedTokens().Add(disputeFeeMinusBurn), reporterAfterDispute.GetBondedTokens())

	votersBalanceAfter := []sdk.Coin{
		s.bankKeeper.GetBalance(s.ctx, addrs[0], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[2], s.denom),
		s.bankKeeper.GetBalance(s.ctx, addrs[3], s.denom),
	}
	voters := s.disputekeeper.GetVote(s.ctx, 1).Voters
	votersReward, _ := s.disputekeeper.CalculateVoterShare(s.ctx, voters, twoPercentBurn)
	for i := range votersBalanceBefore {
		votersBalanceBefore[i].Amount = votersBalanceBefore[i].Amount.Add(votersReward[addrs[i].String()])
		s.Equal(votersBalanceBefore[i], (votersBalanceAfter[i]))
	}
}

func (s *IntegrationTestSuite) TestDisputeMultipleRounds() {
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{100, 200, 300})
	accBals1 := s.bankKeeper.GetAccountsBalances(s.ctx)
	moduleAccs := s.ModuleAccs()
	for _, acc := range accBals1 {
		if acc.Address == addrs[0].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(900_000_000))
		}
		if acc.Address == addrs[1].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(800_000_000))
		}
		if acc.Address == addrs[2].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(700_000_000))
		}
		if acc.Address == moduleAccs.staking.GetAddress().String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(601_000_000))
		}
	}

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
	burnAmount := disputeFee.MulRaw(1).QuoRaw(20)
	dispute := types.MsgProposeDispute{
		Creator:         addrs[1].String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.denom, disputeFee),
		DisputeCategory: types.Warning,
	}
	balanceBefore := s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(s.ctx, &dispute)
	s.NoError(err)
	accBals2 := s.bankKeeper.GetAccountsBalances(s.ctx)
	for _, acc := range accBals2 {
		if acc.Address == addrs[0].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(900_000_000))
		}
		if acc.Address == addrs[1].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(799_000_000))
		}
		if acc.Address == addrs[2].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(700_000_000))
		}
		if acc.Address == moduleAccs.staking.GetAddress().String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(600_000_000))
		}
		if acc.Address == moduleAccs.dispute.GetAddress().String() {
			s.Equal(acc.Coins.AmountOf(s.denom), disputeFee.MulRaw(2)) // disputeFee + slashAmount
		}
	}
	balanceAfter := s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	s.True(balanceBefore.Amount.Sub(disputeFee).Equal(balanceAfter.Amount))
	// check reporter stake
	s.True(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens().LT(reporterStakeBefore))
	s.Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
	// begin block
	header := tmproto.Header{Height: s.app.LastBlockHeight() + 1, Time: s.ctx.BlockTime()}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})

	votes := []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}

	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime()}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.Equal(err.Error(), "can't start a new round for this dispute 1; dispute status DISPUTE_STATUS_VOTING")
	// check reporter stake
	s.True(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens().LT(reporterStakeBefore))
	s.Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
	// forward time to after vote end
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.NoError(err)
	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, burnAmount.MulRaw(2))), balanceAfter)
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
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.Equal(err.Error(), "can't start a new round for this dispute 2; dispute status DISPUTE_STATUS_VOTING") //fails since hasn't been tallied and executed

	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.NoError(err)
	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, burnAmount.MulRaw(4))), balanceAfter)
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
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.Error(err) //fails since hasn't been tallied and executed
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.NoError(err)
	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, burnAmount.MulRaw(8))), balanceAfter)
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
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.Equal(err.Error(), "can't start a new round for this dispute 4; dispute status DISPUTE_STATUS_VOTING")
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.NoError(err)
	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, burnAmount.MulRaw(16))), balanceAfter)
	// check reporter stake
	s.True(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens().LT(reporterStakeBefore))
	s.Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens(), reporterStakeBefore.Sub(disputeFee))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	// voting that doesn't reach quorum
	votes = []types.MsgVote{
		{
			Voter: addrs[0].String(),
			Id:    5,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.ctx, &votes[i])
		s.NoError(err)
	}
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.Equal(err.Error(), "can't start a new round for this dispute 5; dispute status DISPUTE_STATUS_VOTING") //fails since hasn't been tallied and executed
	// forward time to end vote
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})

	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.NoError(err)
	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, disputeFee)), balanceAfter)

	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.NoError(err)
	accBals3 := s.bankKeeper.GetAccountsBalances(s.ctx)
	for _, acc := range accBals3 {
		if acc.Address == addrs[0].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(900_000_000))
		}
		if acc.Address == addrs[1].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(795_500_000))
		}
		if acc.Address == addrs[2].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(700_000_000))
		}
		if acc.Address == moduleAccs.staking.GetAddress().String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(600_000_000))
		}
		if acc.Address == moduleAccs.dispute.GetAddress().String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(5_500_000)) // disputeFee + slashAmount + round 1(100000) + round 2(200000) + round 3(400000) + round 4(800000) + round 5(1000000) + round 6(1000000)
		}
	}
	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, disputeFee)), balanceAfter)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.TWO_DAYS))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	balanceBefore = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	_, err = msgServer.ProposeDispute(s.ctx, &dispute)
	s.NoError(err)
	balanceAfter = s.bankKeeper.GetBalance(s.ctx, addrs[1], s.denom)
	s.Equal(balanceBefore.Sub(sdk.NewCoin(s.denom, disputeFee)), balanceAfter)
	// forward time and end dispute
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.THREE_DAYS))
	header = tmproto.Header{Height: s.app.LastBlockHeight() + 1, AppHash: s.app.LastCommitID().Hash, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	// check reporter stake, stake should be restored due to invalid vote final result
	s.Equal(s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens(), reporterStakeBefore)
	accBals4 := s.bankKeeper.GetAccountsBalances(s.ctx)
	for _, acc := range accBals4 {
		if acc.Address == addrs[0].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(902_275_000)) // voter reward half the total burn Amount(the 5%)
		}
		if acc.Address == addrs[1].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(795_450_000))
		}
		if acc.Address == addrs[2].String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(700_000_000))
		}
		if acc.Address == moduleAccs.staking.GetAddress().String() {
			s.Equal(acc.Coins.AmountOf(s.denom), sdk.NewInt(601_000_000)) // stake restored
		}
	}
}

func (s *IntegrationTestSuite) TestNoQorumSingleRound() {
	_, msgServer := s.disputeKeeper()
	addrs, valAddrs := s.createValidators([]int64{100, 200, 300})
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
	addrs, valAddrs := s.createValidators([]int64{100, 200, 300})
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
	// forward time to end dispute
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(keeper.THREE_DAYS))
	header := tmproto.Header{Height: s.app.LastBlockHeight() + 1, Time: s.ctx.BlockTime().Add(1)}
	s.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	s.Equal(reporterStakeBefore, s.stakingKeeper.Validator(s.ctx, valAddrs[0]).GetBondedTokens())
}
