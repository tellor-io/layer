package integration_test

import (
	"fmt"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	"github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
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
	_, err = msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: valAccs[0].String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6), Moniker: "reporter_moniker1"})
	s.NoError(err)
	reporter, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, valAccs[0].Bytes())
	s.NoError(err)
	s.Equal(reporter.Moniker, "reporter_moniker1")
	s.Equal(reporter.Jailed, false)
	s.Equal(reporter.CommissionRate, reportertypes.DefaultMinCommissionRate)
	s.Equal(reporter.MinTokensRequired, math.NewIntWithDecimal(1, 6))

	// delegator is not self reporting but delegated to another reporter
	_, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, newDelegator)
	s.Error(err)
	_, err = msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: newDelegator.String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6), Moniker: "reporter_moniker2"})
	s.NoError(err)
	reporter, err = s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, newDelegator)
	s.NoError(err)
	s.Equal(reporter.Moniker, "reporter_moniker2")
	s.Equal(reporter.Jailed, false)
	s.Equal(reporter.CommissionRate, reportertypes.DefaultMinCommissionRate)
	s.Equal(reporter.MinTokensRequired, math.NewIntWithDecimal(1, 6))

	delBonded, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, newDelegator)
	s.NoError(err)

	// check validator reporting tokens after delegator has moved
	val2, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[0])
	s.NoError(err)
	// staked tokens should be same as before
	s.Equal(val1.Tokens, val2.Tokens)
	validatorReporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[0], []byte{})
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

	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	_, err := stakingMsgServer.Delegate(s.Setup.Ctx, msgDelegate)
	s.NoError(err)
	val1, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[0])
	s.NoError(err)

	// register reporter
	_, err = msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: valAccs[0].String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6), Moniker: "reporter_moniker1"})
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(2)
	// add selector to the reporter
	_, err = msgServer.SelectReporter(s.Setup.Ctx, &reportertypes.MsgSelectReporter{SelectorAddress: newDelegator.String(), ReporterAddress: valAccs[0].String()})
	s.NoError(err)

	// check validator reporting status
	validatorReporter1, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[0], []byte{})
	s.NoError(err)
	// validator reporter should have self tokens and delegator tokens as their total
	s.Equal(validatorReporter1, val1.Tokens)

	// check second reporter tokens
	val2, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[1])
	s.NoError(err)
	// register second reporter
	_, err = msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: valAccs[1].String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6), Moniker: "reporter_moniker2"})
	s.NoError(err)
	validatorReporter2, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[1], []byte{})
	s.NoError(err)
	// validator reporter should have self tokens and delegator tokens as their total
	s.Equal(validatorReporter2, val2.Tokens)
	// valrep1 should have more tokens than valrep2
	s.True(validatorReporter1.GT(validatorReporter2))

	// change reporter
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	_, err = msgServer.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{SelectorAddress: newDelegator.String(), ReporterAddress: valAccs[1].String()})
	s.NoError(err)
	// forward time to bypass the lock time that the delegator has

	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(1814400 * time.Second).Add(1))
	// check validator reporting tokens after delegator has moved
	validatorReporter1, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[0], []byte{})
	s.NoError(err)
	validatorReporter2, err = s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[1], []byte{})
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
	_, err := msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: delegatorAddr.String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6), Moniker: "reporter_moniker1"})
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

	_, err := msgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: sdk.AccAddress(valAddrs[0]).String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6), Moniker: "reporter_moniker"})
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
	s.Setup.Ctx = s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: 200000,
			MaxGas:   100_000_000,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 302400,
			MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
			MaxBytes:        10000,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: []string{
				cmttypes.ABCIPubKeyTypeEd25519,
			},
		},
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})
	ctx = ctx.WithConsensusParams(s.Setup.Ctx.ConsensusParams())
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
		Moniker:           "reporter_moniker1",
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
	reporterStake, err := rk.ReporterStake(ctx, reporterAddr, []byte{})
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
	reporterStake, err = rk.ReporterStake(ctx, reporterAddr, []byte{})
	s.NoError(err)
	s.Equal(math.NewIntWithDecimal(2000, 6), reporterStake)

	// redelegate delegator3, creates redelegation object
	msgReDelegate3 := stakingtypes.NewMsgBeginRedelegate(
		delegator3.String(),
		valAddr1.String(),
		valAddr2.String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)
	fmt.Println("begin redelegate", "dstVal", valAddr2.String())
	redelres, err := stakingmsgServer.BeginRedelegate(ctx, msgReDelegate3)
	s.NoError(err)
	fmt.Println(redelres.CompletionTime)
	ctx, err = simtestutil.NextBlock(s.Setup.App, ctx, time.Minute)
	s.NoError(err)
	// what happens when the delegator tries to unbond from the new validator
	msgUndelegatedelegator3 := stakingtypes.NewMsgUndelegate(
		delegator3.String(),
		valAddr2.String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)
	fmt.Println("undelegate", "srcVal", valAddr2.String())
	undelres, err := stakingmsgServer.Undelegate(ctx, msgUndelegatedelegator3)
	s.NoError(err)
	fmt.Println(undelres.CompletionTime)
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
		uint64(blockHeightAtFullPower), math.NewIntWithDecimal(1500, 6), []byte{}, []byte("hashId"))
	s.NoError(err)
	// tokens are moved to dispute module
	disputeBal = s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress("dispute"), s.Setup.Denom)
	s.Equal(math.NewIntWithDecimal(1500, 6), disputeBal.Amount)

	// slash delegator3, infraction height before escrowReporterStake was called
	_, err = sk.Slash(ctx, sdk.ConsAddress(cmtPk.Address()).Bytes(), blockHeightAtFullPower, valPower, math.LegacyNewDecWithPrec(5, 1))
	s.NoError(err)
}

func (s *IntegrationTestSuite) TestEscrowReporterStake2() {
	ctx := s.Setup.Ctx
	rk := s.Setup.Reporterkeeper
	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	height := uint64(ctx.BlockHeight())
	s.Equal(height, uint64(1))

	delAddr, valAddrs, _ := s.createValidatorAccs([]uint64{100, 200, 300, 400, 500})
	for _, val := range valAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
	reporter := delAddr[0]
	delAddr = delAddr[1:]

	err := rk.Reporters.Set(ctx, reporter, reportertypes.OracleReporter{
		MinTokensRequired: reportertypes.DefaultMinLoya,
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
	})
	s.NoError(err)

	for _, selector := range delAddr {
		err = rk.Selectors.Set(ctx, selector, reportertypes.Selection{
			Reporter:         reporter,
			DelegationsCount: 1,
		})
		s.NoError(err)
	}

	reporterStake, err := rk.ReporterStake(ctx, reporter, []byte("queryid1"))
	s.NoError(err)
	s.Equal(math.NewInt(2_800_000_000), reporterStake)
	// -------------------------------------------------
	err = rk.EscrowReporterStake(ctx, reporter, math.NewInt(2_800_000_000).Quo(layertypes.PowerReduction).Uint64(), height, math.NewInt(800), []byte("queryid1"), []byte("hashId"))
	s.NoError(err)

	reporterStake, err = rk.ReporterStake(ctx, reporter, []byte("queryid2"))
	s.NoError(err)
	stakeleft := 2_800_000_000 - 800
	s.Equal(math.NewInt(int64(stakeleft)), reporterStake)
	// -------------------------------------------------
	err = rk.EscrowReporterStake(ctx, reporter, reporterStake.Quo(layertypes.PowerReduction).Uint64(), height, math.NewInt(8000), []byte("queryid2"), []byte("hashId2"))
	s.NoError(err)

	reporterStake, err = rk.ReporterStake(ctx, reporter, []byte("queryid3"))
	s.NoError(err)
	stakeleft -= 8000
	s.Equal(math.NewInt(int64(stakeleft)), reporterStake)
	// -------------------------------------------------
	err = rk.EscrowReporterStake(ctx, reporter, reporterStake.Quo(layertypes.PowerReduction).Uint64(), height, math.NewInt(1234), []byte("queryid3"), []byte("hashId3"))
	s.NoError(err)

	reporterStake, err = rk.ReporterStake(ctx, reporter, []byte("queryid4"))
	s.NoError(err)
	stakeleft -= 1234
	s.Equal(math.NewInt(int64(stakeleft)), reporterStake)
	// -------------------------------------------------
	err = rk.EscrowReporterStake(ctx, reporter, reporterStake.Quo(layertypes.PowerReduction).Uint64(), height, math.NewInt(85023), []byte("queryid4"), []byte("hashId4"))
	s.NoError(err)

	reporterStake, err = rk.ReporterStake(ctx, reporter, []byte("queryid5"))
	s.NoError(err)
	stakeleft -= 85023
	s.Equal(math.NewInt(int64(stakeleft)), reporterStake)
	// -------------------------------------------------
	rPower := reporterStake.Quo(layertypes.PowerReduction)
	err = rk.EscrowReporterStake(ctx, reporter, rPower.Uint64(), height, rPower.Mul(layertypes.PowerReduction), []byte("queryid5"), []byte("hashId5"))
	s.NoError(err)

	reporterStake, err = rk.ReporterStake(ctx, reporter, []byte("queryid6"))
	s.NoError(err)
	s.True(reporterStake.LT(math.NewIntWithDecimal(1, 6)))
	// leftover less than 1 trb
	leftover := reporterStake.ToLegacyDec().Sub(reporterStake.Quo(layertypes.PowerReduction).ToLegacyDec()).TruncateInt()
	s.Equal(leftover, reporterStake)
}

func (s *IntegrationTestSuite) TestCreateAndSwitchReporterMsg() {
	require := s.Require()
	s.Setup.Ctx = s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: 200000,
			MaxGas:   100_000_000,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 302400,
			MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
			MaxBytes:        10000,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: []string{
				cmttypes.ABCIPubKeyTypeEd25519,
			},
		},
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})
	msReporter := keeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	require.NotNil(msReporter)

	msStaking := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	require.NotNil(msStaking)

	msOracle := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	require.NotNil(msOracle)

	valAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 200})
	newDelegator := sample.AccAddressBytes()
	s.Setup.MintTokens(newDelegator, math.NewInt(1000*1e6))
	msgDelegate := stakingtypes.NewMsgDelegate(
		newDelegator.String(),
		valAddrs[0].String(),
		sdk.NewInt64Coin(s.Setup.Denom, 1000*1e6),
	)

	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	_, err := msStaking.Delegate(s.Setup.Ctx, msgDelegate)
	s.NoError(err)
	val1, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[0])
	s.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	s.NoError(err)

	// val 1 becomes a reporter
	_, err = msReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: valAccs[0].String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6), Moniker: "reporter_moniker1"})
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(2)
	// delegator selects val 1 as their reporter
	_, err = msReporter.SelectReporter(s.Setup.Ctx, &reportertypes.MsgSelectReporter{SelectorAddress: newDelegator.String(), ReporterAddress: valAccs[0].String()})
	s.NoError(err)

	// check validator reporting status
	validatorReporter1, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[0], []byte{})
	s.NoError(err)
	// validator reporter should have self tokens and delegator tokens as their total
	s.Equal(validatorReporter1, val1.Tokens)

	// check second reporter tokens
	val2, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddrs[1])
	s.NoError(err)
	// val 2 becomes a reporter
	_, err = msReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: valAccs[1].String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: math.NewIntWithDecimal(1, 6), Moniker: "reporter_moniker2"})
	s.NoError(err)
	validatorReporter2, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, valAccs[1], []byte{})
	s.NoError(err)
	// validator reporter should have self tokens and delegator tokens as their total
	s.Equal(validatorReporter2, val2.Tokens)
	// valrep1 should have more tokens than valrep2
	s.True(validatorReporter1.GT(validatorReporter2))

	// selector becomes a reporter
	_, err = msReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   newDelegator.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewIntWithDecimal(1, 6),
		Moniker:           "used_to_be_a_selector",
	})
	s.NoError(err)

	// check delegator reporter in selectors collections
	formerSelector, err := s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, newDelegator)
	s.NoError(err)
	s.Equal(formerSelector.Reporter, newDelegator.Bytes())
	// check delegator reporter exists in reporters collections
	reporterExists, err := s.Setup.Reporterkeeper.Reporters.Has(s.Setup.Ctx, newDelegator)
	s.NoError(err)
	s.True(reporterExists)

	// delegator reporter decides to go back to delegator selector
	_, err = msReporter.SwitchReporter(s.Setup.Ctx, &reportertypes.MsgSwitchReporter{SelectorAddress: newDelegator.String(), ReporterAddress: valAccs[0].String()})
	s.NoError(err)

	// check delegator reporter in selectors collections
	formerSelector, err = s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, newDelegator)
	s.NoError(err)
	s.Equal(formerSelector.Reporter, valAccs[0].Bytes())
	// check delegator reporter does not exist in reporters collections
	reporterExists, err = s.Setup.Reporterkeeper.Reporters.Has(s.Setup.Ctx, newDelegator)
	s.NoError(err)
	s.False(reporterExists)

	// delegator becomes reporter again
	_, err = msReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   newDelegator.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewIntWithDecimal(1, 6),
		Moniker:           "back_again_to_report_and_then_leave",
	})
	s.NoError(err)

	// check delegator reporter in selectors collections
	formerSelector, err = s.Setup.Reporterkeeper.Selectors.Get(s.Setup.Ctx, newDelegator)
	s.NoError(err)
	s.Equal(formerSelector.Reporter, newDelegator.Bytes())
}

func (s *IntegrationTestSuite) TestPruneOldReports() {
	require := s.Require()
	ctx := s.Setup.Ctx

	rk := s.Setup.Reporterkeeper
	ok := s.Setup.Oraclekeeper

	reporter := sample.AccAddressBytes()

	startTime := time.Now()

	queryId1 := []byte("queryid1")
	queryId2 := []byte("queryid2")
	queryId3 := []byte("queryid3")

	// Timestamps for each block
	timestamp1 := uint64(startTime.UnixMilli())                          // block 100: startTime
	timestamp2 := uint64(startTime.Add(30 * 24 * time.Hour).UnixMilli()) // block 200: 30 days later
	timestamp3 := uint64(startTime.Add(50 * 24 * time.Hour).UnixMilli()) // block 300: 50 days later

	// Set up Report entries directly
	delegationsAmounts := reportertypes.DelegationsAmounts{
		Total: math.NewInt(1000000),
	}

	// Report at block 100
	err := rk.Report.Set(ctx, collections.Join(queryId1, collections.Join(reporter.Bytes(), uint64(100))), delegationsAmounts)
	require.NoError(err)

	// Report at block 200
	err = rk.Report.Set(ctx, collections.Join(queryId2, collections.Join(reporter.Bytes(), uint64(200))), delegationsAmounts)
	require.NoError(err)

	// Report at block 300
	err = rk.Report.Set(ctx, collections.Join(queryId3, collections.Join(reporter.Bytes(), uint64(300))), delegationsAmounts)
	require.NoError(err)

	// Set up ETH/USD aggregate entries in oracle keeper for GetBlockHeightFromTimestamp.
	// GetBlockHeightFromTimestamp uses ETH/USD queryId to resolve timestamps to block heights.
	ethUsdQueryId := utils.QueryIDFromData(ethQueryData)

	// Aggregate at startTime with Height 50 (before block 100)
	err = ok.Aggregates.Set(ctx, collections.Join(ethUsdQueryId, timestamp1), oracletypes.Aggregate{
		QueryId:           ethUsdQueryId,
		AggregateValue:    "100",
		AggregateReporter: reporter.String(),
		Height:            50,
	})
	require.NoError(err)

	// Aggregate at startTime+30d with Height 150 (between block 100 and 200)
	err = ok.Aggregates.Set(ctx, collections.Join(ethUsdQueryId, timestamp2), oracletypes.Aggregate{
		QueryId:           ethUsdQueryId,
		AggregateValue:    "100",
		AggregateReporter: reporter.String(),
		Height:            150,
	})
	require.NoError(err)

	// Aggregate at startTime+50d with Height 250 (between block 200 and 300)
	err = ok.Aggregates.Set(ctx, collections.Join(ethUsdQueryId, timestamp3), oracletypes.Aggregate{
		QueryId:           ethUsdQueryId,
		AggregateValue:    "100",
		AggregateReporter: reporter.String(),
		Height:            250,
	})
	require.NoError(err)

	// Aggregate at startTime+80d with Height 350 (above block 300)
	// Used when pruning at 120 days: cutoff=startTime+90d, nearest before is startTime+80d -> Height 350
	timestamp4 := uint64(startTime.Add(80 * 24 * time.Hour).UnixMilli())
	err = ok.Aggregates.Set(ctx, collections.Join(ethUsdQueryId, timestamp4), oracletypes.Aggregate{
		QueryId:           ethUsdQueryId,
		AggregateValue:    "100",
		AggregateReporter: reporter.String(),
		Height:            350,
	})
	require.NoError(err)

	// Verify all 3 Report entries exist
	_, err = rk.Report.Get(ctx, collections.Join(queryId1, collections.Join(reporter.Bytes(), uint64(100))))
	require.NoError(err, "Report at block 100 should exist")
	_, err = rk.Report.Get(ctx, collections.Join(queryId2, collections.Join(reporter.Bytes(), uint64(200))))
	require.NoError(err, "Report at block 200 should exist")
	_, err = rk.Report.Get(ctx, collections.Join(queryId3, collections.Join(reporter.Bytes(), uint64(300))))
	require.NoError(err, "Report at block 300 should exist")

	// Count total reports before pruning
	countBefore := 0
	err = rk.Report.Walk(ctx, nil, func(key collections.Pair[[]byte, collections.Pair[[]byte, uint64]], value reportertypes.DelegationsAmounts) (stop bool, err error) {
		countBefore++
		return false, nil
	})
	require.NoError(err)
	require.Equal(3, countBefore, "Should have exactly 3 reports before pruning")

	// Set context to 70 days from start and call PruneOldReports
	// Block 100 (timestamp1 = startTime) should be pruned (70 days old > 60 day cutoff)
	ctx = ctx.WithBlockTime(startTime.Add(70 * 24 * time.Hour))
	err = rk.PruneOldReports(ctx, 100)
	require.NoError(err)

	// Verify block 100 report is deleted (it's 70 days old, > 60 day cutoff)
	_, err = rk.Report.Get(ctx, collections.Join(queryId1, collections.Join(reporter.Bytes(), uint64(100))))
	require.Error(err, "Report at block 100 should be deleted (70 days old)")

	// Block 200 report is 40 days old (70 - 30), should still exist
	_, err = rk.Report.Get(ctx, collections.Join(queryId2, collections.Join(reporter.Bytes(), uint64(200))))
	require.NoError(err, "Report at block 200 should still exist (40 days old)")

	// Block 300 report is 20 days old (70 - 50), should still exist
	_, err = rk.Report.Get(ctx, collections.Join(queryId3, collections.Join(reporter.Bytes(), uint64(300))))
	require.NoError(err, "Report at block 300 should still exist (20 days old)")

	// Set context to 120 days from start - now block 200 and 300 reports should be old too
	ctx = ctx.WithBlockTime(startTime.Add(120 * 24 * time.Hour))
	err = rk.PruneOldReports(ctx, 100)
	require.NoError(err)

	// Block 200 report is now 90 days old (120 - 30), should be deleted
	_, err = rk.Report.Get(ctx, collections.Join(queryId2, collections.Join(reporter.Bytes(), uint64(200))))
	require.Error(err, "Report at block 200 should be deleted (90 days old)")

	// Block 300 report is 70 days old (120 - 50), should be deleted
	_, err = rk.Report.Get(ctx, collections.Join(queryId3, collections.Join(reporter.Bytes(), uint64(300))))
	require.Error(err, "Report at block 300 should be deleted (70 days old)")
}
