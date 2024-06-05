package e2e_test

import (
	"encoding/hex"

	"time"

	"github.com/tellor-io/layer/testutil"
	utils "github.com/tellor-io/layer/utils"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *E2ETestSuite) TestBasicReporting() {
	require := s.Require()
	msgServerStaking := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)

	//---------------------------------------------------------------------------
	// Height 0
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

	_, err = s.Setup.Stakingkeeper.EndBlocker(s.Setup.Ctx)
	s.NoError(err)

	type Delegator struct {
		delegatorAddress sdk.AccAddress
		validator        stakingtypes.Validator
		tokenAmount      math.Int
	}
	pk := ed25519.GenPrivKey()
	reporterAccount := sdk.AccAddress(pk.PubKey().Address())
	// mint 5000*1e6 tokens for reporter
	s.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, reporterAccount, sdk.NewCoins(initCoins)))
	// delegate to validator so reporter can delegate to themselves
	reporterDelToVal := Delegator{delegatorAddress: reporterAccount, validator: validator, tokenAmount: math.NewInt(4000 * 1e6)}
	msgDelegate := stakingtypes.NewMsgDelegate(reporterAccount.String(), validator.OperatorAddress, sdk.NewCoin(s.Setup.Denom, math.NewInt(4000*1e6)))
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)
	// set up reporter module msgServer
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	require.NotNil(msgServerReporter)
	// define createReporterMsg params
	reporterAddress := reporterDelToVal.delegatorAddress.String()
	// 0% commission for reporter staking to validator
	commission := stakingtypes.NewCommissionWithTime(math.LegacyNewDecWithPrec(0, 0), math.LegacyNewDecWithPrec(3, 1),
		math.LegacyNewDecWithPrec(1, 1), s.Setup.Ctx.BlockTime())
	createReporterMsg := reportertypes.MsgCreateReporter{ReporterAddress: reporterAddress, Commission: &commission}
	// send createreporter msg
	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &createReporterMsg)
	require.NoError(err)
	// check that reporter was created in Reporters collections
	reporter, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.TotalTokens, math.NewInt(4000*1e6))
	require.Equal(reporter.Jailed, false)
	// check on reporter in Delegators collections
	rkDelegation, err := s.Setup.Reporterkeeper.Delegators.Get(s.Setup.Ctx, reporterAccount)
	require.NoError(err)
	require.Equal(rkDelegation.Reporter, reporterAccount.Bytes())
	require.Equal(rkDelegation.Amount, math.NewInt(4000*1e6))
	// check on reporter/validator delegation
	valBz, err := s.Setup.Stakingkeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
	require.NoError(err)
	skDelegation, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, reporterAccount, valBz)
	require.NoError(err)
	require.Equal(skDelegation.GetDelegatorAddr(), reporterAccount.String())
	require.Equal(skDelegation.GetValidatorAddr(), validator.GetOperator())

	// setup oracle msgServer
	msgServerOracle := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	require.NotNil(msgServerOracle)

	// case 1: commit/reveal for cycle list
	//---------------------------------------------------------------------------
	// Height 1
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// check that no time based rewards have been minted yet
	tbrModuleAccount := s.Setup.Accountkeeper.GetModuleAddress(minttypes.TimeBasedRewards)
	tbrModuleAccountBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
	require.Equal(int64(0), tbrModuleAccountBalance.Amount.Int64())

	// begin report
	cycleListEth, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	// create hash for commit
	salt1, err := oracleutils.Salt(32)
	require.NoError(err)
	value1 := testutil.EncodeValue(4500)
	hash1 := oracleutils.CalculateCommitment(value1, salt1)
	// create commit1 msg
	commit1 := oracletypes.MsgCommitReport{
		Creator:   reporterAccount.String(),
		QueryData: cycleListEth,
		Hash:      hash1,
	}
	// send commit tx
	commitResponse1, err := msgServerOracle.CommitReport(s.Setup.Ctx, &commit1)
	require.NoError(err)
	require.NotNil(commitResponse1)
	commitHeight := s.Setup.Ctx.BlockHeight()
	require.Equal(int64(1), commitHeight)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(commitHeight + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// check that 1 second worth of tbr has been minted
	// expected tbr = (daily mint rate * time elapsed) / (# of ms in a day)
	expectedBlockProvision := int64(146940000 * (1 * time.Second) / (24 * 60 * 60 * 1000))
	expectedTbr := sdk.NewCoin(s.Setup.Denom, math.NewInt((expectedBlockProvision)).MulRaw(75).QuoRaw(100).Quo(sdk.DefaultPowerReduction))
	tbrModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
	require.Equal(expectedTbr, tbrModuleAccountBalance)
	// check that the cycle list has rotated
	cycleListBtc, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NotEqual(cycleListEth, cycleListBtc)
	require.NoError(err)

	// create reveal msg
	require.NoError(err)
	reveal1 := oracletypes.MsgSubmitValue{
		Creator:   reporterAccount.String(),
		QueryData: cycleListEth,
		Value:     value1,
		Salt:      salt1,
	}
	// send reveal tx
	revealResponse1, err := msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal1)
	require.NoError(err)
	require.NotNil(revealResponse1)
	// advance time and block height to expire the query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryIdEth := utils.QueryIDFromData(cycleListEth)
	s.NoError(err)
	// check that aggregated report is stored
	getAggReportRequest1 := oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: hex.EncodeToString(queryIdEth),
	}
	queryServer := oraclekeeper.NewQuerier(s.Setup.Oraclekeeper)
	result1, err := queryServer.GetAggregatedReport(s.Setup.Ctx, &getAggReportRequest1)
	require.NoError(err)
	require.Equal(result1.Report.Height, int64(2))
	require.Equal(result1.Report.AggregateReportIndex, int64(0))
	require.Equal(result1.Report.AggregateValue, testutil.EncodeValue(4500))
	require.Equal(result1.Report.AggregateReporter, reporterAccount.String())
	require.Equal(result1.Report.QueryId, queryIdEth)
	require.Equal(int64(4000), result1.Report.ReporterPower)
	// check that tbr is no longer in timeBasedRewards module acct
	tbrModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
	require.Equal(int64(0), tbrModuleAccountBalance.Amount.Int64())
	// check that tbr was sent to reporter module account
	reporterModuleAccount := s.Setup.Accountkeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
	reporterModuleAccountBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, reporterModuleAccount, s.Setup.Denom)
	require.Equal(expectedTbr, reporterModuleAccountBalance)
	// check reporters outstaning rewards
	outstandingRewards, err := s.Setup.Reporterkeeper.DelegatorTips.Get(s.Setup.Ctx, reporterAccount.Bytes())
	require.NoError(err)
	require.Equal(outstandingRewards, expectedTbr.Amount)
	// withdraw tbr
	rewards, err := msgServerReporter.WithdrawTip(s.Setup.Ctx, &reportertypes.MsgWithdrawTip{DelegatorAddress: reporterAddress, ValidatorAddress: validator.OperatorAddress})
	require.NoError(err)
	tbrEarned := outstandingRewards
	// check that there is only one reward to claim
	require.NotNil(rewards)
	// check that reporter module account balance is now empty
	reporterModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, reporterModuleAccount, s.Setup.Denom)
	require.Equal(int64(0), reporterModuleAccountBalance.Amount.Int64())
	// check that reporter now has more bonded tokens

	// case 2: direct reveal for cycle list
	//---------------------------------------------------------------------------
	// Height 3
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// check that 8 sec of tbr has been minted
	tbrModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
	expectedBlockProvision = int64(146940000 * (8 * time.Second) / (24 * 60 * 60 * 1000))
	expectedTbr = sdk.NewCoin(s.Setup.Denom, math.NewInt((expectedBlockProvision)).MulRaw(75).QuoRaw(100).Quo(sdk.DefaultPowerReduction))

	require.Equal(expectedTbr, tbrModuleAccountBalance)

	// get new cycle list query data
	cycleListTrb, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	require.NotEqual(cycleListEth, cycleListTrb)
	require.NotEqual(cycleListBtc, cycleListTrb)
	// create reveal message
	value2 := testutil.EncodeValue(100_000)
	require.NoError(err)
	reveal2 := oracletypes.MsgSubmitValue{
		Creator:   reporterAccount.String(),
		QueryData: cycleListTrb,
		Value:     value2,
	}
	// send reveal message
	revealResponse2, err := msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal2)
	require.NoError(err)
	require.NotNil(revealResponse2)
	// advance time and block height to expire the query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryIdTrb := utils.QueryIDFromData(cycleListTrb)
	s.NoError(err)
	// create get aggregated report query
	getAggReportRequest2 := oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: hex.EncodeToString(queryIdTrb),
	}
	// check that aggregated report is stored correctly
	result2, err := queryServer.GetAggregatedReport(s.Setup.Ctx, &getAggReportRequest2)
	require.NoError(err)
	require.Equal(int64(0), result2.Report.AggregateReportIndex)
	require.Equal(testutil.EncodeValue(100_000), result2.Report.AggregateValue)
	require.Equal(reporterAccount.String(), result2.Report.AggregateReporter)
	require.Equal(queryIdTrb, result2.Report.QueryId)
	require.Equal(int64(4000), result2.Report.ReporterPower)
	require.Equal(int64(3), result2.Report.Height)
	// check that tbr is no longer in timeBasedRewards module acct
	tbrModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
	require.Equal(int64(0), tbrModuleAccountBalance.Amount.Int64())
	// check that tbr was sent to reporter module account
	reporterModuleAccount = s.Setup.Accountkeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
	reporterModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, reporterModuleAccount, s.Setup.Denom)
	require.Equal(expectedTbr, reporterModuleAccountBalance)

	// check reporters outstaning rewards
	outstandingRewards, err = s.Setup.Reporterkeeper.DelegatorTips.Get(s.Setup.Ctx, reporterAccount.Bytes())
	require.NoError(err)
	require.Equal(outstandingRewards, expectedTbr.Amount)
	// withdraw tbr
	tbrEarned = tbrEarned.Add(outstandingRewards)
	rewards, err = msgServerReporter.WithdrawTip(s.Setup.Ctx, &reportertypes.MsgWithdrawTip{DelegatorAddress: reporterAddress, ValidatorAddress: validator.OperatorAddress})
	require.NoError(err)
	require.NotNil(rewards)
	// check that reporter module account balance is now empty
	reporterModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, reporterModuleAccount, s.Setup.Denom)
	require.Equal(int64(0), reporterModuleAccountBalance.Amount.Int64())

	// case 3: commit/reveal for tipped query
	//---------------------------------------------------------------------------
	// Height 4
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// get reporters shares
	deleBeforeReport, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, reporterAccount.Bytes(), valBz)
	require.NoError(err)
	require.Equal(deleBeforeReport.GetShares(), math.LegacyNewDecFromInt(math.NewInt(4000*1e6).Add(tbrEarned)))

	// create tip msg
	balanceBeforetip := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
	tipAmount := sdk.NewCoin(s.Setup.Denom, math.NewInt(100))
	msgTip := oracletypes.MsgTip{
		Tipper:    reporterAccount.String(),
		QueryData: cycleListEth,
		Amount:    tipAmount,
	}
	// send tip tx
	tipRes, err := msgServerOracle.Tip(s.Setup.Ctx, &msgTip)
	require.NoError(err)
	require.NotNil(tipRes)

	// check that tip is in oracle module account
	twoPercent := sdk.NewCoin(s.Setup.Denom, tipAmount.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	tipModuleAcct := s.Setup.Accountkeeper.GetModuleAddress(oracletypes.ModuleName)
	tipAcctBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipModuleAcct, s.Setup.Denom)
	require.Equal(tipAcctBalance, tipAmount.Sub(twoPercent))
	// create commit for tipped eth query
	salt1, err = oracleutils.Salt(32)
	require.NoError(err)
	value1 = testutil.EncodeValue(5000)
	hash1 = oracleutils.CalculateCommitment(value1, salt1)

	queryId := utils.QueryIDFromData(cycleListEth)
	quertip, err := s.Setup.Oraclekeeper.GetQueryTip(s.Setup.Ctx, queryId)
	require.NoError(err)
	require.Equal(quertip, tipAmount.Amount.Sub(twoPercent.Amount))

	commit1 = oracletypes.MsgCommitReport{
		Creator:   reporterAccount.String(),
		QueryData: cycleListEth,
		Hash:      hash1,
	}
	// send commit tx
	commitResponse1, err = msgServerOracle.CommitReport(s.Setup.Ctx, &commit1)
	require.NoError(err)
	require.NotNil(commitResponse1)
	commitHeight = s.Setup.Ctx.BlockHeight()
	require.Equal(int64(4), commitHeight)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(commitHeight + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// create reveal msg
	value1 = testutil.EncodeValue(5000)
	reveal1 = oracletypes.MsgSubmitValue{
		Creator:   reporterAccount.String(),
		QueryData: cycleListEth,
		Value:     value1,
		Salt:      salt1,
	}
	// send reveal tx
	revealResponse1, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal1)
	require.NoError(err)
	require.NotNil(revealResponse1)

	// advance time and block height to expire the query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	// create get aggreagted report query
	getAggReportRequest1 = oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: hex.EncodeToString(queryIdEth),
	}
	// check that the aggregated report is stored correctly
	result1, err = queryServer.GetAggregatedReport(s.Setup.Ctx, &getAggReportRequest1)
	require.NoError(err)
	require.Equal(result1.Report.AggregateReportIndex, int64(0))
	require.Equal(result1.Report.AggregateValue, testutil.EncodeValue(5000))
	require.Equal(result1.Report.AggregateReporter, reporterAccount.String())
	require.Equal(queryIdEth, result1.Report.QueryId)
	require.Equal(int64(4000), result1.Report.ReporterPower)
	require.Equal(int64(5), result1.Report.Height)
	// check that the tip is in tip escrow
	tipEscrowAcct := s.Setup.Accountkeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
	tipEscrowBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipEscrowAcct, s.Setup.Denom) // 98 loya
	require.Equal(tipAmount.Amount.Sub(twoPercent.Amount), tipEscrowBalance.Amount.Sub(balanceBeforetip.Amount))
	// withdraw tip
	msgWithdrawTip := reportertypes.MsgWithdrawTip{
		DelegatorAddress: reporterAddress,
		ValidatorAddress: validator.OperatorAddress,
	}
	_, err = msgServerReporter.WithdrawTip(s.Setup.Ctx, &msgWithdrawTip)
	require.NoError(err)

	// check that tip is no longer in escrow pool
	tipEscrowBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipEscrowAcct, s.Setup.Denom)
	require.Equal(int64(0), tipEscrowBalance.Amount.Int64())
	// check that reporter now has more bonded tokens
	deleAfter, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, reporterAccount.Bytes(), valBz)
	require.NoError(err)
	require.Equal(deleBeforeReport.GetShares().Add(math.LegacyNewDec(98+8928)), deleAfter.GetShares())

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	// case 4: submit without committing for tipped query
	//---------------------------------------------------------------------------
	// Height 6
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// check reporter starting shares
	deleBeforeReport2, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, reporterAccount.Bytes(), valBz)
	require.NoError(err)
	expectedShares := math.LegacyNewDecFromInt(deleBeforeReport.GetShares().TruncateInt().Add(math.NewInt(98 + 8928))) // 8928 is the tbr that was earned
	require.Equal(deleBeforeReport2.GetShares(), expectedShares)

	// create tip msg
	msgTip = oracletypes.MsgTip{
		Tipper:    reporterAccount.String(),
		QueryData: cycleListTrb,
		Amount:    tipAmount,
	}
	// send tip tx
	tipRes, err = msgServerOracle.Tip(s.Setup.Ctx, &msgTip)
	require.NoError(err)
	require.NotNil(tipRes)
	// check that tip is in oracle module account
	tipModuleAcct = s.Setup.Accountkeeper.GetModuleAddress(oracletypes.ModuleName)
	tipAcctBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipModuleAcct, s.Setup.Denom)
	require.Equal(tipAcctBalance, tipAmount.Sub(twoPercent))
	// create submit msg
	revealMsgTrb := oracletypes.MsgSubmitValue{
		Creator:   reporterAccount.String(),
		QueryData: cycleListTrb,
		Value:     testutil.EncodeValue(1_000_000),
	}
	// send submit msg
	revealTrb, err := msgServerOracle.SubmitValue(s.Setup.Ctx, &revealMsgTrb)
	require.NoError(err)
	require.NotNil(revealTrb)
	// advance time and block height to expire the query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	// create get aggregated report query
	getAggReportRequestTrb := oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: hex.EncodeToString(queryIdTrb),
	}
	// query aggregated report
	resultTrb, err := queryServer.GetAggregatedReport(s.Setup.Ctx, &getAggReportRequestTrb)
	require.NoError(err)
	require.Equal(resultTrb.Report.AggregateReportIndex, int64(0))
	require.Equal(resultTrb.Report.AggregateValue, testutil.EncodeValue(1_000_000))
	require.Equal(resultTrb.Report.AggregateReporter, reporterAccount.String())
	require.Equal(queryIdTrb, resultTrb.Report.QueryId)
	require.Equal(int64(4000), resultTrb.Report.ReporterPower)
	require.Equal(int64(6), resultTrb.Report.Height)
	// check that the tip is in tip escrow
	tipEscrowBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipEscrowAcct, s.Setup.Denom) // 98 loya
	require.Equal(tipAmount.Amount.Sub(twoPercent.Amount), tipEscrowBalance.Amount)
	// withdraw tip
	_, err = msgServerReporter.WithdrawTip(s.Setup.Ctx, &msgWithdrawTip)
	require.NoError(err)
	// check that tip is no longer in escrow pool
	tipEscrowBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipEscrowAcct, s.Setup.Denom)
	require.Equal(int64(0), tipEscrowBalance.Amount.Int64())
	// check that reporter now has more bonded tokens
	deleAfter, err = s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, reporterAccount.Bytes(), valBz)
	require.NoError(err)
	require.Equal(deleBeforeReport2.GetShares().Add(math.LegacyNewDec(98)), deleAfter.GetShares())

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
}
