package integration_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"
)

const (
	reporter     = "reporter"
	delegatorI   = "delegator1"
	delegatorII  = "delegator2"
	delegatorIII = "delegator3"
	delegatorIV  = "delegator4"
)

type Delegator struct {
	delegatorAddress sdk.AccAddress
	validator        stakingtypes.Validator
	tokenAmount      math.Int
}

func (s *IntegrationTestSuite) createdelegators(delegators map[string]Delegator) map[string]Delegator {
	for _, del := range delegators {
		_, err := s.stakingKeeper.Delegate(s.ctx, del.delegatorAddress, del.tokenAmount, stakingtypes.Unbonded, del.validator, true)
		s.NoError(err)
	}
	return delegators
}

func (s *IntegrationTestSuite) TestRegisteringReporterDelegators() map[string]Delegator {
	_, valAddr, _ := s.createValidatorAccs([]int64{1000})
	val, err := s.stakingKeeper.GetValidator(s.ctx, valAddr[0])
	s.NoError(err)
	// create delegator funded accounts
	delAcc := s.CreateAccountsWithTokens(5, 100*1e6)
	delegators := map[string]Delegator{
		reporter:     {delegatorAddress: delAcc[0], validator: val, tokenAmount: math.NewInt(100 * 1e6)},
		delegatorI:   {delegatorAddress: delAcc[1], validator: val, tokenAmount: math.NewInt(100 * 1e6)},
		delegatorII:  {delegatorAddress: delAcc[2], validator: val, tokenAmount: math.NewInt(100 * 1e6)},
		delegatorIII: {delegatorAddress: delAcc[3], validator: val, tokenAmount: math.NewInt(100 * 1e6)},
		delegatorIV:  {delegatorAddress: delAcc[4], validator: val, tokenAmount: math.NewInt(100 * 1e6)},
	}
	delegators = s.createdelegators(delegators)

	// register reporter in reporter module
	commission := stakingtypes.NewCommissionWithTime(math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(3, 1),
		math.LegacyNewDecWithPrec(1, 1), s.ctx.BlockTime())

	source := reportertypes.TokenOrigin{ValidatorAddress: val.GetOperator(), Amount: 100 * 1e6}
	createReporterMsg := reportertypes.NewMsgCreateReporter(delegators[reporter].delegatorAddress.String(), 100*1e6, []*reportertypes.TokenOrigin{&source}, &commission)
	server := keeper.NewMsgServerImpl(s.reporterkeeper)
	_, err = server.CreateReporter(s.ctx, createReporterMsg)
	s.NoError(err)

	oracleReporter, err := s.reporterkeeper.Reporters.Get(s.ctx, delegators[reporter].delegatorAddress)
	s.NoError(err)
	s.Equal(oracleReporter.Reporter, delegators[reporter].delegatorAddress.String())
	s.Equal(oracleReporter.TotalTokens, uint64(100*1e6))

	// add delegation to reporter
	source = reportertypes.TokenOrigin{ValidatorAddress: val.GetOperator(), Amount: 25 * 1e6}
	delegationI := reportertypes.NewMsgDelegateReporter(
		delegators[delegatorI].delegatorAddress.String(),
		delegators[reporter].delegatorAddress.String(),
		25*1e6,
		[]*reportertypes.TokenOrigin{&source},
	)
	_, err = server.DelegateReporter(s.ctx, delegationI)
	s.NoError(err)
	delegationIReporter, err := s.reporterkeeper.Delegators.Get(s.ctx, delegators[delegatorI].delegatorAddress)
	s.NoError(err)
	s.Equal(delegationIReporter.Reporter, delegators[reporter].delegatorAddress.String())

	oracleReporter, err = s.reporterkeeper.Reporters.Get(s.ctx, delegators[reporter].delegatorAddress)
	s.NoError(err)
	s.Equal(oracleReporter.TotalTokens, uint64(125*1e6))
	// add 2nd delegation to reporter
	delegationII := reportertypes.NewMsgDelegateReporter(
		delegators[delegatorII].delegatorAddress.String(),
		delegators[reporter].delegatorAddress.String(),
		25*1e6,
		[]*reportertypes.TokenOrigin{&source},
	)
	_, err = server.DelegateReporter(s.ctx, delegationII)
	s.NoError(err)
	delegationIIReporter, err := s.reporterkeeper.Delegators.Get(s.ctx, delegators[delegatorII].delegatorAddress)
	s.NoError(err)
	s.Equal(delegationIIReporter.Reporter, delegators[reporter].delegatorAddress.String())

	oracleReporter, err = s.reporterkeeper.Reporters.Get(s.ctx, delegators[reporter].delegatorAddress)
	s.NoError(err)
	s.Equal(oracleReporter.TotalTokens, uint64(150*1e6))
	// add 3rd delegation to reporter
	delegationIII := reportertypes.NewMsgDelegateReporter(
		delegators[delegatorIII].delegatorAddress.String(),
		delegators[reporter].delegatorAddress.String(),
		25*1e6,
		[]*reportertypes.TokenOrigin{&source},
	)
	_, err = server.DelegateReporter(s.ctx, delegationIII)
	s.NoError(err)
	delegationIIIReporter, err := s.reporterkeeper.Delegators.Get(s.ctx, delegators[delegatorIII].delegatorAddress)
	s.NoError(err)
	s.Equal(delegationIIIReporter.Reporter, delegators[reporter].delegatorAddress.String())

	oracleReporter, err = s.reporterkeeper.Reporters.Get(s.ctx, delegators[reporter].delegatorAddress)
	s.NoError(err)
	s.Equal(oracleReporter.TotalTokens, uint64(175*1e6))
	// add 4th delegation to reporter
	delegationIV := reportertypes.NewMsgDelegateReporter(
		delegators[delegatorIV].delegatorAddress.String(),
		delegators[reporter].delegatorAddress.String(),
		25*1e6,
		[]*reportertypes.TokenOrigin{&source},
	)
	_, err = server.DelegateReporter(s.ctx, delegationIV)
	s.NoError(err)
	delegationIVReporter, err := s.reporterkeeper.Delegators.Get(s.ctx, delegators[delegatorIV].delegatorAddress)
	s.NoError(err)
	s.Equal(delegationIVReporter.Reporter, delegators[reporter].delegatorAddress.String())

	oracleReporter, err = s.reporterkeeper.Reporters.Get(s.ctx, delegators[reporter].delegatorAddress)
	s.NoError(err)
	s.Equal(oracleReporter.TotalTokens, uint64(200*1e6))

	return delegators
}

func (s *IntegrationTestSuite) TestDelegatorIundelegatesFromValidator() {
	delegators := s.TestRegisteringReporterDelegators()
	// delegatorI undelegates from validator
	shares, err := delegators[delegatorI].validator.SharesFromTokens(math.NewInt(10 * 1e6))
	s.NoError(err)
	valBz, err := s.stakingKeeper.ValidatorAddressCodec().StringToBytes(delegators[delegatorI].validator.GetOperator())
	s.NoError(err)
	// delegatorI undelegates from validator but is still has more tokens with validator than the reporter so reporter tokens should not be affected
	_, amt, err := s.stakingKeeper.Undelegate(s.ctx, delegators[delegatorI].delegatorAddress, valBz, shares)
	s.NoError(err)
	s.Equal(amt, math.NewInt(10*1e6))
	// call the staking hook
	s.stakingKeeper.SetHooks(s.reporterkeeper.Hooks())
	err = s.stakingKeeper.Hooks().AfterDelegationModified(s.ctx, delegators[delegatorI].delegatorAddress, valBz)
	s.NoError(err)
	oracleReporter, err := s.reporterkeeper.Reporters.Get(s.ctx, delegators[reporter].delegatorAddress)
	s.NoError(err)
	s.Equal(oracleReporter.TotalTokens, uint64(200*1e6))
	shares, err = delegators[delegatorI].validator.SharesFromTokens(math.NewInt(75 * 1e6))
	s.NoError(err)
	// delegatorI undelegates from validator and is left with 15 tokens less than the 25 delegated with reporter
	_, amt, err = s.stakingKeeper.Undelegate(s.ctx, delegators[delegatorI].delegatorAddress, valBz, shares)
	s.NoError(err)
	s.Equal(amt, math.NewInt(75*1e6))
	err = s.stakingKeeper.Hooks().AfterDelegationModified(s.ctx, delegators[delegatorI].delegatorAddress, valBz)
	s.NoError(err)
	// reporter total tokens should go down by 10 since delegatorI undelegated 85 total tokens
	// from validator remaining 15, which also mean delegation should have only 15 tokens
	oracleReporter, err = s.reporterkeeper.Reporters.Get(s.ctx, delegators[reporter].delegatorAddress)
	s.NoError(err)
	s.Equal(oracleReporter.TotalTokens, uint64(190*1e6))
	delegationIReporter, err := s.reporterkeeper.Delegators.Get(s.ctx, delegators[delegatorI].delegatorAddress)
	s.NoError(err)
	s.Equal(delegationIReporter.Amount, uint64(15*1e6))
}

func (s *IntegrationTestSuite) TestDelegatorIundelegatesFromReporter() {
	delegators := s.TestRegisteringReporterDelegators()
	server := keeper.NewMsgServerImpl(s.reporterkeeper)
	source := reportertypes.TokenOrigin{ValidatorAddress: delegators[delegatorI].validator.GetOperator()}
	// delegatorI undelegates from reporter
	source.Amount = 5 * 1e6
	delegationI := reportertypes.NewMsgUndelegateReporter(
		delegators[delegatorI].delegatorAddress.String(),
		[]*reportertypes.TokenOrigin{&source},
	)
	_, err := server.UndelegateReporter(s.ctx, delegationI)
	s.NoError(err)

	delegationIReporter, err := s.reporterkeeper.Delegators.Get(s.ctx, delegators[delegatorI].delegatorAddress)
	s.NoError(err)
	s.Equal(delegationIReporter.Amount, uint64(20*1e6))
	// undelegate the remaining 15 tokens
	source.Amount = 20 * 1e6
	delegationI.TokenOrigins[0] = &source
	_, err = server.UndelegateReporter(s.ctx, delegationI)
	s.NoError(err)
	//  delegatorI shouldn't exist in delegators table since they are fully undelegated
	_, err = s.reporterkeeper.Delegators.Get(s.ctx, delegators[delegatorI].delegatorAddress)
	s.Error(err)
	// check if reporter total tokens went down by 25
	oracleReporter, err := s.reporterkeeper.Reporters.Get(s.ctx, delegators[reporter].delegatorAddress)
	s.NoError(err)
	s.Equal(oracleReporter.TotalTokens, uint64(175*1e6))
}
