package integration_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *IntegrationTestSuite) TestCreatingReporter() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	valAccs, valAddrs, _ := s.createValidatorAccs([]int64{1000})

	newDelegator := sample.AccAddressBytes()
	s.Setup.MintTokens(newDelegator, math.NewInt(1000*1e6))
	msgDelegate := stakingtypes.NewMsgDelegate(
		newDelegator.String(),
		valAddrs[0].String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)
	_, err := stakingMsgServer.Delegate(s.Setup.Ctx, msgDelegate)
	s.NoError(err)
	val1, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[0])
	s.NoError(err)
	// check validator reporting status
	validatorReporter, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[0])
	s.NoError(err)
	// validator reporter should have self tokens and delegator tokens as their total
	s.Equal(validatorReporter.TotalTokens, val1.Tokens)
	// delegator is not self reporting but delegated to another reporter
	_, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, newDelegator)
	s.Error(err)
	_, err = msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: newDelegator.String(), Commission: keeper.DefaultCommission()})
	s.NoError(err)

	rep, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, newDelegator)
	s.NoError(err)
	delBonded, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, newDelegator)
	s.NoError(err)
	s.Equal(rep.TotalTokens, delBonded)

	// check validator reporting tokens after delegator has moved
	val2, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[0])
	s.NoError(err)
	// staked tokens should be same as before
	s.Equal(val1.Tokens, val2.Tokens)
	validatorReporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[0])
	s.NoError(err)
	// reporting tokens should be less than before
	s.True(validatorReporter.TotalTokens.LT(val1.Tokens))
	s.True(validatorReporter.TotalTokens.Equal(val1.Tokens.Sub(delBonded)))
}

func (s *IntegrationTestSuite) TestChangeReporterMsg() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	valAccs, valAddrs, _ := s.createValidatorAccs([]int64{100, 200})

	newDelegator := sample.AccAddressBytes()
	s.Setup.MintTokens(newDelegator, math.NewInt(1000*1e6))
	msgDelegate := stakingtypes.NewMsgDelegate(
		newDelegator.String(),
		valAddrs[0].String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)

	_, err := stakingMsgServer.Delegate(s.Setup.Ctx, msgDelegate)
	s.NoError(err)
	val1, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[0])
	s.NoError(err)
	// check validator reporting status
	validatorReporter1, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[0])
	s.NoError(err)
	// validator reporter should have self tokens and delegator tokens as their total
	s.Equal(validatorReporter1.TotalTokens, val1.Tokens)

	// check second reporter tokens
	val2, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[1])
	s.NoError(err)
	validatorReporter2, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[1])
	s.NoError(err)
	// validator reporter should have self tokens and delegator tokens as their total
	s.Equal(validatorReporter2.TotalTokens, val2.Tokens)
	// valrep1 should have more tokens than valrep2
	s.True(validatorReporter1.TotalTokens.GT(validatorReporter2.TotalTokens))

	// change reporter
	_, err = msgServer.ChangeReporter(s.Setup.Ctx, &reportertypes.MsgChangeReporter{DelegatorAddress: newDelegator.String(), ReporterAddress: valAccs[1].String()})
	s.NoError(err)

	// check validator reporting tokens after delegator has moved
	validatorReporter1, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[0])
	s.NoError(err)
	validatorReporter2, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[1])
	s.NoError(err)
	// reporting tokens should be less than before
	s.True(validatorReporter1.TotalTokens.LT(val1.Tokens))
	s.True(validatorReporter2.TotalTokens.GT(val2.Tokens))
	// valrep2 should have more tokens than valrep1
	s.True(validatorReporter2.TotalTokens.GT(validatorReporter1.TotalTokens))
}

func (s *IntegrationTestSuite) TestAddAmountToStake() {
	s.Setup.CreateValidators(5)

	addr := sample.AccAddressBytes()
	delbefore, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, addr)
	s.NoError(err)
	s.True(delbefore.IsZero())
	delAmount := math.NewInt(1000)
	s.NoError(s.Setup.Reporterkeeper.AddAmountToStake(s.Setup.Ctx, addr, delAmount))
	delAfter, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, addr)
	s.NoError(err)
	s.True(delAfter.Equal(delAmount))
}

func (s *IntegrationTestSuite) TestGetBondedValidators() {
	s.Setup.CreateValidators(5)
	testCases := []struct {
		name        string
		num         uint32
		expectedlen int
	}{
		{
			name:        "one bonded validator",
			num:         1,
			expectedlen: 1,
		},
		{
			name:        "two bonded validators",
			num:         2,
			expectedlen: 2,
		},
		{
			name:        "five bonded validators",
			num:         5,
			expectedlen: 5,
		},
		{
			name:        "ten bonded validators",
			num:         10,
			expectedlen: 5 + 1, // 1 for genesis validator
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			vals, err := s.Setup.Reporterkeeper.GetBondedValidators(s.Setup.Ctx, tc.num)
			s.NoError(err)
			s.Equal(tc.expectedlen, len(vals))
		})
	}
}

// see if delegators when they stake the reporter tokens increase
func (s *IntegrationTestSuite) TestAddReporterTokens() {
	valAccs, valAddrs, _ := s.Setup.CreateValidators(1)
	stakingmsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	valAcc := valAccs[0]
	valAdd := valAddrs[0]
	testCases := []struct {
		name      string
		delegator sdk.AccAddress
	}{
		{
			name:      "one",
			delegator: sample.AccAddressBytes(),
		},
		{
			name:      "two",
			delegator: sample.AccAddressBytes(),
		},
		{
			name:      "three",
			delegator: sample.AccAddressBytes(),
		},
	}
	amt := math.NewInt(1000 * 1e6)
	repBefore, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAcc)
	s.NoError(err)
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.Setup.MintTokens(tc.delegator, amt)
			msgDelegate := stakingtypes.NewMsgDelegate(
				tc.delegator.String(),
				valAdd.String(),
				sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
			)
			_, err := stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate)
			s.NoError(err)
			repBefore.TotalTokens = repBefore.TotalTokens.Add(amt)
			rep, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAcc)
			s.NoError(err)
			s.Equal(rep.TotalTokens, repBefore.TotalTokens)
		})
	}
}

// one delegator stakes with multiple validators, check the delegation count
func (s *IntegrationTestSuite) TestDelegatorCount() {
	_, valAddrs, _ := s.Setup.CreateValidators(5)
	stakingmsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)

	delegatorAddr := sample.AccAddressBytes()
	s.Setup.MintTokens(delegatorAddr, math.NewInt(5000*1e6))

	for _, val := range valAddrs {
		msgDelegate := stakingtypes.NewMsgDelegate(
			delegatorAddr.String(),
			val.String(),
			sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
		)
		_, err := stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate)
		s.NoError(err)
	}
	del, err := s.Setup.Reporterkeeper.Delegators.Get(s.Setup.Ctx, delegatorAddr.Bytes())
	s.NoError(err)
	s.Equal(uint64(5), del.DelegationCount)
}

// add 100 delegators to a reporter and check if the delegator count is 100
// and what happens when the 101st delegator tries to delegate
func (s *IntegrationTestSuite) TestMaxDelegatorCount() {
	valAccs, valAddrs, _ := s.Setup.CreateValidator(1)
	stakingmsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	valAcc := valAccs[0]
	valAdd := valAddrs[0]
	k1 := valAcc
	var k2 sdk.AccAddress
	// delegate a 100 delegators
	for i := 0; i < 100; i++ {
		delegatorAddr := sample.AccAddressBytes()
		s.Setup.MintTokens(delegatorAddr, math.NewInt(1000*1e6))
		msgDelegate := stakingtypes.NewMsgDelegate(
			delegatorAddr.String(),
			valAdd.String(),
			sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
		)
		if i == 99 {
			k2 = delegatorAddr
		}
		_, err := stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate)
		s.NoError(err)
	}

	// check delegator count
	rep, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAcc)
	s.NoError(err)
	s.Equal(uint64(100), rep.DelegatorsCount)
	// check how many reporters are there
	iter, err := s.Setup.Reporterkeeper.Reporters.Iterate(s.Setup.Ctx, nil)
	s.NoError(err)
	list, err := iter.Keys()
	s.NoError(err)
	s.Equal(2, len(list))

	rep1, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, k1)
	s.NoError(err)
	s.Equal(uint64(100), rep1.DelegatorsCount)
	rep2, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, k2)
	s.NoError(err)
	s.Equal(uint64(1), rep2.DelegatorsCount)

	// delegate 102nd delegator
	delegatorAddr := sample.AccAddressBytes()
	s.Setup.MintTokens(delegatorAddr, math.NewInt(1000*1e6))
	msgDelegate := stakingtypes.NewMsgDelegate(
		delegatorAddr.String(),
		valAdd.String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)
	_, err = stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate)
	s.NoError(err)

	// check delegator count
	iter, err = s.Setup.Reporterkeeper.Reporters.Iterate(s.Setup.Ctx, nil)
	s.NoError(err)
	list, err = iter.Keys()
	s.NoError(err)
	s.Equal(3, len(list))
	// delegator should be come a third reporter
	rep3, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, delegatorAddr)
	s.NoError(err)
	s.Equal(uint64(1), rep3.DelegatorsCount)
}

func (s *IntegrationTestSuite) TestIncorrectReportsUpdated() {
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	valAccs, valAddrs, _ := s.createValidatorAccs([]int64{1000, 1000})

	newDelegator1 := sample.AccAddressBytes()
	newDelegator2 := sample.AccAddressBytes()
	s.Setup.MintTokens(newDelegator1, math.NewInt(10000*1e6))
	s.Setup.MintTokens(newDelegator2, math.NewInt(10000*1e6))
	msgDelegateFromDel1toVal1 := stakingtypes.NewMsgDelegate(
		newDelegator1.String(),
		valAddrs[0].String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)

	msgDelegateFromDel1toVal2 := stakingtypes.NewMsgDelegate(
		newDelegator1.String(),
		valAddrs[1].String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)

	_, err := stakingMsgServer.Delegate(s.Setup.Ctx, msgDelegateFromDel1toVal1)
	s.NoError(err)
	_, err1 := stakingMsgServer.Delegate(s.Setup.Ctx, msgDelegateFromDel1toVal2)
	s.NoError(err1)

	// Test
	rep, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[0])
	println(rep.DelegatorsCount)
	s.NoError(err)

	rep, err3 := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[1])
	println(rep.DelegatorsCount)
	s.NoError(err3)
}

func (s *IntegrationTestSuite) TestTwoValidators() {
	valAccs, valAddrs, _ := s.Setup.CreateValidators(2)
	stakingmsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)

	valAddr1 := valAddrs[0]
	valAddr2 := valAddrs[1]

	for i := 0; i < 2; i++ {
		delegatorAddr := sample.AccAddressBytes()
		s.Setup.MintTokens(delegatorAddr, math.NewInt(1000*1e6))
		msgDelegate := stakingtypes.NewMsgDelegate(
			delegatorAddr.String(),
			valAddr1.String(),
			sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
		)

		_, err := stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate)
		s.NoError(err)
	}

	for i := 0; i < 100; i++ {
		delegatorAddr := sample.AccAddressBytes()
		s.Setup.MintTokens(delegatorAddr, math.NewInt(1000*1e6))
		msgDelegate := stakingtypes.NewMsgDelegate(
			delegatorAddr.String(),
			valAddr2.String(),
			sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
		)
		_, err := stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate)
		s.NoError(err)
	}
	// check delegator count
	rep1, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[0])
	s.NoError(err)
	s.Equal(uint64(3), rep1.DelegatorsCount) // should be 3 delegators plus the self delegation
	del1, err := s.Setup.Reporterkeeper.Delegators.Get(s.Setup.Ctx, valAccs[0])
	s.Equal(uint64(1), del1.DelegationCount)
	s.NoError(err)

	rep2, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[1])
	s.NoError(err)
	s.Equal(uint64(100), rep2.DelegatorsCount) // should be 101 delegators plus the self delegation
	s.Setup.MintTokens(valAccs[0], math.NewInt(1000*1e6))
	msg := stakingtypes.MsgDelegate{
		DelegatorAddress: valAccs[0].String(),
		ValidatorAddress: valAddr2.String(),
		Amount:           sdk.NewCoin(s.Setup.Denom, math.NewInt(1000*1e6)),
	}
	_, err = stakingmsgServer.Delegate(s.Setup.Ctx, &msg)
	s.NoError(err)

	// check delegator count, should be the same as before
	rep1, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[0])
	s.NoError(err)

	del1, err = s.Setup.Reporterkeeper.Delegators.Get(s.Setup.Ctx, valAccs[0])
	s.NoError(err)

	rep2, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[1])
	s.NoError(err)
	println(rep2.DelegatorsCount)
}

func (s *IntegrationTestSuite) TestMaxDelegatorCountBug() {
	valAccs, valAddrs, _ := s.Setup.CreateValidator(2)
	stakingmsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	valAcc1 := valAccs[1]
	valAdd1 := valAddrs[1]

	for i := 0; i < 2; i++ {
		delegatorAddr := sample.AccAddressBytes()
		s.Setup.MintTokens(delegatorAddr, math.NewInt(1000*1e6))
		msgDelegate := stakingtypes.NewMsgDelegate(
			delegatorAddr.String(),
			valAddrs[0].String(),
			sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
		)
		_, err := stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate)
		s.NoError(err)
	}

	// delegate 99 delegators to val1
	for i := 0; i < 99; i++ {
		delegatorAddr := sample.AccAddressBytes()
		s.Setup.MintTokens(delegatorAddr, math.NewInt(1000*1e6))
		msgDelegate := stakingtypes.NewMsgDelegate(
			delegatorAddr.String(),
			valAdd1.String(),
			sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
		)
		_, err := stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate)
		s.NoError(err)
	}

	// check delegator count
	rep, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAcc1)
	s.NoError(err)
	s.Equal(uint64(100), rep.DelegatorsCount)

	rep1, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[0])
	s.Equal(rep1.DelegatorsCount, uint64(3))
	s.NoError(err)

	// undelegate val 1 to val 1
	_, bz, err := bech32.DecodeAndConvert(valAddrs[0].String())
	s.NoError(err)
	valAddress_to_accAddress := sdk.AccAddress(bz)

	undel := stakingtypes.NewMsgUndelegate(
		valAddress_to_accAddress.String(),
		valAddrs[0].String(),
		sdk.NewInt64Coin(s.Setup.Denom, 100),
	)
	// get the delegator
	delegator, err := s.Setup.Reporterkeeper.Delegators.Get(s.Setup.Ctx, valAccs[0])
	s.NoError(err)
	s.Equal(uint64(1), delegator.DelegationCount)
	_, err2 := stakingmsgServer.Undelegate(s.Setup.Ctx, undel)
	s.NoError(err2)
	// get the delegator
	delegator, err = s.Setup.Reporterkeeper.Delegators.Get(s.Setup.Ctx, valAccs[0])
	s.NoError(err)
	s.Equal(uint64(0), delegator.DelegationCount)
	rep4, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[0])
	s.Equal(rep4.DelegatorsCount, uint64(3))
	s.NoError(err)

	// delegate from val 1 to val 2

	msgDelegate1 := stakingtypes.NewMsgDelegate(
		valAddress_to_accAddress.String(),
		valAdd1.String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)
	_, err1 := stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate1)
	s.NoError(err1)
	delegator, err = s.Setup.Reporterkeeper.Delegators.Get(s.Setup.Ctx, valAccs[0])
	s.NoError(err)
	s.Equal(uint64(1), delegator.DelegationCount)

	rep2, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[0])
	s.Equal(rep2.DelegatorsCount, uint64(3))
	s.NoError(err)
}
