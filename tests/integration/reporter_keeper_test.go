package integration_test

import (
	"fmt"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *IntegrationTestSuite) TestCreatingReporter() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	valAccs, valAddrs, _ := s.createValidatorAccs([]uint64{1000})

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
	_, err = msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: valAccs[0].String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6)})
	s.NoError(err)

	// delegator is not self reporting but delegated to another reporter
	_, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, newDelegator)
	s.Error(err)
	_, err = msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: newDelegator.String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6)})
	s.NoError(err)

	delBonded, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, newDelegator)
	s.NoError(err)

	// check validator reporting tokens after delegator has moved
	val2, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[0])
	s.NoError(err)
	// staked tokens should be same as before
	s.Equal(val1.Tokens, val2.Tokens)
	validatorReporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[0])
	s.NoError(err)
	// reporting tokens should be less than before
	s.True(validatorReporterStake.LT(val1.Tokens))
	s.True(validatorReporterStake.Equal(val1.Tokens.Sub(delBonded)))
}

func (s *IntegrationTestSuite) TestSwitchReporterMsg() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	valAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 200})

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
	// register reporter
	_, err = msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: valAccs[0].String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6)})
	s.NoError(err)
	// add selector to the reporter
	_, err = msgServer.SelectReporter(s.Setup.Ctx, &reportertypes.MsgSelectReporter{SelectorAddress: newDelegator.String(), ReporterAddress: valAccs[0].String()})
	s.NoError(err)
	// check validator reporting status
	validatorReporter1, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[0])
	s.NoError(err)
	// validator reporter should have self tokens and delegator tokens as their total
	s.Equal(validatorReporter1, val1.Tokens)

	// check second reporter tokens
	val2, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[1])
	s.NoError(err)
	// register second reporter
	_, err = msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: valAccs[1].String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6)})
	s.NoError(err)
	validatorReporter2, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[1])
	s.NoError(err)
	// validator reporter should have self tokens and delegator tokens as their total
	s.Equal(validatorReporter2, val2.Tokens)
	// valrep1 should have more tokens than valrep2
	s.True(validatorReporter1.GT(validatorReporter2))

	// change reporter
	_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{SelectorAddress: newDelegator.String(), ReporterAddress: valAccs[1].String()})
	s.NoError(err)
	// forward time to bypass the lock time that the delegator has
	maxbuffer, err := s.Setup.Registrykeeper.MaxReportBufferWindow(s.Setup.Ctx)
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(int64(maxbuffer))
	// check validator reporting tokens after delegator has moved
	validatorReporter1, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[0])
	s.NoError(err)
	validatorReporter2, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[1])
	s.NoError(err)
	// reporting tokens should be less than before
	s.True(validatorReporter1.LT(val1.Tokens))
	s.True(validatorReporter2.GT(val2.Tokens))
	// valrep2 should have more tokens than valrep1
	s.True(validatorReporter2.GT(validatorReporter1))
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
	// register reporter
	msgServer := keeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	_, err := msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: delegatorAddr.String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6)})
	s.NoError(err)
	del, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, delegatorAddr.Bytes())
	s.NoError(err)
	s.Equal(uint64(5), del.DelegationsCount)
}

// add 100 delegators to a reporter and check if the delegator count is 100
// and what happens when the 101st delegator tries to delegate
func (s *IntegrationTestSuite) TestMaxSelectorsCount() {
	valAccs, valAddrs, _ := s.Setup.CreateValidators(1)
	msgServer := keeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	stakingmsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	val, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[0])
	s.NoError(err)
	fmt.Println(val.Tokens)
	_, err = msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: sdk.AccAddress(valAddrs[0]).String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6)})
	s.NoError(err)
	valAcc := valAccs[0]
	valAdd := valAddrs[0]
	// delegate a 100 delegators
	for i := 0; i < 99; i++ {
		delegatorAddr := sample.AccAddressBytes()
		s.Setup.MintTokens(delegatorAddr, math.NewInt(1000*1e6))
		msgDelegate := stakingtypes.NewMsgDelegate(
			delegatorAddr.String(),
			valAdd.String(),
			sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
		)
		_, err := stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate)
		s.NoError(err)

		_, err = msgServer.SelectReporter(s.Setup.Ctx, &reportertypes.MsgSelectReporter{SelectorAddress: delegatorAddr.String(), ReporterAddress: valAcc.String()})
		s.NoError(err)

	}
	iterSelectors, err := s.Setup.Reporterkeeper.Selectors.Indexes.Reporter.MatchExact(s.Setup.Ctx, valAcc.Bytes())
	s.NoError(err)
	selectors, err := iterSelectors.FullKeys()
	s.NoError(err)
	s.Equal(100, len(selectors))
	// try to add 1 more selector, should fail since max reached
	_, err = msgServer.SelectReporter(s.Setup.Ctx, &reportertypes.MsgSelectReporter{SelectorAddress: sample.AccAddress(), ReporterAddress: valAcc.String()})
	s.ErrorContains(err, "reporter has reached max selectors")
}

func (s *IntegrationTestSuite) TestEscrowReporterStake() {
	ctx := s.Setup.Ctx
	app := s.Setup.App
	rk := s.Setup.Reporterkeeper
	sk := s.Setup.Stakingkeeper
	reportermsgServer := keeper.NewMsgServerImpl(rk)
	// create two validators
	_, valAddrs, _ := s.Setup.CreateValidators(2)
	// the amount doesn't mean anything specific, just how much is in the pool after calling CreateValidators()
	startedBondedPoolAmount := math.NewInt(10_001_000_200)
	bondedpool := s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(stakingtypes.BondedPoolName), s.Setup.Denom)
	s.Equal(startedBondedPoolAmount, bondedpool.Amount)
	unbondedpool := s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(stakingtypes.NotBondedPoolName), s.Setup.Denom)
	s.Equal(math.ZeroInt(), unbondedpool.Amount)

	valAddr1 := valAddrs[0]
	valAddr2 := valAddrs[1]
	// create three new addresses and delegate them to the first validator
	delegator1, delegator2, delegator3 := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	reporterAddr := delegator1
	s.Setup.MintTokens(delegator1, math.NewInt(1000*1e6))
	s.Setup.MintTokens(delegator2, math.NewInt(1000*1e6))
	s.Setup.MintTokens(delegator3, math.NewInt(1000*1e6))

	stakingmsgServer := stakingkeeper.NewMsgServerImpl(sk)
	msgDelegate1 := stakingtypes.NewMsgDelegate(
		delegator1.String(),
		valAddr1.String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)
	msgDelegate2 := stakingtypes.NewMsgDelegate(
		delegator2.String(),
		valAddr1.String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)
	msgDelegate3 := stakingtypes.NewMsgDelegate(
		delegator3.String(),
		valAddr1.String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)

	_, err := stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate1)
	s.NoError(err)
	_, err = stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate2)
	s.NoError(err)
	_, err = stakingmsgServer.Delegate(s.Setup.Ctx, msgDelegate3)
	s.NoError(err)
	bondedpoolAmountafterDelegating := startedBondedPoolAmount.Add(math.NewIntWithDecimal(3_000, 6))
	bondedpool = s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(stakingtypes.BondedPoolName), s.Setup.Denom)
	s.Equal(bondedpoolAmountafterDelegating, bondedpool.Amount)
	unbondedpool = s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(stakingtypes.NotBondedPoolName), s.Setup.Denom)
	s.Equal(math.ZeroInt(), unbondedpool.Amount)
	// create reporter, automatically self selects
	msgCreateReporter := reportertypes.MsgCreateReporter{
		ReporterAddress:   reporterAddr.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewIntWithDecimal(1, 6),
	}
	_, err = reportermsgServer.CreateReporter(ctx, &msgCreateReporter)
	s.NoError(err)
	// select reporter other two delegators
	msgSelectReporter := reportertypes.MsgSelectReporter{
		SelectorAddress: delegator2.String(),
		ReporterAddress: reporterAddr.String(),
	}
	_, err = reportermsgServer.SelectReporter(ctx, &msgSelectReporter)
	s.NoError(err)

	msgSelectReporter = reportertypes.MsgSelectReporter{
		SelectorAddress: delegator3.String(),
		ReporterAddress: reporterAddr.String(),
	}
	_, err = reportermsgServer.SelectReporter(ctx, &msgSelectReporter)
	s.NoError(err)

	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	// sanity check of reporter stake, this also sets k.Report.Set
	blockHeightAtFullPower := ctx.BlockHeight()
	reporterStake, err := rk.ReporterStake(ctx, reporterAddr)
	s.NoError(err)
	s.Equal(math.NewIntWithDecimal(3_000, 6), reporterStake)
	// undelegate delegator2 sends tokens to unbonded pool and creates unbonding delegation object
	msgUndelegatedelegator2 := stakingtypes.NewMsgUndelegate(
		delegator2.String(),
		valAddr1.String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)
	_, err = stakingmsgServer.Undelegate(ctx, msgUndelegatedelegator2)
	s.NoError(err)

	// check staking module accounts
	bondedpool = s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(stakingtypes.BondedPoolName), s.Setup.Denom)
	s.Equal(bondedpoolAmountafterDelegating.Sub(math.NewIntWithDecimal(1_000, 6)), bondedpool.Amount)
	unbondedpool = s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(stakingtypes.NotBondedPoolName), s.Setup.Denom)
	s.Equal(math.NewIntWithDecimal(1_000, 6), unbondedpool.Amount)

	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: ctx.BlockHeight() + 1, Time: ctx.BlockTime().Add(1)})
	reporterStake, err = rk.ReporterStake(ctx, reporterAddr)
	s.NoError(err)
	s.Equal(math.NewIntWithDecimal(2000, 6), reporterStake)

	// redelegate delegator3, creates redelegation object
	msgReDelegate3 := stakingtypes.NewMsgBeginRedelegate(
		delegator3.String(),
		valAddr1.String(),
		valAddr2.String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)
	_, err = stakingmsgServer.BeginRedelegate(ctx, msgReDelegate3)
	s.NoError(err)

	// what happens when the delegator tries to unbond from the new validator
	msgUndelegatedelegator3 := stakingtypes.NewMsgUndelegate(
		delegator3.String(),
		valAddr2.String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)
	_, err = stakingmsgServer.Undelegate(ctx, msgUndelegatedelegator3)
	s.NoError(err)

	bondedpool = s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(stakingtypes.BondedPoolName), s.Setup.Denom)
	s.Equal(bondedpoolAmountafterDelegating.Sub(math.NewIntWithDecimal(2_000, 6)), bondedpool.Amount)
	unbondedpool = s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(stakingtypes.NotBondedPoolName), s.Setup.Denom)
	s.Equal(math.NewIntWithDecimal(2_000, 6), unbondedpool.Amount)

	disputeBal := s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress("dispute"), s.Setup.Denom)
	s.Equal(math.ZeroInt(), disputeBal.Amount)

	// get validator power before escrowing reporter stake
	val, err := sk.GetValidator(ctx, valAddr1)
	s.NoError(err)
	valPower := val.ConsensusPower(val.Tokens)
	pk, err := val.ConsPubKey()
	s.NoError(err)
	cmtPk, err := cryptocodec.ToCmtPubKeyInterface(pk)
	s.NoError(err)

	err = rk.EscrowReporterStake(
		ctx, reporterAddr, uint64(sdk.TokensToConsensusPower(math.NewIntWithDecimal(3000, 6), sdk.DefaultPowerReduction)),
		uint64(blockHeightAtFullPower), math.NewIntWithDecimal(1500, 6), []byte("hashId"))
	s.NoError(err)
	// tokens are moved to dispute module
	disputeBal = s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress("dispute"), s.Setup.Denom)
	s.Equal(math.NewIntWithDecimal(1500, 6), disputeBal.Amount)

	// slash delegator3, infraction height before escrowReporterStake was called
	_, err = sk.Slash(ctx, sdk.ConsAddress(cmtPk.Address()).Bytes(), blockHeightAtFullPower, valPower, math.LegacyNewDecWithPrec(5, 1))
	s.NoError(err)
}
