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

func (s *IntegrationTestSuite) TestRegisteringReporterDelegators() {
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
}
