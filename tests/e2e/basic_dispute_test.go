package e2e_test

import (
	// "encoding/hex"
	// "time"

	// utils "github.com/tellor-io/layer/utils"
	"fmt"

	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	// disputetypes "github.com/tellor-io/layer/x/dispute/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	// oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

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
	msgServerOracle := oraclekeeper.NewMsgServerImpl(s.oraclekeeper)
	require.NotNil(msgServerOracle)
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.reporterkeeper)
	require.NotNil(msgServerReporter)
	msgServerDispute := disputekeeper.NewMsgServerImpl(s.disputekeeper)
	require.NotNil(msgServerDispute)
	msgServerStaking := stakingkeeper.NewMsgServerImpl(s.stakingKeeper)
	require.NotNil(msgServerStaking)

	//---------------------------------------------------------------------------
	// Height 0 - create validator and 2 reporters
	//---------------------------------------------------------------------------
	_, err := s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// create a validator
	valAccount := simtestutil.CreateIncrementalAccounts(1)
	// mint 5000*1e8 tokens for validator
	initCoins := sdk.NewCoin(s.denom, math.NewInt(5000*1e8))
	require.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
	require.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, valAccount[0], sdk.NewCoins(initCoins)))
	// get val address
	valAccountValAddrs := simtestutil.ConvertAddrsToValAddrs(valAccount)
	// create pub key for validator
	pubKey := simtestutil.CreateTestPubKeys(1)
	// tell keepers about the new validator
	s.accountKeeper.NewAccountWithAddress(s.ctx, valAccount[0])
	msgCreateValidaotr, err := stakingtypes.NewMsgCreateValidator(
		valAccountValAddrs[0].String(),
		pubKey[0],
		sdk.NewCoin(s.denom, math.NewInt(4000*1e8)),
		stakingtypes.Description{Moniker: "created validator"},
		stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(0, 0), math.LegacyNewDecWithPrec(3, 1), math.LegacyNewDecWithPrec(1, 1)),
		math.OneInt(),
	)
	require.NoError(err)
	_, err = msgServerStaking.CreateValidator(s.ctx, msgCreateValidaotr)
	require.NoError(err)
	validator, err := s.stakingKeeper.GetValidator(s.ctx, valAccountValAddrs[0])
	require.NoError(err)

	type Delegator struct {
		delegatorAddress sdk.AccAddress
		validator        stakingtypes.Validator
		tokenAmount      math.Int
	}
	pk := secp256k1.GenPrivKey()
	reporterAccount := sdk.AccAddress(pk.PubKey().Address())
	// mint 5000*1e6 tokens for reporter
	s.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
	s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, reporterAccount, sdk.NewCoins(initCoins)))
	// delegate 5k trb to validator so reporter can delegate to themselves
	reporterDelToVal := Delegator{delegatorAddress: reporterAccount, validator: validator, tokenAmount: math.NewInt(5000 * 1e6)}
	msgDelegate := stakingtypes.NewMsgDelegate(
		reporterDelToVal.delegatorAddress.String(),
		reporterDelToVal.validator.GetOperator(), sdk.NewCoin(s.denom, reporterDelToVal.tokenAmount),
	)
	_, err = msgServerStaking.Delegate(s.ctx, msgDelegate)
	require.NoError(err)

	// // self delegate in reporter module with 4k trb
	// var createReporterMsg reportertypes.MsgCreateReporter
	// reporterAddress := reporterDelToVal.delegatorAddress.String()
	// amount := math.NewInt(4000 * 1e6)
	// source := reportertypes.TokenOrigin{ValidatorAddress: valAccountValAddrs[0], Amount: math.NewInt(4000 * 1e6)}
	// // 0% commission for reporter staking to validator
	// commission := stakingtypes.NewCommissionWithTime(math.LegacyNewDecWithPrec(0, 0), math.LegacyNewDecWithPrec(3, 1),
	// 	math.LegacyNewDecWithPrec(1, 1), s.ctx.BlockTime())
	// // fill in createReporterMsg
	// createReporterMsg.Reporter = reporterAddress
	// createReporterMsg.Amount = amount
	// createReporterMsg.TokenOrigins = []*reportertypes.TokenOrigin{&source}
	// createReporterMsg.Commission = &commission
	// // send createreporter msg
	// _, err = msgServerReporter.CreateReporter(s.ctx, &createReporterMsg)
	// require.NoError(err)
	// // check that reporter was created in Reporters collections
	_, err = msgServerReporter.CreateReporter(s.ctx, &reportertypes.MsgCreateReporter{ReporterAddress: reporterAccount.String(), Commission: reporterkeeper.DefaultCommission()})
	require.NoError(err)
	reporter, err := s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	require.NoError(err)
	fmt.Println(reporter.TotalTokens)
	require.Equal(reporter.TotalTokens, math.NewInt(4000*1e6))
	require.Equal(reporter.Jailed, false)
	// // check on reporter in Delegators collections
	// rkDelegation, err := s.reporterkeeper.Delegators.Get(s.ctx, reporterAccount)
	// require.NoError(err)
	// require.Equal(rkDelegation.Reporter, reporterAccount.Bytes())
	// require.Equal(rkDelegation.Amount, math.NewInt(4000*1e6))
	// // check on reporter/validator delegation
	// skDelegation, err := s.stakingKeeper.Delegation(s.ctx, reporterAccount, valBz)
	// require.NoError(err)
	// require.Equal(skDelegation.GetDelegatorAddr(), reporterAccount.String())
	// require.Equal(skDelegation.GetValidatorAddr(), validator.GetOperator())

	// //---------------------------------------------------------------------------
	// // Height 1 - direct reveal for cycle list
	// //---------------------------------------------------------------------------
	// s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	// _, err = s.app.BeginBlocker(s.ctx)
	// require.NoError(err)
	// // get new cycle list query data
	// cycleListQuery, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	// require.NoError(err)
	// // create reveal message
	// value := encodeValue(100_000)
	// require.NoError(err)
	// reveal := oracletypes.MsgSubmitValue{
	// 	Creator:   sdk.AccAddress(reporter.Reporter).String(),
	// 	QueryData: cycleListQuery,
	// 	Value:     value,
	// }
	// // send reveal message
	// revealResponse, err := msgServerOracle.SubmitValue(s.ctx, &reveal)
	// require.NoError(err)
	// require.NotNil(revealResponse)
	// // advance time and block height to expire the query and aggregate report
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	// _, err = s.app.EndBlocker(s.ctx)
	// require.NoError(err)
	// // get queryId for GetAggregatedReportRequest
	// queryId := utils.QueryIDFromData(cycleListQuery)
	// s.NoError(err)
	// // create get aggregated report query
	// getAggReportRequest := oracletypes.QueryGetCurrentAggregatedReportRequest{
	// 	QueryId: hex.EncodeToString(queryId),
	// }
	// // aggregated report is stored correctly
	// queryServer := oraclekeeper.NewQuerier(s.oraclekeeper)
	// result, err := queryServer.GetAggregatedReport(s.ctx, &getAggReportRequest)
	// require.NoError(err)
	// require.Equal(int64(0), result.Report.AggregateReportIndex)
	// require.Equal(encodeValue(100_000), result.Report.AggregateValue)
	// require.Equal(sdk.AccAddress(reporter.Reporter).String(), result.Report.AggregateReporter)
	// require.Equal(queryId, result.Report.QueryId)
	// require.Equal(int64(4000), result.Report.ReporterPower)
	// require.Equal(int64(1), result.Report.Height)

	// //---------------------------------------------------------------------------
	// // Height 2 - create a dispute
	// //---------------------------------------------------------------------------
	// s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	// _, err = s.app.BeginBlocker(s.ctx)
	// require.NoError(err)

	// reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// require.NoError(err)
	// require.Equal(reporter.Jailed, false)
	// freeFloatingBalanceBefore := s.bankKeeper.GetBalance(s.ctx, reporterAccount, s.denom)

	// balBeforeDispute := reporter.TotalTokens
	// onePercent := balBeforeDispute.Mul(math.NewInt(1)).Quo(math.NewInt(100))
	// disputeFee := sdk.NewCoin(s.denom, onePercent) // warning should be 1% of bonded tokens

	// // todo: is there a getter for this ?
	// // get microreport for dispute
	// report := oracletypes.MicroReport{
	// 	Reporter:  sdk.AccAddress(reporter.Reporter).String(),
	// 	Power:     reporter.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
	// 	QueryId:   queryId,
	// 	Value:     value,
	// 	Timestamp: s.ctx.BlockTime(),
	// }

	// // create msg for propose dispute tx
	// msgProposeDispute := disputetypes.MsgProposeDispute{
	// 	Creator:         sdk.AccAddress(reporter.Reporter).String(),
	// 	Report:          &report,
	// 	DisputeCategory: disputetypes.Warning,
	// 	Fee:             disputeFee,
	// 	PayFromBond:     false,
	// }

	// // send propose dispute tx
	// _, err = msgServerDispute.ProposeDispute(s.ctx, &msgProposeDispute)
	// require.NoError(err)

	// burnAmount := disputeFee.Amount.MulRaw(1).QuoRaw(20)
	// disputes, err := s.disputekeeper.GetOpenDisputes(s.ctx)
	// require.NoError(err)
	// require.NotNil(disputes)
	// // dispute is created correctly
	// dispute, err := s.disputekeeper.Disputes.Get(s.ctx, 1)
	// require.NoError(err)
	// require.Equal(dispute.DisputeId, uint64(1))
	// require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	// require.Equal(dispute.DisputeCategory, disputetypes.Warning)
	// require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	// require.Equal(dispute.FeePayers, []disputetypes.PayerInfo{{PayerAddress: reporter.Reporter, Amount: disputeFee.Amount, FromBond: false, BlockNumber: 2}})

	// _, err = s.app.EndBlocker(s.ctx)
	// require.NoError(err)

	// //---------------------------------------------------------------------------
	// // Height 3 - unjail reporter
	// //---------------------------------------------------------------------------
	// s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	// err = s.disputekeeper.Tallyvote(s.ctx, dispute.DisputeId)
	// require.Error(err, "vote period not ended and quorum not reached")
	// _, err = s.app.BeginBlocker(s.ctx)
	// require.NoError(err)

	// // reporter is in jail
	// reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// require.NoError(err)
	// require.Equal(reporter.Jailed, true)
	// // reporter lost 1% of their free floating tokens
	// freeFloatingBalanceAfter := s.bankKeeper.GetBalance(s.ctx, reporterAccount, s.denom)
	// require.Equal(freeFloatingBalanceAfter, freeFloatingBalanceBefore.Sub(disputeFee))

	// // create msgUnJailReporter
	// msgUnjailReporter := reportertypes.MsgUnjailReporter{
	// 	ReporterAddress: sdk.AccAddress(reporter.Reporter).String(),
	// }
	// // send unjailreporter tx
	// _, err = msgServerReporter.UnjailReporter(s.ctx, &msgUnjailReporter)
	// require.NoError(err)

	// // reporter is now unjailed
	// reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// require.NoError(err)
	// require.Equal(reporter.Jailed, false)
	// freeFloatingBalanceAfter = s.bankKeeper.GetBalance(s.ctx, reporterAccount, s.denom)
	// require.Equal(freeFloatingBalanceAfter, freeFloatingBalanceBefore.Sub(disputeFee))
	// _, err = s.app.EndBlocker(s.ctx)
	// require.NoError(err)
	// // todo: more balance checks at each step

	// //---------------------------------------------------------------------------
	// // Height 4 - direct reveal for cycle list again
	// //---------------------------------------------------------------------------
	// s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	// err = s.disputekeeper.Tallyvote(s.ctx, 1)
	// require.Error(err, "vote period not ended and quorum not reached")
	// _, err = s.app.BeginBlocker(s.ctx)
	// require.NoError(err)

	// // get new cycle list query data
	// cycleListQuery, err = s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	// require.NoError(err)
	// // create reveal message
	// value = encodeValue(100_000)
	// require.NoError(err)
	// reveal = oracletypes.MsgSubmitValue{
	// 	Creator:   sdk.AccAddress(reporter.Reporter).String(),
	// 	QueryData: cycleListQuery,
	// 	Value:     value,
	// }
	// // send reveal message
	// revealResponse, err = msgServerOracle.SubmitValue(s.ctx, &reveal)
	// require.NoError(err)
	// require.NotNil(revealResponse)
	// // advance time and block height to expire the query and aggregate report
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	// _, err = s.app.EndBlocker(s.ctx)
	// require.NoError(err)
	// // get queryId for GetAggregatedReportRequest
	// queryId = utils.QueryIDFromData(cycleListQuery)
	// s.NoError(err)
	// // create get aggregated report query
	// getAggReportRequest = oracletypes.QueryGetCurrentAggregatedReportRequest{
	// 	QueryId: hex.EncodeToString(queryId),
	// }
	// // aggregated report is stored correctly
	// result, err = queryServer.GetAggregatedReport(s.ctx, &getAggReportRequest)
	// require.NoError(err)
	// require.Equal(int64(0), result.Report.AggregateReportIndex)
	// require.Equal(encodeValue(100_000), result.Report.AggregateValue)
	// require.Equal(sdk.AccAddress(reporter.Reporter).String(), result.Report.AggregateReporter)
	// require.Equal(queryId, result.Report.QueryId)
	// require.Equal(int64(4000), result.Report.ReporterPower)
	// require.Equal(int64(4), result.Report.Height)

	// _, err = s.app.EndBlocker(s.ctx)
	// require.NoError(err)

	// //---------------------------------------------------------------------------
	// // Height 5 - open minor dispute
	// //---------------------------------------------------------------------------
	// s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	// _, err = s.app.BeginBlocker(s.ctx)
	// require.NoError(err)

	// balBeforeDispute = reporter.TotalTokens
	// fivePercent := balBeforeDispute.Mul(math.NewInt(5)).Quo(math.NewInt(100))
	// disputeFee = sdk.NewCoin(s.denom, fivePercent)

	// report = oracletypes.MicroReport{
	// 	Reporter:  sdk.AccAddress(reporter.Reporter).String(),
	// 	Power:     reporter.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
	// 	QueryId:   queryId,
	// 	Value:     value,
	// 	Timestamp: s.ctx.BlockTime(),
	// }

	// // create msg for propose dispute tx
	// msgProposeDispute = disputetypes.MsgProposeDispute{
	// 	Creator:         sdk.AccAddress(reporter.Reporter).String(),
	// 	Report:          &report,
	// 	DisputeCategory: disputetypes.Minor,
	// 	Fee:             disputeFee,
	// 	PayFromBond:     false,
	// }

	// // send propose dispute tx
	// _, err = msgServerDispute.ProposeDispute(s.ctx, &msgProposeDispute)
	// require.NoError(err)
	// disputeStartTime := s.ctx.BlockTime()

	// _, err = s.app.EndBlocker(s.ctx)
	// require.NoError(err)

	// //---------------------------------------------------------------------------
	// // Height 6 - vote on minor dispute
	// //---------------------------------------------------------------------------
	// s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	// err = s.disputekeeper.Tallyvote(s.ctx, 1)
	// require.Error(err, "vote period not ended and quorum not reached")
	// err = s.disputekeeper.Tallyvote(s.ctx, 2)
	// require.Error(err, "vote period not ended and quorum not reached")
	// _, err = s.app.BeginBlocker(s.ctx)
	// require.NoError(err)

	// // reporter is in jail
	// reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// require.NoError(err)
	// require.Equal(reporter.Jailed, true)
	// // dispute is created correctly
	// burnAmount = disputeFee.Amount.MulRaw(1).QuoRaw(20)
	// dispute, err = s.disputekeeper.GetDisputeByReporter(s.ctx, report, disputetypes.Minor)
	// require.NoError(err)
	// require.Equal(dispute.DisputeCategory, disputetypes.Minor)
	// require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	// require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	// require.Equal(dispute.FeePayers, []disputetypes.PayerInfo{{PayerAddress: reporter.Reporter, Amount: disputeFee.Amount, FromBond: false, BlockNumber: 5}})

	// // create vote tx msg
	// msgVote := disputetypes.MsgVote{
	// 	Voter: sdk.AccAddress(reporter.Reporter).String(),
	// 	Id:    dispute.DisputeId,
	// 	Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	// }
	// // send vote tx
	// voteResponse, err := msgServerDispute.Vote(s.ctx, &msgVote)
	// require.NoError(err)
	// require.NotNil(voteResponse)

	// // vote is properly stored
	// vote, err := s.disputekeeper.Votes.Get(s.ctx, dispute.DisputeId)
	// require.NoError(err)
	// require.NotNil(vote)
	// require.Equal(vote.Executed, false)
	// require.Equal(vote.Id, dispute.DisputeId)
	// require.Equal(vote.VoteStart, disputeStartTime)
	// require.Equal(vote.VoteEnd, disputeStartTime.Add(disputekeeper.TWO_DAYS))

	// // advance 2 days to expire vote
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(disputekeeper.THREE_DAYS))
	// // call unjail function
	// msgUnjailReporter = reportertypes.MsgUnjailReporter{
	// 	ReporterAddress: sdk.AccAddress(reporter.Reporter).String(),
	// }
	// _, err = msgServerReporter.UnjailReporter(s.ctx, &msgUnjailReporter)
	// require.NoError(err)

	// // reporter no longer in jail
	// reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// require.NoError(err)
	// require.Equal(reporter.Jailed, false)

	// _, err = s.app.EndBlocker(s.ctx)
	// require.NoError(err)

	// //---------------------------------------------------------------------------
	// // Height 7 - minor dispute ends and another direct reveal for cycle list
	// //---------------------------------------------------------------------------
	// s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))

	// require.NoError(s.disputekeeper.Tallyvote(s.ctx, 1))
	// require.NoError(s.disputekeeper.Tallyvote(s.ctx, 2))
	// require.NoError(s.disputekeeper.ExecuteVote(s.ctx, 1))
	// require.NoError(s.disputekeeper.ExecuteVote(s.ctx, 2))

	// // vote is executed
	// vote, err = s.disputekeeper.Votes.Get(s.ctx, dispute.DisputeId)
	// require.NoError(err)
	// require.NotNil(vote)
	// require.Equal(vote.Executed, true)
	// require.Equal(vote.Id, dispute.DisputeId)
	// // reporter no longer in jail
	// reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// require.NoError(err)
	// require.Equal(reporter.Jailed, false)

	// // get open disputes
	// disputes, err = s.disputekeeper.GetOpenDisputes(s.ctx)
	// require.NoError(err)
	// require.NotNil(disputes)

	// // get new cycle list query data
	// cycleListQuery, err = s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	// require.NoError(err)
	// // create reveal message
	// value = encodeValue(100_000)
	// require.NoError(err)
	// reveal = oracletypes.MsgSubmitValue{
	// 	Creator:   sdk.AccAddress(reporter.Reporter).String(),
	// 	QueryData: cycleListQuery,
	// 	Value:     value,
	// }
	// // send reveal message
	// revealResponse, err = msgServerOracle.SubmitValue(s.ctx, &reveal)
	// require.NoError(err)
	// require.NotNil(revealResponse)
	// // advance time and block height to expire the query and aggregate report
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	// _, err = s.app.EndBlocker(s.ctx)
	// require.NoError(err)
	// // get queryId for GetAggregatedReportRequest
	// queryId = utils.QueryIDFromData(cycleListQuery)
	// s.NoError(err)
	// // create get aggregated report query
	// getAggReportRequest = oracletypes.QueryGetCurrentAggregatedReportRequest{
	// 	QueryId: hex.EncodeToString(queryId),
	// }
	// // check that aggregated report is stored correctly
	// result, err = queryServer.GetAggregatedReport(s.ctx, &getAggReportRequest)
	// require.NoError(err)
	// require.Equal(int64(0), result.Report.AggregateReportIndex)
	// require.Equal(encodeValue(100_000), result.Report.AggregateValue)
	// require.Equal(sdk.AccAddress(reporter.Reporter).String(), result.Report.AggregateReporter)
	// require.Equal(queryId, result.Report.QueryId)
	// require.Equal(int64(7), result.Report.Height)

	// _, err = s.app.EndBlocker(s.ctx)
	// require.NoError(err)

	// //---------------------------------------------------------------------------
	// // Height 8 - open major dispute for report
	// //---------------------------------------------------------------------------
	// s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	// _, err = s.app.BeginBlocker(s.ctx)
	// require.NoError(err)

	// reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// require.NoError(err)
	// require.Equal(reporter.Jailed, false)

	// oneHundredPercent := reporter.TotalTokens
	// disputeFee = sdk.NewCoin(s.denom, oneHundredPercent)

	// report = oracletypes.MicroReport{
	// 	Reporter:    sdk.AccAddress(reporter.Reporter).String(),
	// 	Power:       reporter.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
	// 	QueryId:     queryId,
	// 	Value:       value,
	// 	Timestamp:   s.ctx.BlockTime(),
	// 	BlockNumber: s.ctx.BlockHeight(),
	// }
	// // create msg for propose dispute tx

	// msgProposeDispute = disputetypes.MsgProposeDispute{
	// 	Creator:         sdk.AccAddress(reporter.Reporter).String(),
	// 	Report:          &report,
	// 	DisputeCategory: disputetypes.Major,
	// 	Fee:             disputeFee,
	// 	PayFromBond:     false,
	// }

	// // send propose dispute tx
	// _, err = msgServerDispute.ProposeDispute(s.ctx, &msgProposeDispute)
	// require.NoError(err)
	// disputeStartTime = s.ctx.BlockTime()
	// disputeStartHeight := s.ctx.BlockHeight()

	// _, err = s.app.EndBlocker(s.ctx)
	// require.NoError(err)

	// //---------------------------------------------------------------------------
	// // Height 9 - vote on major dispute
	// //---------------------------------------------------------------------------
	// s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))

	// err = s.disputekeeper.Tallyvote(s.ctx, 3)
	// require.Error(err, "vote period not ended and quorum not reached")
	// _, err = s.app.BeginBlocker(s.ctx)
	// require.NoError(err)

	// fee, err := s.disputekeeper.GetDisputeFee(s.ctx, sdk.AccAddress(reporter.Reporter).String(), disputetypes.Major)
	// require.NoError(err)
	// require.GreaterOrEqual(msgProposeDispute.Fee.Amount.Uint64(), fee.Uint64())

	// // dispute is created and open for voting
	// dispute, err = s.disputekeeper.GetDisputeByReporter(s.ctx, report, disputetypes.Major)
	// require.NoError(err)
	// burnAmount = disputeFee.Amount.MulRaw(1).QuoRaw(20)
	// require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	// require.Equal(dispute.DisputeStartTime, disputeStartTime)
	// require.Equal(dispute.DisputeEndTime, disputeStartTime.Add(disputekeeper.THREE_DAYS))
	// require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	// require.Equal(dispute.DisputeStartBlock, disputeStartHeight)
	// // todo: handle reporter removal
	// reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// require.NoError(err)

	// // create vote tx msg
	// msgVote = disputetypes.MsgVote{
	// 	Voter: sdk.AccAddress(reporter.Reporter).String(),
	// 	Id:    dispute.DisputeId,
	// 	Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	// }
	// // send vote tx
	// voteResponse, err = msgServerDispute.Vote(s.ctx, &msgVote)
	// require.NoError(err)
	// require.NotNil(voteResponse)

	// // vote is properly stored
	// vote, err = s.disputekeeper.Votes.Get(s.ctx, dispute.DisputeId)
	// require.NoError(err)
	// require.NotNil(vote)
	// require.Equal(vote.Executed, false)
	// require.Equal(vote.Id, dispute.DisputeId)
	// require.Equal(vote.VoteStart, disputeStartTime)
	// require.Equal(vote.VoteEnd, disputeStartTime.Add(disputekeeper.TWO_DAYS))

	// // advance 3 days to expire vote
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(disputekeeper.THREE_DAYS))

	// _, err = s.app.EndBlocker(s.ctx)
	// require.NoError(err)

	// // ---------------------------------------------------------------------------
	// // Height 10 - dispute is resolved, reporter no longer a reporter
	// // ---------------------------------------------------------------------------
	// // s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	// // s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	// // _, err = s.app.BeginBlocker(s.ctx)
	// // require.NoError(err)

	// err = s.disputekeeper.Tallyvote(s.ctx, 3)
	// require.NoError(err)
	// _, err = s.app.BeginBlocker(s.ctx)
	// require.NoError(err)
	// // reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// // require.NoError(err)
}
