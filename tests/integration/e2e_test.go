package integration_test

import (
	"math/big"
	"math/rand"
	"strconv"
	"time"

	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *IntegrationTestSuite) TestSetUpValidatorAndReporter() {
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
	valAddr1, err := sdk.ValAddressFromBech32(validatorSet[1].GetOperator())
	require.NoError(err)
	valAddr2, err := sdk.ValAddressFromBech32(validatorSet[2].GetOperator())
	require.NoError(err)
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, valAddr1, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker1")))
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, valAddr2, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker2")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, valAddr1, reportertypes.NewSelection(valAddr1, 1)))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, valAddr2, reportertypes.NewSelection(valAddr2, 1)))

	reporterserver := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	// delegate to validators
	for _, del := range delegators {
		_, err := stakingserver.Delegate(s.Setup.Ctx, &stakingtypes.MsgDelegate{DelegatorAddress: del.delegatorAddress.String(), ValidatorAddress: del.validator.GetOperator(), Amount: sdk.NewCoin(s.Setup.Denom, del.tokenAmount)})
		require.NoError(err)
		valAddrs, err := sdk.ValAddressFromBech32(del.validator.GetOperator())
		require.NoError(err)
		_, err = reporterserver.SelectReporter(s.Setup.Ctx, &reportertypes.MsgSelectReporter{ReporterAddress: sdk.AccAddress(valAddrs).String(), SelectorAddress: del.delegatorAddress.String()})
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
	oracleReporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAddr.Bytes(), []byte{})
	require.NoError(err)
	require.Equal(oracleReporterStake, val.Tokens)
	require.Equal(oracleReporter.Jailed, false)
	delegationReporter, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, delegators[delegatorI].delegatorAddress)
	require.NoError(err)
	require.Equal(delegationReporter.Reporter, valAddr.Bytes())
}

func (s *IntegrationTestSuite) TestUnstaking() {
	require := s.Require()
	// create 5 validators with 5_000 TRB
	accaddr, valaddr, _ := s.Setup.CreateValidators(5)
	for _, val := range valaddr {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
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
