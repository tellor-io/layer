package integration_test

import (
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// enableValidatorCap sets MaxValidatorPowerShare to the 0.30 default.
func (s *IntegrationTestSuite) enableValidatorCap() {
	params, err := s.Setup.Reporterkeeper.Params.Get(s.Setup.Ctx)
	s.Require().NoError(err)
	params.MaxValidatorPowerShare = reportertypes.DefaultMaxValidatorPowerShare
	s.Require().NoError(s.Setup.Reporterkeeper.Params.Set(s.Setup.Ctx, params))
}

func (s *IntegrationTestSuite) TestValidatorPowerCapRedelegate() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	accAddrs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 100, 100, 100})
	s.setStakeTrackerToCurrentBonded()
	s.enableValidatorCap()

	totalBonded, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.Require().NoError(err)
	// 10% of bonded: val1 would go 25% -> 35%.
	move := totalBonded.QuoRaw(10)
	redelegateMsg := stakingtypes.NewMsgBeginRedelegate(accAddrs[0].String(), valAddrs[0].String(), valAddrs[1].String(), sdk.NewCoin(s.Setup.Denom, move))

	s.Require().ErrorIs(s.runStakeAnte(s.T(), redelegateMsg), reportertypes.ErrExceedsMaxValidatorPowerShare)
	cacheCtx, err := s.runStakingMsgsOnCache(redelegateMsg)
	s.Require().NoError(err)
	dstVal, err := s.Setup.Stakingkeeper.GetValidator(cacheCtx, valAddrs[1])
	s.Require().NoError(err)
	s.Require().True(dstVal.Tokens.MulRaw(100).GT(totalBonded.MulRaw(30)))
}

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

func (s *IntegrationTestSuite) TestValidatorPowerCapRejectsMixedWithdrawTipThenRedelegate() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	accAddrs, valAddrs, _ := s.createValidatorAccs([]uint64{1, 60, 39})
	totalBefore := s.setStakeTrackerToCurrentBonded()
	s.enableValidatorCap()

	targetBefore, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[0])
	s.Require().NoError(err)
	unit := s.Setup.Stakingkeeper.TokensFromConsensusPower(s.Setup.Ctx, 1)
	s.Require().Equal(unit.MulRaw(2), targetBefore.Tokens)
	s.Require().Equal(unit.MulRaw(201), totalBefore)

	// Direct leg under 30%: (2+83)/(201+83).
	tipAmount := unit.MulRaw(83)
	s.Require().NoError(s.Setup.Reporterkeeper.CheckValidatorPowerShareDelegation(
		s.Setup.Ctx,
		targetBefore,
		tipAmount,
	))

	selector := s.newAccount(10_050, math.OneInt())
	s.Require().NoError(s.Setup.Reporterkeeper.Selectors.Set(
		s.Setup.Ctx,
		selector,
		reportertypes.NewSelection(selector.Bytes(), 1),
	))
	s.Require().NoError(s.Setup.Reporterkeeper.SelectorTips.Set(
		s.Setup.Ctx,
		selector,
		math.LegacyNewDecFromInt(tipAmount),
	))
	tipCoins := sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, tipAmount))
	s.Require().NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, tipCoins))
	s.Require().NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToModule(
		s.Setup.Ctx,
		authtypes.Minter,
		reportertypes.TipsEscrowPool,
		tipCoins,
	))

	withdrawTipMsg := &reportertypes.MsgWithdrawTip{
		SelectorAddress:  selector.String(),
		ValidatorAddress: valAddrs[0].String(),
	}

	// Ante leg under 30%: (2+58)/201.
	redelegateAmount := unit.MulRaw(58)
	redelegateMsg := stakingtypes.NewMsgBeginRedelegate(
		accAddrs[1].String(),
		valAddrs[1].String(),
		valAddrs[0].String(),
		sdk.NewCoin(s.Setup.Denom, redelegateAmount),
	)
	s.Require().NoError(s.runStakeAnte(s.T(), redelegateMsg))

	// Mixed path must reject; if not, fall through and expose the combined cap breach.
	if err := s.runStakeAnte(s.T(), withdrawTipMsg, redelegateMsg); err != nil {
		s.Require().ErrorIs(err, reportertypes.ErrMixedStakeAcquisitionPaths)
		return
	}

	cacheCtx, _ := s.Setup.Ctx.CacheContext()
	reporterServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	_, err = reporterServer.WithdrawTip(cacheCtx, withdrawTipMsg)
	s.Require().NoError(err)

	stakingServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	_, err = stakingServer.BeginRedelegate(cacheCtx, redelegateMsg)
	s.Require().NoError(err)

	targetAfter, err := s.Setup.Stakingkeeper.GetValidator(cacheCtx, valAddrs[0])
	s.Require().NoError(err)
	totalAfter, err := s.Setup.Stakingkeeper.TotalBondedTokens(cacheCtx)
	s.Require().NoError(err)
	s.Require().Equal(unit.MulRaw(143), targetAfter.Tokens)
	s.Require().Equal(unit.MulRaw(284), totalAfter)

	s.Require().False(
		targetAfter.Tokens.MulRaw(100).GT(totalAfter.MulRaw(30)),
		"combined execution admitted validator share 143/284 (50.35%), above the 30% cap",
	)
}
