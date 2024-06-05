package integration_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
