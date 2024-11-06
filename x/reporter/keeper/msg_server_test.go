package keeper_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func setupMsgServer(tb testing.TB) (keeper.Keeper, *mocks.StakingKeeper, *mocks.BankKeeper, *mocks.RegistryKeeper, types.MsgServer, sdk.Context) {
	tb.Helper()
	k, sk, bk, rk, ctx, _ := setupKeeper(tb)
	return k, sk, bk, rk, keeper.NewMsgServerImpl(k), ctx
}

func TestMsgServer(t *testing.T) {
	k, sk, bk, rk, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
	require.NotNil(t, sk)
	require.NotNil(t, bk)
	require.NotNil(t, rk)
}

func TestCreateReporter(t *testing.T) {
	k, sk, _, _, ms, ctx := setupMsgServer(t)
	addr := sample.AccAddressBytes()
	sk.On("IterateDelegatorDelegations", ctx, addr, mock.Anything).Return(nil)
	_, err := ms.CreateReporter(ctx, &types.MsgCreateReporter{ReporterAddress: addr.String(), CommissionRate: types.DefaultMinCommissionRate, MinTokensRequired: types.DefaultMinTrb})
	require.ErrorContains(t, err, "address does not have min tokens required to be a reporter with a BONDED validator")

	ctx = ctx.WithBlockHeight(1)
	sk.On("IterateDelegatorDelegations", ctx, addr, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		delegations := []stakingtypes.Delegation{
			{
				DelegatorAddress: addr.String(),
				ValidatorAddress: sdk.ValAddress(addr).String(),
				Shares:           math.LegacyNewDec(1000),
			},
		}
		for _, delegation := range delegations {
			val := stakingtypes.Validator{
				OperatorAddress: sdk.ValAddress(addr).String(),
				Status:          stakingtypes.Bonded,
				Tokens:          math.NewInt(1_000_000),
				DelegatorShares: math.LegacyNewDec(1_000),
			}

			sk.On("GetValidator", ctx, sdk.ValAddress(addr)).Return(val, nil)

			if fn(delegation) {
				break
			}
		}
	})

	_, err = ms.CreateReporter(ctx, &types.MsgCreateReporter{ReporterAddress: addr.String(), CommissionRate: math.NewUint(1e6 + 1), MinTokensRequired: types.DefaultMinTrb})
	require.Equal(t, err.Error(), "commission rate must be below 1000000 as that is a 100 percent commission rate")

	_, err = k.Reporters.Get(ctx, addr)
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = ms.CreateReporter(ctx, &types.MsgCreateReporter{ReporterAddress: addr.String(), CommissionRate: types.DefaultMinCommissionRate, MinTokensRequired: types.DefaultMinTrb})
	require.NoError(t, err)

	reporter, err := k.Reporters.Get(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, types.DefaultMinCommissionRate, reporter.CommissionRate)
	require.Equal(t, types.DefaultMinTrb, reporter.MinTokensRequired)
}

func TestSelectReporter(t *testing.T) {
	k, sk, _, _, ms, ctx := setupMsgServer(t)
	selector, reporter := sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Reporters.Set(ctx, reporter, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinTrb)))
	sk.On("IterateDelegatorDelegations", ctx, selector, mock.Anything).Return(nil)
	_, err := ms.SelectReporter(ctx, &types.MsgSelectReporter{SelectorAddress: selector.String(), ReporterAddress: reporter.String()})
	require.ErrorContains(t, err, "reporter's min requirement 1000000 not met by selector")

	ctx = ctx.WithBlockHeight(1)
	sk.On("IterateDelegatorDelegations", ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		delegations := []stakingtypes.Delegation{
			{
				DelegatorAddress: selector.String(),
				ValidatorAddress: sdk.ValAddress(selector).String(),
				Shares:           math.LegacyNewDec(1000),
			},
		}
		for _, delegation := range delegations {
			val := stakingtypes.Validator{
				OperatorAddress: sdk.ValAddress(selector).String(),
				Status:          stakingtypes.Bonded,
				Tokens:          math.NewInt(1_000_000),
				DelegatorShares: math.LegacyNewDec(1_000),
			}

			sk.On("GetValidator", ctx, sdk.ValAddress(selector)).Return(val, nil)

			if fn(delegation) {
				break
			}
		}
	})
	require.NoError(t, k.Params.Set(ctx, types.Params{MaxSelectors: 0}))
	_, err = ms.SelectReporter(ctx, &types.MsgSelectReporter{SelectorAddress: selector.String(), ReporterAddress: reporter.String()})
	require.ErrorContains(t, err, "reporter has reached max selectors")

	require.NoError(t, k.Params.Set(ctx, types.Params{MaxSelectors: 10}))
	_, err = ms.SelectReporter(ctx, &types.MsgSelectReporter{SelectorAddress: selector.String(), ReporterAddress: reporter.String()})
	require.NoError(t, err)

	selection, err := k.Selectors.Get(ctx, selector)
	require.NoError(t, err)
	require.True(t, bytes.Equal(reporter.Bytes(), selection.Reporter))
}

func TestSwitchReporter(t *testing.T) {
	k, sk, _, rk, ms, ctx := setupMsgServer(t)
	ctx = ctx.WithBlockTime(time.Now())
	selector, reporter, reporter2 := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()

	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporter, 1)))
	// reporter2 does not exist
	_, err := ms.SwitchReporter(ctx, &types.MsgSwitchReporter{SelectorAddress: selector.String(), ReporterAddress: reporter2.String()})
	require.ErrorIs(t, err, collections.ErrNotFound)

	require.NoError(t, k.Reporters.Set(ctx, reporter2, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinTrb)))
	require.NoError(t, k.Params.Set(ctx, types.Params{MaxSelectors: 0}))

	_, err = ms.SwitchReporter(ctx, &types.MsgSwitchReporter{SelectorAddress: selector.String(), ReporterAddress: reporter2.String()})
	require.ErrorContains(t, err, "reporter has reached max selectors")

	require.NoError(t, k.Params.Set(ctx, types.Params{MaxSelectors: 1}))
	sk.On("IterateDelegatorDelegations", ctx, selector, mock.Anything).Return(nil)
	_, err = ms.SwitchReporter(ctx, &types.MsgSwitchReporter{SelectorAddress: selector.String(), ReporterAddress: reporter2.String()})
	require.ErrorContains(t, err, "reporter's min requirement 1000000 not met by selector")

	ctx = ctx.WithBlockHeight(1)
	sk.On("IterateDelegatorDelegations", ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		delegations := []stakingtypes.Delegation{
			{
				DelegatorAddress: selector.String(),
				ValidatorAddress: sdk.ValAddress(selector).String(),
				Shares:           math.LegacyNewDec(1000),
			},
		}
		for _, delegation := range delegations {
			val := stakingtypes.Validator{
				OperatorAddress: sdk.ValAddress(selector).String(),
				Status:          stakingtypes.Bonded,
				Tokens:          math.NewInt(1_000_000),
				DelegatorShares: math.LegacyNewDec(1_000),
			}

			sk.On("GetValidator", ctx, sdk.ValAddress(selector)).Return(val, nil)

			if fn(delegation) {
				break
			}
		}
	})
	// no previous reports
	_, err = ms.SwitchReporter(ctx, &types.MsgSwitchReporter{SelectorAddress: selector.String(), ReporterAddress: reporter2.String()})
	require.NoError(t, err)

	selection, err := k.Selectors.Get(ctx, selector)
	require.NoError(t, err)
	require.True(t, bytes.Equal(reporter2.Bytes(), selection.Reporter))
	require.True(t, selection.LockedUntilTime.IsZero())

	// reset reporter for selector
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporter, 1)))

	// this time selector was part of previous reporting
	tokenOrigin := &types.TokenOriginInfo{
		DelegatorAddress: selector.Bytes(),
		ValidatorAddress: selector.Bytes(),
		Amount:           math.NewInt(1000 * 1e6),
	}
	tokenOrigins := []*types.TokenOriginInfo{tokenOrigin}

	delegationAmounts := types.DelegationsAmounts{TokenOrigins: tokenOrigins, Total: math.NewInt(1000 * 1e6)}
	require.NoError(t, k.Report.Set(ctx, collections.Join(reporter.Bytes(), uint64(ctx.BlockHeight())), delegationAmounts))

	rk.On("MaxReportBufferWindow", ctx).Return(700_000, nil)
	sk.On("UnbondingTime", ctx).Return(1814400*time.Second, nil)
	_, err = ms.SwitchReporter(ctx, &types.MsgSwitchReporter{SelectorAddress: selector.String(), ReporterAddress: reporter2.String()})
	require.NoError(t, err)

	selection, err = k.Selectors.Get(ctx, selector)
	require.NoError(t, err)
	require.True(t, bytes.Equal(reporter2.Bytes(), selection.Reporter))
	require.False(t, selection.LockedUntilTime.IsZero())
	require.Equal(t, selection.LockedUntilTime, ctx.BlockTime().Add(1814400*time.Second))
}

func TestRemoveSelector(t *testing.T) {
	k, sk, _, _, ms, ctx := setupMsgServer(t)
	reporter, selector := sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Reporters.Set(ctx, reporter, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinTrb)))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporter, 1)))

	sk.On("IterateDelegatorDelegations", ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		delegations := []stakingtypes.Delegation{
			{
				DelegatorAddress: selector.String(),
				ValidatorAddress: sdk.ValAddress(selector).String(),
				Shares:           math.LegacyNewDec(1000),
			},
		}
		for _, delegation := range delegations {
			val := stakingtypes.Validator{
				OperatorAddress: sdk.ValAddress(selector).String(),
				Status:          stakingtypes.Bonded,
				Tokens:          math.NewInt(1_000_000),
				DelegatorShares: math.LegacyNewDec(1_000),
			}

			sk.On("GetValidator", ctx, sdk.ValAddress(selector)).Return(val, nil)

			if fn(delegation) {
				break
			}
		}
	})
	// no previous reports
	_, err := ms.RemoveSelector(ctx, &types.MsgRemoveSelector{SelectorAddress: selector.String()})
	require.ErrorContains(t, err, "selector can't be removed if reporter's min requirement is met")
	// selector not removed
	_, err = k.Selectors.Get(ctx, selector)
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(1)
	// selector does not meet min requirement, however reporter is not capped
	sk.On("IterateDelegatorDelegations", ctx, selector, mock.Anything).Return(nil)
	_, err = ms.RemoveSelector(ctx, &types.MsgRemoveSelector{SelectorAddress: selector.String()})
	require.ErrorContains(t, err, "selector can only be removed if reporter has reached max selectors and doesn't meet min requirement")

	require.NoError(t, k.Params.Set(ctx, types.Params{MaxSelectors: 0}))
	_, err = ms.RemoveSelector(ctx, &types.MsgRemoveSelector{SelectorAddress: selector.String()})
	require.NoError(t, err)

	_, err = k.Selectors.Get(ctx, selector)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestUnjailReporter(t *testing.T) {
	k, _, _, _, msg, ctx := setupMsgServer(t)
	addr := sample.AccAddressBytes()
	require.NoError(t, k.Reporters.Set(ctx, addr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinTrb)))
	reporter, err := k.Reporters.Get(ctx, addr)
	require.NoError(t, err)
	require.False(t, reporter.Jailed)
	_, err = msg.UnjailReporter(ctx, &types.MsgUnjailReporter{ReporterAddress: addr.String()})
	require.ErrorContains(t, err, "cannot unjail already unjailed reporter, false: reporter not jailed")

	reporter.Jailed = true
	reporter.JailedUntil = ctx.BlockTime().Add(time.Hour)
	require.NoError(t, k.Reporters.Set(ctx, addr, reporter))

	_, err = msg.UnjailReporter(ctx, &types.MsgUnjailReporter{ReporterAddress: addr.String()})
	require.ErrorContains(t, err, "cannot unjail reporter before jail time is up")

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Hour))
	_, err = msg.UnjailReporter(ctx, &types.MsgUnjailReporter{ReporterAddress: addr.String()})
	require.NoError(t, err)

	reporter, err = k.Reporters.Get(ctx, addr)
	require.NoError(t, err)
	require.False(t, reporter.Jailed)
}

func TestWithdrawTip(t *testing.T) {
	k, sk, bk, _, msg, ctx := setupMsgServer(t)
	selector, valAddr := sample.AccAddressBytes(), sdk.ValAddress(sample.AccAddressBytes())

	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(selector, 1)))

	_, err := msg.WithdrawTip(ctx, &types.MsgWithdrawTip{SelectorAddress: selector.String(), ValidatorAddress: valAddr.String()})
	require.ErrorIs(t, err, collections.ErrNotFound)

	require.NoError(t, k.SelectorTips.Set(ctx, selector, types.BigUint{Value: math.NewUint(1 * 1e6)}))

	validator := stakingtypes.Validator{Status: stakingtypes.Bonded}
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	sk.On("Delegate", ctx, selector, math.OneInt(), stakingtypes.Bonded, validator, false).Return(math.LegacyZeroDec(), nil)
	bk.On("SendCoinsFromModuleToModule", ctx, types.TipsEscrowPool, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin("loya", math.OneInt()))).Return(nil)
	_, err = msg.WithdrawTip(ctx, &types.MsgWithdrawTip{SelectorAddress: selector.String(), ValidatorAddress: valAddr.String()})
	require.NoError(t, err)
}
