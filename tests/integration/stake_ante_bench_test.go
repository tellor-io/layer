package integration_test

import (
	"fmt"
	"testing"
	"time"

	setup "github.com/tellor-io/layer/tests"
	"github.com/tellor-io/layer/testutil/encoding"
	reporterante "github.com/tellor-io/layer/x/reporter/ante"
	_ "github.com/tellor-io/layer/x/reporter/module"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func benchSetup(t *testing.T, numValidators int) (*setup.SharedSetup, sdk.AccAddress, sdk.ValAddress) {
	t.Helper()
	s := &setup.SharedSetup{}
	s.SetupTest(t)
	ctx := s.Ctx.WithBlockHeight(1)

	powers := make([]uint64, numValidators)
	for i := range powers {
		powers[i] = 100
	}
	privKeys := make([]ed25519.PrivKey, numValidators)
	addrs := make([]sdk.AccAddress, numValidators)
	for i := range privKeys {
		pk := ed25519.GenPrivKey()
		privKeys[i] = *pk
		addrs[i] = sdk.AccAddress(pk.PubKey().Address())
		s.MintTokens(addrs[i], math.NewInt(1_000_000_000_000))
	}
	valAddrs := simtestutil.ConvertAddrsToValAddrs(addrs)
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Stakingkeeper)
	for i, pk := range privKeys {
		s.Accountkeeper.SetAccount(ctx, &authtypes.BaseAccount{
			Address:       addrs[i].String(),
			PubKey:        codectypes.UnsafePackAny(pk.PubKey()),
			AccountNumber: uint64(i + 1000),
		})
		valMsg, err := stakingtypes.NewMsgCreateValidator(
			valAddrs[i].String(),
			pk.PubKey(),
			sdk.NewCoin(s.Denom, math.NewInt(1_000_000_000)),
			stakingtypes.Description{Moniker: fmt.Sprintf("v%d", i)},
			stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1), math.LegacyZeroDec()),
			math.OneInt(),
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = stakingServer.CreateValidator(ctx, valMsg); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Stakingkeeper.EndBlocker(ctx); err != nil {
		t.Fatal(err)
	}

	totalBonded, err := s.Stakingkeeper.TotalBondedTokens(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Reporterkeeper.Tracker.Set(ctx, reportertypes.StakeTracker{Amount: totalBonded}); err != nil {
		t.Fatal(err)
	}

	delegator := addrs[0]
	targetVal := valAddrs[0]
	return s, delegator, targetVal
}

func runAnte(tb testing.TB, s *setup.SharedSetup, msg sdk.Msg) (storetypes.Gas, time.Duration, error) {
	tb.Helper()
	txBuilder := client.Context{}.WithTxConfig(encoding.GetTestEncodingCfg().TxConfig).TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg); err != nil {
		return 0, 0, err
	}
	gasMeter := storetypes.NewGasMeter(1_000_000_000_000)
	ctx := s.Ctx.WithBlockHeight(1).WithGasMeter(gasMeter)
	decorator := reporterante.NewTrackStakeChangesDecorator(s.Reporterkeeper, s.Stakingkeeper)
	start := time.Now()
	_, err := decorator.AnteHandle(ctx, txBuilder.GetTx(), false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	return gasMeter.GasConsumed(), time.Since(start), err
}

func TestBenchStakeAnte(t *testing.T) {
	s, delegator, targetVal := benchSetup(t, 5)

	delegateMsg := &stakingtypes.MsgDelegate{
		DelegatorAddress: delegator.String(),
		ValidatorAddress: targetVal.String(),
		Amount:           sdk.NewCoin(s.Denom, math.NewInt(1_000_000)),
	}
	gas, dur, err := runAnte(t, s, delegateMsg)
	if err != nil {
		t.Fatalf("delegate ante: %v", err)
	}
	t.Logf("simple bonded delegate: gas=%d duration=%s", gas, dur)

	// Active-set triggering redelegate
	srcVal := targetVal
	dstPriv := ed25519.GenPrivKey()
	dstAcc := sdk.AccAddress(dstPriv.PubKey().Address())
	dstVal := sdk.ValAddress(dstAcc)
	s.MintTokens(dstAcc, math.NewInt(1_000_000_000_000))
	account := authtypes.BaseAccount{
		Address:       dstAcc.String(),
		PubKey:        codectypes.UnsafePackAny(dstPriv.PubKey()),
		AccountNumber: 999,
	}
	s.Accountkeeper.SetAccount(s.Ctx, &account)
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Stakingkeeper)
	valMsg, _ := stakingtypes.NewMsgCreateValidator(
		dstVal.String(),
		dstPriv.PubKey(),
		sdk.NewCoin(s.Denom, math.NewInt(500_000_000)),
		stakingtypes.Description{Moniker: "candidate"},
		stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1), math.LegacyZeroDec()),
		math.OneInt(),
	)
	if _, err := stakingServer.CreateValidator(s.Ctx, valMsg); err != nil {
		t.Fatal(err)
	}

	redelegateMsg := &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    delegator.String(),
		ValidatorSrcAddress: srcVal.String(),
		ValidatorDstAddress: dstVal.String(),
		Amount:              sdk.NewCoin(s.Denom, math.NewInt(1_000)),
	}
	gas2, dur2, err := runAnte(t, s, redelegateMsg)
	t.Logf("redelegate (active-set path): gas=%d duration=%s err=%v", gas2, dur2, err)

	// fresh setup for undelegate-only measurement
	s2, delegator2, val2 := benchSetup(t, 5)
	undelegateMsg := &stakingtypes.MsgUndelegate{
		DelegatorAddress: delegator2.String(),
		ValidatorAddress: val2.String(),
		Amount:           sdk.NewCoin(s2.Denom, math.NewInt(1_000)),
	}
	gas3, dur3, err := runAnte(t, s2, undelegateMsg)
	t.Logf("undelegate (active-set flagged): gas=%d duration=%s err=%v", gas3, dur3, err)

	// Many delegations on entering validator (worst-case delegation expansion path)
	const numDelegators = 100
	s3, _, _ := benchSetup(t, 3)
	ctx3 := s3.Ctx.WithBlockHeight(1)
	candidatePriv := ed25519.GenPrivKey()
	candidateAcc := sdk.AccAddress(candidatePriv.PubKey().Address())
	candidateVal := sdk.ValAddress(candidateAcc)
	s3.MintTokens(candidateAcc, math.NewInt(10_000_000_000_000))
	s3.Accountkeeper.SetAccount(ctx3, &authtypes.BaseAccount{
		Address: candidateAcc.String(), PubKey: codectypes.UnsafePackAny(candidatePriv.PubKey()), AccountNumber: 5000,
	})
	stakingServer3 := stakingkeeper.NewMsgServerImpl(s3.Stakingkeeper)
	valMsg3, _ := stakingtypes.NewMsgCreateValidator(
		candidateVal.String(), candidatePriv.PubKey(), sdk.NewCoin(s3.Denom, math.NewInt(1_000_000_000)),
		stakingtypes.Description{Moniker: "many-dels"}, stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1), math.LegacyZeroDec()), math.OneInt(),
	)
	if _, err := stakingServer3.CreateValidator(ctx3, valMsg3); err != nil {
		t.Fatal(err)
	}
	delegators := make([]sdk.AccAddress, 0, numDelegators)
	for i := 0; i < numDelegators; i++ {
		pk := ed25519.GenPrivKey()
		acc := sdk.AccAddress(pk.PubKey().Address())
		delegators = append(delegators, acc)
		s3.MintTokens(acc, math.NewInt(10_000_000_000))
		s3.Accountkeeper.SetAccount(ctx3, &authtypes.BaseAccount{
			Address: acc.String(), PubKey: codectypes.UnsafePackAny(pk.PubKey()), AccountNumber: uint64(6000 + i),
		})
		_, err := stakingServer3.Delegate(ctx3, &stakingtypes.MsgDelegate{
			DelegatorAddress: acc.String(), ValidatorAddress: candidateVal.String(), Amount: sdk.NewCoin(s3.Denom, math.NewInt(1_000_000)),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	totalBonded3, _ := s3.Stakingkeeper.TotalBondedTokens(ctx3)
	_ = s3.Reporterkeeper.Tracker.Set(ctx3, reportertypes.StakeTracker{Amount: totalBonded3})
	// Small delegate to bonded validator triggers active-set scan; candidate stays inactive in 3-val setup
	bondedVals, _ := s3.Stakingkeeper.GetBondedValidatorsByPower(ctx3)
	triggerDelegator := delegators[0]
	triggerMsg := &stakingtypes.MsgDelegate{
		DelegatorAddress: triggerDelegator.String(),
		ValidatorAddress: bondedVals[0].OperatorAddress,
		Amount:           sdk.NewCoin(s3.Denom, math.NewInt(1_000)),
	}
	gas4, dur4, err := runAnte(t, s3, triggerMsg)
	t.Logf("delegate with %d delegations on candidate val in state: gas=%d duration=%s err=%v", numDelegators, gas4, dur4, err)
}

func benchSetupTB(tb testing.TB, numValidators int) (*setup.SharedSetup, sdk.AccAddress, sdk.ValAddress) {
	tb.Helper()
	s := &setup.SharedSetup{}
	s.SetupTest(tb)
	ctx := s.Ctx.WithBlockHeight(1)

	params, err := s.Stakingkeeper.GetParams(ctx)
	if err != nil {
		tb.Fatal(err)
	}
	params.MaxValidators = uint32(numValidators)
	if err := s.Stakingkeeper.SetParams(ctx, params); err != nil {
		tb.Fatal(err)
	}

	pk := ed25519.GenPrivKey()
	addr := sdk.AccAddress(pk.PubKey().Address())
	valAddr := sdk.ValAddress(addr)
	s.MintTokens(addr, math.NewInt(1_000_000_000_000_000))
	setPubKey(tb, s, ctx, addr, pk)
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Stakingkeeper)
	valMsg, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		pk.PubKey(),
		sdk.NewCoin(s.Denom, math.NewInt(1_000_000_000)),
		stakingtypes.Description{Moniker: "active"},
		stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1), math.LegacyZeroDec()),
		math.OneInt(),
	)
	if err != nil {
		tb.Fatal(err)
	}
	if _, err = stakingServer.CreateValidator(ctx, valMsg); err != nil {
		tb.Fatal(err)
	}
	if _, err := s.Stakingkeeper.EndBlocker(ctx); err != nil {
		tb.Fatal(err)
	}
	totalBonded, err := s.Stakingkeeper.TotalBondedTokens(ctx)
	if err != nil {
		tb.Fatal(err)
	}
	if err := s.Reporterkeeper.Tracker.Set(ctx, reportertypes.StakeTracker{Amount: totalBonded}); err != nil {
		tb.Fatal(err)
	}
	return s, addr, valAddr
}

func setPubKey(tb testing.TB, s *setup.SharedSetup, ctx sdk.Context, addr sdk.AccAddress, pk *ed25519.PrivKey) {
	tb.Helper()
	account := s.Accountkeeper.GetAccount(ctx, addr)
	if account == nil {
		account = authtypes.NewBaseAccountWithAddress(addr)
	}
	if err := account.SetPubKey(pk.PubKey()); err != nil {
		tb.Fatal(err)
	}
	s.Accountkeeper.SetAccount(ctx, account)
}

func createValidatorWithDelegations(tb testing.TB, s *setup.SharedSetup, moniker string, selfStake math.Int, delegationCount int, delegationAmount math.Int) (sdk.AccAddress, sdk.ValAddress) {
	tb.Helper()
	ctx := s.Ctx.WithBlockHeight(1)
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Stakingkeeper)
	pk := ed25519.GenPrivKey()
	addr := sdk.AccAddress(pk.PubKey().Address())
	valAddr := sdk.ValAddress(addr)
	s.MintTokens(addr, math.NewInt(1_000_000_000_000_000))
	setPubKey(tb, s, ctx, addr, pk)
	valMsg, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		pk.PubKey(),
		sdk.NewCoin(s.Denom, selfStake),
		stakingtypes.Description{Moniker: moniker},
		stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1), math.LegacyZeroDec()),
		math.OneInt(),
	)
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := stakingServer.CreateValidator(ctx, valMsg); err != nil {
		tb.Fatal(err)
	}
	for i := 0; i < delegationCount; i++ {
		delPK := ed25519.GenPrivKey()
		delAddr := sdk.AccAddress(delPK.PubKey().Address())
		s.MintTokens(delAddr, math.NewInt(1_000_000_000_000))
		setPubKey(tb, s, ctx, delAddr, delPK)
		if _, err := stakingServer.Delegate(ctx, &stakingtypes.MsgDelegate{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valAddr.String(),
			Amount:           sdk.NewCoin(s.Denom, delegationAmount),
		}); err != nil {
			tb.Fatal(err)
		}
	}
	return addr, valAddr
}

func benchmarkAnte(b *testing.B, s *setup.SharedSetup, msg sdk.Msg) {
	b.Helper()
	txBuilder := client.Context{}.WithTxConfig(encoding.GetTestEncodingCfg().TxConfig).TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg); err != nil {
		b.Fatal(err)
	}
	tx := txBuilder.GetTx()
	decorator := reporterante.NewTrackStakeChangesDecorator(s.Reporterkeeper, s.Stakingkeeper)

	gas, _, err := runAnte(b, s, msg)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gasMeter := storetypes.NewGasMeter(1_000_000_000_000)
		ctx := s.Ctx.WithBlockHeight(1).WithGasMeter(gasMeter)
		if _, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			return ctx, nil
		}); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(gas), "ante_gas/op")
}

func BenchmarkStakeAnteActiveSetDelegations(b *testing.B) {
	for _, n := range []int{100, 1_000, 10_000} {
		b.Run(fmt.Sprintf("enter/%d", n), func(b *testing.B) {
			s, _, _ := benchSetupTB(b, 1)
			delegationAmount := math.NewInt((1_100_000_000 / int64(n)) + 1)
			_, candidateVal := createValidatorWithDelegations(b, s, "entering", math.NewInt(1_000_000), n, delegationAmount)
			if err := s.Reporterkeeper.Tracker.Set(s.Ctx.WithBlockHeight(1), reportertypes.StakeTracker{Amount: math.NewInt(1_000_000_000_000)}); err != nil {
				b.Fatal(err)
			}
			msg := &stakingtypes.MsgDelegate{
				DelegatorAddress: sdk.AccAddress(candidateVal).String(),
				ValidatorAddress: candidateVal.String(),
				Amount:           sdk.NewCoin(s.Denom, math.NewInt(1_000)),
			}
			benchmarkAnte(b, s, msg)
		})
		b.Run(fmt.Sprintf("leave/%d", n), func(b *testing.B) {
			s, activeDelegator, activeVal := benchSetupTB(b, 1)
			stakingServer := stakingkeeper.NewMsgServerImpl(s.Stakingkeeper)
			for i := 0; i < n; i++ {
				delPK := ed25519.GenPrivKey()
				delAddr := sdk.AccAddress(delPK.PubKey().Address())
				s.MintTokens(delAddr, math.NewInt(1_000_000_000_000))
				setPubKey(b, s, s.Ctx.WithBlockHeight(1), delAddr, delPK)
				if _, err := stakingServer.Delegate(s.Ctx.WithBlockHeight(1), &stakingtypes.MsgDelegate{
					DelegatorAddress: delAddr.String(),
					ValidatorAddress: activeVal.String(),
					Amount:           sdk.NewCoin(s.Denom, math.NewInt(1_000_000)),
				}); err != nil {
					b.Fatal(err)
				}
			}
			activeTotal := int64(1_000_000_000 + n*1_000_000)
			replacementDelegationAmount := math.NewInt(((activeTotal - 2_000_000) / int64(n)) + 1)
			createValidatorWithDelegations(b, s, "replacement", math.NewInt(1_000_000), n, replacementDelegationAmount)
			totalBonded, err := s.Stakingkeeper.TotalBondedTokens(s.Ctx.WithBlockHeight(1))
			if err != nil {
				b.Fatal(err)
			}
			if totalBonded.IsZero() {
				b.Fatal("expected bonded tokens")
			}
			if err := s.Reporterkeeper.Tracker.Set(s.Ctx.WithBlockHeight(1), reportertypes.StakeTracker{Amount: math.OneInt()}); err != nil {
				b.Fatal(err)
			}
			msg := &stakingtypes.MsgUndelegate{
				DelegatorAddress: activeDelegator.String(),
				ValidatorAddress: activeVal.String(),
				Amount:           sdk.NewCoin(s.Denom, math.NewInt(2_000_000)),
			}
			benchmarkAnte(b, s, msg)
		})
	}
}
