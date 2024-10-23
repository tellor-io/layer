package integration_test

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/x/dispute"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reporterKeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *IntegrationTestSuite) TestVotingOnDispute() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, valAddrs, _ := s.createValidatorAccs([]uint64{50}) // creates validator with 100 power
	valAddr := valAddrs[0]
	repAddr := sdk.AccAddress(valAddr)
	valBond, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)

	s.NoError(err)
	dels, err := s.Setup.Stakingkeeper.GetValidatorDelegations(s.Setup.Ctx, valAddr)
	s.NoError(err)

	srcs := make([]*reportertypes.TokenOriginInfo, len(dels))
	total := math.ZeroInt()
	for i, del := range dels {
		srcs[i] = &reportertypes.TokenOriginInfo{
			DelegatorAddress: sdk.MustAccAddressFromBech32(del.DelegatorAddress).Bytes(),
			ValidatorAddress: valAddr.Bytes(),
			Amount:           valBond.TokensFromShares(del.Shares).TruncateInt(),
		}
		total = total.Add(srcs[i].Amount)
	}
	err = s.Setup.Reporterkeeper.Report.Set(s.Setup.Ctx, collections.Join(repAddr.Bytes(), uint64(s.Setup.Ctx.BlockHeight())), reportertypes.DelegationsAmounts{TokenOrigins: srcs, Total: total})
	s.NoError(err)
	// assemble report with reporter to dispute
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:  repAddr.String(),
		Power:     100,
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.Unix(1696516597, 0),
	}
	// disputer with tokens to pay fee
	disputer := s.newKeysWithTokens()

	// Propose dispute pay half of the fee from account
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.Setup.Denom, math.NewInt(500_000)),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)

	// 2 here because dispute count starts from 1 and dispute count gives the next dispute id
	s.Equal(uint64(2), s.Setup.Disputekeeper.NextDisputeId(s.Setup.Ctx))
	open, err := s.Setup.Disputekeeper.GetOpenDisputes(s.Setup.Ctx)
	s.NoError(err)
	s.Equal(1, len(open))

	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAddr.Bytes(), reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAddr.Bytes(), reportertypes.NewSelection(repAddr.Bytes(), 1)))
	// check validator wasn't slashed/jailed
	rep, err := s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, repAddr.Bytes())
	s.NoError(err)
	repTokens, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr)
	s.NoError(err)
	// reporter tokens should be the same as the stake amount since fee wasn't fully paid
	s.Equal(repTokens, valBond.Tokens)
	s.False(rep.Jailed)
	// Add dispute fee to complete the fee and jail/slash validator
	_, err = msgServer.AddFeeToDispute(s.Setup.Ctx, &types.MsgAddFeeToDispute{
		Creator:   disputer.String(),
		DisputeId: 1,
		Amount:    sdk.NewCoin(s.Setup.Denom, math.NewInt(500_000)),
	})
	s.NoError(err)
	// check reporter was slashed/jailed after fee was added
	rep, err = s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, repAddr.Bytes())
	s.NoError(err)
	s.True(rep.Jailed)

	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(types.Voting, dispute.DisputeStatus)
	// vote on dispute
	// mint more tokens to disputer to give voting power
	s.Setup.MintTokens(disputer, math.NewInt(1_000_000))
	_, _ = msgServer.Vote(s.Setup.Ctx, &types.MsgVote{
		Voter: disputer.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	})
	vtr, err := s.Setup.Disputekeeper.Voter.Get(s.Setup.Ctx, collections.Join(uint64(1), disputer.Bytes()))
	s.NoError(err)
	s.Equal(types.VoteEnum_VOTE_SUPPORT, vtr.Vote)
	v, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(v.VoteResult, types.VoteResult_NO_TALLY)
	iter, err := s.Setup.Disputekeeper.Voter.Indexes.VotersById.MatchExact(s.Setup.Ctx, uint64(1))
	s.NoError(err)
	voters, err := iter.PrimaryKeys()
	s.NoError(err)
	s.Equal(voters[0].K2(), disputer.Bytes())
}

func (s *IntegrationTestSuite) TestProposeDisputeFromBond() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)

	_, valAddrs, _ := s.createValidatorAccs([]uint64{500})

	valAddr := valAddrs[0]
	repAddr := sdk.AccAddress(valAddr)

	valBond, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)
	s.NoError(err)
	dels, err := s.Setup.Stakingkeeper.GetValidatorDelegations(s.Setup.Ctx, valAddr)
	s.NoError(err)

	srcs := make([]*reportertypes.TokenOriginInfo, len(dels))
	total := math.ZeroInt()
	for i, del := range dels {
		srcs[i] = &reportertypes.TokenOriginInfo{
			DelegatorAddress: sdk.MustAccAddressFromBech32(del.DelegatorAddress).Bytes(),
			ValidatorAddress: valAddr.Bytes(),
			Amount:           valBond.TokensFromShares(del.Shares).TruncateInt(),
		}
		total = total.Add(srcs[i].Amount)
	}
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAddr, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAddr, reportertypes.NewSelection(repAddr, 1)))
	err = s.Setup.Reporterkeeper.Report.Set(s.Setup.Ctx, collections.Join(repAddr.Bytes(), uint64(s.Setup.Ctx.BlockHeight())), reportertypes.DelegationsAmounts{TokenOrigins: srcs, Total: total})
	s.NoError(err)
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    repAddr.String(),
		Power:       1000,
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:         repAddr.String(),
		Report:          &report,
		DisputeCategory: types.Warning,
		Fee:             sdk.NewCoin(s.Setup.Denom, math.NewInt(10_000_000)), // one percent dispute fee
		PayFromBond:     true,
	})
	s.NoError(err)

	// check reporter was slashed/jailed after fee was added
	rep, err := s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, repAddr)
	s.NoError(err)
	s.True(rep.Jailed)

	reporterServer := reporterKeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	req := &reportertypes.MsgUnjailReporter{
		ReporterAddress: repAddr.String(),
	}
	_, err = reporterServer.UnjailReporter(s.Setup.Ctx, req)
	s.NoError(err)
	rep, err = s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, repAddr)
	s.NoError(err)
	s.False(rep.Jailed)
}

func (s *IntegrationTestSuite) TestExecuteVoteInvalid() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)

	_, valAddrs, _ := s.createValidatorAccs([]uint64{50})
	repAccs := s.CreateAccountsWithTokens(3, 100*1e6)
	disputer := s.newKeysWithTokens()
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))
	delegators := repAccs
	valAddr := valAddrs[0]
	repAddr := sdk.AccAddress(valAddr)

	valBond, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)
	s.NoError(err)
	dels, err := s.Setup.Stakingkeeper.GetValidatorDelegations(s.Setup.Ctx, valAddr)
	s.NoError(err)

	srcs := make([]*reportertypes.TokenOriginInfo, len(dels))
	total := math.ZeroInt()
	for i, del := range dels {
		srcs[i] = &reportertypes.TokenOriginInfo{
			DelegatorAddress: sdk.MustAccAddressFromBech32(del.DelegatorAddress).Bytes(),
			ValidatorAddress: valAddr.Bytes(),
			Amount:           valBond.TokensFromShares(del.Shares).TruncateInt(),
		}
		total = total.Add(srcs[i].Amount)
	}
	err = s.Setup.Reporterkeeper.Report.Set(s.Setup.Ctx, collections.Join(repAddr.Bytes(), uint64(s.Setup.Ctx.BlockHeight())), reportertypes.DelegationsAmounts{TokenOrigins: srcs, Total: total})
	s.NoError(err)
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAddr, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAddr, reportertypes.NewSelection(repAddr, uint64(len(dels)))))

	repTokensBeforePropose, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr)
	s.NoError(err)
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    repAddr.String(),
		Power:       100,
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)
	burnAmount := disputeFee.MulRaw(1).QuoRaw(20)
	disputerBalanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	// Propose dispute pay half of the fee from account
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)
	s.True(s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom).IsLT(disputerBalanceBefore))

	s.NoError(dispute.CheckOpenDisputesForExpiration(s.Setup.Ctx, s.Setup.Disputekeeper))

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
		_, err = msgServer.Vote(s.Setup.Ctx, &votes[i])
		if err != nil {
			s.Error(err, "voter power is zero")
		}

	}
	valTokensBeforeExecuteVote, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)
	s.NoError(err)
	disputerBalanceBeforeExecuteVote := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	// only 25 percent of the total power voted so vote should not be tallied unless it's expired
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	s.NoError(err)
	voteInfo, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.True(voteInfo.Executed)
	s.Equal(types.VoteResult_NO_QUORUM_MAJORITY_INVALID, voteInfo.VoteResult)
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{CallerAddress: disputer.String(), PayerAddress: disputer.String(), Id: 1})
	s.NoError(err)

	reporterServer := reporterKeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	req := &reportertypes.MsgUnjailReporter{
		ReporterAddress: repAddr.String(),
	}
	_, err = reporterServer.UnjailReporter(s.Setup.Ctx, req)
	s.NoError(err)
	repTokensAfterExecuteVote, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr)
	s.NoError(err)
	s.True(repTokensBeforePropose.Equal(repTokensAfterExecuteVote))
	valTokensAfterExecuteVote, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)
	s.NoError(err)
	s.True(valTokensAfterExecuteVote.Tokens.GT(valTokensBeforeExecuteVote.Tokens))
	disputerBalanceAfterExecuteVote := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	iter, err := s.Setup.Disputekeeper.Voter.Indexes.VotersById.MatchExact(s.Setup.Ctx, uint64(1))
	s.NoError(err)
	keys, err := iter.PrimaryKeys()
	s.NoError(err)
	voters := make([]keeper.VoterInfo, len(keys))
	totalVoterPower := math.ZeroInt()
	for i := range keys {
		v, err := s.Setup.Disputekeeper.Voter.Get(s.Setup.Ctx, keys[i])
		s.NoError(err)
		voters[i] = keeper.VoterInfo{Voter: keys[i].K2(), Power: v.VoterPower}
		totalVoterPower = totalVoterPower.Add(v.VoterPower)
	}
	expectedDisputerBalAfterExecute := disputerBalanceBeforeExecuteVote.Amount.Add(disputeFee.Sub(burnAmount))
	s.Equal(expectedDisputerBalAfterExecute, disputerBalanceAfterExecuteVote.Amount)
	disputerVoterReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, disputer, 1)
	s.NoError(err)
	reporterVoterReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, repAddr, 1)
	s.NoError(err)
	delegator1VoterReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, delegators[1], 1)
	s.NoError(err)
	delegator2VoterReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, delegators[2], 1)
	s.NoError(err)
	_, err = msgServer.ClaimReward(s.Setup.Ctx, &types.MsgClaimReward{CallerAddress: disputer.String(), DisputeId: 1})
	s.NoError(err)
	disputerBalAfterClaim := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	expectedDisputerBalAfterClaim := disputerBalanceAfterExecuteVote.Amount.Add(disputerVoterReward)
	s.Equal(expectedDisputerBalAfterClaim, disputerBalAfterClaim.Amount)
	sumVoterRewards := disputerVoterReward.Add(reporterVoterReward).Add(delegator1VoterReward).Add(delegator2VoterReward)
	s.True(sumVoterRewards.LTE(burnAmount.Quo(math.NewInt(2))))
	s.True(sumVoterRewards.GTE(burnAmount.Quo(math.NewInt(2)).Sub(math.NewInt(4)))) // max one loya per voter lost via rounding
}

func (s *IntegrationTestSuite) TestExecuteVoteNoQuorumInvalid() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)

	_, valAddrs, _ := s.createValidatorAccs([]uint64{1000})

	disputer := s.newKeysWithTokens()
	s.Setup.MintTokens(disputer, math.NewInt(20_000_000))

	valAddr := valAddrs[0]
	repAddr := sdk.AccAddress(valAddr)
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAddr, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAddr, reportertypes.NewSelection(repAddr, 1)))

	repStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr)
	fmt.Println("\nrepStake", repStake)
	valStakeBeforePropose, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)
	s.NoError(err)
	fmt.Println("\nvalStakeBeforePropose", valStakeBeforePropose.Tokens)
	s.NoError(err)
	currentBlock := s.Setup.Ctx.BlockHeight()
	delTokensAtBlock, err := s.Setup.Reporterkeeper.GetDelegatorTokensAtBlock(s.Setup.Ctx, valAddr.Bytes(), uint64(currentBlock))
	s.NoError(err)
	fmt.Println("\ndelTokensAtBlock", delTokensAtBlock)
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:  repAddr.String(),
		Power:     repStake.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.Unix(1696516597, 0),
	}

	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)

	// Propose dispute pay half of the fee from account
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory: types.Warning,
	})
	s.NoError(err)

	vote := []types.MsgVote{
		{
			Voter: repAddr.String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_INVALID,
		},
	}
	// start vote
	_, _ = msgServer.Vote(s.Setup.Ctx, &vote[0])

	ctx := s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	err = s.Setup.Disputekeeper.TallyVote(ctx, 1)
	s.NoError(err)

	bond := sdk.DefaultPowerReduction.MulRaw(int64(report.Power))
	// execute vote
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(ctx, 1))

	voteInfo, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(types.VoteResult_NO_QUORUM_MAJORITY_INVALID, voteInfo.VoteResult)

	val, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)
	s.NoError(err)
	s.True(val.Tokens.Equal(bond))
}

// func (s *IntegrationTestSuite) TestExecuteVoteSupport() {
// 	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)

// 	_, valAddrs, _ := s.createValidatorAccs([]uint64{1000})

// 	repAccs := s.CreateAccountsWithTokens(3, 100*1e6)
// 	disputer := s.newKeysWithTokens()

// 	delegators := repAccs
// 	valAddr := valAddrs[0]
// 	repAddr := sdk.AccAddress(valAddr)
// 	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAddr, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
// 	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAddr, reportertypes.NewSelection(repAddr, 1)))

// 	stake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr)
// 	s.NoError(err)
// 	disputerBefore, err := s.Setup.Stakingkeeper.GetAllDelegatorDelegations(s.Setup.Ctx, disputer)
// 	s.NoError(err)
// 	s.True(len(disputerBefore) == 0)

// 	// mint tokens to voters
// 	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))
// 	oracleServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
// 	msg := oracletypes.MsgTip{
// 		Tipper:    disputer.String(),
// 		QueryData: ethQueryData,
// 		Amount:    sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000)),
// 	}
// 	_, err = oracleServer.Tip(s.Setup.Ctx, &msg)
// 	s.Nil(err)

// 	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
// 	report := oracletypes.MicroReport{
// 		Reporter:  repAddr.String(),
// 		Power:     stake.Quo(sdk.DefaultPowerReduction).Uint64(),
// 		QueryId:   qId,
// 		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
// 		Timestamp: time.Unix(1696516597, 0),
// 	}
// 	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
// 	s.NoError(err)
// 	fivePercentBurn := disputeFee.MulRaw(1).QuoRaw(20)
// 	twoPercentBurn := fivePercentBurn.QuoRaw(2)
// 	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
// 		Creator:         disputer.String(),
// 		Report:          &report,
// 		Fee:             sdk.NewCoin(s.Setup.Denom, disputeFee),
// 		DisputeCategory: types.Warning,
// 	})
// 	s.NoError(err)
// 	s.NoError(dispute.CheckOpenDisputesForExpiration(s.Setup.Ctx, s.Setup.Disputekeeper))

// 	votersBalanceBefore := map[string]sdk.Coin{
// 		repAddr.String():       s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, repAddr, s.Setup.Denom),
// 		disputer.String():      s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom),
// 		delegators[1].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[1], s.Setup.Denom),
// 		delegators[2].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[2], s.Setup.Denom),
// 	}
// 	votes := []types.MsgVote{
// 		{
// 			Voter: repAddr.String(),
// 			Id:    1,
// 			Vote:  types.VoteEnum_VOTE_SUPPORT,
// 		},
// 		{
// 			Voter: disputer.String(),
// 			Id:    1,
// 			Vote:  types.VoteEnum_VOTE_SUPPORT,
// 		},
// 		{
// 			Voter: delegators[1].String(),
// 			Id:    1,
// 			Vote:  types.VoteEnum_VOTE_SUPPORT,
// 		},
// 		{
// 			Voter: delegators[2].String(),
// 			Id:    1,
// 			Vote:  types.VoteEnum_VOTE_SUPPORT,
// 		},
// 	}
// 	for i := range votes {
// 		_, err = msgServer.Vote(s.Setup.Ctx, &votes[i])
// 		if err != nil {
// 			s.Error(err, "voter power is zero")
// 		}
// 	}
// 	fmt.Println("rep", repAddr.String())
// 	err = s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1)
// 	s.NoError(err)
// 	// execute vote
// 	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 1))

// 	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{CallerAddress: disputer.String(), PayerAddress: disputer.String(), Id: 1})
// 	s.NoError(err)

// 	reporterAfter, err := s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, repAddr)
// 	s.NoError(err)
// 	// should still be jailed
// 	s.True(reporterAfter.Jailed)

// 	votersBalanceAfter := map[string]sdk.Coin{
// 		repAddr.String():       s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, repAddr, s.Setup.Denom),
// 		disputer.String():      s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom),
// 		delegators[1].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[1], s.Setup.Denom),
// 		delegators[2].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[2], s.Setup.Denom),
// 	}

// 	iter, err := s.Setup.Disputekeeper.Voter.Indexes.VotersById.MatchExact(s.Setup.Ctx, uint64(1))
// 	s.NoError(err)
// 	keys, err := iter.PrimaryKeys()
// 	s.NoError(err)
// 	voters := make([]keeper.VoterInfo, len(keys))
// 	totalVoterPower := math.ZeroInt()
// 	for i := range keys {
// 		v, err := s.Setup.Disputekeeper.Voter.Get(s.Setup.Ctx, keys[i])
// 		s.NoError(err)
// 		voters[i] = keeper.VoterInfo{Voter: keys[i].K2(), Power: v.VoterPower}
// 		totalVoterPower = totalVoterPower.Add(v.VoterPower)
// 	}
// 	votersReward, _ := s.Setup.Disputekeeper.CalculateVoterShare(s.Setup.Ctx, voters, twoPercentBurn, totalVoterPower)
// 	for i, v := range voters {
// 		voterBal := votersBalanceBefore[v.Voter.String()].AddAmount(votersReward[i].Share)
// 		if bytes.Equal(disputer, votersReward[i].Voter) {
// 			// disputer gets the dispute fee they paid minus the 5% burn for a one rounder dispute
// 			voterBal = voterBal.AddAmount(disputeFee.Sub(fivePercentBurn))
// 		}
// 		s.Equal(voterBal, votersBalanceAfter[v.Voter.String()])
// 	}
// 	disputerDelgation, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, disputer)
// 	s.NoError(err)
// 	fmt.Println(disputerDelgation)
// 	s.True(disputerDelgation.Equal(math.NewInt(20_000_000)))
// }

// func (s *IntegrationTestSuite) TestExecuteVoteAgainst() {
// 	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)

// 	_, valAddrs, _ := s.createValidatorAccs([]uint64{1000})

// 	repAccs := s.CreateAccountsWithTokens(3, 100*1e6)
// 	disputer := s.newKeysWithTokens()

// 	valAddr := valAddrs[0]
// 	repAddr := sdk.AccAddress(valAddr)
// 	delegators := repAccs
// 	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAddr, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
// 	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAddr, reportertypes.NewSelection(repAddr, 1)))

// 	stake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr)
// 	s.NoError(err)

// 	// tip to capture other group of voters 25% of the total power
// 	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))
// 	oracleServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
// 	msg := oracletypes.MsgTip{
// 		Tipper:    disputer.String(),
// 		QueryData: ethQueryData,
// 		Amount:    sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000)),
// 	}
// 	_, err = oracleServer.Tip(s.Setup.Ctx, &msg)
// 	s.Nil(err)

// 	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
// 	report := oracletypes.MicroReport{
// 		Reporter:  repAddr.String(),
// 		Power:     stake.Quo(sdk.DefaultPowerReduction).Uint64(),
// 		QueryId:   qId,
// 		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
// 		Timestamp: time.Unix(1696516597, 0),
// 	}
// 	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
// 	s.NoError(err)

// 	fivePercentBurn := disputeFee.MulRaw(1).QuoRaw(20)
// 	twoPercentBurn := fivePercentBurn.QuoRaw(2)
// 	// disputeFeeMinusBurn := disputeFee.Sub(disputeFee.MulRaw(1).QuoRaw(20))

// 	// Propose dispute pay half of the fee from account
// 	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
// 		Creator:         disputer.String(),
// 		Report:          &report,
// 		Fee:             sdk.NewCoin(s.Setup.Denom, disputeFee),
// 		DisputeCategory: types.Warning,
// 	})
// 	s.NoError(err)
// 	votersBalanceBefore := map[string]sdk.Coin{
// 		repAddr.String():       s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, repAddr, s.Setup.Denom),
// 		disputer.String():      s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom),
// 		delegators[1].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[1], s.Setup.Denom),
// 		delegators[2].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[2], s.Setup.Denom),
// 	}
// 	votes := []types.MsgVote{
// 		{
// 			Voter: repAddr.String(),
// 			Id:    1,
// 			Vote:  types.VoteEnum_VOTE_AGAINST,
// 		},
// 		{
// 			Voter: disputer.String(),
// 			Id:    1,
// 			Vote:  types.VoteEnum_VOTE_AGAINST,
// 		},
// 		{
// 			Voter: delegators[1].String(),
// 			Id:    1,
// 			Vote:  types.VoteEnum_VOTE_AGAINST,
// 		},
// 		{
// 			Voter: delegators[2].String(),
// 			Id:    1,
// 			Vote:  types.VoteEnum_VOTE_AGAINST,
// 		},
// 	}
// 	for i := range votes {
// 		_, err = msgServer.Vote(s.Setup.Ctx, &votes[i])
// 		if err != nil {
// 			s.Error(err, "voter power is zero")
// 		}
// 	}
// 	val, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)
// 	s.NoError(err)
// 	fmt.Println(val.Tokens)
// 	// tally vote
// 	err = s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1)
// 	s.NoError(err)
// 	// execute vote
// 	err = s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 1)
// 	s.NoError(err)
// 	// reporterAfterDispute, err := s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, repAddr)
// 	// s.NoError(err)

// 	// s.Equal(stake.Add(disputeFeeMinusBurn), reporterAfterDispute.TotalTokens)
// 	votersBalanceAfter := map[string]sdk.Coin{
// 		repAddr.String():       s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, repAddr, s.Setup.Denom),
// 		disputer.String():      s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom),
// 		delegators[1].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[1], s.Setup.Denom),
// 		delegators[2].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[2], s.Setup.Denom),
// 	}

// 	iter, err := s.Setup.Disputekeeper.Voter.Indexes.VotersById.MatchExact(s.Setup.Ctx, uint64(1))
// 	s.NoError(err)
// 	keys, err := iter.PrimaryKeys()
// 	s.NoError(err)
// 	voters := make([]keeper.VoterInfo, len(keys))
// 	totalVoterPower := math.ZeroInt()
// 	for i := range keys {
// 		v, err := s.Setup.Disputekeeper.Voter.Get(s.Setup.Ctx, keys[i])
// 		s.NoError(err)
// 		voters[i] = keeper.VoterInfo{Voter: keys[i].K2(), Power: v.VoterPower, Share: math.ZeroInt()}
// 		totalVoterPower = totalVoterPower.Add(v.VoterPower)
// 	}
// 	votersReward, _ := s.Setup.Disputekeeper.CalculateVoterShare(s.Setup.Ctx, voters, twoPercentBurn, totalVoterPower)

// 	for _, v := range votersReward {
// 		newBal := votersBalanceBefore[v.Voter.String()].Amount.Add(v.Share)
// 		// votersBalanceBefore[votersReward[i].Voter.String()].Amount = votersBalanceBefore[i].Amount.Add(votersReward[i].Share)
// 		s.Equal(newBal, votersBalanceAfter[v.Voter.String()].Amount)
// 	}
// }

func (s *IntegrationTestSuite) TestDisputeMultipleRounds() {
	repAccs, _, _ := s.createValidatorAccs([]uint64{100, 200})
	reporter1Acc := repAccs[0]
	reporter2Acc := repAccs[1]
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewSelection(reporter1Acc, 1)))

	reporter1StakeBefore, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    reporter1Acc.String(),
		Power:       reporter1StakeBefore.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)
	burnAmount := disputeFee.MulRaw(1).QuoRaw(20)
	disputer := s.newKeysWithTokens()
	// mint disputer tokens
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))
	// disputer balance before proposing dispute
	disputerBalanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	disputeMsg := types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory: types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)
	// check disputer balance after proposing dispute
	disputerBalanceAfter1stRound := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	s.True(disputerBalanceBefore.Amount.GT(disputerBalanceAfter1stRound.Amount))
	// assert reporter tokens slashed and reporter jailed
	// rep1Tokens, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, reporter1Acc)
	// s.NoError(err)
	reporter1, err := s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	// reporter1TokensAfterDispute1stround := rep1Tokens
	s.True(reporter1.Jailed)
	// s.True(reporter1.TotalTokens.LT(reporter1StakeBefore))
	// s.Equal(reporter1.TotalTokens, reporter1StakeBefore.Sub(disputeFee))

	voteMsg := types.MsgVote{
		Voter: reporter2Acc.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}
	_, err = msgServer.Vote(s.Setup.Ctx, &voteMsg)
	s.NoError(err)

	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.Error(err, "can't start a new round for this dispute 1; dispute status DISPUTE_STATUS_VOTING")
	// forward time to end voting period pre execute vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.TWO_DAYS + 1))
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1))
	s.ErrorContains(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1), "vote already tallied")
	s.Error(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 1), "dispute is not resolved yet")
	// start another dispute round
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)
	disputerBalanceAfter2ndRound := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	s.Equal(disputerBalanceAfter1stRound.Amount.Sub(burnAmount.MulRaw(2)), disputerBalanceAfter2ndRound.Amount)
	reporter1, err = s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)

	s.Error(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 2), "vote period not ended and quorum not reached")

	// voting that doesn't reach quorum
	voteMsg = types.MsgVote{
		Voter: reporter2Acc.String(),
		Id:    2,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}

	_, err = msgServer.Vote(s.Setup.Ctx, &voteMsg)
	s.NoError(err)

	// expire vote period
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 2))
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))
	// attempt to start another round
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.Error(err, "can't start a new round for this dispute 2; dispute status DISPUTE_STATUS_RESOLVED")
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)
	vote, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.True(vote.Executed)
}

func (s *IntegrationTestSuite) TestNoQorumSingleRound() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	repAccs, _, _ := s.createValidatorAccs([]uint64{100, 200})
	reporter1Acc := repAccs[0]
	reporter2Acc := repAccs[1]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewSelection(reporter1Acc, 1)))

	reporter1StakeBefore, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    reporter1Acc.String(),
		Power:       reporter1StakeBefore.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)

	disputer := s.newKeysWithTokens()
	// mint disputer tokens
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))

	disputeMsg := types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory: types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)

	voteMsg := types.MsgVote{
		Voter: reporter2Acc.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}

	_, err = msgServer.Vote(s.Setup.Ctx, &voteMsg)
	s.NoError(err)
	// forward time to expire dispute
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1))
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 1))
}

func (s *IntegrationTestSuite) TestDisputeButNoVotes() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	repAccs, _, _ := s.createValidatorAccs([]uint64{100})
	reporter1Acc := repAccs[0]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewSelection(reporter1Acc, 1)))

	reporterStakeBefore, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    reporter1Acc.String(),
		Power:       reporterStakeBefore.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}

	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)

	disputer := s.newKeysWithTokens()
	// mint disputer tokens
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))

	disputeMsg := types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory: types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)

	// forward time to expire dispute
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))

	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1))
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 1))
}

func (s *IntegrationTestSuite) TestFlagReport() {
	// three micro reports
	// setAggregate
	// then dispute report to check if its flagged
	valAccs, _, _ := s.createValidatorAccs([]uint64{100, 200, 300})
	reporter1 := valAccs[0]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter1, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter1, reportertypes.NewSelection(reporter1, 1)))

	stake1, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1)
	s.NoError(err)
	reporter2 := valAccs[1]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter2, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter2, reportertypes.NewSelection(reporter2, 1)))
	stake2, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter2)
	s.NoError(err)
	reporter3 := valAccs[2]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter3, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter3, reportertypes.NewSelection(reporter3, 1)))
	stake3, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter3)
	s.NoError(err)

	queryid, err := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	s.NoError(err)
	aggmethod := "weighted-median"
	s.NoError(err)

	report1 := oracletypes.MicroReport{
		Reporter:        reporter1.String(),
		Power:           uint64(sdk.TokensToConsensusPower(stake1, sdk.DefaultPowerReduction)),
		QueryId:         queryid,
		QueryType:       "SpotPrice",
		AggregateMethod: aggmethod,
		Value:           testutil.EncodeValue(1.00),
		Timestamp:       s.Setup.Ctx.BlockTime(),
		Cyclelist:       true,
		BlockNumber:     uint64(s.Setup.Ctx.BlockHeight()),
	}
	report2 := oracletypes.MicroReport{
		Reporter:        reporter2.String(),
		Power:           uint64(sdk.TokensToConsensusPower(stake2, sdk.DefaultPowerReduction)),
		QueryId:         queryid,
		QueryType:       "SpotPrice",
		AggregateMethod: aggmethod,
		Value:           testutil.EncodeValue(2.00),
		Timestamp:       s.Setup.Ctx.BlockTime(),
		Cyclelist:       true,
		BlockNumber:     uint64(s.Setup.Ctx.BlockHeight()),
	}
	report3 := oracletypes.MicroReport{
		Reporter:        reporter3.String(),
		Power:           uint64(sdk.TokensToConsensusPower(stake3, sdk.DefaultPowerReduction)),
		QueryId:         queryid,
		QueryType:       "SpotPrice",
		AggregateMethod: aggmethod,
		Value:           testutil.EncodeValue(3.00),
		Timestamp:       s.Setup.Ctx.BlockTime(),
		Cyclelist:       true,
		BlockNumber:     uint64(s.Setup.Ctx.BlockHeight()),
	}

	// forward time
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	// set report
	err = s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report1.QueryId, reporter1.Bytes(), uint64(1)), report1)
	s.NoError(err)
	err = s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report2.QueryId, reporter2.Bytes(), uint64(1)), report2)
	s.NoError(err)
	err = s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report3.QueryId, reporter3.Bytes(), uint64(1)), report3)
	s.NoError(err)

	// add query
	s.NoError(s.Setup.Oraclekeeper.Query.Set(s.Setup.Ctx, collections.Join(queryid, uint64(1)), oracletypes.QueryMeta{Id: 1, HasRevealedReports: true}))
	// set aggregate
	err = s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx)
	s.NoError(err)

	// get aggregate
	agg, err := s.Setup.Oraclekeeper.Aggregates.Get(s.Setup.Ctx, collections.Join(queryid, uint64(s.Setup.Ctx.BlockTime().UnixMilli())))
	s.NoError(err)
	s.Equal(agg.AggregateReporter, reporter2.String())
	s.False(agg.Flagged)

	// dispute reporter2 report
	disputer := s.newKeysWithTokens()
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))

	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report2, types.Warning)
	s.NoError(err)

	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	disputeMsg := types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report2,
		Fee:             sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory: types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)

	// check if aggregate is flagged
	agg, err = s.Setup.Oraclekeeper.Aggregates.Get(s.Setup.Ctx, collections.Join(queryid, uint64(s.Setup.Ctx.BlockTime().UnixMilli())))
	s.NoError(err)
	s.True(agg.Flagged)
}

func (s *IntegrationTestSuite) TestAddFeeToDisputeNotBond() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	repAccs, _, _ := s.createValidatorAccs([]uint64{100})
	reporter1Acc := repAccs[0]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewSelection(reporter1Acc, 1)))
	reporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    reporter1Acc.String(),
		Power:       reporterStake.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}

	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)

	disputer := s.newKeysWithTokens()
	// mint disputer tokens
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))
	// propose dispute with half the fee
	disputeMsg := types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.Setup.Denom, disputeFee.QuoRaw(2)),
		DisputeCategory: types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)

	// check if dispute is started
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(types.Prevote, dispute.DisputeStatus)

	// disputer balance before adding fee
	disputerBalanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	// add fee to dispute with more than left over
	msgAddFee := types.MsgAddFeeToDispute{
		Creator:     disputer.String(),
		DisputeId:   1,
		Amount:      sdk.NewCoin(s.Setup.Denom, disputeFee),
		PayFromBond: false,
	}
	_, err = msgServer.AddFeeToDispute(s.Setup.Ctx, &msgAddFee)
	s.NoError(err)

	// balance should only decrease by half the fee (remaining fee)
	disputerBalanceAfter := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	s.Equal(disputerBalanceBefore.Amount.Sub(disputeFee.QuoRaw(2)), disputerBalanceAfter.Amount)
}

func (s *IntegrationTestSuite) TestAddFeeToDisputeBond() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	repAccs, _, _ := s.createValidatorAccs([]uint64{100, 200})
	reporter1Acc := repAccs[0]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewSelection(reporter1Acc, 1)))
	reporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:    reporter1Acc.String(),
		Power:       reporterStake.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}

	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)

	disputer := repAccs[1]
	// mint disputer tokens
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))
	// propose dispute with half the fee
	disputeMsg := types.MsgProposeDispute{
		Creator:         disputer.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(s.Setup.Denom, disputeFee.QuoRaw(2)),
		DisputeCategory: types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)

	// check if dispute is started
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(types.Prevote, dispute.DisputeStatus)

	// disputer balance before adding fee
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, disputer, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, disputer, reportertypes.NewSelection(disputer, 1)))
	feePayerStakeBefore, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, disputer)
	s.NoError(err)
	// add fee to dispute with more than left over
	msgAddFee := types.MsgAddFeeToDispute{
		Creator:     disputer.String(),
		DisputeId:   1,
		Amount:      sdk.NewCoin(s.Setup.Denom, disputeFee),
		PayFromBond: true,
	}
	_, err = msgServer.AddFeeToDispute(s.Setup.Ctx, &msgAddFee)
	s.NoError(err)

	// balance should only decrease by half the fee (remaining fee)
	feePayerStakeAfter, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, disputer)
	s.NoError(err)
	s.Equal(feePayerStakeBefore.Sub(disputeFee.QuoRaw(2)), feePayerStakeAfter)
}
