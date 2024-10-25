package e2e_test

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/tellor-io/layer/testutil"
	utils "github.com/tellor-io/layer/utils"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	collections "cosmossdk.io/collections"
	math "cosmossdk.io/math"

	secp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *E2ETestSuite) TestDisputes() {
	require := s.Require()
	msgServerOracle := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	require.NotNil(msgServerOracle)
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	require.NotNil(msgServerReporter)
	msgServerDispute := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	require.NotNil(msgServerDispute)
	msgServerStaking := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	require.NotNil(msgServerStaking)

	//---------------------------------------------------------------------------
	// Height 0 - create validator and 2 reporters
	//---------------------------------------------------------------------------
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// create a validator
	valAccount := simtestutil.CreateIncrementalAccounts(1)
	// mint 5000*1e8 tokens for validator
	initCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(5000*1e8))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, valAccount[0], sdk.NewCoins(initCoins)))
	// get val address
	valAccountValAddrs := simtestutil.ConvertAddrsToValAddrs(valAccount)
	// create pub key for validator
	pubKey := simtestutil.CreateTestPubKeys(1)
	// tell keepers about the new validator
	s.Setup.Accountkeeper.NewAccountWithAddress(s.Setup.Ctx, valAccount[0])
	msgCreateValidaotr, err := stakingtypes.NewMsgCreateValidator(
		valAccountValAddrs[0].String(),
		pubKey[0],
		sdk.NewCoin(s.Setup.Denom, math.NewInt(4000*1e8)),
		stakingtypes.Description{Moniker: "created validator"},
		stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(0, 0), math.LegacyNewDecWithPrec(3, 1), math.LegacyNewDecWithPrec(1, 1)),
		math.OneInt(),
	)
	require.NoError(err)
	_, err = msgServerStaking.CreateValidator(s.Setup.Ctx, msgCreateValidaotr)
	require.NoError(err)
	for _, val := range valAccountValAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	validator, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAccountValAddrs[0])
	require.NoError(err)

	type Delegator struct {
		delegatorAddress sdk.AccAddress
		validator        stakingtypes.Validator
		tokenAmount      math.Int
	}
	pk := secp256k1.GenPrivKey()
	reporterAccount := sdk.AccAddress(pk.PubKey().Address())
	// mint 5000*1e6 tokens for reporter
	s.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, reporterAccount, sdk.NewCoins(initCoins)))
	// delegate 5k trb to validator so reporter can delegate to themselves
	reporterDelToVal := Delegator{delegatorAddress: reporterAccount, validator: validator, tokenAmount: math.NewInt(4000 * 1e6)}
	msgDelegate := stakingtypes.NewMsgDelegate(
		reporterDelToVal.delegatorAddress.String(),
		reporterDelToVal.validator.GetOperator(), sdk.NewCoin(s.Setup.Denom, reporterDelToVal.tokenAmount),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)
	// // check that reporter was created in Reporters collections
	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: reporterAccount.String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewInt(4000 * 1e6)})
	require.NoError(err)
	reporter, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	fmt.Println(reporterAccount.String())
	require.NoError(err)
	require.Equal(reporter.Jailed, false)
	// // check on reporter in Delegators collections
	rkDelegation, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	require.Equal(rkDelegation.Reporter, reporterAccount.Bytes())

	// check on reporter/validator delegation
	skDelegation, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, reporterAccount, valAccountValAddrs[0])
	require.NoError(err)
	require.Equal(skDelegation.GetDelegatorAddr(), reporterAccount.String())
	require.Equal(skDelegation.GetValidatorAddr(), validator.GetOperator())

	//---------------------------------------------------------------------------
	// Height 1 - direct reveal for cycle list
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	// get new cycle list query data
	cycleListQuery, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	// create reveal message
	value := testutil.EncodeValue(100_000)
	require.NoError(err)
	reveal := oracletypes.MsgSubmitValue{
		Creator:   reporterAccount.String(),
		QueryData: cycleListQuery,
		Value:     value,
	}
	// send reveal message
	revealResponse, err := msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal)
	revealBlock := s.Setup.Ctx.BlockHeight()
	require.NoError(err)
	require.NotNil(revealResponse)
	// advance block height to expire the query and aggregate report

	//---------------------------------------------------------------------------
	// Height 2  - advance block to expire query
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	//---------------------------------------------------------------------------
	// Height 3 - advance block to expire query
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	//---------------------------------------------------------------------------
	// Height 4 - check on aggregate
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// get queryId for GetAggregatedReportRequest
	queryId := utils.QueryIDFromData(cycleListQuery)
	s.NoError(err)
	// create get aggregated report query
	getAggReportRequest := oracletypes.QueryGetCurrentAggregateReportRequest{
		QueryId: hex.EncodeToString(queryId),
	}
	// aggregated report is stored correctly
	queryServer := oraclekeeper.NewQuerier(s.Setup.Oraclekeeper)
	result, err := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(uint64(0), result.Aggregate.AggregateReportIndex)
	require.Equal(testutil.EncodeValue(100_000), result.Aggregate.AggregateValue)
	require.Equal(reporterAccount.String(), result.Aggregate.AggregateReporter)
	require.Equal(queryId, result.Aggregate.QueryId)
	require.Equal(uint64(4000), result.Aggregate.ReporterPower)
	require.Equal(uint64(3), result.Aggregate.Height)

	//---------------------------------------------------------------------------
	// Height 5 - create a dispute
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, false)
	freeFloatingBalanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, reporterAccount, s.Setup.Denom)

	balBeforeDispute, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	onePercent := balBeforeDispute.Mul(math.NewInt(1)).Quo(math.NewInt(100))
	disputeFee := sdk.NewCoin(s.Setup.Denom, onePercent) // warning should be 1% of bonded tokens

	// todo: is there a getter for this ?
	// get microreport for dispute
	report := oracletypes.MicroReport{
		Reporter:    reporterAccount.String(),
		Power:       balBeforeDispute.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     queryId,
		Value:       value,
		Timestamp:   s.Setup.Ctx.BlockTime(),
		BlockNumber: uint64(revealBlock),
	}

	// create msg for propose dispute tx
	msgProposeDispute := disputetypes.MsgProposeDispute{
		Creator:         reporterAccount.String(),
		Report:          &report,
		DisputeCategory: disputetypes.Warning,
		Fee:             disputeFee,
		PayFromBond:     false,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.Setup.Ctx, &msgProposeDispute)
	require.NoError(err)

	burnAmount := disputeFee.Amount.MulRaw(1).QuoRaw(20)
	disputes, err := s.Setup.Disputekeeper.GetOpenDisputes(s.Setup.Ctx)
	require.NoError(err)
	require.NotNil(disputes)
	require.Equal(1, len(disputes))
	// dispute is created correctly
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	require.NoError(err)
	require.Equal(dispute.DisputeId, uint64(1))
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeCategory, disputetypes.Warning)
	require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	feepayer, err := s.Setup.Disputekeeper.DisputeFeePayer.Get(s.Setup.Ctx, collections.Join(uint64(1), reporterAccount.Bytes()))
	require.NoError(err)
	require.Equal(feepayer.Amount, disputeFee.Amount)
	require.Equal(feepayer.FromBond, false)
	slashAmount := dispute.SlashAmount
	firstDisputeVoteMsg := disputetypes.MsgVote{
		Voter: reporterAccount.String(),
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 6 - unjail reporter
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	err = s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, dispute.DisputeId)
	require.Error(err, "vote period not ended and quorum not reached")
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// reporter is in jail
	reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, true)
	// reporter lost 1% of their free floating tokens
	freeFloatingBalanceAfter := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, reporterAccount, s.Setup.Denom)
	require.Equal(freeFloatingBalanceAfter, freeFloatingBalanceBefore.Sub(disputeFee))

	// create msgUnJailReporter
	msgUnjailReporter := reportertypes.MsgUnjailReporter{
		ReporterAddress: reporterAccount.String(),
	}
	// send unjailreporter tx
	_, err = msgServerReporter.UnjailReporter(s.Setup.Ctx, &msgUnjailReporter)
	require.NoError(err)

	// reporter is now unjailed
	reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, false)
	freeFloatingBalanceAfter = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, reporterAccount, s.Setup.Denom)
	require.Equal(freeFloatingBalanceAfter, freeFloatingBalanceBefore.Sub(disputeFee))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	// todo: more balance checks at each step

	//---------------------------------------------------------------------------
	// Height 7 - direct reveal for cycle list again
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	err = s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1)
	require.Error(err, "vote period not ended and quorum not reached")
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// get new cycle list query data
	cycleListQuery, err = s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	// create reveal message
	value = testutil.EncodeValue(100_000)
	require.NoError(err)
	reveal = oracletypes.MsgSubmitValue{
		Creator:   reporterAccount.String(),
		QueryData: cycleListQuery,
		Value:     value,
	}
	secReportQueryId := utils.QueryIDFromData(cycleListQuery)
	// send reveal message
	revealResponse, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal)
	revealBlock = s.Setup.Ctx.BlockHeight()
	require.NoError(err)
	require.NotNil(revealResponse)
	// advance block height to expire the query and aggregate report
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 8  - advance block to expire query
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	//---------------------------------------------------------------------------
	// Height 9 - advance block to expire query
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryId = utils.QueryIDFromData(cycleListQuery)
	s.NoError(err)
	// create get aggregated report query
	getAggReportRequest = oracletypes.QueryGetCurrentAggregateReportRequest{
		QueryId: hex.EncodeToString(queryId),
	}
	// aggregated report is stored correctly
	result, err = queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(uint64(0), result.Aggregate.AggregateReportIndex)
	require.Equal(testutil.EncodeValue(100_000), result.Aggregate.AggregateValue)
	require.Equal(reporterAccount.String(), result.Aggregate.AggregateReporter)
	require.Equal(queryId, result.Aggregate.QueryId)
	require.Equal(uint64(4000)-slashAmount.Quo(sdk.DefaultPowerReduction).Uint64(), result.Aggregate.ReporterPower)
	require.Equal(uint64(7), result.Aggregate.Height)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 10 - open minor dispute
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	balBeforeDispute, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterAccount)
	fmt.Println("Balance before dispute: ", balBeforeDispute)
	require.NoError(err)
	fivePercent := balBeforeDispute.Mul(math.NewInt(5)).Quo(math.NewInt(100))
	disputeFee = sdk.NewCoin(s.Setup.Denom, fivePercent)

	report = oracletypes.MicroReport{
		Reporter:    reporterAccount.String(),
		Power:       balBeforeDispute.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     secReportQueryId,
		Value:       value,
		Timestamp:   s.Setup.Ctx.BlockTime(),
		BlockNumber: uint64(revealBlock),
	}

	fmt.Println("Report power: ", report.Power)

	// create msg for propose dispute tx
	msgProposeDispute = disputetypes.MsgProposeDispute{
		Creator:         reporterAccount.String(),
		Report:          &report,
		DisputeCategory: disputetypes.Minor,
		Fee:             disputeFee,
		PayFromBond:     false,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.Setup.Ctx, &msgProposeDispute)
	require.NoError(err)
	disputeStartTime := s.Setup.Ctx.BlockTime()

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 11 - vote on minor dispute
	// ---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	err = s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1)
	require.Error(err, "vote period not ended and quorum not reached")
	err = s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 2)
	require.Error(err, "vote period not ended and quorum not reached")
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// reporter is in jail
	reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, true)
	// dispute is created correctly
	burnAmount = disputeFee.Amount.MulRaw(1).QuoRaw(20)
	dispute, err = s.Setup.Disputekeeper.GetDisputeByReporter(s.Setup.Ctx, report, disputetypes.Minor)
	fmt.Printf("Dispute: %v,\r Report: %v\r", dispute, report)
	require.NoError(err)
	require.Equal(dispute.DisputeCategory, disputetypes.Minor)
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	feepayer, err = s.Setup.Disputekeeper.DisputeFeePayer.Get(s.Setup.Ctx, collections.Join(dispute.DisputeId, reporterAccount.Bytes()))
	require.NoError(err)
	require.Equal(feepayer.Amount, disputeFee.Amount)
	require.Equal(feepayer.FromBond, false)

	firstVoteReponse, err := msgServerDispute.Vote(s.Setup.Ctx, &firstDisputeVoteMsg)
	require.NoError(err)
	require.NotNil(firstVoteReponse)

	// create vote tx msg
	msgVote := disputetypes.MsgVote{
		Voter: reporterAccount.String(),
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}
	// send vote tx for second dispute
	voteResponse, err := msgServerDispute.Vote(s.Setup.Ctx, &msgVote)
	require.NoError(err)
	require.NotNil(voteResponse)

	// vote is properly stored
	vote, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, dispute.DisputeId)
	require.NoError(err)
	require.NotNil(vote)
	require.Equal(vote.Executed, false)
	require.Equal(vote.Id, dispute.DisputeId)
	require.Equal(vote.VoteStart, disputeStartTime)
	require.Equal(vote.VoteEnd, disputeStartTime.Add(disputekeeper.TWO_DAYS))

	// advance 2 days to expire vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(disputekeeper.THREE_DAYS))
	// call unjail function
	msgUnjailReporter = reportertypes.MsgUnjailReporter{
		ReporterAddress: reporterAccount.String(),
	}
	_, err = msgServerReporter.UnjailReporter(s.Setup.Ctx, &msgUnjailReporter)
	require.NoError(err)

	// reporter no longer in jail
	reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, false)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 12 - minor dispute ends and another direct reveal for cycle list
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	require.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1))
	require.NoError(s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 2))
	require.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 1))
	require.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, 2))

	// vote is executed
	vote, err = s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, dispute.DisputeId)
	require.NoError(err)
	require.NotNil(vote)
	require.Equal(vote.Executed, true)
	require.Equal(vote.Id, dispute.DisputeId)
	// reporter no longer in jail
	reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, false)

	// get open disputes
	disputes, err = s.Setup.Disputekeeper.GetOpenDisputes(s.Setup.Ctx)
	require.NoError(err)
	require.NotNil(disputes)

	cycleListQuery, err = s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	// create reveal message
	value = testutil.EncodeValue(100_000)
	require.NoError(err)
	reveal = oracletypes.MsgSubmitValue{
		Creator:   reporterAccount.String(),
		QueryData: cycleListQuery,
		Value:     value,
	}
	// send reveal message
	revealResponse, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal)
	revealBlock = s.Setup.Ctx.BlockHeight()
	require.NoError(err)
	require.NotNil(revealResponse)
	// advance time and block height to expire the query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(21 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryId = utils.QueryIDFromData(cycleListQuery)
	s.NoError(err)
	// create get aggregated report query
	getAggReportRequest = oracletypes.QueryGetCurrentAggregateReportRequest{
		QueryId: hex.EncodeToString(queryId),
	}
	// check that aggregated report is stored correctly
	result, err = queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(uint64(0), result.Aggregate.AggregateReportIndex)
	require.Equal(testutil.EncodeValue(100_000), result.Aggregate.AggregateValue)
	require.Equal(reporterAccount.String(), result.Aggregate.AggregateReporter)
	require.Equal(queryId, result.Aggregate.QueryId)
	require.Equal(uint64(7), result.Aggregate.Height)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 13 - open major dispute for report
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, false)

	oneHundredPercent, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	disputeFee = sdk.NewCoin(s.Setup.Denom, oneHundredPercent)

	report = oracletypes.MicroReport{
		Reporter:    reporterAccount.String(),
		Power:       oneHundredPercent.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     queryId,
		Value:       value,
		Timestamp:   s.Setup.Ctx.BlockTime(),
		BlockNumber: uint64(revealBlock),
	}
	// create msg for propose dispute tx

	msgProposeDispute = disputetypes.MsgProposeDispute{
		Creator:         reporterAccount.String(),
		Report:          &report,
		DisputeCategory: disputetypes.Major,
		Fee:             disputeFee,
		PayFromBond:     false,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.Setup.Ctx, &msgProposeDispute)
	require.NoError(err)
	disputeStartTime = s.Setup.Ctx.BlockTime()
	disputeStartHeight := uint64(s.Setup.Ctx.BlockHeight())

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 9 - vote on major dispute
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	err = s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 3)
	require.Error(err, "vote period not ended and quorum not reached")
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	fee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, disputetypes.Major)
	require.NoError(err)
	require.GreaterOrEqual(msgProposeDispute.Fee.Amount.Uint64(), fee.Uint64())

	// dispute is created and open for voting
	dispute, err = s.Setup.Disputekeeper.GetDisputeByReporter(s.Setup.Ctx, report, disputetypes.Major)
	require.NoError(err)
	burnAmount = disputeFee.Amount.MulRaw(1).QuoRaw(20)
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeStartTime, disputeStartTime)
	require.Equal(dispute.DisputeEndTime, disputeStartTime.Add(disputekeeper.THREE_DAYS))
	require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	require.Equal(dispute.DisputeStartBlock, disputeStartHeight)

	// create vote tx msg
	msgVote = disputetypes.MsgVote{
		Voter: reporterAccount.String(),
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}
	// send vote tx
	voteResponse, err = msgServerDispute.Vote(s.Setup.Ctx, &msgVote)
	require.NoError(err)
	require.NotNil(voteResponse)

	// vote is properly stored
	vote, err = s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, dispute.DisputeId)
	require.NoError(err)
	require.NotNil(vote)
	require.Equal(vote.Executed, false)
	require.Equal(vote.Id, dispute.DisputeId)
	require.Equal(vote.VoteStart, disputeStartTime)
	require.Equal(vote.VoteEnd, disputeStartTime.Add(disputekeeper.TWO_DAYS))

	// advance 3 days to expire vote
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(disputekeeper.THREE_DAYS))

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	// ---------------------------------------------------------------------------
	// Height 10 - dispute is resolved, reporter no longer a reporter
	// ---------------------------------------------------------------------------
	// s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	// s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	// _, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	// require.NoError(err)

	err = s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 3)
	require.NoError(err)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	// reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	// require.NoError(err)
}


// Vicky the Validator just produces blocks, has 1000 trb staked
// Rob the Reporter has 100 trb staked with Vicky so he can select himself as the reporter
// Delwood the Delegator has 250 trb delegated to Rob
// Rob stakes 10 more trb through a different reporter account to vicky and makes Rob2 a reporter
// Rob creates a dispute for a good report from Rob2
// The dispute settles to `No`, moving tokens from Delwood to Rob through proxy of Rob2 ?
func (s *E2ETestSuite) TestDisputeSettlesToNo() {
	// Setup msgServers
	require := s.Require()
	msgServerOracle := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	require.NotNil(msgServerOracle)
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	require.NotNil(msgServerReporter)
	msgServerDispute := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	require.NotNil(msgServerDispute)
	msgServerStaking := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	require.NotNil(msgServerStaking)

	//---------------------------------------------------------------------------
	// Height 0 - vicky becomes a validator
	//---------------------------------------------------------------------------
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	vickyAccAddr := simtestutil.CreateIncrementalAccounts(1)
	vickyInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(2000*1e6)) // give vicky extra to act as free floating token voting group
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(vickyInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, vickyAccAddr[0], sdk.NewCoins(vickyInitCoins)))
	s.Setup.Accountkeeper.NewAccountWithAddress(s.Setup.Ctx, vickyAccAddr[0])

	pubKey := simtestutil.CreateTestPubKeys(1)
	vickyValAddr := simtestutil.ConvertAddrsToValAddrs(vickyAccAddr)
	msgCreateValidator, err := stakingtypes.NewMsgCreateValidator(
		vickyValAddr[0].String(),
		pubKey[0],
		sdk.NewCoin(s.Setup.Denom, math.NewInt(1000*1e6)),
		stakingtypes.Description{Moniker: "created validator"},
		stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(0, 0), math.LegacyNewDecWithPrec(3, 1), math.LegacyNewDecWithPrec(1, 1)),
		math.OneInt(),
	)
	require.NoError(err)

	_, err = msgServerStaking.CreateValidator(s.Setup.Ctx, msgCreateValidator)
	require.NoError(err)

	require.NoError(s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, vickyValAddr[0].String(), []byte("vickyEvmAddr")))

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - Rob delegates to Vicky and selects himself to become a reporter
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// verify vicky is a bonded validator
	vickyValidatorInfo, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, vickyValAddr[0])
	require.NoError(err)
	require.Equal(vickyValidatorInfo.Status, stakingtypes.Bonded)
	require.Equal(vickyValidatorInfo.Tokens, math.NewInt(1000*1e6))

	robPrivKey := secp256k1.GenPrivKey()
	robAccAddr := sdk.AccAddress(robPrivKey.PubKey().Address())
	robInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(100*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(robInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, robAccAddr, sdk.NewCoins(robInitCoins)))

	// rob delegates to vicky
	msgDelegate := stakingtypes.NewMsgDelegate(
		robAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(100*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	// rob becomes a reporter
	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   robAccAddr.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewInt(1 * 1e6),
	})
	require.NoError(err)
	robReporterInfo, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, robAccAddr)
	require.NoError(err)
	require.Equal(robReporterInfo.Jailed, false)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - Delwood delegates 250 trb to Vicky
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(2)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	delwoodPrivKey := secp256k1.GenPrivKey()
	delwoodAccAddr := sdk.AccAddress(delwoodPrivKey.PubKey().Address())
	delwoodInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(250*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(delwoodInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, delwoodAccAddr, sdk.NewCoins(delwoodInitCoins)))

	msgDelegate = stakingtypes.NewMsgDelegate(
		delwoodAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(250*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - Delwood selects 250 trb to Rob
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(3)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = msgServerReporter.SelectReporter(s.Setup.Ctx, &reportertypes.MsgSelectReporter{
		SelectorAddress: delwoodAccAddr.String(),
		ReporterAddress: robAccAddr.String(),
	})
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - Rob creates a second reporter
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(4)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	rob2PrivKey := secp256k1.GenPrivKey()
	rob2AccAddr := sdk.AccAddress(rob2PrivKey.PubKey().Address())
	rob2InitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(50*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(rob2InitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, rob2AccAddr, sdk.NewCoins(rob2InitCoins)))

	// rob delegates to vicky
	msgDelegate = stakingtypes.NewMsgDelegate(
		rob2AccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(50*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	// rob2 becomes a reporter
	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   rob2AccAddr.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewInt(1 * 1e6),
	})
	require.NoError(err)
	rob2ReporterInfo, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, rob2AccAddr)
	require.NoError(err)
	require.Equal(rob2ReporterInfo.Jailed, false)

	// create third party ricky the reporter to vote from reporter group
	rickyPrivKey := secp256k1.GenPrivKey()
	rickyAccAddr := sdk.AccAddress(rickyPrivKey.PubKey().Address())
	rickyInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(2000*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(rickyInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, rickyAccAddr, sdk.NewCoins(rickyInitCoins)))

	// ricky delegates to vicky
	msgDelegate = stakingtypes.NewMsgDelegate(
		rickyAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(1000*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	// ricky becomes a reporter
	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   rickyAccAddr.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewInt(1 * 1e6),
	})
	require.NoError(err)
	rickyReporterInfo, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, rickyAccAddr)
	require.NoError(err)
	require.Equal(rickyReporterInfo.Jailed, false)

	// ricky tips for more voting power
	queryData, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	msgTip := oracletypes.MsgTip{
		Tipper:    rickyAccAddr.String(),
		QueryData: queryData,
		Amount:    sdk.NewCoin(s.Setup.Denom, math.NewInt(10*1e6)),
	}
	_, err = msgServerOracle.Tip(s.Setup.Ctx, &msgTip)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5 - Rob2 makes a fine report
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(5)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// balances before reporting/disputing
	robReporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, robAccAddr)
	require.NoError(err)
	require.Equal(robReporterStake, math.NewInt(350*1e6))

	rob2ReporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, rob2AccAddr)
	require.NoError(err)
	require.Equal(rob2ReporterStake, math.NewInt(50*1e6))

	delwoodSelectionStake, err := s.Setup.Reporterkeeper.GetDelegatorTokensAtBlock(s.Setup.Ctx, delwoodAccAddr, 5)
	require.NoError(err)
	require.Equal(delwoodSelectionStake, math.NewInt(250*1e6))

	vickyValidatorStakedTokens, err := s.Setup.Stakingkeeper.GetLastValidatorPower(s.Setup.Ctx, vickyValAddr[0])
	require.NoError(err)
	require.Equal(vickyValidatorStakedTokens, int64(2400))

	// rob2 makes a report
	currentCycleList, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryId := utils.QueryIDFromData(currentCycleList)
	msgSubmitValue := oracletypes.MsgSubmitValue{
		Creator:   rob2AccAddr.String(),
		QueryData: currentCycleList,
		Value:     testutil.EncodeValue(100_000),
	}
	_, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &msgSubmitValue)
	require.NoError(err)

	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	getAggReportRequest := oracletypes.QueryGetCurrentAggregateReportRequest{
		QueryId: hex.EncodeToString(queryId),
	}
	queryServer := oraclekeeper.NewQuerier(s.Setup.Oraclekeeper)
	result, err := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(int64(0), result.Aggregate.AggregateReportIndex)
	require.Equal(testutil.EncodeValue(100_000), result.Aggregate.AggregateValue)
	require.Equal(rob2AccAddr.String(), result.Aggregate.AggregateReporter)
	require.Equal(queryId, result.Aggregate.QueryId)
	require.Equal(int64(50), result.Aggregate.ReporterPower)
	require.Equal(int64(5), result.Aggregate.Height)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 6 - Rob disputes Rob2s report (minor - 5%)
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(6)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// double check balances
	robReporterStake, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, robAccAddr)
	require.NoError(err)
	require.Equal(robReporterStake, math.NewInt(350*1e6))

	rob2ReporterStake, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, rob2AccAddr)
	require.NoError(err)
	require.Equal(rob2ReporterStake, math.NewInt(50*1e6))

	delwoodSelectionStake, err = s.Setup.Reporterkeeper.GetDelegatorTokensAtBlock(s.Setup.Ctx, delwoodAccAddr, 6)
	require.NoError(err)
	require.Equal(delwoodSelectionStake, math.NewInt(250*1e6))

	vickyValidatorStakedTokens, err = s.Setup.Stakingkeeper.GetLastValidatorPower(s.Setup.Ctx, vickyValAddr[0])
	require.NoError(err)
	require.Equal(vickyValidatorStakedTokens, int64(2400))

	// rob proposes a minor dispute
	report := oracletypes.MicroReport{
		Reporter:    rob2AccAddr.String(),
		Power:       rob2ReporterStake.Quo(layertypes.PowerReduction).Int64(),
		QueryId:     queryId,
		Value:       testutil.EncodeValue(100_000),
		Timestamp:   s.Setup.Ctx.BlockTime(),
		BlockNumber: int64(5),
	}

	// create msg for propose dispute tx
	msgProposeDispute := disputetypes.MsgProposeDispute{
		Creator:         robAccAddr.String(),
		Report:          &report,
		DisputeCategory: disputetypes.Minor,
		Fee:             sdk.NewCoin(s.Setup.Denom, math.NewInt(2.5*1e6)),
		PayFromBond:     true,
	}

	_, err = msgServerDispute.ProposeDispute(s.Setup.Ctx, &msgProposeDispute)
	require.NoError(err)

	disputes, err := s.Setup.Disputekeeper.GetOpenDisputes(s.Setup.Ctx)
	require.NoError(err)
	require.NotNil(disputes)
	require.Equal(len(disputes), 1)

	burnAmount := msgProposeDispute.Fee.Amount.MulRaw(1).QuoRaw(20)
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	require.NoError(err)
	require.Equal(dispute.DisputeId, uint64(1))
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeCategory, disputetypes.Minor)
	require.Equal(dispute.DisputeFee, msgProposeDispute.Fee.Amount.Sub(burnAmount))
	// require.Equal(dispute.BlockNumber, int64(5))
	require.Equal(dispute.DisputeStartBlock, int64(6))
	feepayer, err := s.Setup.Disputekeeper.DisputeFeePayer.Get(s.Setup.Ctx, collections.Join(uint64(1), robAccAddr.Bytes()))
	require.NoError(err)
	require.Equal(feepayer.Amount, msgProposeDispute.Fee.Amount)
	require.Equal(feepayer.FromBond, true)
	slashAmount := dispute.SlashAmount
	fmt.Println("slashAmount", slashAmount)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - Everyone votes No on the dispute
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(7)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// rob2 unjails himself
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Hour))
	msgUnjailReporter := reportertypes.MsgUnjailReporter{
		ReporterAddress: rob2AccAddr.String(),
	}
	_, err = msgServerReporter.UnjailReporter(s.Setup.Ctx, &msgUnjailReporter)
	require.NoError(err)
	rob2ReporterInfo, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, rob2AccAddr)
	require.NoError(err)
	require.Equal(rob2ReporterInfo.Jailed, false)

	// check balances
	vickyValidatorStakedTokens, err = s.Setup.Stakingkeeper.GetLastValidatorPower(s.Setup.Ctx, vickyValAddr[0])
	require.NoError(err)
	require.Equal(vickyValidatorStakedTokens, int64(2395)) // 2.5 trb gone from Rob, 2.5 trb gone from Rob2

	robReporterStake, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, robAccAddr)
	require.NoError(err)
	require.Equal(robReporterStake, math.NewInt(347.5*1e6)) // Paid 5% of 50 trb to open dispute

	rob2ReporterStake, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, rob2AccAddr)
	require.NoError(err)
	require.Equal(rob2ReporterStake, math.NewInt(47.5*1e6)) // 5% slashed from 50 trb

	delwoodSelectionStake, err = s.Setup.Reporterkeeper.GetDelegatorTokensAtBlock(s.Setup.Ctx, delwoodAccAddr, 7)
	require.NoError(err)
	delwoodsFee := ((250 * 2.5) / 350) * 1e6 // ((250/350) * 2.5) trb = ~1.785 trb
	fmt.Println("delwoodsFee", delwoodsFee)
	require.Equal(delwoodSelectionStake, math.NewInt(250*1e6).Sub(math.NewInt(int64(delwoodsFee)))) // ~1.785 of delwoods trb used towards dispute

	// free floating token group votes no
	fmt.Println("Vicky's Vote")
	msgVoteVicky := disputetypes.MsgVote{
		Voter: vickyAccAddr[0].String(),
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_AGAINST,
	}
	voteResponse, err := msgServerDispute.Vote(s.Setup.Ctx, &msgVoteVicky)
	require.NoError(err)
	require.NotNil(voteResponse)

	// team votes no
	fmt.Println("Team's Vote")
	teamAddr, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	require.NoError(err)
	msgVoteTeam := disputetypes.MsgVote{
		Voter: teamAddr.String(),
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_AGAINST,
	}
	voteResponse, err = msgServerDispute.Vote(s.Setup.Ctx, &msgVoteTeam)
	require.NoError(err)
	require.NotNil(voteResponse)

	// // ricky votes no -- no voting power ?
	fmt.Println("Ricky's Vote")
	msgVoteRicky := disputetypes.MsgVote{
		Voter: rickyAccAddr.String(),
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_AGAINST,
	}
	voteResponse, err = msgServerDispute.Vote(s.Setup.Ctx, &msgVoteRicky)
	require.NoError(err)
	require.NotNil(voteResponse)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	// ---------------------------------------------------------------------------
	// Height 8 - quorom reached, advance 3 days, tally votes
	// ---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(8)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// check on dispute status
	disputeByReporter, err := s.Setup.Disputekeeper.GetDisputeByReporter(s.Setup.Ctx, report, disputetypes.Minor)
	require.NoError(err)
	require.Equal(disputeByReporter.DisputeId, uint64(1))
	require.Equal(disputeByReporter.DisputeStatus, disputetypes.Voting)
	require.Equal(disputeByReporter.DisputeCategory, disputetypes.Minor)
	require.Equal(disputeByReporter.DisputeFee, msgProposeDispute.Fee.Amount.Sub(burnAmount))
	require.Equal(disputeByReporter.DisputeStartBlock, int64(6))

	// fast forward 3 days
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(disputekeeper.THREE_DAYS))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	// tally votes
	err = s.Setup.Disputekeeper.TallyVote(s.Setup.Ctx, 1)
	require.NoError(err)

	// execute vote
	msgExecuteDispute := disputetypes.MsgExecuteDispute{
		CallerAddress: rob2AccAddr.String(),
		DisputeId:     1,
	}
	_, err = msgServerDispute.ExecuteDispute(s.Setup.Ctx, &msgExecuteDispute)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	// ---------------------------------------------------------------------------
	// Height 9 - rob2 unjails himself, check balances
	// ---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(9)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// check on dispute status
	disputeByReporter, err = s.Setup.Disputekeeper.GetDisputeByReporter(s.Setup.Ctx, report, disputetypes.Minor)
	require.NoError(err)
	require.Equal(disputeByReporter.DisputeId, uint64(1))
	require.Equal(disputeByReporter.DisputeStatus, disputetypes.Resolved)

	// robReporterStake, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, robAccAddr)
	// require.NoError(err)
	// require.Equal(robReporterStake, math.NewInt(350*1e6))

	// rob2ReporterStake, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, rob2AccAddr)
	// require.NoError(err)
	// require.Equal(rob2ReporterStake, math.NewInt(50*1e6))

	// delwoodSelectionStake, err = s.Setup.Reporterkeeper.GetDelegatorTokensAtBlock(s.Setup.Ctx, delwoodAccAddr, 6)
	// require.NoError(err)
	// require.Equal(delwoodSelectionStake, math.NewInt(250*1e6))

	// vickyValidatorStakedTokens, err = s.Setup.Stakingkeeper.GetLastValidatorPower(s.Setup.Ctx, vickyValAddr[0])
	// require.NoError(err)
	// require.Equal(vickyValidatorStakedTokens, int64(2400))

	// FINAL RESULTS:
	vickyFreeFloating := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, vickyAccAddr[0], s.Setup.Denom)
	vickyStaked, err := s.Setup.Stakingkeeper.GetLastValidatorPower(s.Setup.Ctx, vickyValAddr[0])
	require.NoError(err)
	fmt.Println("\nvicky before the dispute: ", 2400)
	fmt.Println("vickyStaked: ", vickyStaked)
	fmt.Println("vickyFreeFloating: ", vickyFreeFloating.Amount)
	fmt.Println("vicky total after dispute: ", vickyStaked+vickyFreeFloating.Amount.Int64())

	robFreeFloating := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, robAccAddr, s.Setup.Denom)
	robReporterStake, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, robAccAddr)
	require.NoError(err)
	fmt.Println("\nrob before the dispute: ", 350)
	fmt.Println("robReporterStake", robReporterStake)
	fmt.Println("robFreeFloating: ", robFreeFloating.Amount)
	fmt.Println("rob total after dispute: ", robReporterStake.Int64()+robFreeFloating.Amount.Int64())

	rob2FreeFloating := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, rob2AccAddr, s.Setup.Denom)
	rob2ReporterStake, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, rob2AccAddr)
	require.NoError(err)
	fmt.Println("\nrob2 before the dispute: ", 50)
	fmt.Println("rob2ReporterStake", rob2ReporterStake)
	fmt.Println("rob2FreeFloating: ", rob2FreeFloating.Amount)
	fmt.Println("rob2 total after dispute: ", rob2ReporterStake.Int64()+rob2FreeFloating.Amount.Int64())

	delwoodFreeFloating := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, delwoodAccAddr, s.Setup.Denom)
	delwoodSelectionStake, err = s.Setup.Reporterkeeper.GetDelegatorTokensAtBlock(s.Setup.Ctx, delwoodAccAddr, 9)
	require.NoError(err)
	fmt.Println("\ndelwood before the dispute: ", 250)
	fmt.Println("delwoodSelectionStake", delwoodSelectionStake)
	fmt.Println("delwoodFreeFloating: ", delwoodFreeFloating.Amount)
	fmt.Println("delwood total after dispute: ", delwoodSelectionStake.Int64()+delwoodFreeFloating.Amount.Int64())

	rickyFreeFloating := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, rickyAccAddr, s.Setup.Denom)
	rickyReporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, rickyAccAddr)
	require.NoError(err)
	fmt.Println("\nricky before the dispute: ", 1999)
	fmt.Println("rickyReporterStake", rickyReporterStake)
	fmt.Println("rickyFreeFloating: ", rickyFreeFloating.Amount)
	fmt.Println("ricky total after dispute: ", rickyReporterStake.Int64()+rickyFreeFloating.Amount.Int64())
}

// Vicky the Validator has 1000 trb staked
// Rob the Reporter has 100 trb staked with Vicky, selects himself as a reporter
// Ricky the Reporter has 100 trb staked with Vicky, selects himself as a reporter
// Delwood the Delegator has 250 trb delegated to Rob
// Delwood tries to dispute a report from Ricky to eliminate his competition
// fails
func (s *E2ETestSuite) TestDisputeFromDelegatorPayFromBond() {
	// Setup msgServers
	require := s.Require()
	msgServerOracle := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	require.NotNil(msgServerOracle)
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	require.NotNil(msgServerReporter)
	msgServerDispute := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	require.NotNil(msgServerDispute)
	msgServerStaking := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	require.NotNil(msgServerStaking)

	//---------------------------------------------------------------------------
	// Height 0 - vicky becomes a validator
	//---------------------------------------------------------------------------
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	vickyAccAddr := simtestutil.CreateIncrementalAccounts(1)
	vickyInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(2000*1e6)) // give vicky extra to act as free floating token voting group
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(vickyInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, vickyAccAddr[0], sdk.NewCoins(vickyInitCoins)))
	s.Setup.Accountkeeper.NewAccountWithAddress(s.Setup.Ctx, vickyAccAddr[0])

	pubKey := simtestutil.CreateTestPubKeys(1)
	vickyValAddr := simtestutil.ConvertAddrsToValAddrs(vickyAccAddr)
	msgCreateValidator, err := stakingtypes.NewMsgCreateValidator(
		vickyValAddr[0].String(),
		pubKey[0],
		sdk.NewCoin(s.Setup.Denom, math.NewInt(1000*1e6)),
		stakingtypes.Description{Moniker: "created validator"},
		stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(0, 0), math.LegacyNewDecWithPrec(3, 1), math.LegacyNewDecWithPrec(1, 1)),
		math.OneInt(),
	)
	require.NoError(err)

	_, err = msgServerStaking.CreateValidator(s.Setup.Ctx, msgCreateValidator)
	require.NoError(err)

	require.NoError(s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, vickyValAddr[0].String(), []byte("vickyEvmAddr")))

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - Rob delegates to Vicky and selects himself to become a reporter
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// verify vicky is a bonded validator
	vickyValidatorInfo, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, vickyValAddr[0])
	require.NoError(err)
	require.Equal(vickyValidatorInfo.Status, stakingtypes.Bonded)
	require.Equal(vickyValidatorInfo.Tokens, math.NewInt(1000*1e6))

	robPrivKey := secp256k1.GenPrivKey()
	robAccAddr := sdk.AccAddress(robPrivKey.PubKey().Address())
	robInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(100*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(robInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, robAccAddr, sdk.NewCoins(robInitCoins)))

	// rob delegates to vicky
	msgDelegate := stakingtypes.NewMsgDelegate(
		robAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(100*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	// rob becomes a reporter
	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   robAccAddr.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewInt(1 * 1e6),
	})
	require.NoError(err)
	robReporterInfo, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, robAccAddr)
	require.NoError(err)
	require.Equal(robReporterInfo.Jailed, false)

	rickyPrivKey := secp256k1.GenPrivKey()
	rickyAccAddr := sdk.AccAddress(rickyPrivKey.PubKey().Address())
	rickyInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(2000*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(rickyInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, rickyAccAddr, sdk.NewCoins(rickyInitCoins)))

	// ricky delegates to vicky
	msgDelegate = stakingtypes.NewMsgDelegate(
		rickyAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(1000*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	// ricky becomes a reporter
	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   rickyAccAddr.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewInt(1 * 1e6),
	})
	require.NoError(err)
	rickyReporterInfo, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, rickyAccAddr)
	require.NoError(err)
	require.Equal(rickyReporterInfo.Jailed, false)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - Delwood delegates 250 trb to Vicky
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(2)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	delwoodPrivKey := secp256k1.GenPrivKey()
	delwoodAccAddr := sdk.AccAddress(delwoodPrivKey.PubKey().Address())
	delwoodInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(250*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(delwoodInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, delwoodAccAddr, sdk.NewCoins(delwoodInitCoins)))

	msgDelegate = stakingtypes.NewMsgDelegate(
		delwoodAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(250*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - Delwood selects 250 trb to Rob
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(3)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = msgServerReporter.SelectReporter(s.Setup.Ctx, &reportertypes.MsgSelectReporter{
		SelectorAddress: delwoodAccAddr.String(),
		ReporterAddress: robAccAddr.String(),
	})
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - Ricky reports for the cycle list
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(4)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	currentCycleList, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryId := utils.QueryIDFromData(currentCycleList)
	msgSubmitValue := oracletypes.MsgSubmitValue{
		Creator:   rickyAccAddr.String(),
		QueryData: currentCycleList,
		Value:     testutil.EncodeValue(100_000),
	}
	_, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &msgSubmitValue)
	require.NoError(err)

	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	getAggReportRequest := oracletypes.QueryGetCurrentAggregateReportRequest{
		QueryId: hex.EncodeToString(queryId),
	}
	queryServer := oraclekeeper.NewQuerier(s.Setup.Oraclekeeper)
	result, err := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(int64(0), result.Aggregate.AggregateReportIndex)
	require.Equal(testutil.EncodeValue(100_000), result.Aggregate.AggregateValue)
	require.Equal(rickyAccAddr.String(), result.Aggregate.AggregateReporter)
	require.Equal(queryId, result.Aggregate.QueryId)
	require.Equal(int64(1000), result.Aggregate.ReporterPower)
	require.Equal(int64(4), result.Aggregate.Height)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5 - Delwood proposes a dispute from bond
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(5)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	rickyReporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, rickyAccAddr)
	require.NoError(err)

	report := oracletypes.MicroReport{
		Reporter:    rickyAccAddr.String(),
		Power:       rickyReporterStake.Quo(layertypes.PowerReduction).Int64(),
		QueryId:     queryId,
		Value:       testutil.EncodeValue(100_000),
		Timestamp:   s.Setup.Ctx.BlockTime(),
		BlockNumber: int64(4),
	}

	msgProposeDispute := disputetypes.MsgProposeDispute{
		Creator:         delwoodAccAddr.String(),
		Report:          &report,
		DisputeCategory: disputetypes.Warning,
		Fee:             sdk.NewCoin(s.Setup.Denom, math.NewInt(10*1e6)),
		PayFromBond:     true,
	}

	_, err = msgServerDispute.ProposeDispute(s.Setup.Ctx, &msgProposeDispute)
	require.Error(err)
}

// test precision loss throughout tip/report/dispute/claim process

