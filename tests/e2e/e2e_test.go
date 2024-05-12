package e2e_test

import (
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
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/utils"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
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

	// height 0
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

	// height 1
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

	// height 2
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

	// loop through more times
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
		Creator:   reporter.Reporter,
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
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// check that 1 second worth of tbr has been minted
	// expected tbr = (daily mint rate * time elapsed) / (# of ms in a day)
	expectedBlockProvision := int64(146940000 * (1 * time.Second) / (24 * 60 * 60 * 1000))
	expectedTbr := sdk.NewCoin(s.denom, math.NewInt((expectedBlockProvision)).Quo(sdk.DefaultPowerReduction))
	tbrModuleAccountBalance = s.bankKeeper.GetBalance(s.ctx, tbrModuleAccount, sdk.DefaultBondDenom)
	require.Equal(expectedTbr, tbrModuleAccountBalance)
	// check that the cycle list has rotated
	cycleListBtc, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NotEqual(cycleListEth, cycleListBtc)
	require.NoError(err)

	// create reveal msg
	require.NoError(err)
	reveal1 := oracletypes.MsgSubmitValue{
		Creator:   reporter.Reporter,
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
		QueryId: queryIdEth,
	}
	result1, err := s.oraclekeeper.GetAggregatedReport(s.ctx, &getAggReportRequest1)
	require.NoError(err)
	require.Equal(result1.Report.Height, int64(2))
	require.Equal(result1.Report.AggregateReportIndex, int64(0))
	require.Equal(result1.Report.AggregateValue, encodeValue(4500))
	require.Equal(result1.Report.AggregateReporter, reporter.Reporter)
	require.Equal(result1.Report.QueryId, queryIdEth)
	require.Equal(int64(4000000000), result1.Report.ReporterPower)
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
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// check that 8 sec of tbr has been minted
	tbrModuleAccountBalance = s.bankKeeper.GetBalance(s.ctx, tbrModuleAccount, sdk.DefaultBondDenom)
	expectedBlockProvision = int64(146940000 * (8 * time.Second) / (24 * 60 * 60 * 1000))
	expectedTbr = sdk.NewCoin(s.denom, math.NewInt((expectedBlockProvision)).Quo(sdk.DefaultPowerReduction))
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
		Creator:   reporter.Reporter,
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
		QueryId: queryIdTrb,
	}
	// check that aggregated report is stored correctly
	result2, err := s.oraclekeeper.GetAggregatedReport(s.ctx, &getAggReportRequest2)
	require.NoError(err)
	require.Equal(int64(0), result2.Report.AggregateReportIndex)
	require.Equal(encodeValue(100_000), result2.Report.AggregateValue)
	require.Equal(reporter.Reporter, result2.Report.AggregateReporter)
	require.Equal(queryIdTrb, result2.Report.QueryId)
	require.Equal(int64(4000000000), result2.Report.ReporterPower)
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
		Tipper:    reporter.Reporter,
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
		Creator:   reporter.Reporter,
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
		Creator:   reporter.Reporter,
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
		QueryId: queryIdEth,
	}
	// check that the aggregated report is stored correctly
	result1, err = s.oraclekeeper.GetAggregatedReport(s.ctx, &getAggReportRequest1)
	require.NoError(err)
	require.Equal(result1.Report.AggregateReportIndex, int64(0))
	require.Equal(result1.Report.AggregateValue, encodeValue(5000))
	require.Equal(result1.Report.AggregateReporter, reporter.Reporter)
	require.Equal(queryIdEth, result1.Report.QueryId)
	require.Equal(int64(4000000000), result1.Report.ReporterPower)
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
		Tipper:    reporter.Reporter,
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
		Creator:   reporter.Reporter,
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
		QueryId: queryIdTrb,
	}
	// query aggregated report
	resultTrb, err := s.oraclekeeper.GetAggregatedReport(s.ctx, &getAggReportRequestTrb)
	require.NoError(err)
	require.Equal(resultTrb.Report.AggregateReportIndex, int64(0))
	require.Equal(resultTrb.Report.AggregateValue, encodeValue(1_000_000))
	require.Equal(resultTrb.Report.AggregateReporter, reporter.Reporter)
	require.Equal(queryIdTrb, resultTrb.Report.QueryId)
	require.Equal(int64(4000000000), resultTrb.Report.ReporterPower)
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

	// define createReporterMsg params
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

	// votingPower := reporter.TotalTokens.Quo(layertypes.PowerReduction).Int64()
	// fmt.Println("voting power: ", votingPower)

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
	report := types.MicroReport{
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
	err = s.disputekeeper.Tallyvote(s.ctx, 1)
	require.Error(err, "vote period not ended and quorum not reached")
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// reporter is in jail
	reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	require.NoError(err)
	require.Equal(reporter.Jailed, true)
	// reporter lost 1% of their free floating tokens
	freeFloatingBalanceAfter := s.bankKeeper.GetBalance(s.ctx, reporterAccount, s.denom)
	require.Equal(freeFloatingBalanceAfter, freeFloatingBalanceBefore.Sub(disputeFee))
	// disputeModAcct := s.accountKeeper.GetModuleAddress(disputetypes.ModuleName)
	// disputeModAcctBal := s.bankKeeper.GetBalance(s.ctx, disputeModAcct, s.denom)
	// fmt.Println("dispute Module Acct Bal: ", disputeModAcctBal)
	// require.Equal(disputeModAcctBal.Amount, disputeFee.Amount)

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
	err = s.disputekeeper.Tallyvote(s.ctx, 1)
	require.Error(err, "vote period not ended and quorum not reached")
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

	report = types.MicroReport{
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
	err = s.disputekeeper.Tallyvote(s.ctx, 1)
	require.Error(err, "vote period not ended and quorum not reached")
	err = s.disputekeeper.Tallyvote(s.ctx, 2)
	require.Error(err, "vote period not ended and quorum not reached")
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

	err = s.disputekeeper.Tallyvote(s.ctx, 1)
	require.NoError(err)
	err = s.disputekeeper.Tallyvote(s.ctx, 2)
	require.NoError(err)
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
	// fmt.Println("disputes: ", disputes)

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

	report = types.MicroReport{
		Reporter:    reporter.Reporter,
		Power:       reporter.TotalTokens.Int64(),
		QueryId:     queryId,
		Value:       value,
		Timestamp:   s.ctx.BlockTime(),
		BlockNumber: s.ctx.BlockHeight(),
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

	err = s.disputekeeper.Tallyvote(s.ctx, 3)
	require.Error(err, "vote period not ended and quorum not reached")
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
	// fmt.Println(reporter)
	// reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// require.NoError(err)
	// fmt.Println("reporter.TotalTokens during dispute: ", reporter.TotalTokens)
	// require.NoError(err)
	// freeFloatingBalanceBefore = s.bankKeeper.GetBalance(s.ctx, reporterAccount, s.denom)
	// fmt.Println("freeFloatingBalance during dispute: ", freeFloatingBalanceBefore)

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

	err = s.disputekeeper.Tallyvote(s.ctx, 3)
	require.NoError(err)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	// reporter, err = s.reporterkeeper.Reporters.Get(s.ctx, reporterAccount)
	// require.NoError(err)

}

func (s *E2ETestSuite) TestUnstaking() {

}
