package e2e_test

import (
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
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/utils"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	// oracleutils "github.com/tellor-io/layer/x/oracle/utils"

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
