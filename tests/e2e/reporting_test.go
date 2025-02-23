package e2e_test

import (
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/testutil"
	utils "github.com/tellor-io/layer/utils"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	secp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// func (s *E2ETestSuite) TestBasicReporting() {
// 	require := s.Require()
// 	minter, err := s.Setup.Mintkeeper.Minter.Get(s.Setup.Ctx)
// 	require.NoError(err)
// 	minter.Initialized = true
// 	require.NoError(s.Setup.Mintkeeper.Minter.Set(s.Setup.Ctx, minter))
// 	msgServerStaking := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)

// 	//---------------------------------------------------------------------------
// 	// Height 0
// 	//---------------------------------------------------------------------------
// 	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
// 	require.NoError(err)

// 	// create a validator
// 	valAccount := simtestutil.CreateIncrementalAccounts(1)
// 	// mint 5000*1e8 tokens for validator
// 	initCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(5000*1e8))
// 	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
// 	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, valAccount[0], sdk.NewCoins(initCoins)))
// 	// get val address
// 	valAccountValAddrs := simtestutil.ConvertAddrsToValAddrs(valAccount)
// 	// create pub key for validator
// 	pubKey := simtestutil.CreateTestPubKeys(1)
// 	// tell keepers about the new validator
// 	s.Setup.Accountkeeper.NewAccountWithAddress(s.Setup.Ctx, valAccount[0])
// 	msgCreateValidaotr, err := stakingtypes.NewMsgCreateValidator(
// 		valAccountValAddrs[0].String(),
// 		pubKey[0],
// 		sdk.NewCoin(s.Setup.Denom, math.NewInt(4000*1e8)),
// 		stakingtypes.Description{Moniker: "created validator"},
// 		stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(0, 0), math.LegacyNewDecWithPrec(3, 1), math.LegacyNewDecWithPrec(1, 1)),
// 		math.OneInt(),
// 	)
// 	require.NoError(err)
// 	_, err = msgServerStaking.CreateValidator(s.Setup.Ctx, msgCreateValidaotr)
// 	require.NoError(err)
// 	_, _ = s.Setup.App.EndBlocker(s.Setup.Ctx)
// 	s.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx))
// 	validator, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAccountValAddrs[0])
// 	require.NoError(err)

// 	_, err = s.Setup.Stakingkeeper.EndBlocker(s.Setup.Ctx)
// 	s.NoError(err)

// 	type Delegator struct {
// 		delegatorAddress sdk.AccAddress
// 		validator        stakingtypes.Validator
// 		tokenAmount      math.Int
// 	}
// 	pk := ed25519.GenPrivKey()
// 	reporterAccount := sdk.AccAddress(pk.PubKey().Address())
// 	// mint 5000*1e6 tokens for reporter
// 	s.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
// 	s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, reporterAccount, sdk.NewCoins(initCoins)))
// 	// delegate to validator so reporter can delegate to themselves
// 	reporterDelToVal := Delegator{delegatorAddress: reporterAccount, validator: validator, tokenAmount: math.NewInt(4000 * 1e6)}
// 	msgDelegate := stakingtypes.NewMsgDelegate(reporterAccount.String(), validator.OperatorAddress, sdk.NewCoin(s.Setup.Denom, math.NewInt(4000*1e6)))
// 	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
// 	require.NoError(err)
// 	// set up reporter module msgServer
// 	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
// 	require.NotNil(msgServerReporter)
// 	// define createReporterMsg params
// 	reporterAddress := reporterDelToVal.delegatorAddress.String()

// 	createReporterMsg := reportertypes.MsgCreateReporter{ReporterAddress: reporterAddress, CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: reportertypes.DefaultMinTrb}
// 	// send createreporter msg
// 	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &createReporterMsg)
// 	require.NoError(err)
// 	// check that reporter was created in Reporters collections
// 	reporter, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, reporterAccount)
// 	require.NoError(err)
// 	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporterAccount, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
// 	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporterAccount, reportertypes.NewSelection(reporterAccount, 1)))
// 	reporterTokens, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporterAccount)
// 	require.NoError(err)
// 	require.Equal(reporterTokens, math.NewInt(4000*1e6))
// 	require.Equal(reporter.Jailed, false)
// 	// check on reporter in Delegators collections
// 	rkDelegation, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, reporterAccount)
// 	require.NoError(err)
// 	require.Equal(rkDelegation.Reporter, reporterAccount.Bytes())
// 	// check on reporter/validator delegation
// 	valBz, err := s.Setup.Stakingkeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
// 	require.NoError(err)
// 	skDelegation, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, reporterAccount, valBz)
// 	require.NoError(err)
// 	require.Equal(skDelegation.GetDelegatorAddr(), reporterAccount.String())
// 	require.Equal(skDelegation.GetValidatorAddr(), validator.GetOperator())

// 	// setup oracle msgServer
// 	msgServerOracle := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
// 	require.NotNil(msgServerOracle)

// 	// case 1: commit/reveal for cycle list
// 	//---------------------------------------------------------------------------
// 	// Height 1
// 	//---------------------------------------------------------------------------
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
// 	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
// 	require.NoError(err)

// 	// check that no time based rewards have been minted yet
// 	tbrModuleAccount := s.Setup.Accountkeeper.GetModuleAddress(minttypes.TimeBasedRewards)
// 	tbrModuleAccountBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
// 	require.Equal(int64(0), tbrModuleAccountBalance.Amount.Int64())

// 	// begin report
// 	cycleListEth, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
// 	require.NoError(err)
// 	// create hash for commit
// 	salt1, err := oracleutils.Salt(32)
// 	require.NoError(err)
// 	value1 := testutil.EncodeValue(4500)
// 	hash1 := oracleutils.CalculateCommitment(value1, salt1)
// 	// create commit1 msg
// 	commit1 := oracletypes.MsgCommitReport{
// 		Creator:   reporterAccount.String(),
// 		QueryData: cycleListEth,
// 		Hash:      hash1,
// 	}
// 	// send commit tx
// 	commitResponse1, err := msgServerOracle.CommitReport(s.Setup.Ctx, &commit1)
// 	require.NoError(err)
// 	require.NotNil(commitResponse1)
// 	commitHeight := s.Setup.Ctx.BlockHeight()
// 	require.Equal(int64(1), commitHeight)

// 	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
// 	require.NoError(err)
// 	s.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx))

// 	//---------------------------------------------------------------------------
// 	// Height 2
// 	//---------------------------------------------------------------------------
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(commitHeight + 1)
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 14))
// 	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
// 	require.NoError(err)
// 	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
// 	require.NoError(err)

// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(commitHeight + 2)
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
// 	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
// 	require.NoError(err)

// 	// check that 1 second worth of tbr has been minted
// 	// expected tbr = (daily mint rate * time elapsed) / (# of ms in a day)
// 	expectedBlockProvision := int64(146940000 * (15 * time.Second) / (24 * 60 * 60 * 1000))
// 	expectedTbr := sdk.NewCoin(s.Setup.Denom, math.NewInt((expectedBlockProvision)).MulRaw(75).QuoRaw(100).Quo(sdk.DefaultPowerReduction))
// 	tbrModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
// 	require.GreaterOrEqual(tbrModuleAccountBalance.Amount.Int64(), expectedTbr.Amount.Int64()-1)
// 	require.LessOrEqual(tbrModuleAccountBalance.Amount.Int64(), expectedTbr.Amount.Int64()+1)
// 	// check that the cycle list has rotated
// 	cycleListBtc, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
// 	require.NotEqual(cycleListEth, cycleListBtc)
// 	require.NoError(err)

// 	// create reveal msg
// 	require.NoError(err)
// 	reveal1 := oracletypes.MsgSubmitValue{
// 		Creator:   reporterAccount.String(),
// 		QueryData: cycleListEth,
// 		Value:     value1,
// 		Salt:      salt1,
// 	}
// 	// send reveal tx
// 	revealResponse1, err := msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal1)
// 	require.NoError(err)
// 	require.NotNil(revealResponse1)
// 	// advance time and block height to expire the query and aggregate report
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(21 * time.Second))
// 	_, _ = s.Setup.App.EndBlocker(s.Setup.Ctx)
// 	s.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx))

// 	// get queryId for GetAggregatedReportRequest
// 	queryIdEth := utils.QueryIDFromData(cycleListEth)

// 	// check that aggregated report is stored
// 	getAggReportRequest1 := oracletypes.QueryGetCurrentAggregateReportRequest{
// 		QueryId: hex.EncodeToString(queryIdEth),
// 	}
// 	queryServer := oraclekeeper.NewQuerier(s.Setup.Oraclekeeper)
// 	result1, err := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &getAggReportRequest1)
// 	require.NoError(err)
// 	require.Equal(result1.Aggregate.Height, uint64(2))
// 	require.Equal(result1.Aggregate.AggregateReportIndex, uint64(0))
// 	require.Equal(result1.Aggregate.AggregateValue, testutil.EncodeValue(4500))
// 	require.Equal(result1.Aggregate.AggregateReporter, reporterAccount.String())
// 	require.Equal(result1.Aggregate.QueryId, queryIdEth)
// 	require.Equal(uint64(4000), result1.Aggregate.ReporterPower)
// 	// check that tbr is no longer in timeBasedRewards module acct
// 	tbrModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
// 	require.Equal(int64(0), tbrModuleAccountBalance.Amount.Int64())
// 	// check that tbr was sent to reporter module account
// 	reporterModuleAccount := s.Setup.Accountkeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
// 	reporterModuleAccountBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, reporterModuleAccount, s.Setup.Denom)
// 	require.Equal(expectedTbr, reporterModuleAccountBalance)
// 	// check reporters outstaning rewards
// 	outstandingRewards, err := s.Setup.Reporterkeeper.SelectorTips.Get(s.Setup.Ctx, reporterAccount.Bytes())
// 	require.NoError(err)
// 	require.Equal(outstandingRewards.TruncateInt(), expectedTbr.Amount)
// 	// withdraw tbr
// 	rewards, err := msgServerReporter.WithdrawTip(s.Setup.Ctx, &reportertypes.MsgWithdrawTip{SelectorAddress: reporterAddress, ValidatorAddress: validator.OperatorAddress})
// 	require.NoError(err)
// 	tbrEarned := outstandingRewards
// 	// check that there is only one reward to claim
// 	require.NotNil(rewards)
// 	// check that reporter module account balance is now empty
// 	reporterModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, reporterModuleAccount, s.Setup.Denom)
// 	require.Equal(int64(0), reporterModuleAccountBalance.Amount.Int64())
// 	// check that reporter now has more bonded tokens

// 	// case 2: direct reveal for cycle list
// 	//---------------------------------------------------------------------------
// 	// Height 3
// 	//---------------------------------------------------------------------------
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
// 	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
// 	require.NoError(err)

// 	// check that 8 sec of tbr has been minted
// 	tbrModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
// 	expectedBlockProvision = int64(146940000 * (22 * time.Second) / (24 * 60 * 60 * 1000))
// 	expectedTbr = sdk.NewCoin(s.Setup.Denom, (math.NewInt((expectedBlockProvision)).MulRaw(75).QuoRaw(100).Quo(sdk.DefaultPowerReduction)).Add(math.NewInt(1)))

// 	require.Equal(expectedTbr, tbrModuleAccountBalance)

// 	// get new cycle list query data
// 	cycleListTrb, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
// 	require.NoError(err)
// 	require.NotEqual(cycleListEth, cycleListTrb)
// 	require.NotEqual(cycleListBtc, cycleListTrb)
// 	// create reveal message
// 	value2 := testutil.EncodeValue(100_000)
// 	require.NoError(err)
// 	reveal2 := oracletypes.MsgSubmitValue{
// 		Creator:   reporterAccount.String(),
// 		QueryData: cycleListTrb,
// 		Value:     value2,
// 	}
// 	// send reveal message
// 	revealResponse2, err := msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal2)
// 	require.NoError(err)
// 	require.NotNil(revealResponse2)
// 	// advance time and block height to expire the query and aggregate report
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(21 * time.Second))
// 	_, _ = s.Setup.App.EndBlocker(s.Setup.Ctx)
// 	s.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx))

// 	// get queryId for GetAggregatedReportRequest
// 	queryIdTrb := utils.QueryIDFromData(cycleListTrb)

// 	// create get aggregated report query
// 	getAggReportRequest2 := oracletypes.QueryGetCurrentAggregateReportRequest{
// 		QueryId: hex.EncodeToString(queryIdTrb),
// 	}
// 	// check that aggregated report is stored correctly
// 	result2, err := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &getAggReportRequest2)
// 	require.NoError(err)
// 	require.Equal(uint64(0), result2.Aggregate.AggregateReportIndex)
// 	require.Equal(testutil.EncodeValue(100_000), result2.Aggregate.AggregateValue)
// 	require.Equal(reporterAccount.String(), result2.Aggregate.AggregateReporter)
// 	require.Equal(queryIdTrb, result2.Aggregate.QueryId)
// 	require.Equal(uint64(4000), result2.Aggregate.ReporterPower)
// 	require.Equal(uint64(3), result2.Aggregate.Height)
// 	// check that tbr is no longer in timeBasedRewards module acct
// 	tbrModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
// 	require.Equal(int64(0), tbrModuleAccountBalance.Amount.Int64())
// 	// check that tbr was sent to reporter module account
// 	reporterModuleAccount = s.Setup.Accountkeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
// 	reporterModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, reporterModuleAccount, s.Setup.Denom)
// 	require.Equal(expectedTbr, reporterModuleAccountBalance)

// 	// check reporters outstaning rewards
// 	outstandingRewards, err = s.Setup.Reporterkeeper.SelectorTips.Get(s.Setup.Ctx, reporterAccount.Bytes())
// 	require.NoError(err)
// 	require.Equal(outstandingRewards.TruncateInt(), expectedTbr.Amount)
// 	// withdraw tbr
// 	tbrEarned = tbrEarned.Add(outstandingRewards)
// 	rewards, err = msgServerReporter.WithdrawTip(s.Setup.Ctx, &reportertypes.MsgWithdrawTip{SelectorAddress: reporterAddress, ValidatorAddress: validator.OperatorAddress})
// 	require.NoError(err)
// 	require.NotNil(rewards)
// 	// check that reporter module account balance is now empty
// 	reporterModuleAccountBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, reporterModuleAccount, s.Setup.Denom)
// 	require.Equal(int64(0), reporterModuleAccountBalance.Amount.Int64())

// 	// case 3: commit/reveal for tipped query
// 	//---------------------------------------------------------------------------
// 	// Height 4
// 	//---------------------------------------------------------------------------
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
// 	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
// 	require.NoError(err)

// 	// get reporters shares
// 	deleBeforeReport, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, reporterAccount.Bytes(), valBz)
// 	require.NoError(err)
// 	require.Equal(deleBeforeReport.GetShares(), math.LegacyNewDec(4000*1e6).Add(tbrEarned))

// 	// create tip msg
// 	balanceBeforetip := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrModuleAccount, s.Setup.Denom)
// 	tipAmount := sdk.NewCoin(s.Setup.Denom, math.NewInt(100))
// 	msgTip := oracletypes.MsgTip{
// 		Tipper:    reporterAccount.String(),
// 		QueryData: cycleListEth,
// 		Amount:    tipAmount,
// 	}
// 	// send tip tx
// 	tipRes, err := msgServerOracle.Tip(s.Setup.Ctx, &msgTip)
// 	require.NoError(err)
// 	require.NotNil(tipRes)

// 	// check that tip is in oracle module account
// 	twoPercent := sdk.NewCoin(s.Setup.Denom, tipAmount.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
// 	tipModuleAcct := s.Setup.Accountkeeper.GetModuleAddress(oracletypes.ModuleName)
// 	tipAcctBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipModuleAcct, s.Setup.Denom)
// 	require.Equal(tipAcctBalance, tipAmount.Sub(twoPercent))
// 	// create commit for tipped eth query
// 	salt1, err = oracleutils.Salt(32)
// 	require.NoError(err)
// 	value1 = testutil.EncodeValue(5000)
// 	hash1 = oracleutils.CalculateCommitment(value1, salt1)

// 	queryId := utils.QueryIDFromData(cycleListEth)
// 	quertip, err := s.Setup.Oraclekeeper.GetQueryTip(s.Setup.Ctx, queryId)
// 	require.NoError(err)
// 	require.Equal(quertip, tipAmount.Amount.Sub(twoPercent.Amount))

// 	commit1 = oracletypes.MsgCommitReport{
// 		Creator:   reporterAccount.String(),
// 		QueryData: cycleListEth,
// 		Hash:      hash1,
// 	}
// 	// send commit tx
// 	commitResponse1, err = msgServerOracle.CommitReport(s.Setup.Ctx, &commit1)
// 	require.NoError(err)
// 	require.NotNil(commitResponse1)
// 	commitHeight = s.Setup.Ctx.BlockHeight()
// 	require.Equal(int64(4), commitHeight)
// 	_, _ = s.Setup.App.EndBlocker(s.Setup.Ctx)
// 	s.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx))

// 	//---------------------------------------------------------------------------
// 	// Height 5
// 	//---------------------------------------------------------------------------
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(commitHeight + 1)
// 	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
// 	require.NoError(err)

// 	// create reveal msg
// 	value1 = testutil.EncodeValue(5000)
// 	reveal1 = oracletypes.MsgSubmitValue{
// 		Creator:   reporterAccount.String(),
// 		QueryData: cycleListEth,
// 		Value:     value1,
// 		Salt:      salt1,
// 	}
// 	// send reveal tx
// 	revealResponse1, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &reveal1)
// 	require.NoError(err)
// 	require.NotNil(revealResponse1)

// 	// advance time and block height to expire the query and aggregate report
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(21 * time.Second))
// 	_, _ = s.Setup.App.EndBlocker(s.Setup.Ctx)
// 	s.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx))

// 	// create get aggreagted report query
// 	getAggReportRequest1 = oracletypes.QueryGetCurrentAggregateReportRequest{
// 		QueryId: hex.EncodeToString(queryIdEth),
// 	}
// 	// check that the aggregated report is stored correctly
// 	result1, err = queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &getAggReportRequest1)
// 	require.NoError(err)
// 	require.Equal(result1.Aggregate.AggregateReportIndex, uint64(0))
// 	require.Equal(result1.Aggregate.AggregateValue, testutil.EncodeValue(5000))
// 	require.Equal(result1.Aggregate.AggregateReporter, reporterAccount.String())
// 	require.Equal(queryIdEth, result1.Aggregate.QueryId)
// 	require.Equal(uint64(4000), result1.Aggregate.ReporterPower)
// 	require.Equal(uint64(5), result1.Aggregate.Height)
// 	// check that the tip is in tip escrow
// 	tipEscrowAcct := s.Setup.Accountkeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
// 	tipEscrowBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipEscrowAcct, s.Setup.Denom) // 98 loya
// 	require.Equal(tipAmount.Amount.Sub(twoPercent.Amount), tipEscrowBalance.Amount.Sub(balanceBeforetip.Amount))
// 	// withdraw tip
// 	msgWithdrawTip := reportertypes.MsgWithdrawTip{
// 		SelectorAddress:  reporterAddress,
// 		ValidatorAddress: validator.OperatorAddress,
// 	}
// 	_, err = msgServerReporter.WithdrawTip(s.Setup.Ctx, &msgWithdrawTip)
// 	require.NoError(err)

// 	// check that tip is no longer in escrow pool
// 	tipEscrowBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipEscrowAcct, s.Setup.Denom)
// 	require.Equal(int64(0), tipEscrowBalance.Amount.Int64())
// 	// check that reporter now has more bonded tokens
// 	deleAfter, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, reporterAccount.Bytes(), valBz)
// 	require.NoError(err)
// 	tipPlusTbr := math.NewInt(98 + 26786)
// 	require.Equal(deleBeforeReport.GetShares().Add(math.LegacyNewDecFromInt(tipPlusTbr)), deleAfter.GetShares())

// 	_, _ = s.Setup.App.EndBlocker(s.Setup.Ctx)
// 	s.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx))

// 	// case 4: submit without committing for tipped query
// 	//---------------------------------------------------------------------------
// 	// Height 6
// 	//---------------------------------------------------------------------------
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
// 	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
// 	require.NoError(err)

// 	// check reporter starting shares
// 	deleBeforeReport2, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, reporterAccount.Bytes(), valBz)
// 	require.NoError(err)

// 	twoPercentTip := sdk.NewCoin(s.Setup.Denom, tipAmount.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
// 	twoPercentTipPlusTbr := sdk.NewCoin(s.Setup.Denom, tipAmount.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(tipPlusTbr.Int64())))
// 	expectedShares := math.LegacyNewDecFromInt(deleBeforeReport.GetShares().TruncateInt().Add(tipPlusTbr)) // 8928 is the tbr that was earned
// 	require.Equal(deleBeforeReport2.GetShares(), expectedShares)

// 	// create tip msg
// 	msgTip = oracletypes.MsgTip{
// 		Tipper:    reporterAccount.String(),
// 		QueryData: cycleListTrb,
// 		Amount:    tipAmount,
// 	}
// 	// send tip tx
// 	tipRes, err = msgServerOracle.Tip(s.Setup.Ctx, &msgTip)
// 	require.NoError(err)
// 	require.NotNil(tipRes)
// 	// check that tip is in oracle module account
// 	tipModuleAcct = s.Setup.Accountkeeper.GetModuleAddress(oracletypes.ModuleName)
// 	tipAcctBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipModuleAcct, s.Setup.Denom)
// 	require.Equal(tipAcctBalance.Amount, tipAmount.Amount.Sub(twoPercentTip.Amount))
// 	// create submit msg
// 	revealMsgTrb := oracletypes.MsgSubmitValue{
// 		Creator:   reporterAccount.String(),
// 		QueryData: cycleListTrb,
// 		Value:     testutil.EncodeValue(1_000_000),
// 	}
// 	// send submit msg
// 	revealTrb, err := msgServerOracle.SubmitValue(s.Setup.Ctx, &revealMsgTrb)
// 	require.NoError(err)
// 	require.NotNil(revealTrb)
// 	// advance time and block height to expire the query and aggregate report
// 	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(21 * time.Second))
// 	_, _ = s.Setup.App.EndBlocker(s.Setup.Ctx)
// 	s.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx))
// 	// create get aggregated report query
// 	getAggReportRequestTrb := oracletypes.QueryGetCurrentAggregateReportRequest{
// 		QueryId: hex.EncodeToString(queryIdTrb),
// 	}
// 	// query aggregated report
// 	reportTrb, err := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &getAggReportRequestTrb)
// 	require.NoError(err)
// 	require.Equal(reportTrb.Aggregate.AggregateReportIndex, uint64(0))
// 	require.Equal(reportTrb.Aggregate.AggregateValue, testutil.EncodeValue(1_000_000))
// 	require.Equal(reportTrb.Aggregate.AggregateReporter, reporterAccount.String())
// 	require.Equal(queryIdTrb, reportTrb.Aggregate.QueryId)
// 	require.Equal(uint64(4000), reportTrb.Aggregate.ReporterPower)
// 	require.Equal(uint64(6), reportTrb.Aggregate.Height)
// 	// check that the tip is in tip escrow
// 	tipEscrowBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipEscrowAcct, s.Setup.Denom) // 98 loya
// 	require.Equal(tipPlusTbr.Sub(twoPercentTipPlusTbr.Amount), tipEscrowBalance.Amount)
// 	// withdraw tip
// 	_, err = msgServerReporter.WithdrawTip(s.Setup.Ctx, &msgWithdrawTip)
// 	require.NoError(err)
// 	// check that tip is no longer in escrow pool
// 	tipEscrowBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tipEscrowAcct, s.Setup.Denom)
// 	require.Equal(int64(0), tipEscrowBalance.Amount.Int64())
// 	// check that reporter now has more bonded tokens
// 	deleAfter, err = s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, reporterAccount.Bytes(), valBz)
// 	require.NoError(err)
// 	require.Equal(deleBeforeReport2.GetShares().Add(math.LegacyNewDecFromInt(tipPlusTbr)), deleAfter.GetShares())
// }

func (s *E2ETestSuite) TestAggregateOverMultipleBlocks() {
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
	s.Setup.Ctx = s.Setup.Ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	//---------------------------------------------------------------------------
	// Height 0 - vicky becomes a validator
	//---------------------------------------------------------------------------
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	vickyAccAddr := simtestutil.CreateIncrementalAccounts(1)
	vickyInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(2000*1e6))
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
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
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
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
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
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
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
	// Height 4 - Roman and Ricky become reporters
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(4)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	romanPrivKey := secp256k1.GenPrivKey()
	romanAccAddr := sdk.AccAddress(romanPrivKey.PubKey().Address())
	romanInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(200*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(romanInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, romanAccAddr, sdk.NewCoins(romanInitCoins)))

	// roman delegates to vicky
	msgDelegate = stakingtypes.NewMsgDelegate(
		romanAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(200*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   romanAccAddr.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewInt(1 * 1e6),
	})
	require.NoError(err)
	romanReporterInfo, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, romanAccAddr)
	require.NoError(err)
	require.Equal(romanReporterInfo.Jailed, false)

	rickyPrivKey := secp256k1.GenPrivKey()
	rickyAccAddr := sdk.AccAddress(rickyPrivKey.PubKey().Address())
	rickyInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(300*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(rickyInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, rickyAccAddr, sdk.NewCoins(rickyInitCoins)))

	// ricky delegates to vicky
	msgDelegate = stakingtypes.NewMsgDelegate(
		rickyAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(300*1e6)),
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
	// Height 5 - only one block left in this cycle list query, pretend empty block
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(5)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 6 - Rob direct reveals for cycle list at height 6
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(6)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// Rob direct reveals for cycle list
	currentCycleList, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryId := utils.QueryIDFromData(currentCycleList)
	msgSubmitValue := oracletypes.MsgSubmitValue{
		Creator:   robAccAddr.String(),
		QueryData: currentCycleList,
		Value:     testutil.EncodeValue(90_000),
	}
	_, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &msgSubmitValue)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - Roman and Ricky direct reveal for the same cycle list at height 7
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(7)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	msgSubmitValue = oracletypes.MsgSubmitValue{
		Creator:   romanAccAddr.String(),
		QueryData: currentCycleList,
		Value:     testutil.EncodeValue(100_000),
	}
	_, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &msgSubmitValue)
	require.NoError(err)

	msgSubmitValue = oracletypes.MsgSubmitValue{
		Creator:   rickyAccAddr.String(),
		QueryData: currentCycleList,
		Value:     testutil.EncodeValue(110_000),
	}
	_, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &msgSubmitValue)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - Commit window expires, report gets aggregated in endblock
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(8)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	aggregate, time, err := s.Setup.Oraclekeeper.GetCurrentAggregateReport(s.Setup.Ctx, queryId)
	require.NoError(err)
	// require.Equal(3, len(aggregate.Reporters))
	// require.Equal(aggregate.AggregateReportIndex, uint64(1))
	require.Equal(aggregate.AggregateValue, testutil.EncodeValue(100_000))
	require.Equal(aggregate.AggregateReporter, romanAccAddr.String())
	require.Equal(aggregate.Height, uint64(7))
	require.Equal(aggregate.AggregatePower, uint64(850))
	require.Equal(aggregate.QueryId, queryId)
	require.Equal(aggregate.MetaId, uint64(2))

	// agg, err := s.Setup.Oraclekeeper.Aggregates.Get(s.Setup.Ctx, collections.Join(queryId, uint64(time.UnixMilli())))
	// require.NoError(err)
	// require.Equal(3, len(agg.Reporters))

	oracleQuerier := oraclekeeper.NewQuerier(s.Setup.Oraclekeeper)
	microreports, err := oracleQuerier.GetReportsByAggregate(s.Setup.Ctx, &oracletypes.QueryGetReportsByAggregateRequest{
		QueryId:    hex.EncodeToString(queryId),
		Timestamp:  uint64(time.UnixMilli()),
		Pagination: &query.PageRequest{Limit: 100, CountTotal: true},
	})
	require.NoError(err)
	require.Equal(3, len(microreports.MicroReports))
}
