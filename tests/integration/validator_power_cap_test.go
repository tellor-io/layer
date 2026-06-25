package integration_test

import (
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// enableValidatorCap sets the validator power cap to the 0.30 default for
// dedicated cap tests; the shared integration setup disables it by default.
func (s *IntegrationTestSuite) enableValidatorCap() {
	params, err := s.Setup.Reporterkeeper.Params.Get(s.Setup.Ctx)
	s.Require().NoError(err)
	params.MaxValidatorPowerShare = reportertypes.DefaultMaxValidatorPowerShare
	s.Require().NoError(s.Setup.Reporterkeeper.Params.Set(s.Setup.Ctx, params))
}

// TestValidatorPowerCapRedelegate bonds four equal validators (25% each) and
// redelegates enough from one bonded validator to another to push the
// destination strictly above 30%. The redelegator is the source validator's own
// operator, so its per-delegator bonded delta is zero (bonded->bonded) and the
// per-delegator 30% cap does not fire first. The ante rejects with
// ErrExceedsMaxValidatorPowerShare, while real staking would otherwise let the
// destination exceed the cap.
func (s *IntegrationTestSuite) TestValidatorPowerCapRedelegate() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	accAddrs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 100, 100, 100})
	s.setStakeTrackerToCurrentBonded()
	s.enableValidatorCap()

	totalBonded, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.Require().NoError(err)
	// move 10% of total bonded from val0 to val1: val1 goes 25% -> 35%.
	move := totalBonded.QuoRaw(10)
	redelegateMsg := stakingtypes.NewMsgBeginRedelegate(accAddrs[0].String(), valAddrs[0].String(), valAddrs[1].String(), sdk.NewCoin(s.Setup.Denom, move))

	s.Require().ErrorIs(s.runStakeAnte(s.T(), redelegateMsg), reportertypes.ErrExceedsMaxValidatorPowerShare)
	// staking itself would make val1 exceed the cap if the ante did not block.
	cacheCtx, err := s.runStakingMsgsOnCache(redelegateMsg)
	s.Require().NoError(err)
	dstVal, err := s.Setup.Stakingkeeper.GetValidator(cacheCtx, valAddrs[1])
	s.Require().NoError(err)
	s.Require().True(dstVal.Tokens.MulRaw(100).GT(totalBonded.MulRaw(30)))
}

// TestValidatorPowerCapDisabled accepts a bonded-to-bonded redelegation over
// 30% when the validator cap is disabled (>=1, the shared-setup default).
func (s *IntegrationTestSuite) TestValidatorPowerCapDisabled() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	accAddrs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 100, 100, 100})
	s.setStakeTrackerToCurrentBonded()

	totalBonded, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.Require().NoError(err)
	move := totalBonded.QuoRaw(10)
	redelegateMsg := stakingtypes.NewMsgBeginRedelegate(accAddrs[0].String(), valAddrs[0].String(), valAddrs[1].String(), sdk.NewCoin(s.Setup.Denom, move))

	s.Require().NoError(s.runStakeAnte(s.T(), redelegateMsg))
}
