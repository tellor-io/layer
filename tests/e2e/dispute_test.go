package e2e_test

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"github.com/tellor-io/layer/testutil"
	layertypes "github.com/tellor-io/layer/types"
	utils "github.com/tellor-io/layer/utils"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	collections "cosmossdk.io/collections"
	math "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	secp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
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
	require.Equal(testutil.EncodeValue(100_000), result.Aggregate.AggregateValue)
	require.Equal(reporterAccount.String(), result.Aggregate.AggregateReporter)
	require.Equal(queryId, result.Aggregate.QueryId)
	require.Equal(uint64(4000), result.Aggregate.AggregatePower)
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

	balBeforeDispute, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterAccount, []byte{})
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
	require.Equal(testutil.EncodeValue(100_000), result.Aggregate.AggregateValue)
	require.Equal(reporterAccount.String(), result.Aggregate.AggregateReporter)
	require.Equal(queryId, result.Aggregate.QueryId)
	require.Equal(uint64(4000)-slashAmount.Quo(sdk.DefaultPowerReduction).Uint64(), result.Aggregate.AggregatePower)
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

	balBeforeDispute, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterAccount, queryId)
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

	oneHundredPercent, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterAccount, queryId)
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

// Vicky the Validator has 1000 trb staked
// Rob the Reporter has 100 trb staked with Vicky, selects himself as a reporter
// Ricky the Reporter has 100 trb staked with Vicky, selects himself as a reporter
// Delwood the Delegator has 250 trb selected to Rob
// Delwood tries to dispute Rickys report
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

	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 2) // advance to 6, call 6 endblocker
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	getAggReportRequest := oracletypes.QueryGetCurrentAggregateReportRequest{
		QueryId: hex.EncodeToString(queryId),
	}
	queryServer := oraclekeeper.NewQuerier(s.Setup.Oraclekeeper)
	result, err := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(testutil.EncodeValue(100_000), result.Aggregate.AggregateValue)
	require.Equal(rickyAccAddr.String(), result.Aggregate.AggregateReporter)
	require.Equal(queryId, result.Aggregate.QueryId)
	require.Equal(uint64(1000), result.Aggregate.AggregatePower)
	require.Equal(uint64(6), result.Aggregate.Height)

	//---------------------------------------------------------------------------
	// Height 5 - Delwood proposes a dispute from bond
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(7)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	rickyReporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, rickyAccAddr, queryId)
	require.NoError(err)

	report := oracletypes.MicroReport{
		Reporter:    rickyAccAddr.String(),
		Power:       rickyReporterStake.Quo(layertypes.PowerReduction).Uint64(),
		QueryId:     queryId,
		Value:       testutil.EncodeValue(100_000),
		Timestamp:   s.Setup.Ctx.BlockTime(),
		BlockNumber: uint64(4),
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
// 2 validators anna and bob become reporters
// chris delegates and selects to anna
// chris tips between 1 loya and 1 trb to get matic/usd spot price
// bob reports matic/usd price inaccurately
// anna disputes bob's report
func (s *E2ETestSuite) TestOpenDisputePrecision() {
	require := s.Require()

	// setup msgServers
	msgServerOracle := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	require.NotNil(msgServerOracle)
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	require.NotNil(msgServerReporter)
	msgServerDispute := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	require.NotNil(msgServerDispute)
	msgServerStaking := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	require.NotNil(msgServerStaking)

	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	ctx := s.Setup.Ctx

	// create 2 validators with random stakes 1 trb to 200,001 trb
	valAccAddrs, valAddrs, _, stakes := s.Setup.CreateValidatorsRandomStake(2)
	annaAccAddr := valAccAddrs[0]
	bobAccAddr := valAccAddrs[1]
	annaValAddr := valAddrs[0]
	// bobValAddr := valAddrs[1]
	annaInitialStake := stakes[0]
	bobInitialStake := stakes[1]

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	// ________________________________________________________________________
	// Block 2 - anna and bob become reporters
	// ________________________________________________________________________
	ctx = ctx.WithBlockHeight(2)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	// anna becomes a reporter
	_, err = s.Setup.CreateReporter(ctx, annaAccAddr, reportertypes.DefaultMinCommissionRate, math.NewInt(1*1e6))
	require.NoError(err)
	// verify anna's reporter power
	annaReporterStake, err := s.Setup.Reporterkeeper.ReporterStake(ctx, annaAccAddr, []byte{})
	require.NoError(err)
	require.Equal(math.NewInt(annaInitialStake).String(), annaReporterStake.String())

	// bob becomes a reporter
	_, err = s.Setup.CreateReporter(ctx, bobAccAddr, reportertypes.DefaultMinCommissionRate, math.NewInt(1*1e6))
	require.NoError(err)
	// verify bobs reporter power
	bobReporterStake, err := s.Setup.Reporterkeeper.ReporterStake(ctx, bobAccAddr, []byte{})
	require.NoError(err)
	require.Equal(math.NewInt(bobInitialStake).String(), bobReporterStake.String())

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	// ________________________________________________________________________
	// Block 3 - chris delegates to anna and selects annas reporter
	// ________________________________________________________________________
	ctx = ctx.WithBlockHeight(3)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	// chris has 250k trb
	chrisAccAddr, err := s.Setup.CreateFundedAccount(250_000)
	require.NoError(err)

	// chris delegates a random amount of trb to annas validator, selects the same amount to annas reporter
	// amount is between 1 trb and 200k trb
	randomAmountLoya := rand.Int63n(200_000*1e6) + 1*1e6
	s.Setup.DelegateAndSelect(msgServerStaking, msgServerReporter, math.NewInt(randomAmountLoya), chrisAccAddr, annaValAddr, annaAccAddr)

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	// ________________________________________________________________________
	// Block 4 - chris tips small amount of trb to get matic/usd spot price
	// ________________________________________________________________________
	ctx = ctx.WithBlockHeight(4)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	// check on delegation from chris to anna validator
	chrisDelegation, err := s.Setup.Stakingkeeper.Delegation(ctx, chrisAccAddr, annaValAddr)
	require.NoError(err)
	require.Equal(randomAmountLoya, chrisDelegation.GetShares().TruncateInt64())
	// check on selection from chris to anna reporter
	annaReporterStake, err = s.Setup.Reporterkeeper.ReporterStake(ctx, annaAccAddr, []byte{})
	require.NoError(err)
	expectedAnnaPower := math.NewInt(randomAmountLoya).Add(math.NewInt(annaInitialStake))
	require.Equal(expectedAnnaPower.String(), annaReporterStake.String())

	// chris tips random fraction of trb to get matic/usd spot price
	// tip is between 1 loya and 1 trb
	randomTipAmount := math.NewInt(rand.Int63n(1*1e6) + 1)
	maticQueryData := s.Setup.CreateSpotPriceTip(ctx, chrisAccAddr, `["matic","usd"]`, randomTipAmount)

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	// ________________________________________________________________________
	// Block 5 - bob reports a bad matic/usd price
	// ________________________________________________________________________
	ctx = ctx.WithBlockHeight(5)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	// check on tip
	maticQueryId := utils.QueryIDFromData(maticQueryData)
	maticTip, err := s.Setup.Oraclekeeper.GetQueryTip(ctx, maticQueryId)
	require.NoError(err)
	// 2% of tip is burned
	burn := randomTipAmount.Mul(math.NewInt(2)).Quo(math.NewInt(100))
	expectedTip := randomTipAmount.Sub(burn)
	require.Equal(expectedTip.String(), maticTip.String())

	// bob reports 100 as matic/usd spot price
	bobReportValue := testutil.EncodeValue(100)
	s.Setup.Report(ctx, bobAccAddr, maticQueryData, bobReportValue)

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	// ________________________________________________________________________
	// Block 6 - last block to report matic/usd, check on query in collections
	// ________________________________________________________________________
	ctx = ctx.WithBlockHeight(6)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	// check on matic/usd query
	query, err := s.Setup.Oraclekeeper.CurrentQuery(ctx, maticQueryId)
	require.NoError(err)
	require.Equal(query.QueryData, maticQueryData)
	require.Equal(query.Amount, expectedTip)
	require.Equal(query.HasRevealedReports, true)
	require.Equal(query.CycleList, false)
	require.Equal(query.RegistrySpecBlockWindow, uint64(2))
	require.Equal(query.Expiration, uint64(6))

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	// ________________________________________________________________________
	// Block 7 - report gets aggregated, anna opens dispute
	// ________________________________________________________________________
	ctx = ctx.WithBlockHeight(7)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	// query should no longer be in query collections because it got aggregated
	query, err = s.Setup.Oraclekeeper.Query.Get(ctx, collections.Join(maticQueryId, uint64(2)))
	require.ErrorContains(err, "not found")

	// get microreport to submit in dispute
	oracleQuerier := oraclekeeper.NewQuerier(s.Setup.Oraclekeeper)
	report, err := oracleQuerier.GetReportsbyQid(ctx, &oracletypes.QueryGetReportsbyQidRequest{
		QueryId: hex.EncodeToString(maticQueryId),
	})
	require.NoError(err)

	// for a warning dispute, 1% of the report's power is the dispute fee (or 1 trb if 1% is less than 1 trb)
	fmt.Println("report power: ", report.MicroReports[0].Power)
	stake := layertypes.PowerReduction.MulRaw(int64(report.MicroReports[0].Power))
	disputeFeeTotal := stake.Mul(math.NewInt(1)).Quo(math.NewInt(100))
	if disputeFeeTotal.LT(layertypes.OnePercent) {
		disputeFeeTotal = layertypes.OnePercent
	}
	fmt.Println("dispute fee total: ", disputeFeeTotal)
	// anna opens dispute
	disputeStartTime := ctx.BlockTime()
	s.Setup.OpenDispute(ctx, annaAccAddr, report.MicroReports[0], disputetypes.Warning, disputeFeeTotal, true)

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	// ________________________________________________________________________
	// Block 8 - check on dispute, everybody votes
	// _________________________________________________________________________
	ctx = ctx.WithBlockHeight(8)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	// check on dispute
	disputes, err := s.Setup.Disputekeeper.GetOpenDisputes(ctx)
	require.NoError(err)
	require.Equal(len(disputes), 1)
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(ctx, disputes[0])
	require.NoError(err)
	require.Equal(dispute.DisputeId, disputes[0])
	fmt.Println("dispute status: ", dispute.DisputeStatus)
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeCategory, disputetypes.Warning)
	require.Equal(dispute.FeeTotal, disputeFeeTotal)
	// disputeFee should be 95% of fee total, 5% is burned
	disputeFee := disputeFeeTotal.Mul(math.NewInt(95)).Quo(math.NewInt(100))
	require.Equal(dispute.DisputeFee, disputeFee)
	disputeFeeBurn := disputeFeeTotal.Sub(disputeFee)
	require.Equal(dispute.BurnAmount, disputeFeeBurn)
	require.Equal(dispute.BlockNumber, uint64(7))
	// dispute ends in 3 days if fully paid
	require.Equal(dispute.DisputeEndTime, disputeStartTime.Add(3*time.Hour*24))
	require.Equal(dispute.DisputeStartTime, disputeStartTime)
	require.Equal(dispute.DisputeRound, uint64(1))
	require.Equal(dispute.SlashAmount, disputeFeeTotal)
	require.Equal(dispute.InitialEvidence, report.MicroReports[0])

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)
}

func (s *E2ETestSuite) TestDisputes2() {
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
	// Height 0 - create 3 validators and 3 reporters
	//---------------------------------------------------------------------------
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	valsAcctAddrs, valsValAddrs, _ := s.Setup.CreateValidators(3)
	require.NotNil(valsAcctAddrs)
	repsAccs := valsAcctAddrs
	for _, val := range valsValAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
	badReporter := repsAccs[0]
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, badReporter, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, badReporter, reportertypes.NewSelection(badReporter, 1)))
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repsAccs[1], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repsAccs[1], reportertypes.NewSelection(repsAccs[1], 1)))
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repsAccs[2], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repsAccs[2], reportertypes.NewSelection(repsAccs[2], 1)))
	// mapping to track reporter delegation balance
	// reporterToBalanceMap := make(map[string]math.Int)
	// for _, acc := range repsAccs {
	// 	rkDelegation, err := s.Setup.Reporterkeeper.Delegators.Get(s.Setup.Ctx, acc)
	// 	require.NoError(err)
	// 	reporterToBalanceMap[acc.String()] = rkDelegation.Amount
	// }

	//---------------------------------------------------------------------------
	// Height 1 - delegate 500 trb to validator 0 and bad reporter
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	pk := ed25519.GenPrivKey()
	delAcc := s.Setup.ConvertToAccAddress([]ed25519.PrivKey{*pk})
	delAccAddr := delAcc[0]
	initCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(500*1e6))
	s.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, delAccAddr, sdk.NewCoins(initCoins)))

	// delegate to validator 0
	s.Setup.MintTokens(delAccAddr, math.NewInt(500*1e6))
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, &stakingtypes.MsgDelegate{DelegatorAddress: delAccAddr.String(), ValidatorAddress: valsValAddrs[0].String(), Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(500*1e6))})
	require.NoError(err)
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, delAccAddr, reportertypes.NewSelection(badReporter, 1)))

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	val, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valsValAddrs[0])
	require.NoError(err)
	repTokens, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, badReporter, []byte{})
	require.NoError(err)
	require.Equal(repTokens, val.Tokens)

	//---------------------------------------------------------------------------
	// Height 2 - direct reveal for cycle list
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(2)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, repsAccs[0])
	require.NoError(err)

	// get new cycle list query data
	cycleListQuery, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryId := utils.QueryIDFromData(cycleListQuery)
	// create reveal message
	value := testutil.EncodeValue(10_000)
	require.NoError(err)
	reveal := oracletypes.MsgSubmitValue{
		Creator:   repsAccs[0].String(),
		QueryData: cycleListQuery,
		Value:     value,
	}
	reportBlock := s.Setup.Ctx.BlockHeight()
	// send reveal message
	revealResponse, err := msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal)
	require.NoError(err)
	require.NotNil(revealResponse)
	revealTime := s.Setup.Ctx.BlockTime()
	// advance time and block height to expire the query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - open warning, pay from bond from reporter 1
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(3)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	// todo: is there a getter for this ?
	// get microreport for dispute
	report := oracletypes.MicroReport{
		Reporter:    repsAccs[0].String(),
		Power:       repTokens.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     queryId,
		Value:       value,
		Timestamp:   revealTime,
		BlockNumber: uint64(reportBlock),
	}

	// disputedBal := disputedRep.TotalTokens
	// onePercent := disputedBal.Mul(math.NewInt(1)).Quo(math.NewInt(100))
	fee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, disputetypes.Warning)
	require.NoError(err)
	disputeFee := sdk.NewCoin(s.Setup.Denom, fee) // warning should be 1% of bonded tokens

	// create msg for propose dispute tx
	msgProposeDispute := disputetypes.MsgProposeDispute{
		Creator:         repsAccs[0].String(),
		Report:          &report,
		DisputeCategory: disputetypes.Warning,
		Fee:             disputeFee,
		PayFromBond:     true,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.Setup.Ctx, &msgProposeDispute)
	require.NoError(err)

	burnAmount := disputeFee.Amount.MulRaw(1).QuoRaw(20)
	disputes, err := s.Setup.Disputekeeper.GetOpenDisputes(s.Setup.Ctx)
	require.NoError(err)
	require.NotNil(disputes)
	// dispute is created correctly
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 1)
	require.NoError(err)
	require.Equal(dispute.DisputeId, uint64(1))
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeCategory, disputetypes.Warning)
	require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	feepayer, err := s.Setup.Disputekeeper.DisputeFeePayer.Get(s.Setup.Ctx, collections.Join(uint64(1), repsAccs[0].Bytes()))
	require.NoError(err)
	require.Equal(feepayer.Amount, disputeFee.Amount)
	require.Equal(feepayer.FromBond, true)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - disputed reporter reports after calling unjail
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(4)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	disputedRep, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, repsAccs[0])
	require.NoError(err)
	require.Equal(disputedRep.Jailed, true)

	// disputed reporter cant report yet
	cycleListQuery, err = s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	value = testutil.EncodeValue(10_000)
	require.NoError(err)
	reveal = oracletypes.MsgSubmitValue{
		Creator:   repsAccs[0].String(),
		QueryData: cycleListQuery,
		Value:     value,
	}
	_, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal)
	require.Error(err)

	// disputed reporter can report after calling unjail function
	msgUnjail := reportertypes.MsgUnjailReporter{
		ReporterAddress: repsAccs[0].String(),
	}
	_, err = msgServerReporter.UnjailReporter(s.Setup.Ctx, &msgUnjail)
	require.NoError(err)
	disputedRep, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, repsAccs[0])
	require.NoError(err)
	require.Equal(disputedRep.Jailed, false)
	// send reveal message
	revealResponse, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal)
	require.NoError(err)
	require.NotNil(revealResponse)
	revealTime = s.Setup.Ctx.BlockTime()
	revealBlock := s.Setup.Ctx.BlockHeight()

	// give disputer tokens to pay for next disputes not from bond
	beforemint := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, repsAccs[1], s.Setup.Denom)
	initCoins = sdk.NewCoin(s.Setup.Denom, math.NewInt(10_000*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
	// send from module to account
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, repsAccs[1], sdk.NewCoins(initCoins)))
	require.Equal(beforemint.Add(initCoins), s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, repsAccs[1], s.Setup.Denom))

	// advance time and block height to expire the query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	// disputer, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, repsAccs[1])
	// require.NoError(err)
	disputedRep, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, repsAccs[0])
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5 - open warning, pay from not bond from reporter 1
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(5)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	report.Power = repTokens.Quo(sdk.DefaultPowerReduction).Uint64()
	fee, err = s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, disputetypes.Warning)
	require.NoError(err)
	disputeFee = sdk.NewCoin(s.Setup.Denom, fee) // warning should be 1% of bonded tokens

	// get microreport for dispute
	report = oracletypes.MicroReport{
		Reporter:    repsAccs[0].String(),
		Power:       repTokens.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     utils.QueryIDFromData(reveal.QueryData),
		Value:       value,
		Timestamp:   revealTime,
		BlockNumber: uint64(revealBlock),
	}

	// create msg for propose dispute tx
	msgProposeDispute = disputetypes.MsgProposeDispute{
		Creator:         repsAccs[1].String(),
		Report:          &report,
		DisputeCategory: disputetypes.Warning,
		Fee:             disputeFee,
		PayFromBond:     false,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.Setup.Ctx, &msgProposeDispute)
	require.NoError(err)

	burnAmount = disputeFee.Amount.MulRaw(1).QuoRaw(20)
	disputes, err = s.Setup.Disputekeeper.GetOpenDisputes(s.Setup.Ctx)
	require.NoError(err)
	require.NotNil(disputes)
	// dispute is created correctly
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, 2)
	require.NoError(err)
	require.Equal(dispute.DisputeId, uint64(2))
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeCategory, disputetypes.Warning)
	require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	feepayer, err = s.Setup.Disputekeeper.DisputeFeePayer.Get(s.Setup.Ctx, collections.Join(uint64(2), repsAccs[1].Bytes()))
	require.NoError(err)
	require.Equal(feepayer.Amount, disputeFee.Amount)
	require.Equal(feepayer.FromBond, false)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 6 - dispute is resolved, direct reveal again
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(6)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	disputedRep, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, repsAccs[0])
	require.NoError(err)
	require.Equal(disputedRep.Jailed, true)

	// disputed reporter cant report yet
	cycleListQuery, err = s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	value = testutil.EncodeValue(10_000)
	require.NoError(err)
	queryId = utils.QueryIDFromData(cycleListQuery)
	reveal = oracletypes.MsgSubmitValue{
		Creator:   repsAccs[0].String(),
		QueryData: cycleListQuery,
		Value:     value,
	}
	_, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal)
	require.Error(err)

	// disputed reporter can report after calling unjail function
	msgUnjail = reportertypes.MsgUnjailReporter{
		ReporterAddress: repsAccs[0].String(),
	}
	_, err = msgServerReporter.UnjailReporter(s.Setup.Ctx, &msgUnjail)
	require.NoError(err)
	disputedRep, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, repsAccs[0])
	require.NoError(err)
	require.Equal(disputedRep.Jailed, false)
	// send reveal message
	revealResponse, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal)
	require.NoError(err)
	require.NotNil(revealResponse)
	revealTime = s.Setup.Ctx.BlockTime()
	revealBlock = s.Setup.Ctx.BlockHeight()

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - open minor dispute, pay from bond from reporter 1
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(7)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	balBeforeDispute := repTokens
	fivePercent := balBeforeDispute.Mul(math.NewInt(5)).Quo(math.NewInt(100))
	disputeFee = sdk.NewCoin(s.Setup.Denom, fivePercent)

	report = oracletypes.MicroReport{
		Reporter:    repsAccs[0].String(),
		Power:       repTokens.Quo(sdk.DefaultPowerReduction).Uint64(),
		QueryId:     queryId,
		Value:       value,
		Timestamp:   revealTime,
		BlockNumber: uint64(revealBlock),
	}

	// create msg for propose dispute tx
	msgProposeDispute = disputetypes.MsgProposeDispute{
		Creator:         repsAccs[1].String(),
		Report:          &report,
		DisputeCategory: disputetypes.Minor,
		Fee:             disputeFee,
		PayFromBond:     true,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.Setup.Ctx, &msgProposeDispute)
	require.NoError(err)
	_ = s.Setup.Ctx.BlockTime()

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 8 - vote on minor dispute -- reaches quorum
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(8)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	// vote from disputer
	msgVote := disputetypes.MsgVote{
		Voter: repsAccs[0].String(),
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}
	fmt.Println("Dispute Id on rep[0] vote: ", dispute.DisputeId)
	voteResponse, err := msgServerDispute.Vote(s.Setup.Ctx, &msgVote)
	require.NoError(err)
	require.NotNil(voteResponse)

	// vote from disputed reporter
	msgVote = disputetypes.MsgVote{
		Voter: repsAccs[1].String(),
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}

	voteResponse, err = msgServerDispute.Vote(s.Setup.Ctx, &msgVote)
	require.NoError(err)
	require.NotNil(voteResponse)

	// vote from third reporter
	// thirdReporter, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, repsAccs[2])
	require.NoError(err)
	msgVote = disputetypes.MsgVote{
		Voter: repsAccs[2].String(),
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}
	voteResponse, err = msgServerDispute.Vote(s.Setup.Ctx, &msgVote)
	require.NoError(err)
	require.NotNil(voteResponse)

	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)

	// vote from team
	// fmt.Println(disputetypes.TeamAddress)
	// msgVote = disputetypes.MsgVote{
	// 	Voter: sdk.MustAccAddressFromBech32(disputetypes.TeamAddress).String(),
	// 	Id:    dispute.DisputeId,
	// 	Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	// }
	// voteResponse, err = msgServerDispute.Vote(s.Setup.Ctx, &msgVote)
	// require.NoError(err)
	// require.NotNil(voteResponse)

	totalTips, err := s.Setup.Disputekeeper.BlockInfo.Get(s.Setup.Ctx, dispute.HashId)
	require.NoError(err)
	fmt.Println("totalTips: ", totalTips)

	totalReporterPower, err := s.Setup.Reporterkeeper.TotalReporterPower(s.Setup.Ctx)
	require.NoError(err)
	fmt.Println("total reporter power: ", totalReporterPower.Quo(sdk.DefaultPowerReduction))

	totalFreeFloatingTokens := s.Setup.Disputekeeper.GetTotalSupply(s.Setup.Ctx)
	fmt.Println("total Free Floating Tokens: ", totalFreeFloatingTokens)
	owners, err := s.Setup.Bankkeeper.DenomOwners(s.Setup.Ctx, &banktypes.QueryDenomOwnersRequest{Denom: s.Setup.Denom})
	require.NoError(err)
	sumFromDenomOwners := math.ZeroInt()
	for _, owner := range owners.DenomOwners {
		fmt.Println("owner: ", owner)
		sumFromDenomOwners = sumFromDenomOwners.Add(owner.Balance.Amount)
	}
	fmt.Println("sumFromDenomOwners: ", sumFromDenomOwners)

	// print all reporter sdk.AccAddr
	for _, rep := range repsAccs {
		fmt.Println("rep: ", rep.String())
	}
	for _, val := range valsAcctAddrs {
		fmt.Println("val: ", val.String())
	}
	fmt.Println("delegator acc addr: ", delAccAddr.String())

	// print tbr module account address
	tbrModuleAccount := s.Setup.Accountkeeper.GetModuleAddress(minttypes.TimeBasedRewards) // yes
	fmt.Println("tbr module account: ", tbrModuleAccount.String())

	disputeModuleAccount := s.Setup.Accountkeeper.GetModuleAddress(disputetypes.ModuleName) // yes
	fmt.Println("dispute module account: ", disputeModuleAccount.String())

	authModuleAccount := s.Setup.Accountkeeper.GetModuleAddress(authtypes.ModuleName) //
	fmt.Println("auth module account: ", authModuleAccount.String())

	reporterModuleAccount := s.Setup.Accountkeeper.GetModuleAddress(reportertypes.ModuleName) // yes
	fmt.Println("reporter module account: ", reporterModuleAccount.String())

	registryModuleAccount := s.Setup.Accountkeeper.GetModuleAddress(registrytypes.ModuleName) // no
	fmt.Println("registry module account: ", registryModuleAccount.String())

	reporterTipsEscrowAccount := s.Setup.Accountkeeper.GetModuleAddress(reportertypes.TipsEscrowPool) // no
	fmt.Println("reporter tips escrow account: ", reporterTipsEscrowAccount.String())

	oracleModuleAccount := s.Setup.Accountkeeper.GetModuleAddress(oracletypes.ModuleName) // no
	fmt.Println("oracle module account: ", oracleModuleAccount.String())

	stakingModuleAccount := s.Setup.Accountkeeper.GetModuleAddress(stakingtypes.ModuleName) //
	fmt.Println("staking module account: ", stakingModuleAccount.String())

	//---------------------------------------------------------------------------
	// Height 9 - resolve dispute, direct reveal again
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(9)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	//---------------------------------------------------------------------------
	// Height 10 - open minor dispute, pay from not bond from reporter 1
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(10)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	//---------------------------------------------------------------------------
	// Height 11 - vote on minor dispute
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(11)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	//---------------------------------------------------------------------------
	// Height 12 - resolve dispute, direct reveal again
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(12)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	//---------------------------------------------------------------------------
	// Height 13 - open major dispute, pay from bond from reporter 1
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(13)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	//---------------------------------------------------------------------------
	// Height 14 - vote on major dispute
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(14)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	//---------------------------------------------------------------------------
	// Height 15 - resolve dispute, direct reveal again
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(15)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	//---------------------------------------------------------------------------
	// Height 16 - open major dispute, pay from not bond from reporter 1
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(16)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	//---------------------------------------------------------------------------
	// Height 17 - vote on major dispute
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(17)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	//---------------------------------------------------------------------------
	// Height 18 - resolve dispute, direct reveal again
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(18)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
}
