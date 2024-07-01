package e2e_test

import (
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/utils"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	collections "cosmossdk.io/collections"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *E2ETestSuite) TestNoInitialMint() {
	require := s.Require()

	mintToTeamAcc := s.Setup.Accountkeeper.GetModuleAddress(minttypes.MintToTeam)
	require.NotNil(mintToTeamAcc)
	balance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, mintToTeamAcc, s.Setup.Denom)
	require.Equal(balance.Amount, sdk.NewCoin(s.Setup.Denom, math.NewInt(0)).Amount)
}

// func (s *E2ETestSuite) TestTransferAfterMint() {
// 	require := s.Setup.Require()

// 	mintToTeamAcc := s.Setup.Accountkeeper.GetModuleAddress(minttypes.MintToTeam)
// 	require.NotNil(mintToTeamAcc)
// 	balance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, mintToTeamAcc, s.Setup.Denom)
// 	require.Equal(balance.Amount, math.NewInt(300*1e6))

// 	// create 5 accounts
// 	type Accounts struct {
// 		PrivateKey secp256k1.PrivKey
// 		Account    sdk.AccAddress
// 	}
// 	accounts := make([]Accounts, 0, 5)
// 	for i := 0; i < 5; i++ {
// 		privKey := secp256k1.GenPrivKey()
// 		accountAddress := sdk.AccAddress(privKey.PubKey().Address())
// 		account := authtypes.BaseAccount{
// 			Address:       accountAddress.String(),
// 			PubKey:        codectypes.UnsafePackAny(privKey.PubKey()),
// 			AccountNumber: uint64(i + 1),
// 		}
// 		existingAccount := s.Setup.Accountkeeper.GetAccount(s.Setup.Ctx, accountAddress)
// 		if existingAccount == nil {
// 			s.Setup.Accountkeeper.SetAccount(s.Setup.Ctx, &account)
// 			accounts = append(accounts, Accounts{
// 				PrivateKey: *privKey,
// 				Account:    accountAddress,
// 			})
// 		}
// 	}

// 	// transfer 1000 tokens from team to all 5 accounts
// 	for _, acc := range accounts {
// 		startBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, acc.Account, s.Setup.Denom).Amount
// 		err := s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, minttypes.MintToTeam, acc.Account, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, math.NewInt(1000))))
// 		require.NoError(err)
// 		require.Equal(startBalance.Add(math.NewInt(1000)), s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, acc.Account, s.Setup.Denom).Amount)
// 	}
// 	expectedTeamBalance := math.NewInt(300*1e6 - 1000*5)
// 	require.Equal(expectedTeamBalance, s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, mintToTeamAcc, s.Setup.Denom).Amount)

// 	// transfer from account 0 to account 1
// 	s.Setup.Bankkeeper.SendCoins(s.Setup.Ctx, accounts[0].Account, accounts[1].Account, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, math.NewInt(1000))))
// 	require.Equal(math.NewInt(0), s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, accounts[0].Account, s.Setup.Denom).Amount)
// 	require.Equal(math.NewInt(2000), s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, accounts[1].Account, s.Setup.Denom).Amount)

// 	// transfer from account 2 to team
// 	s.Setup.Bankkeeper.SendCoinsFromAccountToModule(s.Setup.Ctx, accounts[2].Account, minttypes.MintToTeam, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, math.NewInt(1000))))
// 	require.Equal(math.NewInt(0), s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, accounts[2].Account, s.Setup.Denom).Amount)
// 	require.Equal(expectedTeamBalance.Add(math.NewInt(1000)), s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, mintToTeamAcc, s.Setup.Denom).Amount)

// 	// try to transfer more than balance from account 3 to 4
// 	err := s.Setup.Bankkeeper.SendCoins(s.Setup.Ctx, accounts[3].Account, accounts[4].Account, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, math.NewInt(1001))))
// 	require.Error(err)
// 	require.Equal(s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, accounts[3].Account, s.Setup.Denom).Amount, math.NewInt(1000))
// 	require.Equal(s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, accounts[4].Account, s.Setup.Denom).Amount, math.NewInt(1000))
// }

func (s *E2ETestSuite) TestSetUpValidatorAndReporter() {
	require := s.Require()

	// Create Validator Accounts
	numValidators := 10
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	_ = new(big.Int).Mul(big.NewInt(1000), base)

	// make addresses
	testAddresses := simtestutil.CreateIncrementalAccounts(numValidators)
	// mint 50k tokens to minter account and send to each address
	initCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(10000*1e6))
	for _, addr := range testAddresses {
		s.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, addr, sdk.NewCoins(initCoins)))
	}
	// get val address for each test address
	valAddresses := simtestutil.ConvertAddrsToValAddrs(testAddresses)
	// create pub keys for each address
	pubKeys := simtestutil.CreateTestPubKeys(numValidators)
	stakingserver := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	// set each account with proper keepers
	for i, pubKey := range pubKeys {
		s.Setup.Accountkeeper.NewAccountWithAddress(s.Setup.Ctx, testAddresses[i])
		valMsg, err := stakingtypes.NewMsgCreateValidator(
			valAddresses[i].String(),
			pubKey,
			sdk.NewInt64Coin(s.Setup.Denom, 5000*1e6),
			stakingtypes.Description{Moniker: strconv.Itoa(i)},
			stakingtypes.CommissionRates{
				Rate:          math.LegacyNewDecWithPrec(5, 1),
				MaxRate:       math.LegacyNewDecWithPrec(5, 1),
				MaxChangeRate: math.LegacyNewDec(0),
			},
			math.OneInt())
		s.NoError(err)
		_, err = stakingserver.CreateValidator(s.Setup.Ctx, valMsg)
		s.NoError(err)
		val, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddresses[i])
		s.NoError(err)

		randomStakeAmount := rand.Intn(5000-1000+1) + 1000
		require.True(randomStakeAmount >= 1000 && randomStakeAmount <= 5000, "randomStakeAmount is not within the expected range")
		msg := stakingtypes.MsgDelegate{DelegatorAddress: testAddresses[i].String(), ValidatorAddress: val.OperatorAddress, Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(5000*1e6))}
		_, err = stakingserver.Delegate(s.Setup.Ctx, &msg)
		s.NoError(err)
	}

	_, err := s.Setup.Stakingkeeper.EndBlocker(s.Setup.Ctx)
	s.NoError(err)

	// check that everyone is a bonded validator
	validatorSet, err := s.Setup.Stakingkeeper.GetAllValidators(s.Setup.Ctx)
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
		s.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, delegatorAccounts[i], sdk.NewCoins(initCoins)))
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
		_, err := stakingserver.Delegate(s.Setup.Ctx, &stakingtypes.MsgDelegate{DelegatorAddress: del.delegatorAddress.String(), ValidatorAddress: del.validator.GetOperator(), Amount: sdk.NewCoin(s.Setup.Denom, del.tokenAmount)})
		require.NoError(err)
	}
	_, err = s.Setup.Stakingkeeper.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	// set up reporter module msgServer
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	require.NotNil(msgServerReporter)

	// check that reporter was created correctly
	valAddr, err := sdk.ValAddressFromBech32(delegators[reporter].validator.GetOperator())
	require.NoError(err)
	oracleReporter, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAddr)
	require.NoError(err)
	val, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)
	require.NoError(err)

	require.Equal(oracleReporter.TotalTokens, val.Tokens)
	require.Equal(oracleReporter.Jailed, false)
	delegationReporter, err := s.Setup.Reporterkeeper.Delegators.Get(s.Setup.Ctx, delegators[delegatorI].delegatorAddress)
	require.NoError(err)
	require.Equal(delegationReporter.Reporter, valAddr.Bytes())
}

func (s *E2ETestSuite) TestUnstaking() {
	require := s.Require()
	// create 5 validators with 5_000 TRB
	accaddr, valaddr, _ := s.Setup.CreateValidators(5)
	_, err := s.Setup.Stakingkeeper.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
	// all validators are bonded
	validators, err := s.Setup.Stakingkeeper.GetAllValidators(s.Setup.Ctx)
	require.NoError(err)
	require.NotNil(validators)
	for _, val := range validators {
		require.Equal(val.Status.String(), stakingtypes.BondStatusBonded)
	}

	// begin unbonding validator 0
	del, err := s.Setup.Stakingkeeper.GetDelegation(s.Setup.Ctx, accaddr[1], valaddr[1])
	require.NoError(err)
	// undelegate all shares except for 1 to avoid getting the validator deleted
	timeToUnbond, _, err := s.Setup.Stakingkeeper.Undelegate(s.Setup.Ctx, accaddr[1], valaddr[1], del.Shares.Sub(math.LegacyNewDec(1)))
	require.NoError(err)

	// unbonding time is 21 days after calling BeginUnbondingValidator
	unbondingStartTime := s.Setup.Ctx.BlockTime()
	twentyOneDays := time.Hour * 24 * 21
	require.Equal(unbondingStartTime.Add(twentyOneDays), timeToUnbond)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	val1, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valaddr[1])
	require.Equal(val1.UnbondingTime, unbondingStartTime.Add(twentyOneDays))
	require.NoError(err)
	require.Equal(val1.IsUnbonding(), true)
	require.Equal(val1.IsBonded(), false)
	require.Equal(val1.IsUnbonded(), false)
	// new block
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(val1.UnbondingHeight + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(twentyOneDays))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	// validator 0 is now unbonded
	val1, err = s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valaddr[1])
	require.NoError(err)
	require.Equal(val1.IsUnbonded(), true)
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

	badReporter := repsAccs[0]
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	// mapping to track reporter delegation balance
	reporterToBalanceMap := make(map[string]math.Int)
	for _, acc := range repsAccs {
		rkDelegation, err := s.Setup.Reporterkeeper.Delegators.Get(s.Setup.Ctx, acc)
		require.NoError(err)
		reporterToBalanceMap[acc.String()] = rkDelegation.Amount
	}

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

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	val, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valsValAddrs[0])
	require.NoError(err)
	rep, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, badReporter)
	require.NoError(err)
	require.Equal(rep.TotalTokens, val.Tokens)

	//---------------------------------------------------------------------------
	// Height 2 - direct reveal for cycle list
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(2)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	disputedRep, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, repsAccs[0])
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
		Power:       disputedRep.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:     queryId,
		Value:       value,
		Timestamp:   revealTime,
		BlockNumber: reportBlock,
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

	disputedRep, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, repsAccs[0])
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

	report.Power = disputedRep.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64()
	fee, err = s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, disputetypes.Warning)
	require.NoError(err)
	disputeFee = sdk.NewCoin(s.Setup.Denom, fee) // warning should be 1% of bonded tokens

	// get microreport for dispute
	report = oracletypes.MicroReport{
		Reporter:    repsAccs[0].String(),
		Power:       disputedRep.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:     queryId,
		Value:       value,
		Timestamp:   revealTime,
		BlockNumber: revealBlock,
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

	balBeforeDispute := disputedRep.TotalTokens
	fivePercent := balBeforeDispute.Mul(math.NewInt(5)).Quo(math.NewInt(100))
	disputeFee = sdk.NewCoin(s.Setup.Denom, fivePercent)

	report = oracletypes.MicroReport{
		Reporter:    repsAccs[0].String(),
		Power:       disputedRep.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryId:     queryId,
		Value:       value,
		Timestamp:   revealTime,
		BlockNumber: revealBlock,
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
	reporter1Power, err := s.Setup.Disputekeeper.ReportersGroup.Get(s.Setup.Ctx, collections.Join(dispute.DisputeId, repsAccs[0].Bytes()))
	require.NoError(err)
	fmt.Println("reporter1 Power: ", reporter1Power)
	reporter2Power, err := s.Setup.Disputekeeper.ReportersGroup.Get(s.Setup.Ctx, collections.Join(dispute.DisputeId, repsAccs[1].Bytes()))
	require.NoError(err)
	fmt.Println("reporter2 Power: ", reporter2Power)
	reporter3Power, err := s.Setup.Disputekeeper.ReportersGroup.Get(s.Setup.Ctx, collections.Join(dispute.DisputeId, repsAccs[2].Bytes()))
	require.NoError(err)
	fmt.Println("reporter3 Power: ", reporter3Power)

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

	teamAccount := s.Setup.Accountkeeper.GetModuleAddress(minttypes.MintToTeam) // yes
	fmt.Println("team account: ", teamAccount.String())

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
