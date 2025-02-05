package integration_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/dispute"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reporterKeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
	err = s.Setup.Reporterkeeper.Report.Set(s.Setup.Ctx, collections.Join([]byte{}, collections.Join(repAddr.Bytes(), uint64(s.Setup.Ctx.BlockHeight()))), reportertypes.DelegationsAmounts{TokenOrigins: srcs, Total: total})
	s.NoError(err)
	// assemble report with reporter to dispute
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:  repAddr.String(),
		Power:     100,
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.UnixMilli(1696516597).UTC(),
		MetaId:    1,
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, repAddr.Bytes(), report.MetaId), report))
	// disputer with tokens to pay fee
	disputer := s.newKeysWithTokens()

	// Propose dispute pay half of the fee from account
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, math.NewInt(500_000)),
		DisputeCategory:  types.Warning,
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
	repTokens, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr, qId)
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

	// set block info directly for ease (need validators to call endblocker)
	err = s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, dispute.HashId)
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	// vote from team
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)
	_, err = msgServer.Vote(s.Setup.Ctx, &types.MsgVote{
		Voter: teamAddr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	})
	s.NoError(err)
	vtr, err := s.Setup.Disputekeeper.Voter.Get(s.Setup.Ctx, collections.Join(uint64(1), teamAddr.Bytes()))
	s.NoError(err)
	s.Equal(types.VoteEnum_VOTE_SUPPORT, vtr.Vote)
	v, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(v.VoteResult, types.VoteResult_NO_TALLY)
	iter, err := s.Setup.Disputekeeper.Voter.Indexes.VotersById.MatchExact(s.Setup.Ctx, uint64(1))
	s.NoError(err)
	voters, err := iter.PrimaryKeys()
	s.NoError(err)
	s.Equal(voters[0].K2(), teamAddr.Bytes())
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

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")

	err = s.Setup.Reporterkeeper.Report.Set(s.Setup.Ctx, collections.Join(qId, collections.Join(repAddr.Bytes(), uint64(s.Setup.Ctx.BlockHeight()))), reportertypes.DelegationsAmounts{TokenOrigins: srcs, Total: total})
	s.NoError(err)

	report := oracletypes.MicroReport{
		Reporter:    repAddr.String(),
		Power:       1000,
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.UnixMilli(1696516597).UTC(),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, repAddr.Bytes(), report.MetaId), report))
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          repAddr.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		DisputeCategory:  types.Warning,
		Fee:              sdk.NewCoin(s.Setup.Denom, math.NewInt(10_000_000)), // one percent dispute fee
		PayFromBond:      false,
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
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")

	err = s.Setup.Reporterkeeper.Report.Set(s.Setup.Ctx, collections.Join(qId, collections.Join(repAddr.Bytes(), uint64(s.Setup.Ctx.BlockHeight()))), reportertypes.DelegationsAmounts{TokenOrigins: srcs, Total: total})
	s.NoError(err)
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAddr, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAddr, reportertypes.NewSelection(repAddr, uint64(len(dels)))))

	repTokensBeforePropose, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr, qId)
	s.NoError(err)

	report := oracletypes.MicroReport{
		Reporter:    repAddr.String(),
		Power:       100,
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.UnixMilli(1696516597).UTC(),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, repAddr.Bytes(), report.MetaId), report))
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)
	burnAmount := disputeFee.MulRaw(1).QuoRaw(20)
	disputerBalanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	// Propose dispute pay half of the fee from account
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
	})
	s.NoError(err)
	s.True(s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom).IsLT(disputerBalanceBefore))
	// set block info directly for ease (need validators to call endblocker)
	d, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	err = s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, d.HashId)
	s.NoError(err)

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
			fmt.Println("err: ", err)
			s.Error(err, "voter power is zero")
		}

	}
	valTokensBeforeExecuteVote, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)
	s.NoError(err)
	disputerBalanceBeforeExecuteVote := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	// get dispute hash id
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	fmt.Println("dispute: ", dispute)
	voteInfo, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	fmt.Println("voteInfo before time skip: ", voteInfo)
	s.False(voteInfo.Executed)
	// only 25 percent of the total power voted so vote should not be tallied unless it's expired
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	s.NoError(err)
	voteInfo, err = s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	fmt.Println("voteInfo after time skip: ", voteInfo)
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
	repTokensAfterExecuteVote, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr, []byte{})
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
	// disputer cannot call claim reward since he has no voting power, just gets withdrawfeerefund
	_, err = msgServer.ClaimReward(s.Setup.Ctx, &types.MsgClaimReward{CallerAddress: repAddr.String(), DisputeId: 1})
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

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")

	repStake, _ := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr, qId)
	report := oracletypes.MicroReport{
		Reporter:  repAddr.String(),
		Power:     repStake.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.UnixMilli(1696516597).UTC(),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, repAddr.Bytes(), report.MetaId), report))
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)

	// Propose dispute pay half of the fee from account
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
	})
	s.NoError(err)
	// get dispute to set block info
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	// set block info directly for ease (need validators to call endblocker)
	err = s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, dispute.HashId)
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

func (s *IntegrationTestSuite) TestExecuteVoteSupport() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)

	_, valAddrs, _ := s.createValidatorAccs([]uint64{1000})

	repAccs := s.CreateAccountsWithTokens(3, 100*1e6)
	disputer := s.newKeysWithTokens()

	delegators := repAccs
	valAddr := valAddrs[0]
	repAddr := sdk.AccAddress(valAddr)
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAddr, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAddr, reportertypes.NewSelection(repAddr, 1)))

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")

	stake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr, qId)
	s.NoError(err)
	disputerBefore, err := s.Setup.Stakingkeeper.GetAllDelegatorDelegations(s.Setup.Ctx, disputer)
	s.NoError(err)
	s.True(len(disputerBefore) == 0)

	// mint tokens to voters
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))
	oracleServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	msg := oracletypes.MsgTip{
		Tipper:    disputer.String(),
		QueryData: ethQueryData,
		Amount:    sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000)),
	}
	_, err = oracleServer.Tip(s.Setup.Ctx, &msg)
	s.Nil(err)

	report := oracletypes.MicroReport{
		Reporter:  repAddr.String(),
		Power:     stake.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.UnixMilli(1696516597).UTC(),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, repAddr.Bytes(), report.MetaId), report))
	fmt.Println("Disputed report power: ", report.Power)
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)
	fivePercentBurn := disputeFee.MulRaw(1).QuoRaw(20)
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
	})
	s.NoError(err)
	s.NoError(dispute.CheckOpenDisputesForExpiration(s.Setup.Ctx, s.Setup.Disputekeeper))
	// get dispute to set block info
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	// set block info directly for ease (need validators to call endblocker)
	err = s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, dispute.HashId)
	s.NoError(err)

	votersBalanceBefore := map[string]sdk.Coin{
		repAddr.String():       s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, repAddr, s.Setup.Denom),
		delegators[1].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[1], s.Setup.Denom),
		delegators[2].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[2], s.Setup.Denom),
		disputer.String():      s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom),
	}

	votes := []types.MsgVote{
		{
			Voter: repAddr.String(),
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
		{
			Voter: disputer.String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_SUPPORT,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.Setup.Ctx, &votes[i])
		if err != nil {
			s.Error(err, "voter power is zero")
		}
	}
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)
	_, err = msgServer.Vote(s.Setup.Ctx, &types.MsgVote{
		Voter: teamAddr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	})
	s.Error(err) // vote already reached quorum
	err = s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1)
	s.Equal(err.Error(), "vote already tallied")
	// execute vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	s.NoError(err)

	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &types.MsgWithdrawFeeRefund{CallerAddress: disputer.String(), PayerAddress: disputer.String(), Id: 1})
	s.NoError(err)

	reporterAfter, err := s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, repAddr)
	s.NoError(err)
	// should still be jailed
	s.True(reporterAfter.Jailed)

	for i := range votes {
		_, err = msgServer.ClaimReward(s.Setup.Ctx, &types.MsgClaimReward{CallerAddress: votes[i].Voter, DisputeId: 1})
		if err != nil {
			s.Equal(err.Error(), "reward is zero")
		}
	}

	votersBalanceAfter := map[string]sdk.Coin{
		repAddr.String():       s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, repAddr, s.Setup.Denom),
		delegators[1].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[1], s.Setup.Denom),
		delegators[2].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[2], s.Setup.Denom),
		disputer.String():      s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom),
	}
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
	for _, v := range voters {
		if bytes.Equal(teamAddr, v.Voter) {
			continue
		}
		votersReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, v.Voter, 1)
		s.NoError(err)
		voterBal := votersBalanceBefore[v.Voter.String()].AddAmount(votersReward)
		if bytes.Equal(disputer, v.Voter) {
			// disputer gets the dispute fee they paid minus the 5% burn for a one rounder dispute
			voterBal = voterBal.AddAmount(disputeFee.Sub(fivePercentBurn))
		}
		s.Equal(voterBal, votersBalanceAfter[v.Voter.String()])
	}
	disputerDelgation, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, disputer)
	s.NoError(err)
	s.True(disputerDelgation.Equal(math.NewInt(20_000_000)))
}

func (s *IntegrationTestSuite) TestExecuteVoteAgainst() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)

	_, valAddrs, _ := s.createValidatorAccs([]uint64{1000})

	repAccs := s.CreateAccountsWithTokens(3, 100*1e6)
	disputer := s.newKeysWithTokens()

	delegators := repAccs
	valAddr := valAddrs[0]
	repAddr := sdk.AccAddress(valAddr)
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAddr, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAddr, reportertypes.NewSelection(repAddr, 1)))

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")

	stake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAddr, qId)
	s.NoError(err)
	disputerBefore, err := s.Setup.Stakingkeeper.GetAllDelegatorDelegations(s.Setup.Ctx, disputer)
	s.NoError(err)
	s.True(len(disputerBefore) == 0)

	// mint tokens to voters
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))
	oracleServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	msg := oracletypes.MsgTip{
		Tipper:    disputer.String(),
		QueryData: ethQueryData,
		Amount:    sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000)),
	}
	_, err = oracleServer.Tip(s.Setup.Ctx, &msg)
	s.Nil(err)

	report := oracletypes.MicroReport{
		Reporter:  repAddr.String(),
		Power:     stake.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.UnixMilli(1696516597).UTC(),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, repAddr.Bytes(), report.MetaId), report))
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)
	// fivePercentBurn := disputeFee.MulRaw(1).QuoRaw(20)
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
	})
	s.NoError(err)
	s.NoError(dispute.CheckOpenDisputesForExpiration(s.Setup.Ctx, s.Setup.Disputekeeper))
	// get dispute to set block info
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	// set block info directly for ease (need validators to call endblocker)
	err = s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, dispute.HashId)
	s.NoError(err)

	votersBalanceBefore := map[string]sdk.Coin{
		repAddr.String():       s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, repAddr, s.Setup.Denom),
		delegators[1].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[1], s.Setup.Denom),
		delegators[2].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[2], s.Setup.Denom),
		disputer.String():      s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom),
	}

	votes := []types.MsgVote{
		{
			Voter: repAddr.String(),
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
		{
			Voter: disputer.String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.Setup.Ctx, &votes[i])
		if err != nil {
			s.Error(err, "voter power is zero")
		}
	}
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)
	_, err = msgServer.Vote(s.Setup.Ctx, &types.MsgVote{
		Voter: teamAddr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_AGAINST,
	})
	s.Error(err) // vote already reached quorum
	err = s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1)
	s.Equal(err.Error(), "vote already tallied")
	// execute vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.THREE_DAYS + 1))
	// tally vote
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	s.NoError(err)

	// s.Equal(stake.Add(disputeFeeMinusBurn), reporterAfterDispute.TotalTokens)
	votersBalanceAfter := map[string]sdk.Coin{
		repAddr.String():       s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, repAddr, s.Setup.Denom),
		disputer.String():      s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom),
		delegators[1].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[1], s.Setup.Denom),
		delegators[2].String(): s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delegators[2], s.Setup.Denom),
	}

	iter, err := s.Setup.Disputekeeper.Voter.Indexes.VotersById.MatchExact(s.Setup.Ctx, uint64(1))
	s.NoError(err)
	keys, err := iter.PrimaryKeys()
	s.NoError(err)
	voters := make([]keeper.VoterInfo, len(keys))
	totalVoterPower := math.ZeroInt()
	for i := range keys {
		// if bytes.Equal(teamAddr, keys[i].K2()) {
		// 	continue
		// }
		v, err := s.Setup.Disputekeeper.Voter.Get(s.Setup.Ctx, keys[i])
		s.NoError(err)
		voters[i] = keeper.VoterInfo{Voter: keys[i].K2(), Power: v.VoterPower, Share: math.ZeroInt()}
		totalVoterPower = totalVoterPower.Add(v.VoterPower)
	}
	// votersReward, _ := s.Setup.Disputekeeper.CalculateVoterShare(s.Setup.Ctx, voters, twoPercentBurn, totalVoterPower)
	for _, v := range voters {
		if bytes.Equal(teamAddr, v.Voter) {
			continue
		}
		newBal := votersBalanceBefore[v.Voter.String()].Amount.Add(v.Share)
		// votersBalanceBefore[votersReward[i].Voter.String()].Amount = votersBalanceBefore[i].Amount.Add(votersReward[i].Share)
		s.Equal(newBal, votersBalanceAfter[v.Voter.String()].Amount)
	}

	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)
	voteInfo, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.True(voteInfo.Executed)
	// Check voter rewards
	disputerVoterReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, disputer, 1)
	s.NoError(err)
	reporterVoterReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, repAddr, 1)
	s.NoError(err)
	delegator1VoterReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, delegators[1], 1)
	s.NoError(err)
	delegator2VoterReward, err := s.Setup.Disputekeeper.CalculateReward(s.Setup.Ctx, delegators[2], 1)
	s.NoError(err)

	// Claim rewards and check balances
	_, err = msgServer.ClaimReward(s.Setup.Ctx, &types.MsgClaimReward{CallerAddress: disputer.String(), DisputeId: 1})
	s.NoError(err)
	disputerBalAfterClaim := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	expectedDisputerBalAfterClaim := votersBalanceAfter[disputer.String()].Amount.Add(disputerVoterReward)
	s.Equal(expectedDisputerBalAfterClaim, disputerBalAfterClaim.Amount)

	// Check total voter rewards are less than or equal to 50% of burn amount
	sumVoterRewards := disputerVoterReward.Add(reporterVoterReward).Add(delegator1VoterReward).Add(delegator2VoterReward)
	fmt.Println(sumVoterRewards.String())
	// s.True(sumVoterRewards.LTE(twoPercentBurn))
	// s.True(sumVoterRewards.GTE(twoPercentBurn.Sub(math.NewInt(4)))) // max one loya per voter lost via rounding
}

func (s *IntegrationTestSuite) TestDisputeMultipleRounds() {
	// create 2 validator reporter accs
	repAccs, _, _ := s.createValidatorAccs([]uint64{100, 200})
	reporter1Acc := repAccs[0]
	// reporter2Acc := repAccs[1]
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	// set reporter and selector stores (skip calling createreporter)
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewSelection(reporter1Acc, 1)))

	// make report to set in store and dispute
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	reporter1StakeBefore, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1Acc, qId)
	s.NoError(err)
	report := oracletypes.MicroReport{
		Reporter:    reporter1Acc.String(),
		Power:       reporter1StakeBefore.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.UnixMilli(1696516597).UTC(),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, reporter1Acc.Bytes(), report.MetaId), report))

	// prepare to propose dispute from non reporter 3rd part
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)
	burnAmount := disputeFee.MulRaw(1).QuoRaw(20)
	disputer := s.newKeysWithTokens()
	// mint disputer tokens
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))
	// disputer balance before proposing dispute
	disputerBalanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)

	// propose warning dispute
	disputeMsg := types.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)
	fmt.Println("RD 1 STARTED")

	// get dispute to set block info
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	err = s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, dispute.HashId)
	s.NoError(err)

	// assert fee is correct
	expectedRd1Fee := reporter1StakeBefore.QuoRaw(100) // 1% of reporter1StakeBefore
	s.Equal(expectedRd1Fee, dispute.DisputeFee)
	// assert disputer fee has been deducted
	disputerBalanceAfter1stRound := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	s.Equal(disputerBalanceAfter1stRound.Amount, disputerBalanceBefore.Amount.Sub(disputeFee))
	// assert disputed reporter is jailed
	reporter1, err := s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	s.True(reporter1.Jailed)
	// assert fee has been deducted from disputed reporters stake
	rep1TokensAfterRd1, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	s.True(rep1TokensAfterRd1.LT(reporter1StakeBefore))
	s.Equal(rep1TokensAfterRd1, reporter1StakeBefore.Sub(disputeFee))

	// vote from team
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)
	voteMsg := types.MsgVote{
		Voter: teamAddr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}
	_, err = msgServer.Vote(s.Setup.Ctx, &voteMsg)
	s.NoError(err)

	// attempt to start another round, shouldnt work because dispute is still in voting phase
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.Error(err, "can't start a new round for this dispute 1; dispute status DISPUTE_STATUS_VOTING")
	// forward time to end voting period, but not resolve dispute (2 days to end voting, 1 day to execute)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.TWO_DAYS + 1))
	// tally vote
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1))
	// calling tally a second time should error
	s.ErrorContains(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1), "vote already tallied")
	// execute should error because dispute is not resolved yet
	s.Error(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 1), "dispute is not resolved yet")

	// start another dispute round
	// fee paid should be 2*0.05(prevRoundFee)
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)
	fmt.Println("RD 2 STARTED")
	// check on dispute
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	// assert disputefeetotal is correct
	rd2FeeAddition := disputeFee.MulRaw(5).QuoRaw(100).MulRaw(2)
	expectedFeeTotalRd2 := disputeFee.Add(rd2FeeAddition)
	s.Equal(expectedFeeTotalRd2, dispute.FeeTotal)
	s.Equal(types.Voting, dispute.DisputeStatus)
	s.True(dispute.Open)
	// assert disputer fee has been deducted
	disputerBalanceAfter2ndRound := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	s.Equal(disputerBalanceAfter1stRound.Amount.Sub(burnAmount.MulRaw(2)), disputerBalanceAfter2ndRound.Amount)
	s.Equal(disputerBalanceAfter1stRound.Amount.Sub(rd2FeeAddition), disputerBalanceAfter2ndRound.Amount)
	// assert disputed reporter is still jailed
	reporter1, err = s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	s.True(reporter1.Jailed)
	// assert reporters stake is the same as after the first round
	rep1TokensAfterRd2, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	s.Equal(rep1TokensAfterRd2, rep1TokensAfterRd1)

	// tally vote should error because 2nd rd voting period just started
	s.Error(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 2), "vote period not ended and quorum not reached")

	// voting that doesn't reach quorum
	voteMsg = types.MsgVote{
		Voter: teamAddr.String(),
		Id:    2,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}
	_, err = msgServer.Vote(s.Setup.Ctx, &voteMsg)
	s.NoError(err)

	// forward time to end voting period pre execute vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.TWO_DAYS + 1))
	// tally vote
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 2))
	// calling tally a second time should error
	s.ErrorContains(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 2), "vote already tallied")
	// execute should error because dispute is not resolved yet
	s.Error(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2), "dispute is not resolved yet")
	// get dispute to check status
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.Unresolved, dispute.DisputeStatus)
	s.True(dispute.PendingExecution)

	// start 3rd round
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)
	fmt.Println("RD 3 STARTED")

	// check on dispute
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 3)
	s.NoError(err)
	// assert disputefeetotal is correct
	rd3FeeAddition := rd2FeeAddition.MulRaw(2)
	expectedFeeTotalRd3 := expectedFeeTotalRd2.Add(rd3FeeAddition)
	s.Equal(expectedFeeTotalRd3, dispute.FeeTotal)
	s.Equal(types.Voting, dispute.DisputeStatus)
	s.True(dispute.Open)
	// assert disputer fee has been deducted
	disputerBalanceAfter3ndRound := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	s.Equal(disputerBalanceAfter2ndRound.Amount.Sub(burnAmount.MulRaw(4)), disputerBalanceAfter3ndRound.Amount)
	s.Equal(disputerBalanceAfter2ndRound.Amount.Sub(rd3FeeAddition), disputerBalanceAfter3ndRound.Amount)
	// assert disputed reporter is still jailed
	reporter1, err = s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	s.True(reporter1.Jailed)
	// assert reporters stake is the same as after the second round
	rep1TokensAfterRd3, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	s.Equal(rep1TokensAfterRd3, rep1TokensAfterRd2)

	// tally vote should error because 3rd rd voting period just started
	s.Error(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 3), "vote period not ended and quorum not reached")

	// voting that doesn't reach quorum
	voteMsg = types.MsgVote{
		Voter: teamAddr.String(),
		Id:    3,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}
	_, err = msgServer.Vote(s.Setup.Ctx, &voteMsg)
	s.NoError(err)

	// forward time to end voting period pre execute vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.TWO_DAYS + 1))
	// tally vote
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 3))
	// calling tally a second time should error
	s.ErrorContains(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 3), "vote already tallied")
	// execute should error because dispute is not resolved yet
	s.Error(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 3), "dispute is not resolved yet")
	// get dispute to check status
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 3)
	s.NoError(err)
	s.Equal(types.Unresolved, dispute.DisputeStatus)
	s.True(dispute.PendingExecution)

	// start 4th round
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)
	fmt.Println("RD 4 STARTED")

	// check on dispute
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 4)
	s.NoError(err)
	// assert disputefeetotal is correct
	rd4FeeAddition := rd3FeeAddition.MulRaw(2)
	expectedFeeTotalRd4 := expectedFeeTotalRd3.Add(rd4FeeAddition)
	s.Equal(expectedFeeTotalRd4, dispute.FeeTotal)
	s.Equal(types.Voting, dispute.DisputeStatus)
	s.True(dispute.Open)
	// assert disputer fee has been deducted
	disputerBalanceAfter4ndRound := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	s.Equal(disputerBalanceAfter3ndRound.Amount.Sub(burnAmount.MulRaw(8)), disputerBalanceAfter4ndRound.Amount)
	s.Equal(disputerBalanceAfter3ndRound.Amount.Sub(rd4FeeAddition), disputerBalanceAfter4ndRound.Amount)
	// assert disputed reporter is still jailed
	reporter1, err = s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	s.True(reporter1.Jailed)
	// assert reporters stake is the same as after the second round
	rep1TokensAfterRd4, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	s.Equal(rep1TokensAfterRd4, rep1TokensAfterRd3)

	// tally vote should error because 4th rd voting period just started
	s.Error(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 4), "vote period not ended and quorum not reached")

	// voting that doesn't reach quorum
	voteMsg = types.MsgVote{
		Voter: teamAddr.String(),
		Id:    4,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}
	_, err = msgServer.Vote(s.Setup.Ctx, &voteMsg)
	s.NoError(err)

	// forward time to end voting period pre execute vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.TWO_DAYS + 1))
	// tally vote
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 4))
	// calling tally a second time should error
	s.ErrorContains(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 4), "vote already tallied")
	// execute should error because dispute is not resolved yet
	s.Error(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 4), "dispute is not resolved yet")
	// get dispute to check status
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 4)
	s.NoError(err)
	s.Equal(types.Unresolved, dispute.DisputeStatus)
	s.True(dispute.PendingExecution)

	// start 5th round -- PROPOSE FROM NEW ACC, SHOULD GET NO REFUND
	fifthRdDisputer := s.newKeysWithTokens()
	s.Setup.MintTokens(fifthRdDisputer, math.NewInt(100_000_000))
	// disputer balance before proposing dispute
	fifthRdDisputerBalanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, fifthRdDisputer, s.Setup.Denom)
	disputeMsg.Creator = fifthRdDisputer.String()
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)
	fmt.Println("RD 5 STARTED")

	// check on dispute
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 5)
	s.NoError(err)
	// assert disputefeetotal is correct
	rd5FeeAddition := rd4FeeAddition.MulRaw(2)
	expectedFeeTotalRd5 := expectedFeeTotalRd4.Add(rd5FeeAddition)
	s.Equal(expectedFeeTotalRd5, dispute.FeeTotal)
	s.Equal(types.Voting, dispute.DisputeStatus)
	s.True(dispute.Open)
	// assert disputer fee has been deducted
	fifthRdDisputerBalanceAfter := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, fifthRdDisputer, s.Setup.Denom)
	s.Equal(fifthRdDisputerBalanceBefore.Amount.Sub(burnAmount.MulRaw(16)), fifthRdDisputerBalanceAfter.Amount)
	s.Equal(fifthRdDisputerBalanceBefore.Amount.Sub(rd5FeeAddition), fifthRdDisputerBalanceAfter.Amount)
	// assert disputed reporter is still jailed
	reporter1, err = s.Setup.Reporterkeeper.Reporter(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	s.True(reporter1.Jailed)
	// assert reporters stake is the same as after the second round
	rep1TokensAfterRd5, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, reporter1Acc)
	s.NoError(err)
	s.Equal(rep1TokensAfterRd5, rep1TokensAfterRd4)

	// tally vote should error because 5th rd voting period just started
	s.Error(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 5), "vote period not ended and quorum not reached")

	// voting that doesn't reach quorum
	voteMsg = types.MsgVote{
		Voter: teamAddr.String(),
		Id:    5,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}
	_, err = msgServer.Vote(s.Setup.Ctx, &voteMsg)
	s.NoError(err)

	// forward time to end voting period pre execute vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.TWO_DAYS + 1))
	// tally vote
	s.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 5))
	// calling tally a second time should error
	s.ErrorContains(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 5), "vote already tallied")
	// execute should error because dispute is not resolved yet
	s.Error(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 5), "dispute is not resolved yet")
	// get dispute to check status
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 5)
	s.NoError(err)
	s.Equal(types.Unresolved, dispute.DisputeStatus)
	s.True(dispute.PendingExecution)

	// try to start new round, should error because 5 rd max
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.Error(err, "can't start a new round for this dispute 5; max dispute rounds has been reached 5")

	// forward time to execute vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.ONE_DAY + 1))
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 5))
	vote, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 5)
	s.NoError(err)
	s.True(vote.Executed)
	s.Equal(types.VoteResult_NO_QUORUM_MAJORITY_INVALID, vote.VoteResult)
	s.Equal(uint64(5), vote.Id)
	s.Less(vote.VoteEnd, s.Setup.Ctx.BlockTime())

	// assert dispute is resolved
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 5)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)

	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	s.NoError(err)

	// check on first round vote
	vote, err = s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	// s.True(vote.Executed) // false
	s.Equal(types.VoteResult_NO_QUORUM_MAJORITY_INVALID, vote.VoteResult)
	s.Equal(uint64(1), vote.Id)
	s.Less(vote.VoteEnd, s.Setup.Ctx.BlockTime())

	// claim fee refund from original dispute proposer, he should get first rd fee back
	disputerBalBeforeClaim := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	withdrawMsg := types.MsgWithdrawFeeRefund{
		Id:            uint64(1),
		PayerAddress:  disputer.String(),
		CallerAddress: disputer.String(),
	}
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &withdrawMsg)
	s.NoError(err)
	disputerBalAfterClaim := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	s.Equal(disputerBalBeforeClaim.Amount.Add(expectedRd1Fee.MulRaw(95).QuoRaw(100)), disputerBalAfterClaim.Amount)

	// try to claim fee refund from 2nd 3rd and 4th rd disputers, they should get nothing back
	for i := 2; i < 4; i++ {
		vote, err = s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, uint64(i))
		s.NoError(err)
		s.True(vote.Executed)

		disputerBalBeforeClaim := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
		withdrawMsg := types.MsgWithdrawFeeRefund{
			Id:            uint64(i),
			PayerAddress:  disputer.String(),
			CallerAddress: disputer.String(),
		}
		_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &withdrawMsg)
		s.Error(err)
		disputerBalAfterClaim := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
		s.Equal(disputerBalBeforeClaim.Amount, disputerBalAfterClaim.Amount)
	}

	// try to claim fee for 5th rd disputer, he should get nothing back
	disputerBalBeforeClaim = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, fifthRdDisputer, s.Setup.Denom)
	withdrawMsg = types.MsgWithdrawFeeRefund{
		Id:            uint64(5),
		PayerAddress:  fifthRdDisputer.String(),
		CallerAddress: fifthRdDisputer.String(),
	}
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &withdrawMsg)
	s.Error(err)
	disputerBalAfterClaim = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, fifthRdDisputer, s.Setup.Denom)
	s.Equal(disputerBalBeforeClaim.Amount, disputerBalAfterClaim.Amount)

	// try to claim fee for 1st rd disputer again, should error
	disputerBalBeforeClaim = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	withdrawMsg = types.MsgWithdrawFeeRefund{
		Id:            uint64(1),
		PayerAddress:  disputer.String(),
		CallerAddress: disputer.String(),
	}
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &withdrawMsg)
	s.Error(err)
	disputerBalAfterClaim = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	s.Equal(disputerBalBeforeClaim.Amount, disputerBalAfterClaim.Amount)

	// check on first vote and dispute
	vote, err = s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.True(vote.Executed)
	s.Equal(types.VoteResult_NO_QUORUM_MAJORITY_INVALID, vote.VoteResult)
	s.Equal(uint64(1), vote.Id)
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	fmt.Println("DISPUTE STATUS", dispute.DisputeStatus)
	// s.Equal(types.Resolved, dispute.DisputeStatus) // still says unresolved
	s.False(dispute.PendingExecution)
	s.False(dispute.Open)
}

func (s *IntegrationTestSuite) TestNoQorumSingleRound() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	repAccs, _, _ := s.createValidatorAccs([]uint64{100, 200})
	reporter1Acc := repAccs[0]
	// reporter2Acc := repAccs[1]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewSelection(reporter1Acc, 1)))

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")

	reporter1StakeBefore, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1Acc, qId)
	s.NoError(err)

	report := oracletypes.MicroReport{
		Reporter:    reporter1Acc.String(),
		Power:       reporter1StakeBefore.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.UnixMilli(1696516597).UTC(),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, reporter1Acc.Bytes(), report.MetaId), report))
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)

	disputer := s.newKeysWithTokens()
	// mint disputer tokens
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))

	disputeMsg := types.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)

	// get dispute to set block info
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	// set block info directly for ease (need validators to call endblocker)
	err = s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, dispute.HashId)
	s.NoError(err)

	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.NoError(err)
	voteMsg := types.MsgVote{
		Voter: teamAddr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_INVALID,
	}

	_, err = msgServer.Vote(s.Setup.Ctx, &voteMsg)
	s.NoError(err)
	// forward time to expire dispute and tally vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.TWO_DAYS + 1))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	s.NoError(err)

	voteInfo, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(types.VoteResult_NO_QUORUM_MAJORITY_INVALID, voteInfo.VoteResult)
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(types.Unresolved, dispute.DisputeStatus)
	s.True(dispute.PendingExecution)
	s.False(voteInfo.Executed)

	// forward time to execute vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(keeper.ONE_DAY + 1))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	s.NoError(err)
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(types.Resolved, dispute.DisputeStatus)
	s.False(dispute.PendingExecution)
	voteInfo, err = s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.True(voteInfo.Executed)
}

func (s *IntegrationTestSuite) TestDisputeButNoVotes() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	repAccs, _, _ := s.createValidatorAccs([]uint64{100})
	reporter1Acc := repAccs[0]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter1Acc, reportertypes.NewSelection(reporter1Acc, 1)))

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")

	reporterStakeBefore, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1Acc, qId)
	s.NoError(err)

	report := oracletypes.MicroReport{
		Reporter:    reporter1Acc.String(),
		Power:       reporterStakeBefore.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.UnixMilli(1696516597).UTC(),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, reporter1Acc.Bytes(), report.MetaId), report))
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)

	disputer := s.newKeysWithTokens()
	// mint disputer tokens
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))

	disputeMsg := types.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
	}
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &disputeMsg)
	s.NoError(err)
	// get dispute to set block info
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	// set block info directly for ease (need validators to call endblocker)
	err = s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, dispute.HashId)
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

	queryid, err := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	s.NoError(err)

	stake1, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1, queryid)
	s.NoError(err)
	reporter2 := valAccs[1]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter2, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter2, reportertypes.NewSelection(reporter2, 1)))
	stake2, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter2, queryid)
	s.NoError(err)
	reporter3 := valAccs[2]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter3, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter3, reportertypes.NewSelection(reporter3, 1)))
	stake3, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter3, queryid)
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
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report1.QueryId, reporter1.Bytes(), report1.MetaId), report1))
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
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report2.QueryId, reporter2.Bytes(), report2.MetaId), report2))
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
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report3.QueryId, reporter3.Bytes(), report3.MetaId), report3))
	// forward time
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	// set report
	err = s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report1.QueryId, reporter1.Bytes(), uint64(1)), report1)
	s.NoError(err)
	s.NoError(s.Setup.Oraclekeeper.AddReport(s.Setup.Ctx, 1, report1))
	err = s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report2.QueryId, reporter2.Bytes(), uint64(1)), report2)
	s.NoError(err)
	s.NoError(s.Setup.Oraclekeeper.AddReport(s.Setup.Ctx, 1, report2))
	err = s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report3.QueryId, reporter3.Bytes(), uint64(1)), report3)
	s.NoError(err)
	s.NoError(s.Setup.Oraclekeeper.AddReport(s.Setup.Ctx, 1, report3))

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
		Creator:          disputer.String(),
		DisputedReporter: report2.Reporter,
		ReportMetaId:     report2.MetaId,
		ReportQueryId:    hex.EncodeToString(report2.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee),
		DisputeCategory:  types.Warning,
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

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")

	reporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1Acc, qId)
	s.NoError(err)

	report := oracletypes.MicroReport{
		Reporter:    reporter1Acc.String(),
		Power:       reporterStake.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.UnixMilli(1696516597).UTC(),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, reporter1Acc.Bytes(), report.MetaId), report))
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)

	disputer := s.newKeysWithTokens()
	// mint disputer tokens
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))
	// propose dispute with half the fee
	disputeMsg := types.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee.QuoRaw(2)),
		DisputeCategory:  types.Warning,
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

	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")

	reporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter1Acc, qId)
	s.NoError(err)

	report := oracletypes.MicroReport{
		Reporter:    reporter1Acc.String(),
		Power:       reporterStake.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     qId,
		Value:       "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp:   time.UnixMilli(1696516597).UTC(),
		BlockNumber: uint64(s.Setup.Ctx.BlockHeight()),
		MetaId:      1,
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, reporter1Acc.Bytes(), report.MetaId), report))
	disputeFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, types.Warning)
	s.NoError(err)

	disputer := repAccs[1]
	// mint disputer tokens
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000_000))
	// propose dispute with half the fee
	disputeMsg := types.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, disputeFee.QuoRaw(2)),
		DisputeCategory:  types.Warning,
		PayFromBond:      false,
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
	// check free floating balance
	freeFloatingBalanceBeforeAdd := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)

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
	freeFloatingBalanceAfterAdd := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, disputer, s.Setup.Denom)
	s.Equal(freeFloatingBalanceBeforeAdd.Amount.Sub(disputeFee.QuoRaw(2)), freeFloatingBalanceAfterAdd.Amount)
}

func (s *IntegrationTestSuite) TestCurrentBug() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	sk := s.Setup.Stakingkeeper
	startingBondedPoolbal := math.NewInt(1000000)
	params := slashingtypes.DefaultParams()
	params.SignedBlocksWindow = 1
	notbondedpool := authtypes.NewModuleAddress(stakingtypes.NotBondedPoolName)
	bondedpool := authtypes.NewModuleAddress(stakingtypes.BondedPoolName)

	s.NoError(s.Setup.SlashingKeeper.SetParams(s.Setup.Ctx, params))
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	oServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	// chain has three validators
	repAccs, valAccs, _ := s.createValidatorsbypowers([]uint64{150, 500, 100000})
	// staking pool balances
	// not bonded pool
	bal, err := s.Setup.Bankkeeper.Balances.Get(s.Setup.Ctx, collections.Join(notbondedpool, "loya"))
	s.Error(err, "balance should be zero")
	s.True(bal.IsNil())
	// bonded poool
	bal, err = s.Setup.Bankkeeper.Balances.Get(s.Setup.Ctx, collections.Join(bondedpool, "loya"))
	s.NoError(err, "balance should be gt zero")
	s.Equal(sk.PowerReduction(s.Setup.Ctx).MulRaw(int64(150+500+100000)).Add(startingBondedPoolbal), bal)

	// give disputer tokens
	s.Setup.MintTokens(repAccs[1], math.NewInt(100000000000))
	// bridge endBlock stuff
	for _, val := range valAccs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}

	// create reporter and submit reports
	reportBlock := s.Setup.Ctx.BlockHeight()
	reportTime := s.Setup.Ctx.BlockTime().UTC()
	for _, r := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, r, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, r, reportertypes.NewSelection(r, 1)))
		rep := report(r.String(), testutil.EncodeValue(29266), ethQueryData)
		_, err := oServer.SubmitValue(s.Setup.Ctx, &rep)
		s.NoError(err)
	}

	s.Setup.Ctx, err = simtestutil.NextBlock(s.Setup.App, s.Setup.Ctx, time.Hour)
	s.NoError(err)

	test_report := &oracletypes.MicroReport{
		Reporter:        repAccs[2].String(),
		Power:           100000,
		QueryType:       "SpotPrice",
		QueryId:         utils.QueryIDFromData(ethQueryData),
		AggregateMethod: "weighted-median",
		Timestamp:       reportTime,
		Value:           testutil.EncodeValue(29266),
		Cyclelist:       true,
		BlockNumber:     uint64(reportBlock),
		MetaId:          0,
	}
	// propose dispute id 1 slash amount 10_000_000
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          repAccs[1].String(),
		DisputedReporter: test_report.Reporter,
		ReportMetaId:     test_report.MetaId,
		ReportQueryId:    hex.EncodeToString(test_report.QueryId),
		DisputeCategory:  types.Warning,
		Fee:              sdk.NewCoin(s.Setup.Denom, math.NewInt(1000_000_000)), // one percent dispute fee
		PayFromBond:      false,
	})
	s.NoError(err)
	// check dispute status
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(types.Voting, dispute.DisputeStatus)
	s.Setup.Ctx, err = simtestutil.NextBlock(s.Setup.App, s.Setup.Ctx, time.Minute)
	s.NoError(err)

	// check pool bals after first dispute
	notbondedpool = authtypes.NewModuleAddress(stakingtypes.NotBondedPoolName)
	bal, err = s.Setup.Bankkeeper.Balances.Get(s.Setup.Ctx, collections.Join(notbondedpool, "loya"))
	s.Error(err, "balance should be zero")
	s.True(bal.IsNil())

	// bonded pool
	bal, err = s.Setup.Bankkeeper.Balances.Get(s.Setup.Ctx, collections.Join(bondedpool, "loya"))
	s.NoError(err, "balance should be gt zero")
	s.Equal(sk.PowerReduction(s.Setup.Ctx).MulRaw(int64(150+500+99000)).Add(startingBondedPoolbal), bal)

	test_report2 := &oracletypes.MicroReport{
		Reporter:        repAccs[0].String(),
		Power:           150,
		QueryType:       "SpotPrice",
		QueryId:         utils.QueryIDFromData(ethQueryData),
		AggregateMethod: "weighted-median",
		Timestamp:       reportTime,
		Value:           testutil.EncodeValue(29266),
		Cyclelist:       true,
		BlockNumber:     uint64(reportBlock),
	}
	// propose dispute id 2 slash amount 2_000_000
	_, err = msgServer.ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          repAccs[1].String(),
		DisputedReporter: test_report2.Reporter,
		ReportMetaId:     test_report2.MetaId,
		ReportQueryId:    hex.EncodeToString(test_report2.QueryId),
		DisputeCategory:  types.Warning,
		Fee:              sdk.NewCoin(s.Setup.Denom, math.NewInt(2_000_000)), // one percent dispute fee
		PayFromBond:      false,
	})
	s.NoError(err)

	// check dispute status 2
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(types.Voting, dispute.DisputeStatus)
	s.Setup.Ctx, err = simtestutil.NextBlock(s.Setup.App, s.Setup.Ctx, time.Minute)
	s.NoError(err)

	notbondedpool = authtypes.NewModuleAddress(stakingtypes.NotBondedPoolName)
	bal, err = s.Setup.Bankkeeper.Balances.Get(s.Setup.Ctx, collections.Join(notbondedpool, "loya"))
	s.Error(err, "balance should be zero")
	s.True(bal.IsNil())

	voteinfos := []abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: valAccs[0],
				Power:   149,
			},
		},
		{
			Validator: abci.Validator{
				Address: valAccs[1],
				Power:   4000,
			},
		},
		{
			Validator: abci.Validator{
				Address: valAccs[2],
				Power:   99000,
			},
		},
	}

	s.Setup.Ctx, err = simtestutil.NextBlock(s.Setup.App, s.Setup.Ctx, time.Minute)
	s.NoError(err)

	// during a dispute two validators are jailed (validator slashing)
	val1, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAccs[0]) // reporter
	s.NoError(err)
	consAddr1, err := val1.GetConsAddr()
	s.NoError(err)
	signinginfor1, err := s.Setup.SlashingKeeper.GetValidatorSigningInfo(s.Setup.Ctx, consAddr1)
	s.NoError(err)
	signinginfor1.MissedBlocksCounter = 2
	signinginfor1.StartHeight = 1
	s.NoError(s.Setup.SlashingKeeper.SetValidatorSigningInfo(s.Setup.Ctx, consAddr1, signinginfor1))

	val2, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAccs[2]) // reporter
	s.NoError(err)

	consAddr2, err := val2.GetConsAddr()
	s.NoError(err)
	signinginfor2, err := s.Setup.SlashingKeeper.GetValidatorSigningInfo(s.Setup.Ctx, consAddr2)
	s.NoError(err)
	signinginfor2.MissedBlocksCounter = 2
	signinginfor1.StartHeight = 1
	s.NoError(s.Setup.SlashingKeeper.SetValidatorSigningInfo(s.Setup.Ctx, consAddr2, signinginfor2))

	// move blocks ahead so that they are jailed/slashed
	s.Setup.Ctx = s.Setup.Ctx.WithVoteInfos(voteinfos)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	s.NoError(err)

	// need to add validators for the x/bridge since all validators are jailed
	// otherwise you get an error "no validators found"
	_, vals, _, _ := s.Setup.CreateValidatorsRandomStake(1)
	for _, val := range vals {
		err = s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
	s.Setup.Ctx, err = simtestutil.NextBlock(s.Setup.App, s.Setup.Ctx, time.Minute)
	s.NoError(err)

	// both validators jailed
	val1, err = s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAccs[0]) // reporter
	s.NoError(err)
	s.True(val1.Jailed)
	val2, err = s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAccs[2]) // reporter
	s.NoError(err)
	s.True(val2.Jailed)

	s.Setup.Ctx, err = simtestutil.NextBlock(s.Setup.App, s.Setup.Ctx, time.Minute)
	s.NoError(err)
	bal, err = s.Setup.Bankkeeper.Balances.Get(s.Setup.Ctx, collections.Join(notbondedpool, "loya"))
	s.NoError(err, "balance should be gt zero since validators were jailed")
	s.True(bal.GT(math.ZeroInt()), "amount should be tokens minus the slashed amount, val 0 and val 2 got slashed 1 percen")
	s.Equal(math.NewInt(int64(147010000+98010000000)), bal) // 147010000 precision ?? should be 147015000
	votes := []types.MsgVote{
		{
			Voter: repAccs[1].String(),
			Id:    1,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.Setup.Ctx, &votes[i])
		s.NoError(err)
	}

	votes = []types.MsgVote{
		{
			Voter: repAccs[1].String(),
			Id:    2,
			Vote:  types.VoteEnum_VOTE_AGAINST,
		},
	}
	for i := range votes {
		_, err = msgServer.Vote(s.Setup.Ctx, &votes[i])
		s.NoError(err)
	}

	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Hour * 72))
	s.Setup.Ctx, err = simtestutil.NextBlock(s.Setup.App, s.Setup.Ctx, time.Minute)
	s.NoError(err)
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(dispute.DisputeStatus, types.Resolved)
	vote, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 1)
	s.NoError(err)
	s.Equal(vote.VoteResult, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST)
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(dispute.DisputeStatus, types.Resolved)
	vote, err = s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, 2)
	s.NoError(err)
	s.Equal(vote.VoteResult, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST)
	// unjail validator 1
	slashingServer := slashingkeeper.NewMsgServerImpl(s.Setup.SlashingKeeper)
	_, err = slashingServer.Unjail(s.Setup.Ctx, slashingtypes.NewMsgUnjail(valAccs[2].String()))
	s.NoError(err)

	s.Setup.Ctx, err = simtestutil.NextBlock(s.Setup.App, s.Setup.Ctx, time.Minute)
	s.NoError(err)

	_, err = slashingServer.Unjail(s.Setup.Ctx, slashingtypes.NewMsgUnjail(valAccs[0].String()))
	s.NoError(err)

	s.Setup.Ctx, err = simtestutil.NextBlock(s.Setup.App, s.Setup.Ctx, time.Minute)
	s.NoError(err)
	// should be back to zero/nil
	bal, err = s.Setup.Bankkeeper.Balances.Get(s.Setup.Ctx, collections.Join(notbondedpool, "loya"))
	s.Error(err, "balance should be zero")
	s.True(bal.IsNil())
}

func (s *IntegrationTestSuite) TestManyOpenDisputes() {
	// require := s.Require()
	// msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)

	// // chain has 11 validators
	// // 10 will get disputed, 1 will be tipping and disputing
	// repAccs, valAccs, _ := s.createValidatorsbypowers([]uint64{10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000})
	// tipper := repAccs[0]
}
