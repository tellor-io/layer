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
	// advance time and block height to expire the query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryId := utils.QueryIDFromData(cycleListQuery)
	s.NoError(err)
	// create get aggregated report query
	getAggReportRequest := oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: hex.EncodeToString(queryId),
	}
	// aggregated report is stored correctly
	queryServer := oraclekeeper.NewQuerier(s.Setup.Oraclekeeper)
	result, err := queryServer.GetAggregatedReport(s.Setup.Ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(int64(0), result.Report.AggregateReportIndex)
	require.Equal(testutil.EncodeValue(100_000), result.Report.AggregateValue)
	require.Equal(reporterAccount.String(), result.Report.AggregateReporter)
	require.Equal(queryId, result.Report.QueryId)
	require.Equal(int64(4000), result.Report.ReporterPower)
	require.Equal(int64(1), result.Report.Height)

	//---------------------------------------------------------------------------
	// Height 2 - create a dispute
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
	onePercent := balBeforeDispute.Mul(math.NewInt(1)).Quo(math.NewInt(100))
	disputeFee := sdk.NewCoin(s.Setup.Denom, onePercent) // warning should be 1% of bonded tokens

	// todo: is there a getter for this ?
	// get microreport for dispute
	report := oracletypes.MicroReport{
		Reporter:    reporterAccount.String(),
		Power:       balBeforeDispute.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:     queryId,
		Value:       value,
		Timestamp:   s.Setup.Ctx.BlockTime(),
		BlockNumber: revealBlock,
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
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - unjail reporter
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	err = s.Setup.Disputekeeper.Tallyvote(s.Setup.Ctx, dispute.DisputeId)
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
	// Height 4 - direct reveal for cycle list again
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	err = s.Setup.Disputekeeper.Tallyvote(s.Setup.Ctx, 1)
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
	// send reveal message
	revealResponse, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal)
	require.NoError(err)
	require.NotNil(revealResponse)
	// advance time and block height to expire the query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryId = utils.QueryIDFromData(cycleListQuery)
	s.NoError(err)
	// create get aggregated report query
	getAggReportRequest = oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: hex.EncodeToString(queryId),
	}
	// aggregated report is stored correctly
	result, err = queryServer.GetAggregatedReport(s.Setup.Ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(int64(0), result.Report.AggregateReportIndex)
	require.Equal(testutil.EncodeValue(100_000), result.Report.AggregateValue)
	require.Equal(reporterAccount.String(), result.Report.AggregateReporter)
	require.Equal(queryId, result.Report.QueryId)
	require.Equal(int64(4000)-slashAmount.Quo(sdk.DefaultPowerReduction).Int64(), result.Report.ReporterPower)
	require.Equal(int64(4), result.Report.Height)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5 - open minor dispute
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	balBeforeDispute, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterAccount)
	fivePercent := balBeforeDispute.Mul(math.NewInt(5)).Quo(math.NewInt(100))
	disputeFee = sdk.NewCoin(s.Setup.Denom, fivePercent)

	report = oracletypes.MicroReport{
		Reporter:    reporterAccount.String(),
		Power:       balBeforeDispute.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:     queryId,
		Value:       value,
		Timestamp:   s.Setup.Ctx.BlockTime(),
		BlockNumber: revealBlock,
	}

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
	// Height 6 - vote on minor dispute
	// ---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	err = s.Setup.Disputekeeper.Tallyvote(s.Setup.Ctx, 1)
	require.Error(err, "vote period not ended and quorum not reached")
	err = s.Setup.Disputekeeper.Tallyvote(s.Setup.Ctx, 2)
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
	require.NoError(err)
	require.Equal(dispute.DisputeCategory, disputetypes.Minor)
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	feepayer, err = s.Setup.Disputekeeper.DisputeFeePayer.Get(s.Setup.Ctx, collections.Join(dispute.DisputeId, reporterAccount.Bytes()))
	require.NoError(err)
	require.Equal(feepayer.Amount, disputeFee.Amount)
	require.Equal(feepayer.FromBond, false)

	// create vote tx msg
	msgVote := disputetypes.MsgVote{
		Voter: reporterAccount.String(),
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}
	// send vote tx
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
	// Height 7 - minor dispute ends and another direct reveal for cycle list
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	require.NoError(s.Setup.Disputekeeper.Tallyvote(s.Setup.Ctx, 1))
	require.NoError(s.Setup.Disputekeeper.Tallyvote(s.Setup.Ctx, 2))
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
	// send reveal message
	revealResponse, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal)
	revealBlock = s.Setup.Ctx.BlockHeight()
	require.NoError(err)
	require.NotNil(revealResponse)
	// advance time and block height to expire the query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryId = utils.QueryIDFromData(cycleListQuery)
	s.NoError(err)
	// create get aggregated report query
	getAggReportRequest = oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: hex.EncodeToString(queryId),
	}
	// check that aggregated report is stored correctly
	result, err = queryServer.GetAggregatedReport(s.Setup.Ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(int64(0), result.Report.AggregateReportIndex)
	require.Equal(testutil.EncodeValue(100_000), result.Report.AggregateValue)
	require.Equal(reporterAccount.String(), result.Report.AggregateReporter)
	require.Equal(queryId, result.Report.QueryId)
	require.Equal(int64(7), result.Report.Height)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 8 - open major dispute for report
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, false)

	oneHundredPercent, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterAccount)
	disputeFee = sdk.NewCoin(s.Setup.Denom, oneHundredPercent)

	report = oracletypes.MicroReport{
		Reporter:    reporterAccount.String(),
		Power:       oneHundredPercent.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:     queryId,
		Value:       value,
		Timestamp:   s.Setup.Ctx.BlockTime(),
		BlockNumber: revealBlock,
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
	disputeStartHeight := s.Setup.Ctx.BlockHeight()

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 9 - vote on major dispute
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	err = s.Setup.Disputekeeper.Tallyvote(s.Setup.Ctx, 3)
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
	// // todo: handle reporter removal
	// reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	// fmt.Println(reporter)
	// require.ErrorContains(err, "not found")

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

	err = s.Setup.Disputekeeper.Tallyvote(s.Setup.Ctx, 3)
	require.NoError(err)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	// reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	// require.NoError(err)
}
