package e2e_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	collections "cosmossdk.io/collections"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/utils"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	// oracleutils "github.com/tellor-io/layer/x/oracle/utils"
	registrykeeper "github.com/tellor-io/layer/x/registry/keeper"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"
)

func (s *E2ETestSuite) TestInitialMint() {
	require := s.Require()

	mintToTeamAcc := s.accountKeeper.GetModuleAddress(minttypes.MintToTeam)
	require.NotNil(mintToTeamAcc)
	balance := s.bankKeeper.GetBalance(s.ctx, mintToTeamAcc, s.denom)
	require.Equal(balance.Amount, math.NewInt(300*1e6))
}

func (s *E2ETestSuite) TestTransferAfterMint() {
	require := s.Require()

	mintToTeamAcc := s.accountKeeper.GetModuleAddress(minttypes.MintToTeam)
	require.NotNil(mintToTeamAcc)
	balance := s.bankKeeper.GetBalance(s.ctx, mintToTeamAcc, s.denom)
	require.Equal(balance.Amount, math.NewInt(300*1e6))

	// create 5 accounts
	type Accounts struct {
		PrivateKey secp256k1.PrivKey
		Account    sdk.AccAddress
	}
	accounts := make([]Accounts, 0, 5)
	for i := 0; i < 5; i++ {
		privKey := secp256k1.GenPrivKey()
		accountAddress := sdk.AccAddress(privKey.PubKey().Address())
		account := authtypes.BaseAccount{
			Address:       accountAddress.String(),
			PubKey:        codectypes.UnsafePackAny(privKey.PubKey()),
			AccountNumber: uint64(i + 1),
		}
		existingAccount := s.accountKeeper.GetAccount(s.ctx, accountAddress)
		if existingAccount == nil {
			s.accountKeeper.SetAccount(s.ctx, &account)
			accounts = append(accounts, Accounts{
				PrivateKey: *privKey,
				Account:    accountAddress,
			})
		}
	}

	// transfer 1000 tokens from team to all 5 accounts
	for _, acc := range accounts {
		startBalance := s.bankKeeper.GetBalance(s.ctx, acc.Account, s.denom).Amount
		err := s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, minttypes.MintToTeam, acc.Account, sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(1000))))
		require.NoError(err)
		require.Equal(startBalance.Add(math.NewInt(1000)), s.bankKeeper.GetBalance(s.ctx, acc.Account, s.denom).Amount)
	}
	expectedTeamBalance := math.NewInt(300*1e6 - 1000*5)
	require.Equal(expectedTeamBalance, s.bankKeeper.GetBalance(s.ctx, mintToTeamAcc, s.denom).Amount)

	// transfer from account 0 to account 1
	s.bankKeeper.SendCoins(s.ctx, accounts[0].Account, accounts[1].Account, sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(1000))))
	require.Equal(math.NewInt(0), s.bankKeeper.GetBalance(s.ctx, accounts[0].Account, s.denom).Amount)
	require.Equal(math.NewInt(2000), s.bankKeeper.GetBalance(s.ctx, accounts[1].Account, s.denom).Amount)

	// transfer from account 2 to team
	s.bankKeeper.SendCoinsFromAccountToModule(s.ctx, accounts[2].Account, minttypes.MintToTeam, sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(1000))))
	require.Equal(math.NewInt(0), s.bankKeeper.GetBalance(s.ctx, accounts[2].Account, s.denom).Amount)
	require.Equal(expectedTeamBalance.Add(math.NewInt(1000)), s.bankKeeper.GetBalance(s.ctx, mintToTeamAcc, s.denom).Amount)

	// try to transfer more than balance from account 3 to 4
	err := s.bankKeeper.SendCoins(s.ctx, accounts[3].Account, accounts[4].Account, sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(1001))))
	require.Error(err)
	require.Equal(s.bankKeeper.GetBalance(s.ctx, accounts[3].Account, s.denom).Amount, math.NewInt(1000))
	require.Equal(s.bankKeeper.GetBalance(s.ctx, accounts[4].Account, s.denom).Amount, math.NewInt(1000))
}

func (s *E2ETestSuite) TestValidateCycleList() {
	require := s.Require()

	//---------------------------------------------------------------------------
	// Height 0 - get initial cycle list query
	//---------------------------------------------------------------------------
	_, err := s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	firstInCycle, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	queryDataBytes, err := utils.QueryBytesFromString(ethQueryData[2:])
	require.NoError(err)
	require.Equal(queryDataBytes, firstInCycle)
	require.Equal(s.ctx.BlockHeight(), int64(0))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - get second cycle list query
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	require.Equal(s.ctx.BlockHeight(), int64(1))
	secondInCycle, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	queryDataBytes, err = utils.QueryBytesFromString(btcQueryData[2:])
	require.NoError(err)
	require.Equal(queryDataBytes, secondInCycle)
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - get third cycle list query
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	require.Equal(s.ctx.BlockHeight(), int64(2))
	thirdInCycle, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	queryDataBytes, err = utils.QueryBytesFromString(trbQueryData[2:])
	require.NoError(err)
	require.Equal(queryDataBytes, thirdInCycle)
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	// loop through 20 more blocks
	list, err := s.oraclekeeper.GetCyclelist(s.ctx)
	require.NoError(err)
	for i := 0; i < 20; i++ {
		s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
		_, err = s.app.BeginBlocker(s.ctx)
		require.NoError(err)

		query, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
		require.NoError(err)
		require.Contains(list, query)

		_, err = s.app.EndBlocker(s.ctx)
		require.NoError(err)
	}
}

func (s *E2ETestSuite) TestSetUpValidatorAndReporter() {
	require := s.Require()

	// Create Validator Accounts
	numValidators := 10
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	_ = new(big.Int).Mul(big.NewInt(1000), base)

	// make addresses
	testAddresses := simtestutil.CreateIncrementalAccounts(numValidators)
	// mint 50k tokens to minter account and send to each address
	initCoins := sdk.NewCoin(s.denom, math.NewInt(5000*1e6))
	for _, addr := range testAddresses {
		s.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, addr, sdk.NewCoins(initCoins)))
	}
	// get val address for each test address
	valAddresses := simtestutil.ConvertAddrsToValAddrs(testAddresses)
	// create pub keys for each address
	pubKeys := simtestutil.CreateTestPubKeys(numValidators)

	// set each account with proper keepers
	for i, pubKey := range pubKeys {
		s.accountKeeper.NewAccountWithAddress(s.ctx, testAddresses[i])
		validator, err := stakingtypes.NewValidator(valAddresses[i].String(), pubKey, stakingtypes.Description{Moniker: strconv.Itoa(i)})
		require.NoError(err)
		s.stakingKeeper.SetValidator(s.ctx, validator)
		s.stakingKeeper.SetValidatorByConsAddr(s.ctx, validator)
		s.stakingKeeper.SetNewValidatorByPowerIndex(s.ctx, validator)

		randomStakeAmount := rand.Intn(5000-1000+1) + 1000
		require.True(randomStakeAmount >= 1000 && randomStakeAmount <= 5000, "randomStakeAmount is not within the expected range")
		_, err = s.stakingKeeper.Delegate(s.ctx, testAddresses[i], math.NewInt(int64(randomStakeAmount)*1e6), stakingtypes.Unbonded, validator, true)
		require.NoError(err)
		// call hooks for distribution init
		valBz, err := s.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
		if err != nil {
			panic(err)
		}
		err = s.distrKeeper.Hooks().AfterValidatorCreated(s.ctx, valBz)
		require.NoError(err)
		err = s.distrKeeper.Hooks().BeforeDelegationCreated(s.ctx, testAddresses[i], valBz)
		require.NoError(err)
		err = s.distrKeeper.Hooks().AfterDelegationModified(s.ctx, testAddresses[i], valBz)
		require.NoError(err)
	}

	_, err := s.stakingKeeper.EndBlocker(s.ctx)
	s.NoError(err)

	// check that everyone is a bonded validator
	validatorSet, err := s.stakingKeeper.GetAllValidators(s.ctx)
	require.NoError(err)
	for _, val := range validatorSet {
		status := val.GetStatus()
		require.Equal(stakingtypes.Bonded.String(), status.String())
	}

	// create 3 delegators
	const (
		reporter     = "reporter"
		delegatorI   = "delegator1"
		delegatorII  = "delegator2"
		delegatorIII = "delegator3"
	)

	type Delegator struct {
		delegatorAddress sdk.AccAddress
		validator        stakingtypes.Validator
		tokenAmount      math.Int
	}

	numDelegators := 4
	// create random private keys for each delegator
	delegatorPrivateKeys := make([]secp256k1.PrivKey, numDelegators)
	for i := 0; i < numDelegators; i++ {
		pk := secp256k1.GenPrivKey()
		delegatorPrivateKeys[i] = *pk
	}
	// turn private keys into accounts
	delegatorAccounts := make([]sdk.AccAddress, numDelegators)
	for i, pk := range delegatorPrivateKeys {
		delegatorAccounts[i] = sdk.AccAddress(pk.PubKey().Address())
		// give each account tokens
		s.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, delegatorAccounts[i], sdk.NewCoins(initCoins)))
	}
	// define each delegator
	delegators := map[string]Delegator{
		reporter:     {delegatorAddress: delegatorAccounts[0], validator: validatorSet[1], tokenAmount: math.NewInt(100 * 1e6)},
		delegatorI:   {delegatorAddress: delegatorAccounts[1], validator: validatorSet[1], tokenAmount: math.NewInt(100 * 1e6)},
		delegatorII:  {delegatorAddress: delegatorAccounts[2], validator: validatorSet[1], tokenAmount: math.NewInt(100 * 1e6)},
		delegatorIII: {delegatorAddress: delegatorAccounts[3], validator: validatorSet[2], tokenAmount: math.NewInt(100 * 1e6)},
	}
	// delegate to validators
	for _, del := range delegators {
		_, err := s.stakingKeeper.Delegate(s.ctx, del.delegatorAddress, del.tokenAmount, stakingtypes.Unbonded, del.validator, true)
		require.NoError(err)
	}

	// set up reporter module msgServer
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.reporterkeeper)
	require.NotNil(msgServerReporter)
	// define reporter params
	var createReporterMsg reportertypes.MsgCreateReporter
	reporterAddress := delegators[reporter].delegatorAddress.String()
	amount := math.NewInt(100 * 1e6)
	source := reportertypes.TokenOrigin{ValidatorAddress: validatorSet[1].GetOperator(), Amount: math.NewInt(100 * 1e6)}
	commission := stakingtypes.NewCommissionWithTime(math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(3, 1),
		math.LegacyNewDecWithPrec(1, 1), s.ctx.BlockTime())
	// fill in createReporterMsg
	createReporterMsg.Reporter = reporterAddress
	createReporterMsg.Amount = amount
	createReporterMsg.TokenOrigins = []*reportertypes.TokenOrigin{&source}
	createReporterMsg.Commission = &commission
	// create reporter through msg server
	_, err = msgServerReporter.CreateReporter(s.ctx, &createReporterMsg)
	require.NoError(err)
	// check that reporter was created correctly
	oracleReporter, err := s.reporterkeeper.Reporters.Get(s.ctx, delegators[reporter].delegatorAddress)
	require.NoError(err)
	require.Equal(oracleReporter.Reporter, delegators[reporter].delegatorAddress.String())
	require.Equal(oracleReporter.TotalTokens, math.NewInt(100*1e6))
	require.Equal(oracleReporter.Jailed, false)

	// define delegation source
	source = reportertypes.TokenOrigin{ValidatorAddress: validatorSet[1].GetOperator(), Amount: math.NewInt(25 * 1e6)}
	delegationMsg := reportertypes.NewMsgDelegateReporter(
		delegators[delegatorI].delegatorAddress.String(),
		delegators[reporter].delegatorAddress.String(),
		math.NewInt(25*1e6),
		[]*reportertypes.TokenOrigin{&source},
	)
	// self delegate as reporter
	_, err = msgServerReporter.DelegateReporter(s.ctx, delegationMsg)
	require.NoError(err)
	delegationReporter, err := s.reporterkeeper.Delegators.Get(s.ctx, delegators[delegatorI].delegatorAddress)
	require.NoError(err)
	require.Equal(delegationReporter.Reporter, delegators[reporter].delegatorAddress.String())
}

// todo: Use helper functions for creating reporters and validators
func (s *E2ETestSuite) TestDisputes() {
	require := s.Require()
	msgServerOracle := oraclekeeper.NewMsgServerImpl(s.oraclekeeper)
	require.NotNil(msgServerOracle)
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.reporterkeeper)
	require.NotNil(msgServerReporter)
	msgServerDispute := disputekeeper.NewMsgServerImpl(s.disputekeeper)
	require.NotNil(msgServerDispute)

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
	validator, err := stakingtypes.NewValidator(valAccountValAddrs[0].String(), pubKey[0], stakingtypes.Description{Moniker: "created validator"})
	require.NoError(err)
	s.stakingKeeper.SetValidator(s.ctx, validator)
	s.stakingKeeper.SetValidatorByConsAddr(s.ctx, validator)
	s.stakingKeeper.SetNewValidatorByPowerIndex(s.ctx, validator)
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

	//create a self delegated reporter from a different account
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
	_, err = s.stakingKeeper.Delegate(s.ctx, reporterDelToVal.delegatorAddress, reporterDelToVal.tokenAmount, stakingtypes.Unbonded, reporterDelToVal.validator, true)
	require.NoError(err)
	// call dist module hooks
	err = s.distrKeeper.Hooks().AfterValidatorCreated(s.ctx, valBz)
	require.NoError(err)
	err = s.distrKeeper.Hooks().BeforeDelegationCreated(s.ctx, reporterAccount, valAccountValAddrs[0])
	require.NoError(err)
	err = s.distrKeeper.Hooks().AfterDelegationModified(s.ctx, reporterAccount, valAccountValAddrs[0])
	require.NoError(err)

	// self delegate in reporter module with 4k trb
	var createReporterMsg reportertypes.MsgCreateReporter
	reporterAddress := reporterDelToVal.delegatorAddress.String()
	amount := math.NewInt(4000 * 1e6)
	source := reportertypes.TokenOrigin{ValidatorAddress: validator.OperatorAddress, Amount: math.NewInt(4000 * 1e6)}
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
	require.Equal(reporter.Reporter, reporterAccount.String())
	require.Equal(reporter.TotalTokens, math.NewInt(4000*1e6))
	require.Equal(reporter.Jailed, false)
	// check on reporter in Delegators collections
	rkDelegation, err := s.reporterkeeper.Delegators.Get(s.ctx, reporterAccount)
	require.NoError(err)
	require.Equal(rkDelegation.Reporter, reporterAccount.String())
	require.Equal(rkDelegation.Amount, math.NewInt(4000*1e6))
	// check on reporter/validator delegation
	skDelegation, err := s.stakingKeeper.Delegation(s.ctx, reporterAccount, valBz)
	require.NoError(err)
	require.Equal(skDelegation.GetDelegatorAddr(), reporterAccount.String())
	require.Equal(skDelegation.GetValidatorAddr(), validator.GetOperator())

	//---------------------------------------------------------------------------
	// Height 1 - direct reveal for cycle list
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	// get new cycle list query data
	cycleListQuery, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	// create reveal message
	value := encodeValue(100_000)
	require.NoError(err)
	reveal := oracletypes.MsgSubmitValue{
		Creator:   reporter.Reporter,
		QueryData: cycleListQuery,
		Value:     value,
	}
	// send reveal message
	revealResponse, err := msgServerOracle.SubmitValue(s.ctx, &reveal)
	require.NoError(err)
	require.NotNil(revealResponse)
	// advance time and block height to expire the query and aggregate report
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryId := utils.QueryIDFromData(cycleListQuery)
	s.NoError(err)
	// create get aggregated report query
	getAggReportRequest := oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: queryId,
	}
	// aggregated report is stored correctly
	result, err := s.oraclekeeper.GetAggregatedReport(s.ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(int64(0), result.Report.AggregateReportIndex)
	require.Equal(encodeValue(100_000), result.Report.AggregateValue)
	require.Equal(reporter.Reporter, result.Report.AggregateReporter)
	require.Equal(queryId, result.Report.QueryId)
	require.Equal(int64(4000000000), result.Report.ReporterPower)
	require.Equal(int64(1), result.Report.Height)

	//---------------------------------------------------------------------------
	// Height 2 - create a dispute
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, false)
	freeFloatingBalanceBefore := s.bankKeeper.GetBalance(s.ctx, reporterAccount, s.denom)

	balBeforeDispute := reporter.TotalTokens
	onePercent := balBeforeDispute.Mul(math.NewInt(1)).Quo(math.NewInt(100))
	disputeFee := sdk.NewCoin(s.denom, onePercent) // warning should be 1% of bonded tokens

	// todo: is there a getter for this ?
	// get microreport for dispute
	report := oracletypes.MicroReport{
		Reporter:  reporter.Reporter,
		Power:     reporter.TotalTokens.Int64(),
		QueryId:   queryId,
		Value:     value,
		Timestamp: s.ctx.BlockTime(),
	}

	// create msg for propose dispute tx
	msgProposeDispute := disputetypes.MsgProposeDispute{
		Creator:         reporter.Reporter,
		Report:          &report,
		DisputeCategory: disputetypes.Warning,
		Fee:             disputeFee,
		PayFromBond:     false,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.ctx, &msgProposeDispute)
	require.NoError(err)

	burnAmount := disputeFee.Amount.MulRaw(1).QuoRaw(20)
	disputes, err := s.disputekeeper.OpenDisputes.Get(s.ctx)
	require.NoError(err)
	require.NotNil(disputes)
	// dispute is created correctly
	dispute, err := s.disputekeeper.Disputes.Get(s.ctx, 1)
	require.NoError(err)
	require.Equal(dispute.DisputeId, uint64(1))
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeCategory, disputetypes.Warning)
	require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	require.Equal(dispute.FeePayers, []disputetypes.PayerInfo{{PayerAddress: reporter.Reporter, Amount: disputeFee.Amount, FromBond: false, BlockNumber: 2}})

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - unjail reporter
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// reporter is in jail
	reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, true)
	// reporter lost 1% of their free floating tokens
	freeFloatingBalanceAfter := s.bankKeeper.GetBalance(s.ctx, reporterAccount, s.denom)
	require.Equal(freeFloatingBalanceAfter, freeFloatingBalanceBefore.Sub(disputeFee))

	// create msgUnJailReporter
	msgUnjailReporter := reportertypes.MsgUnjailReporter{
		ReporterAddress: reporter.Reporter,
	}
	// send unjailreporter tx
	_, err = msgServerReporter.UnjailReporter(s.ctx, &msgUnjailReporter)
	require.NoError(err)

	// reporter is now unjailed
	reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, false)
	freeFloatingBalanceAfter = s.bankKeeper.GetBalance(s.ctx, reporterAccount, s.denom)
	require.Equal(freeFloatingBalanceAfter, freeFloatingBalanceBefore.Sub(disputeFee))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)
	// todo: more balance checks at each step

	//---------------------------------------------------------------------------
	// Height 4 - direct reveal for cycle list again
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// get new cycle list query data
	cycleListQuery, err = s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	// create reveal message
	value = encodeValue(100_000)
	require.NoError(err)
	reveal = oracletypes.MsgSubmitValue{
		Creator:   reporter.Reporter,
		QueryData: cycleListQuery,
		Value:     value,
	}
	// send reveal message
	revealResponse, err = msgServerOracle.SubmitValue(s.ctx, &reveal)
	require.NoError(err)
	require.NotNil(revealResponse)
	// advance time and block height to expire the query and aggregate report
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryId = utils.QueryIDFromData(cycleListQuery)
	s.NoError(err)
	// create get aggregated report query
	getAggReportRequest = oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: queryId,
	}
	// aggregated report is stored correctly
	result, err = s.oraclekeeper.GetAggregatedReport(s.ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(int64(0), result.Report.AggregateReportIndex)
	require.Equal(encodeValue(100_000), result.Report.AggregateValue)
	require.Equal(reporter.Reporter, result.Report.AggregateReporter)
	require.Equal(queryId, result.Report.QueryId)
	require.Equal(int64(4000000000), result.Report.ReporterPower)
	require.Equal(int64(4), result.Report.Height)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5 - open minor dispute
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	balBeforeDispute = reporter.TotalTokens
	fivePercent := balBeforeDispute.Mul(math.NewInt(5)).Quo(math.NewInt(100))
	disputeFee = sdk.NewCoin(s.denom, fivePercent)

	report = oracletypes.MicroReport{
		Reporter:  reporter.Reporter,
		Power:     reporter.TotalTokens.Int64(),
		QueryId:   queryId,
		Value:     value,
		Timestamp: s.ctx.BlockTime(),
	}

	// create msg for propose dispute tx
	msgProposeDispute = disputetypes.MsgProposeDispute{
		Creator:         reporter.Reporter,
		Report:          &report,
		DisputeCategory: disputetypes.Minor,
		Fee:             disputeFee,
		PayFromBond:     false,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.ctx, &msgProposeDispute)
	require.NoError(err)
	disputeStartTime := s.ctx.BlockTime()

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 6 - vote on minor dispute
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// reporter is in jail
	reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, true)
	// dispute is created correctly
	burnAmount = disputeFee.Amount.MulRaw(1).QuoRaw(20)
	dispute, err = s.disputekeeper.GetDisputeByReporter(s.ctx, report, disputetypes.Minor)
	require.NoError(err)
	require.Equal(dispute.DisputeCategory, disputetypes.Minor)
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	require.Equal(dispute.FeePayers, []disputetypes.PayerInfo{{PayerAddress: reporter.Reporter, Amount: disputeFee.Amount, FromBond: false, BlockNumber: 5}})

	// create vote tx msg
	msgVote := disputetypes.MsgVote{
		Voter: reporter.Reporter,
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}
	// send vote tx
	voteResponse, err := msgServerDispute.Vote(s.ctx, &msgVote)
	require.NoError(err)
	require.NotNil(voteResponse)

	// vote is properly stored
	vote, err := s.disputekeeper.Votes.Get(s.ctx, dispute.DisputeId)
	require.NoError(err)
	require.NotNil(vote)
	require.Equal(vote.Executed, false)
	require.Equal(vote.Id, dispute.DisputeId)
	require.Equal(vote.VoteStart, disputeStartTime)
	require.Equal(vote.VoteEnd, disputeStartTime.Add(disputekeeper.TWO_DAYS))

	// advance 2 days to expire vote
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(disputekeeper.THREE_DAYS))
	// call unjail function
	msgUnjailReporter = reportertypes.MsgUnjailReporter{
		ReporterAddress: reporter.Reporter,
	}
	_, err = msgServerReporter.UnjailReporter(s.ctx, &msgUnjailReporter)
	require.NoError(err)

	// reporter no longer in jail
	reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, false)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - minor dispute ends and another direct reveal for cycle list
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// vote is executed
	vote, err = s.disputekeeper.Votes.Get(s.ctx, dispute.DisputeId)
	require.NoError(err)
	require.NotNil(vote)
	require.Equal(vote.Executed, true)
	require.Equal(vote.Id, dispute.DisputeId)
	// reporter no longer in jail
	reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, false)

	// get open disputes
	disputes, err = s.disputekeeper.OpenDisputes.Get(s.ctx)
	require.NoError(err)
	require.NotNil(disputes)

	// get new cycle list query data
	cycleListQuery, err = s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	// create reveal message
	value = encodeValue(100_000)
	require.NoError(err)
	reveal = oracletypes.MsgSubmitValue{
		Creator:   reporter.Reporter,
		QueryData: cycleListQuery,
		Value:     value,
	}
	// send reveal message
	revealResponse, err = msgServerOracle.SubmitValue(s.ctx, &reveal)
	require.NoError(err)
	require.NotNil(revealResponse)
	// advance time and block height to expire the query and aggregate report
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)
	// get queryId for GetAggregatedReportRequest
	queryId = utils.QueryIDFromData(cycleListQuery)
	s.NoError(err)
	// create get aggregated report query
	getAggReportRequest = oracletypes.QueryGetCurrentAggregatedReportRequest{
		QueryId: queryId,
	}
	// check that aggregated report is stored correctly
	result, err = s.oraclekeeper.GetAggregatedReport(s.ctx, &getAggReportRequest)
	require.NoError(err)
	require.Equal(int64(0), result.Report.AggregateReportIndex)
	require.Equal(encodeValue(100_000), result.Report.AggregateValue)
	require.Equal(reporter.Reporter, result.Report.AggregateReporter)
	require.Equal(queryId, result.Report.QueryId)
	require.Equal(int64(7), result.Report.Height)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 8 - open major dispute for report
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, false)

	oneHundredPercent := reporter.TotalTokens
	disputeFee = sdk.NewCoin(s.denom, oneHundredPercent)

	report = oracletypes.MicroReport{
		Reporter:  reporter.Reporter,
		Power:     reporter.TotalTokens.Int64(),
		QueryId:   queryId,
		Value:     value,
		Timestamp: s.ctx.BlockTime(),
	}

	// create msg for propose dispute tx
	msgProposeDispute = disputetypes.MsgProposeDispute{
		Creator:         reporter.Reporter,
		Report:          &report,
		DisputeCategory: disputetypes.Major,
		Fee:             disputeFee,
		PayFromBond:     false,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.ctx, &msgProposeDispute)
	require.NoError(err)
	disputeStartTime = s.ctx.BlockTime()
	disputeStartHeight := s.ctx.BlockHeight()

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 9 - vote on major dispute
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	fee, err := s.disputekeeper.GetDisputeFee(s.ctx, reporter.Reporter, disputetypes.Major)
	require.NoError(err)
	require.GreaterOrEqual(msgProposeDispute.Fee.Amount.Uint64(), fee.Uint64())

	// dispute is created and open for voting
	dispute, err = s.disputekeeper.GetDisputeByReporter(s.ctx, report, disputetypes.Major)
	require.NoError(err)
	burnAmount = disputeFee.Amount.MulRaw(1).QuoRaw(20)
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeStartTime, disputeStartTime)
	require.Equal(dispute.DisputeEndTime, disputeStartTime.Add(disputekeeper.THREE_DAYS))
	require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	require.Equal(dispute.DisputeStartBlock, disputeStartHeight)
	// todo: handle reporter removal
	reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	require.NoError(err)

	// create vote tx msg
	msgVote = disputetypes.MsgVote{
		Voter: reporter.Reporter,
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}
	// send vote tx
	voteResponse, err = msgServerDispute.Vote(s.ctx, &msgVote)
	require.NoError(err)
	require.NotNil(voteResponse)

	// vote is properly stored
	vote, err = s.disputekeeper.Votes.Get(s.ctx, dispute.DisputeId)
	require.NoError(err)
	require.NotNil(vote)
	require.Equal(vote.Executed, false)
	require.Equal(vote.Id, dispute.DisputeId)
	require.Equal(vote.VoteStart, disputeStartTime)
	require.Equal(vote.VoteEnd, disputeStartTime.Add(disputekeeper.TWO_DAYS))

	// advance 3 days to expire vote
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(disputekeeper.THREE_DAYS))

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	// ---------------------------------------------------------------------------
	// Height 10 - dispute is resolved, reporter no longer a reporter
	// ---------------------------------------------------------------------------
	// s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	// s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	// _, err = s.app.BeginBlocker(s.ctx)
	// require.NoError(err)

	// reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// require.NoError(err)

}

func (s *E2ETestSuite) TestUnstaking() {
	require := s.Require()

	// create 5 validators with 5_000 TRB
	accaddr, valaddr, _ := s.CreateValidators(5)
	s.stakingKeeper.EndBlocker(s.ctx)
	// all validators are bonded
	validators, err := s.stakingKeeper.GetAllValidators(s.ctx)
	require.NoError(err)
	require.NotNil(validators)
	for _, val := range validators {
		require.Equal(val.Status.String(), stakingtypes.BondStatusBonded)
	}

	// begin unbonding validator 0
	del, err := s.stakingKeeper.GetDelegation(s.ctx, accaddr[1], valaddr[1])
	require.NoError(err)
	// undelegate all shares except for 1 to avoid getting the validator deleted
	timeToUnbond, _, err := s.stakingKeeper.Undelegate(s.ctx, accaddr[1], valaddr[1], del.Shares.Sub(math.LegacyNewDec(1)))
	require.NoError(err)

	// unbonding time is 21 days after calling BeginUnbondingValidator
	unbondingStartTime := s.ctx.BlockTime()
	twentyOneDays := time.Hour * 24 * 21
	require.Equal(unbondingStartTime.Add(twentyOneDays), timeToUnbond)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	val1, err := s.stakingKeeper.GetValidator(s.ctx, valaddr[1])
	require.Equal(val1.UnbondingTime, unbondingStartTime.Add(twentyOneDays))
	require.NoError(err)
	require.Equal(val1.IsUnbonding(), true)
	require.Equal(val1.IsBonded(), false)
	require.Equal(val1.IsUnbonded(), false)
	// new block
	s.ctx = s.ctx.WithBlockHeight(val1.UnbondingHeight + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(twentyOneDays))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	// validator 0 is now unbonded
	val1, err = s.stakingKeeper.GetValidator(s.ctx, valaddr[1])
	require.NoError(err)
	require.Equal(val1.IsUnbonded(), true)
}

func (s *E2ETestSuite) TestGovernanceChangesCycleList() {
	require := s.Require()

	govMsgServer := govkeeper.NewMsgServerImpl(s.govKeeper)
	require.NotNil(govMsgServer)

	//---------------------------------------------------------------------------
	// Height 0 - create bonded validators and reporters
	//---------------------------------------------------------------------------
	_, err := s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	valAccAddrs, valValAddrs, vals := s.CreateValidators(5)
	repAccAddrs := s.CreateReporters(5, valValAddrs, vals)
	proposer := repAccAddrs[0]
	initCoins := sdk.NewCoin(s.denom, math.NewInt(500*1e6))
	for _, rep := range repAccAddrs {
		s.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, rep, sdk.NewCoins(initCoins)))
	}

	govParams, err := s.govKeeper.Params.Get(s.ctx)
	require.NoError(err)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - submit proposal
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	matic, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000056D6174696300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	msgUpdateCycleList := oracletypes.MsgUpdateCyclelist{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Cyclelist: [][]byte{matic},
	}
	anyMsg, err := codectypes.NewAnyWithValue(&msgUpdateCycleList)
	proposalMsg := []*codectypes.Any{anyMsg}
	require.NoError(err)
	msgSubmitProposal := v1.MsgSubmitProposal{
		Messages:       proposalMsg,
		InitialDeposit: govParams.MinDeposit,
		Proposer:       proposer.String(),
		Metadata:       "test metadata",
		Title:          "test title",
		Summary:        "test summary",
		Expedited:      false,
	}

	proposal, err := govMsgServer.SubmitProposal(s.ctx, &msgSubmitProposal)
	fmt.Println("propRepsonse: ", proposal)
	require.NoError(err)
	require.Equal(proposal.ProposalId, uint64(1))

	proposal1, err := s.govKeeper.Proposals.Get(s.ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusVotingPeriod)
	require.Equal(proposal1.Proposer, proposer.String())
	require.Equal(proposal1.TotalDeposit, govParams.MinDeposit)
	require.Equal(proposal1.Messages, proposalMsg)
	require.Equal(proposal1.Metadata, "test metadata")
	require.Equal(proposal1.Title, "test title")
	require.Equal(proposal1.Summary, "test summary")
	require.Equal(proposal1.Expedited, false)

	_, err = s.app.EndBlocker(s.ctx) // end blocker should emit active proposal event
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - vote on proposal
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// vote from each validator
	for _, val := range valAccAddrs {
		voteResponse, err := govMsgServer.Vote(s.ctx, &v1.MsgVote{
			ProposalId: proposal.ProposalId,
			Voter:      val.String(),
			Option:     v1.VoteOption(1),
			Metadata:   "vote metadata from validator",
		})
		require.NoError(err)
		require.NotNil(voteResponse)
	}

	// check on vote in collections
	vote, err := s.govKeeper.Votes.Get(s.ctx, collections.Join(proposal.ProposalId, valAccAddrs[0]))
	require.NoError(err)
	require.Equal(vote.ProposalId, proposal.ProposalId)
	require.Equal(vote.Voter, valAccAddrs[0].String())
	require.Equal(vote.Metadata, "vote metadata from validator")

	for _, val := range valAccAddrs {
		voteResponse, err := govMsgServer.Vote(s.ctx, &v1.MsgVote{
			ProposalId: proposal.ProposalId,
			Voter:      val.String(),
			Option:     v1.VoteOption(1),
			Metadata:   "vote metadata from validator",
		})
		require.NoError(err)
		require.NotNil(voteResponse)
	}

	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(48 * time.Hour)))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - advance time to expire vote
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// proposal passed
	proposal1, err = s.govKeeper.Proposals.Get(s.ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusPassed)
	require.Equal(proposal1.Proposer, proposer.String())
	require.Equal(proposal1.TotalDeposit, govParams.MinDeposit)
	require.Equal(proposal1.Messages, proposalMsg)
	require.Equal(proposal1.Metadata, "test metadata")
	require.Equal(proposal1.Title, "test title")
	require.Equal(proposal1.Summary, "test summary")
	require.Equal(proposal1.Expedited, false)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - check cycle list
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	cycleList, err := s.oraclekeeper.GetCyclelist(s.ctx)
	require.NoError(err)
	require.Equal(cycleList, [][]byte{matic})

	// add fails

}

func (s *E2ETestSuite) TestEditingSpec() {
	require := s.Require()

	registryMsgServer := registrykeeper.NewMsgServerImpl(s.registrykeeper)
	require.NotNil(registryMsgServer)
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.oraclekeeper)
	require.NotNil(oracleMsgServer)

	//---------------------------------------------------------------------------
	// Height 0 - create 1 validators and 1 reporter
	//---------------------------------------------------------------------------
	_, err := s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	_, valValAddrs, vals := s.CreateValidators(1)
	repAccAddrs := s.CreateReporters(1, valValAddrs, vals)
	require.NotNil(repAccAddrs)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - register a spec for a Historical Price query
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	_, err = registryMsgServer.RegisterSpec(s.ctx, &registrytypes.MsgRegisterSpec{
		Registrar: repAccAddrs[0].String(),
		QueryType: "TWAP",
		Spec: registrytypes.DataSpec{
			ResponseValueType: "uint256",
			AggregationMethod: "weighted-median",
			Registrar:         repAccAddrs[0].String(),
		},
	})
	require.NoError(err)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - report for the spec
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(2)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	// queryDataString := hexutil.Encode([]byte(hexutil.Encode("TWAP", []byte(hexutil.Encode("eth", "usd")))))
	// fmt.Println("queryDataString: ", queryDataString)
	// queryData, err := utils.QueryBytesFromString(queryDataString)
	// require.NoError(err)
	// value := encodeValue(100_000)

	// resp, err := oracleMsgServer.SubmitValue(s.ctx, &oracletypes.MsgSubmitValue{
	// 	Creator:   repAccAddrs[0].String(),
	// 	QueryData: queryData,
	// 	Value:     value,
	// })
	// require.NoError(err)
	// require.NotNil(resp)

}

func (s *E2ETestSuite) TestDisputes2() {
	require := s.Require()
	msgServerOracle := oraclekeeper.NewMsgServerImpl(s.oraclekeeper)
	require.NotNil(msgServerOracle)
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.reporterkeeper)
	require.NotNil(msgServerReporter)
	msgServerDispute := disputekeeper.NewMsgServerImpl(s.disputekeeper)
	require.NotNil(msgServerDispute)

	//---------------------------------------------------------------------------
	// Height 0 - create 3 validators and 3 reporters
	//---------------------------------------------------------------------------
	_, err := s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	valsAcctAddrs, valsValAddrs, vals := s.CreateValidators(3)
	require.NotNil(valsAcctAddrs)
	repsAccs := s.CreateReporters(3, valsValAddrs, vals)
	badReporter := repsAccs[0]
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	// mapping to track reporter delegation balance
	reporterToBalanceMap := make(map[string]math.Int)
	for _, acc := range repsAccs {
		rkDelegation, err := s.reporterkeeper.Delegators.Get(s.ctx, acc)
		require.NoError(err)
		reporterToBalanceMap[acc.String()] = rkDelegation.Amount
	}

	//---------------------------------------------------------------------------
	// Height 1 - delegate 500 trb to validator 0 and bad reporter
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	pk := secp256k1.GenPrivKey()
	delAcc := s.convertToAccAddress([]secp256k1.PrivKey{*pk})
	delAccAddr := sdk.AccAddress(delAcc[0])
	initCoins := sdk.NewCoin(s.denom, math.NewInt(500*1e6))
	s.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
	s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, delAccAddr, sdk.NewCoins(initCoins)))

	// delegate to validator 0
	val, err := s.stakingKeeper.GetValidator(s.ctx, valsValAddrs[0])
	require.NoError(err)
	_, err = s.stakingKeeper.Delegate(s.ctx, delAccAddr, math.NewInt(500*1e6), stakingtypes.Unbonded, val, false)
	require.NoError(err)

	// delegate to bad reporter
	source := reportertypes.TokenOrigin{ValidatorAddress: val.OperatorAddress, Amount: math.NewInt(500 * 1e6)}
	msgDelegate := reportertypes.NewMsgDelegateReporter(delAccAddr.String(), badReporter.String(), math.NewInt(500*1e6), []*reportertypes.TokenOrigin{&source})
	_, err = msgServerReporter.DelegateReporter(s.ctx, msgDelegate)
	require.NoError(err)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	val, err = s.stakingKeeper.GetValidator(s.ctx, valsValAddrs[0])
	require.NoError(err)
	require.Equal(val.Tokens, math.NewInt(1500*1e6))
	rep, err := s.reporterkeeper.Reporters.Get(s.ctx, badReporter)
	require.NoError(err)
	require.Equal(rep.TotalTokens, math.NewInt(1500*1e6))

	//---------------------------------------------------------------------------
	// Height 2 - direct reveal for cycle list
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(2)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	disputedRep, err := s.reporterkeeper.Reporters.Get(s.ctx, repsAccs[0])
	require.NoError(err)

	// get new cycle list query data
	cycleListQuery, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	queryId := utils.QueryIDFromData(cycleListQuery)
	// create reveal message
	value := encodeValue(10_000)
	require.NoError(err)
	reveal := oracletypes.MsgSubmitValue{
		Creator:   disputedRep.Reporter,
		QueryData: cycleListQuery,
		Value:     value,
	}
	// send reveal message
	revealResponse, err := msgServerOracle.SubmitValue(s.ctx, &reveal)
	require.NoError(err)
	require.NotNil(revealResponse)
	revealTime := s.ctx.BlockTime()
	// advance time and block height to expire the query and aggregate report
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - open warning, pay from bond from reporter 1
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(3)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	disputer, err := s.reporterkeeper.Reporters.Get(s.ctx, repsAccs[1])
	require.NoError(err)

	// disputerBal := disputer.TotalTokens
	disputedBal := disputedRep.TotalTokens
	onePercent := disputedBal.Mul(math.NewInt(1)).Quo(math.NewInt(100))
	disputeFee := sdk.NewCoin(s.denom, onePercent) // warning should be 1% of bonded tokens

	// todo: is there a getter for this ?
	// get microreport for dispute
	report := oracletypes.MicroReport{
		Reporter:  disputedRep.Reporter,
		Power:     disputedRep.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:   queryId,
		Value:     value,
		Timestamp: revealTime,
	}

	// create msg for propose dispute tx
	msgProposeDispute := disputetypes.MsgProposeDispute{
		Creator:         disputer.Reporter,
		Report:          &report,
		DisputeCategory: disputetypes.Warning,
		Fee:             disputeFee,
		PayFromBond:     true,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.ctx, &msgProposeDispute)
	require.NoError(err)

	burnAmount := disputeFee.Amount.MulRaw(1).QuoRaw(20)
	disputes, err := s.disputekeeper.OpenDisputes.Get(s.ctx)
	require.NoError(err)
	require.NotNil(disputes)
	// dispute is created correctly
	dispute, err := s.disputekeeper.Disputes.Get(s.ctx, 1)
	require.NoError(err)
	require.Equal(dispute.DisputeId, uint64(1))
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeCategory, disputetypes.Warning)
	require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	require.Equal(dispute.FeePayers, []disputetypes.PayerInfo{{PayerAddress: disputer.Reporter, Amount: disputeFee.Amount, FromBond: true, BlockNumber: 3}})

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - disputed reporter reports after calling unjail
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(4)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	disputedRep, err = s.reporterkeeper.Reporters.Get(s.ctx, repsAccs[0])
	require.NoError(err)
	require.Equal(disputedRep.Jailed, true)

	// disputed reporter cant report yet
	cycleListQuery, err = s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	value = encodeValue(10_000)
	require.NoError(err)
	reveal = oracletypes.MsgSubmitValue{
		Creator:   disputedRep.Reporter,
		QueryData: cycleListQuery,
		Value:     value,
	}
	_, err = msgServerOracle.SubmitValue(s.ctx, &reveal)
	require.Error(err)

	// disputed reporter can report after calling unjail function
	msgUnjail := reportertypes.MsgUnjailReporter{
		ReporterAddress: disputedRep.Reporter,
	}
	_, err = msgServerReporter.UnjailReporter(s.ctx, &msgUnjail)
	require.NoError(err)
	disputedRep, err = s.reporterkeeper.Reporters.Get(s.ctx, repsAccs[0])
	require.NoError(err)
	require.Equal(disputedRep.Jailed, false)
	// send reveal message
	revealResponse, err = msgServerOracle.SubmitValue(s.ctx, &reveal)
	require.NoError(err)
	require.NotNil(revealResponse)
	revealTime = s.ctx.BlockTime()

	// give disputer tokens to pay for next disputes not from bond
	initCoins = sdk.NewCoin(s.denom, math.NewInt(10_000*1e6))
	require.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
	// send from module to account
	require.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, sdk.MustAccAddressFromBech32(disputer.Reporter), sdk.NewCoins(initCoins)))
	require.Equal(initCoins, s.bankKeeper.GetBalance(s.ctx, sdk.MustAccAddressFromBech32(disputer.Reporter), s.denom))

	// advance time and block height to expire the query and aggregate report
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	disputer, err = s.reporterkeeper.Reporters.Get(s.ctx, repsAccs[1])
	require.NoError(err)
	disputedRep, err = s.reporterkeeper.Reporters.Get(s.ctx, repsAccs[0])
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5 - open warning, pay from not bond from reporter 1
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(5)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	// disputerBal := disputer.TotalTokens
	disputedBal = disputedRep.TotalTokens
	onePercent = disputedBal.Mul(math.NewInt(1)).Quo(math.NewInt(100))
	disputeFee = sdk.NewCoin(s.denom, onePercent) // warning should be 1% of bonded tokens

	// get microreport for dispute
	report = oracletypes.MicroReport{
		Reporter:  disputedRep.Reporter,
		Power:     disputedRep.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:   queryId,
		Value:     value,
		Timestamp: revealTime,
	}

	// create msg for propose dispute tx
	msgProposeDispute = disputetypes.MsgProposeDispute{
		Creator:         disputer.Reporter,
		Report:          &report,
		DisputeCategory: disputetypes.Warning,
		Fee:             disputeFee,
		PayFromBond:     false,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.ctx, &msgProposeDispute)
	require.NoError(err)

	burnAmount = disputeFee.Amount.MulRaw(1).QuoRaw(20)
	disputes, err = s.disputekeeper.OpenDisputes.Get(s.ctx)
	require.NoError(err)
	require.NotNil(disputes)
	// dispute is created correctly
	dispute, err = s.disputekeeper.Disputes.Get(s.ctx, 2)
	require.NoError(err)
	require.Equal(dispute.DisputeId, uint64(2))
	require.Equal(dispute.DisputeStatus, disputetypes.Voting)
	require.Equal(dispute.DisputeCategory, disputetypes.Warning)
	require.Equal(dispute.DisputeFee, disputeFee.Amount.Sub(burnAmount))
	require.Equal(dispute.FeePayers, []disputetypes.PayerInfo{{PayerAddress: disputer.Reporter, Amount: disputeFee.Amount, FromBond: false, BlockNumber: 5}})

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 6 - dispute is resolved, direct reveal again
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(6)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	disputedRep, err = s.reporterkeeper.Reporters.Get(s.ctx, repsAccs[0])
	require.NoError(err)
	require.Equal(disputedRep.Jailed, true)

	// disputed reporter cant report yet
	cycleListQuery, err = s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	value = encodeValue(10_000)
	require.NoError(err)
	queryId = utils.QueryIDFromData(cycleListQuery)
	reveal = oracletypes.MsgSubmitValue{
		Creator:   disputedRep.Reporter,
		QueryData: cycleListQuery,
		Value:     value,
	}
	_, err = msgServerOracle.SubmitValue(s.ctx, &reveal)
	require.Error(err)

	// disputed reporter can report after calling unjail function
	msgUnjail = reportertypes.MsgUnjailReporter{
		ReporterAddress: disputedRep.Reporter,
	}
	_, err = msgServerReporter.UnjailReporter(s.ctx, &msgUnjail)
	require.NoError(err)
	disputedRep, err = s.reporterkeeper.Reporters.Get(s.ctx, repsAccs[0])
	require.NoError(err)
	require.Equal(disputedRep.Jailed, false)
	// send reveal message
	revealResponse, err = msgServerOracle.SubmitValue(s.ctx, &reveal)
	require.NoError(err)
	require.NotNil(revealResponse)
	revealTime = s.ctx.BlockTime()

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - open minor dispute, pay from bond from reporter 1
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(7)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	balBeforeDispute := disputedRep.TotalTokens
	fivePercent := balBeforeDispute.Mul(math.NewInt(5)).Quo(math.NewInt(100))
	disputeFee = sdk.NewCoin(s.denom, fivePercent)

	report = oracletypes.MicroReport{
		Reporter:  disputedRep.Reporter,
		Power:     disputedRep.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:   queryId,
		Value:     value,
		Timestamp: revealTime,
	}

	// create msg for propose dispute tx
	msgProposeDispute = disputetypes.MsgProposeDispute{
		Creator:         disputer.Reporter,
		Report:          &report,
		DisputeCategory: disputetypes.Minor,
		Fee:             disputeFee,
		PayFromBond:     true,
	}

	// send propose dispute tx
	_, err = msgServerDispute.ProposeDispute(s.ctx, &msgProposeDispute)
	require.NoError(err)
	_ = s.ctx.BlockTime()

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 8 - vote on minor dispute -- reaches quorum
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(8)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	// vote from disputer
	msgVote := disputetypes.MsgVote{
		Voter: disputer.Reporter,
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}
	voteResponse, err := msgServerDispute.Vote(s.ctx, &msgVote)
	require.NoError(err)
	require.NotNil(voteResponse)

	// vote from disputed reporter
	msgVote = disputetypes.MsgVote{
		Voter: disputedRep.Reporter,
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}
	voteResponse, err = msgServerDispute.Vote(s.ctx, &msgVote)
	require.NoError(err)
	require.NotNil(voteResponse)

	// vote from third reporter
	thirdReporter, err := s.reporterkeeper.Reporters.Get(s.ctx, repsAccs[2])
	require.NoError(err)
	msgVote = disputetypes.MsgVote{
		Voter: thirdReporter.Reporter,
		Id:    dispute.DisputeId,
		Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	}
	voteResponse, err = msgServerDispute.Vote(s.ctx, &msgVote)
	require.NoError(err)
	require.NotNil(voteResponse)

	// vote from team
	// fmt.Println(disputetypes.TeamAddress)
	// msgVote = disputetypes.MsgVote{
	// 	Voter: sdk.MustAccAddressFromBech32(disputetypes.TeamAddress).String(),
	// 	Id:    dispute.DisputeId,
	// 	Vote:  disputetypes.VoteEnum_VOTE_SUPPORT,
	// }
	// voteResponse, err = msgServerDispute.Vote(s.ctx, &msgVote)
	// require.NoError(err)
	// require.NotNil(voteResponse)

	totalTips, err := s.disputekeeper.GetTotalTips(s.ctx)
	require.NoError(err)
	fmt.Println("totalTips: ", totalTips)

	totalReporterPower, err := s.reporterkeeper.TotalReporterPower(s.ctx)
	require.NoError(err)
	fmt.Println("total reporter power: ", totalReporterPower.Quo(sdk.DefaultPowerReduction))
	reporter1Power, err := s.disputekeeper.GetReporterPower(s.ctx, disputedRep.Reporter)
	require.NoError(err)
	fmt.Println("reporter1 Power: ", reporter1Power)
	reporter2Power, err := s.disputekeeper.GetReporterPower(s.ctx, disputer.Reporter)
	require.NoError(err)
	fmt.Println("reporter2 Power: ", reporter2Power)
	reporter3Power, err := s.disputekeeper.GetReporterPower(s.ctx, thirdReporter.Reporter)
	require.NoError(err)
	fmt.Println("reporter3 Power: ", reporter3Power)

	totalFreeFloatingTokens := s.disputekeeper.GetTotalSupply(s.ctx)
	fmt.Println("total Free Floating Tokens: ", totalFreeFloatingTokens)
	owners, err := s.bankKeeper.DenomOwners(s.ctx, &banktypes.QueryDenomOwnersRequest{Denom: s.denom})
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
	tbrModuleAccount := s.accountKeeper.GetModuleAddress(minttypes.TimeBasedRewards) // yes
	fmt.Println("tbr module account: ", tbrModuleAccount.String())

	teamAccount := s.accountKeeper.GetModuleAddress(minttypes.MintToTeam) // yes
	fmt.Println("team account: ", teamAccount.String())

	disputeModuleAccount := s.accountKeeper.GetModuleAddress(disputetypes.ModuleName) // yes
	fmt.Println("dispute module account: ", disputeModuleAccount.String())

	authModuleAccount := s.accountKeeper.GetModuleAddress(authtypes.ModuleName) //
	fmt.Println("auth module account: ", authModuleAccount.String())

	reporterModuleAccount := s.accountKeeper.GetModuleAddress(reportertypes.ModuleName) // yes
	fmt.Println("reporter module account: ", reporterModuleAccount.String())

	registryModuleAccount := s.accountKeeper.GetModuleAddress(registrytypes.ModuleName) // no
	fmt.Println("registry module account: ", registryModuleAccount.String())

	reporterTipsEscrowAccount := s.accountKeeper.GetModuleAddress(reportertypes.TipsEscrowPool) // no
	fmt.Println("reporter tips escrow account: ", reporterTipsEscrowAccount.String())

	oracleModuleAccount := s.accountKeeper.GetModuleAddress(oracletypes.ModuleName) // no
	fmt.Println("oracle module account: ", oracleModuleAccount.String())

	stakingModuleAccount := s.accountKeeper.GetModuleAddress(stakingtypes.ModuleName) //
	fmt.Println("staking module account: ", stakingModuleAccount.String())

	//---------------------------------------------------------------------------
	// Height 9 - resolve dispute, direct reveal again
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(9)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	//---------------------------------------------------------------------------
	// Height 10 - open minor dispute, pay from not bond from reporter 1
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(10)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	//---------------------------------------------------------------------------
	// Height 11 - vote on minor dispute
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(11)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	//---------------------------------------------------------------------------
	// Height 12 - resolve dispute, direct reveal again
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(12)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	//---------------------------------------------------------------------------
	// Height 13 - open major dispute, pay from bond from reporter 1
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(13)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	//---------------------------------------------------------------------------
	// Height 14 - vote on major dispute
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(14)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	//---------------------------------------------------------------------------
	// Height 15 - resolve dispute, direct reveal again
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(15)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	//---------------------------------------------------------------------------
	// Height 16 - open major dispute, pay from not bond from reporter 1
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(16)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	//---------------------------------------------------------------------------
	// Height 17 - vote on major dispute
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(17)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	//---------------------------------------------------------------------------
	// Height 18 - resolve dispute, direct reveal again
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(18)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

}
