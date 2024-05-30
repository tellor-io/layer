package e2e_test

import (
	"encoding/hex"
	"time"

	utils "github.com/tellor-io/layer/utils"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *E2ETestSuite) TestBasicReporting() {
	require := s.Require()

	//---------------------------------------------------------------------------
	// Height 0
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
	validator, err := stakingtypes.NewValidator(valAccountValAddrs[0].String(), pubKey[0], stakingtypes.Description{Moniker: "created validator"})
	require.NoError(err)
	s.NoError(s.stakingKeeper.SetValidator(s.ctx, validator))
	s.NoError(s.stakingKeeper.SetValidatorByConsAddr(s.ctx, validator))
	s.NoError(s.stakingKeeper.SetNewValidatorByPowerIndex(s.ctx, validator))
	// self delegate from validator account to itself
	_, err = s.stakingKeeper.Delegate(s.ctx, valAccount[0], math.NewInt(int64(4000)*1e8), stakingtypes.Unbonded, validator, true)
	require.NoError(err)
	// call hooks for distribution init
	valBz, err := s.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
	if err != nil {
		panic(err)
	}
	err = s.distrKeeper.Hooks().AfterValidatorCreated(s.ctx, valBz)
	require.NoError(err)
	err = s.distrKeeper.Hooks().BeforeDelegationCreated(s.ctx, valAccount[0], valBz)
	require.NoError(err)
	err = s.distrKeeper.Hooks().AfterDelegationModified(s.ctx, valAccount[0], valBz)
	require.NoError(err)
	_, err = s.stakingKeeper.EndBlocker(s.ctx)
	s.NoError(err)

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
	// delegate to validator so reporter can delegate to themselves
	reporterDelToVal := Delegator{delegatorAddress: reporterAccount, validator: validator, tokenAmount: math.NewInt(5000 * 1e6)}
	_, err = s.stakingKeeper.Delegate(s.ctx, reporterDelToVal.delegatorAddress, reporterDelToVal.tokenAmount, stakingtypes.Unbonded, reporterDelToVal.validator, true)
	require.NoError(err)
	// call dist module hooks
	err = s.distrKeeper.Hooks().AfterValidatorCreated(s.ctx, valBz)
	require.NoError(err)
	err = s.distrKeeper.Hooks().BeforeDelegationCreated(s.ctx, reporterAccount, valAccountValAddrs[0])
	require.NoError(err)
	err = s.distrKeeper.Hooks().AfterDelegationModified(s.ctx, reporterAccount, valAccountValAddrs[0])
	require.NoError(err)
	// set up reporter module msgServer
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.reporterkeeper)
	require.NotNil(msgServerReporter)
	// define createReporterMsg params
	var createReporterMsg reportertypes.MsgCreateReporter
	reporterAddress := reporterDelToVal.delegatorAddress.String()
	amount := math.NewInt(4000 * 1e6)
	source := reportertypes.TokenOrigin{ValidatorAddress: valBz, Amount: math.NewInt(4000 * 1e6)}
	// 0% commission for reporter staking to validator
	commission := stakingtypes.NewCommissionWithTime(math.LegacyNewDecWithPrec(0, 0), math.LegacyNewDecWithPrec(3, 1),
		math.LegacyNewDecWithPrec(1, 1), s.ctx.BlockTime())
	// fill in createReporterMsg
	createReporterMsg.Reporter = reporterAddress
	createReporterMsg.Amount = amount
	createReporterMsg.TokenOrigins = []*reportertypes.TokenOrigin{&source}
	createReporterMsg.Commission = &commission
	// send createreporter msg
	_, err = msgServerReporter.CreateReporter(s.ctx, &createReporterMsg)
	require.NoError(err)
	// check that reporter was created in Reporters collections
	reporter, err := s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Reporter, reporterAccount.Bytes())
	require.Equal(reporter.TotalTokens, math.NewInt(4000*1e6))
	require.Equal(reporter.Jailed, false)
	// check on reporter in Delegators collections
	rkDelegation, err := s.reporterkeeper.Delegators.Get(s.ctx, reporterAccount)
	require.NoError(err)
	require.Equal(rkDelegation.Reporter, reporterAccount.Bytes())
	require.Equal(rkDelegation.Amount, math.NewInt(4000*1e6))
	// check on reporter/validator delegation
	skDelegation, err := s.stakingKeeper.Delegation(s.ctx, reporterAccount, valBz)
	require.NoError(err)
	require.Equal(skDelegation.GetDelegatorAddr(), reporterAccount.String())
	require.Equal(skDelegation.GetValidatorAddr(), validator.GetOperator())

	// setup oracle msgServer
	msgServerOracle := oraclekeeper.NewMsgServerImpl(s.oraclekeeper)
	require.NotNil(msgServerOracle)

	// case 1: commit/reveal for cycle list
	//---------------------------------------------------------------------------
	// Height 1
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// check that no time based rewards have been minted yet
	tbrModuleAccount := s.accountKeeper.GetModuleAddress(minttypes.TimeBasedRewards)
	tbrModuleAccountBalance := s.bankKeeper.GetBalance(s.ctx, tbrModuleAccount, sdk.DefaultBondDenom)
	require.Equal(int64(0), tbrModuleAccountBalance.Amount.Int64())

	// begin report
	cycleListEth, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	// create hash for commit
	salt1, err := oracleutils.Salt(32)
	require.NoError(err)
	value1 := encodeValue(4500)
	hash1 := oracleutils.CalculateCommitment(value1, salt1)
	// create commit1 msg
	commit1 := oracletypes.MsgCommitReport{
		Creator:   sdk.AccAddress(reporter.Reporter).String(),
		QueryData: cycleListEth,
		Hash:      hash1,
	}
	// send commit tx
	commitResponse1, err := msgServerOracle.CommitReport(s.ctx, &commit1)
	require.NoError(err)
	require.NotNil(commitResponse1)
	commitHeight := s.ctx.BlockHeight()
	require.Equal(int64(1), commitHeight)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(commitHeight + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// check that 1 second worth of tbr has been minted
	// expected tbr = (daily mint rate * time elapsed) / (# of ms in a day)
	expectedBlockProvision := int64(146940000 * (1 * time.Second) / (24 * 60 * 60 * 1000))
	expectedTbr := sdk.NewCoin(s.denom, math.NewInt((expectedBlockProvision)).MulRaw(75).QuoRaw(100).Quo(sdk.DefaultPowerReduction))
	tbrModuleAccountBalance = s.bankKeeper.GetBalance(s.ctx, tbrModuleAccount, sdk.DefaultBondDenom)
	require.Equal(expectedTbr, tbrModuleAccountBalance)
	// check that the cycle list has rotated
	cycleListBtc, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NotEqual(cycleListEth, cycleListBtc)
	require.NoError(err)

	// create reveal msg
	require.NoError(err)
	reveal1 := oracletypes.MsgSubmitValue{
		Creator:   sdk.AccAddress(reporter.Reporter).String(),
		QueryData: cycleListEth,
		Value:     value1,
		Salt:      salt1,
	}
	// send reveal tx
	revealResponse1, err := msgServerOracle.SubmitValue(s.ctx, &reveal1)
	require.NoError(err)
	require.NotNil(revealResponse1)
	// advance time and block height to expire the query and aggregate report
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryIdEth := utils.QueryIDFromData(cycleListEth)
	s.NoError(err)
	// check that aggregated report is stored
	getAggReportRequest1 := oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: hex.EncodeToString(queryIdEth),
	}
	queryServer := oraclekeeper.NewQuerier(s.oraclekeeper)
	result1, err := queryServer.GetAggregatedReport(s.ctx, &getAggReportRequest1)
	require.NoError(err)
	require.Equal(result1.Report.Height, int64(2))
	require.Equal(result1.Report.AggregateReportIndex, int64(0))
	require.Equal(result1.Report.AggregateValue, encodeValue(4500))
	require.Equal(result1.Report.AggregateReporter, sdk.AccAddress(reporter.Reporter).String())
	require.Equal(result1.Report.QueryId, queryIdEth)
	require.Equal(int64(4000), result1.Report.ReporterPower)
	// check that tbr is no longer in timeBasedRewards module acct
	tbrModuleAccountBalance = s.bankKeeper.GetBalance(s.ctx, tbrModuleAccount, sdk.DefaultBondDenom)
	require.Equal(int64(0), tbrModuleAccountBalance.Amount.Int64())
	// check that tbr was sent to reporter module account
	reporterModuleAccount := s.accountKeeper.GetModuleAddress(reportertypes.ModuleName)
	reporterModuleAccountBalance := s.bankKeeper.GetBalance(s.ctx, reporterModuleAccount, sdk.DefaultBondDenom)
	require.Equal(expectedTbr, reporterModuleAccountBalance)
	// check reporters outstaning rewards
	outstandingRewards, err := s.reporterkeeper.GetReporterOutstandingRewardsCoins(s.ctx, sdk.ValAddress(reporterAccount))
	require.NoError(err)
	require.Equal(outstandingRewards.AmountOf(s.denom).TruncateInt(), expectedTbr.Amount)
	// withdraw tbr
	rewards, err := s.reporterkeeper.WithdrawDelegationRewards(s.ctx, sdk.ValAddress(reporterAccount), reporterAccount)
	require.NoError(err)
	// check that there is only one reward to claim
	require.Equal(len(rewards), 1)
	// check that the reward is the correct amount and denom
	require.Equal(rewards[0].Denom, s.denom)
	require.Equal(rewards.AmountOf(s.denom), expectedTbr.Amount)
	// check that reporter module account balance is now empty
	reporterModuleAccountBalance = s.bankKeeper.GetBalance(s.ctx, reporterModuleAccount, sdk.DefaultBondDenom)
	require.Equal(int64(0), reporterModuleAccountBalance.Amount.Int64())
	// check that reporter now has more bonded tokens

	// case 2: direct reveal for cycle list
	//---------------------------------------------------------------------------
	// Height 3
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// check that 8 sec of tbr has been minted
	tbrModuleAccountBalance = s.bankKeeper.GetBalance(s.ctx, tbrModuleAccount, sdk.DefaultBondDenom)
	expectedBlockProvision = int64(146940000 * (8 * time.Second) / (24 * 60 * 60 * 1000))
	expectedTbr = sdk.NewCoin(s.denom, math.NewInt((expectedBlockProvision)).MulRaw(75).QuoRaw(100).Quo(sdk.DefaultPowerReduction))
	require.Equal(expectedTbr, tbrModuleAccountBalance)

	// get new cycle list query data
	cycleListTrb, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	require.NotEqual(cycleListEth, cycleListTrb)
	require.NotEqual(cycleListBtc, cycleListTrb)
	// create reveal message
	value2 := encodeValue(100_000)
	require.NoError(err)
	reveal2 := oracletypes.MsgSubmitValue{
		Creator:   sdk.AccAddress(reporter.Reporter).String(),
		QueryData: cycleListTrb,
		Value:     value2,
	}
	// send reveal message
	revealResponse2, err := msgServerOracle.SubmitValue(s.ctx, &reveal2)
	require.NoError(err)
	require.NotNil(revealResponse2)
	// advance time and block height to expire the query and aggregate report
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryIdTrb := utils.QueryIDFromData(cycleListTrb)
	s.NoError(err)
	// create get aggregated report query
	getAggReportRequest2 := oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: hex.EncodeToString(queryIdTrb),
	}
	// check that aggregated report is stored correctly
	result2, err := queryServer.GetAggregatedReport(s.ctx, &getAggReportRequest2)
	require.NoError(err)
	require.Equal(int64(0), result2.Report.AggregateReportIndex)
	require.Equal(encodeValue(100_000), result2.Report.AggregateValue)
	require.Equal(sdk.AccAddress(reporter.Reporter).String(), result2.Report.AggregateReporter)
	require.Equal(queryIdTrb, result2.Report.QueryId)
	require.Equal(int64(4000), result2.Report.ReporterPower)
	require.Equal(int64(3), result2.Report.Height)
	// check that tbr is no longer in timeBasedRewards module acct
	tbrModuleAccountBalance = s.bankKeeper.GetBalance(s.ctx, tbrModuleAccount, sdk.DefaultBondDenom)
	require.Equal(int64(0), tbrModuleAccountBalance.Amount.Int64())
	// check that tbr was sent to reporter module account
	reporterModuleAccount = s.accountKeeper.GetModuleAddress(reportertypes.ModuleName)
	reporterModuleAccountBalance = s.bankKeeper.GetBalance(s.ctx, reporterModuleAccount, sdk.DefaultBondDenom)
	require.Equal(expectedTbr, reporterModuleAccountBalance)
	// check reporters outstaning rewards
	outstandingRewards, err = s.reporterkeeper.GetReporterOutstandingRewardsCoins(s.ctx, sdk.ValAddress(reporterAccount))
	require.NoError(err)
	require.Equal(outstandingRewards.AmountOf(s.denom).TruncateInt(), expectedTbr.Amount)
	// withdraw tbr
	rewards, err = s.reporterkeeper.WithdrawDelegationRewards(s.ctx, sdk.ValAddress(reporterAccount), reporterAccount)
	require.NoError(err)
	// check that there is only one reward to claim
	require.Equal(len(rewards), 1)
	// check that the reward is the correct amount and denom
	require.Equal(rewards[0].Denom, s.denom)
	require.Equal(rewards.AmountOf(s.denom), expectedTbr.Amount)
	// check that reporter module account balance is now empty
	reporterModuleAccountBalance = s.bankKeeper.GetBalance(s.ctx, reporterModuleAccount, sdk.DefaultBondDenom)
	require.Equal(int64(0), reporterModuleAccountBalance.Amount.Int64())
	// check that reporter now has more bonded tokens

	// case 3: commit/reveal for tipped query
	//---------------------------------------------------------------------------
	// Height 4
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// get reporters shares
	deleBeforeReport, err := s.stakingKeeper.Delegation(s.ctx, reporterAccount.Bytes(), valBz)
	require.NoError(err)
	require.Equal(deleBeforeReport.GetShares(), math.LegacyNewDecFromInt(math.NewInt(5000*1e6)))

	// create tip msg
	tipAmount := sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100))
	msgTip := oracletypes.MsgTip{
		Tipper:    sdk.AccAddress(reporter.Reporter).String(),
		QueryData: cycleListEth,
		Amount:    tipAmount,
	}
	// send tip tx
	tipRes, err := msgServerOracle.Tip(s.ctx, &msgTip)
	require.NoError(err)
	require.NotNil(tipRes)
	// check that tip is in oracle module account
	twoPercent := sdk.NewCoin(s.denom, tipAmount.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	tipModuleAcct := s.accountKeeper.GetModuleAddress(oracletypes.ModuleName)
	tipAcctBalance := s.bankKeeper.GetBalance(s.ctx, tipModuleAcct, sdk.DefaultBondDenom)
	require.Equal(tipAcctBalance, tipAmount.Sub(twoPercent))
	// create commit for tipped eth query
	salt1, err = oracleutils.Salt(32)
	require.NoError(err)
	value1 = encodeValue(5000)
	hash1 = oracleutils.CalculateCommitment(value1, salt1)
	commit1 = oracletypes.MsgCommitReport{
		Creator:   sdk.AccAddress(reporter.Reporter).String(),
		QueryData: cycleListEth,
		Hash:      hash1,
	}
	// send commit tx
	commitResponse1, err = msgServerOracle.CommitReport(s.ctx, &commit1)
	require.NoError(err)
	require.NotNil(commitResponse1)
	commitHeight = s.ctx.BlockHeight()
	require.Equal(int64(4), commitHeight)
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(commitHeight + 1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// create reveal msg
	value1 = encodeValue(5000)
	reveal1 = oracletypes.MsgSubmitValue{
		Creator:   sdk.AccAddress(reporter.Reporter).String(),
		QueryData: cycleListEth,
		Value:     value1,
		Salt:      salt1,
	}
	// send reveal tx
	revealResponse1, err = msgServerOracle.SubmitValue(s.ctx, &reveal1)
	require.NoError(err)
	require.NotNil(revealResponse1)
	// advance time and block height to expire the query and aggregate report
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)
	// create get aggreagted report query
	getAggReportRequest1 = oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: hex.EncodeToString(queryIdEth),
	}
	// check that the aggregated report is stored correctly
	result1, err = queryServer.GetAggregatedReport(s.ctx, &getAggReportRequest1)
	require.NoError(err)
	require.Equal(result1.Report.AggregateReportIndex, int64(0))
	require.Equal(result1.Report.AggregateValue, encodeValue(5000))
	require.Equal(result1.Report.AggregateReporter, sdk.AccAddress(reporter.Reporter).String())
	require.Equal(queryIdEth, result1.Report.QueryId)
	require.Equal(int64(4000), result1.Report.ReporterPower)
	require.Equal(int64(5), result1.Report.Height)
	// check that the tip is in tip escrow
	tipEscrowAcct := s.accountKeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
	tipEscrowBalance := s.bankKeeper.GetBalance(s.ctx, tipEscrowAcct, sdk.DefaultBondDenom) // 98 loya
	require.Equal(tipAmount.Amount.Sub(twoPercent.Amount), tipEscrowBalance.Amount)
	// withdraw tip
	msgWithdrawTip := reportertypes.MsgWithdrawTip{
		DelegatorAddress: reporterAddress,
		ValidatorAddress: validator.OperatorAddress,
	}
	_, err = msgServerReporter.WithdrawTip(s.ctx, &msgWithdrawTip)
	require.NoError(err)
	// check that tip is no longer in escrow pool
	tipEscrowBalance = s.bankKeeper.GetBalance(s.ctx, tipEscrowAcct, sdk.DefaultBondDenom)
	require.Equal(int64(0), tipEscrowBalance.Amount.Int64())
	// check that reporter now has more bonded tokens
	deleAfter, err := s.stakingKeeper.Delegation(s.ctx, reporterAccount.Bytes(), valBz)
	require.NoError(err)
	require.Equal(deleBeforeReport.GetShares().Add(math.LegacyNewDec(98)), deleAfter.GetShares())

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	// case 4: submit without committing for tipped query
	//---------------------------------------------------------------------------
	// Height 6
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// check reporter starting shares
	deleBeforeReport2, err := s.stakingKeeper.Delegation(s.ctx, reporterAccount.Bytes(), valBz)
	require.NoError(err)
	expectedShares := math.LegacyNewDecFromInt(math.NewInt(5000 * 1e6).Add(math.NewInt(98)))
	require.Equal(deleBeforeReport2.GetShares(), expectedShares)

	// create tip msg
	msgTip = oracletypes.MsgTip{
		Tipper:    sdk.AccAddress(reporter.Reporter).String(),
		QueryData: cycleListTrb,
		Amount:    tipAmount,
	}
	// send tip tx
	tipRes, err = msgServerOracle.Tip(s.ctx, &msgTip)
	require.NoError(err)
	require.NotNil(tipRes)
	// check that tip is in oracle module account
	tipModuleAcct = s.accountKeeper.GetModuleAddress(oracletypes.ModuleName)
	tipAcctBalance = s.bankKeeper.GetBalance(s.ctx, tipModuleAcct, sdk.DefaultBondDenom)
	require.Equal(tipAcctBalance, tipAmount.Sub(twoPercent))
	// create submit msg
	revealMsgTrb := oracletypes.MsgSubmitValue{
		Creator:   sdk.AccAddress(reporter.Reporter).String(),
		QueryData: cycleListTrb,
		Value:     encodeValue(1_000_000),
	}
	// send submit msg
	revealTrb, err := msgServerOracle.SubmitValue(s.ctx, &revealMsgTrb)
	require.NoError(err)
	require.NotNil(revealTrb)
	// advance time and block height to expire the query and aggregate report
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)
	// create get aggregated report query
	getAggReportRequestTrb := oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: hex.EncodeToString(queryIdTrb),
	}
	// query aggregated report
	resultTrb, err := queryServer.GetAggregatedReport(s.ctx, &getAggReportRequestTrb)
	require.NoError(err)
	require.Equal(resultTrb.Report.AggregateReportIndex, int64(0))
	require.Equal(resultTrb.Report.AggregateValue, encodeValue(1_000_000))
	require.Equal(resultTrb.Report.AggregateReporter, sdk.AccAddress(reporter.Reporter).String())
	require.Equal(queryIdTrb, resultTrb.Report.QueryId)
	require.Equal(int64(4000), resultTrb.Report.ReporterPower)
	require.Equal(int64(6), resultTrb.Report.Height)
	// check that the tip is in tip escrow
	tipEscrowBalance = s.bankKeeper.GetBalance(s.ctx, tipEscrowAcct, sdk.DefaultBondDenom) // 98 loya
	require.Equal(tipAmount.Amount.Sub(twoPercent.Amount), tipEscrowBalance.Amount)
	// withdraw tip
	_, err = msgServerReporter.WithdrawTip(s.ctx, &msgWithdrawTip)
	require.NoError(err)
	// check that tip is no longer in escrow pool
	tipEscrowBalance = s.bankKeeper.GetBalance(s.ctx, tipEscrowAcct, sdk.DefaultBondDenom)
	require.Equal(int64(0), tipEscrowBalance.Amount.Int64())
	// check that reporter now has more bonded tokens
	deleAfter, err = s.stakingKeeper.Delegation(s.ctx, reporterAccount.Bytes(), valBz)
	require.NoError(err)
	require.Equal(deleBeforeReport2.GetShares().Add(math.LegacyNewDec(98)), deleAfter.GetShares())

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)
}
