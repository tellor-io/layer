package keeper_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestHasMin(t *testing.T) {
	k, sk, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()

	testCases := []struct {
		name       string
		addr       sdk.AccAddress
		delegation stakingtypes.Delegation
		validator  stakingtypes.Validator
		hasmin     bool
		iterError  error
		valError   error
		err        string
		ctx        sdk.Context
	}{
		{
			name:       "bad val address",
			addr:       addr,
			delegation: stakingtypes.Delegation{DelegatorAddress: sample.AccAddress(), ValidatorAddress: "", Shares: math.LegacyZeroDec()},
			validator:  stakingtypes.Validator{DelegatorShares: math.LegacyZeroDec()},
			err:        "empty address string is not allowed",
			ctx:        ctx.WithBlockHeight(1),
		},
		{
			name:       "bad del address",
			addr:       sdk.AccAddress(""),
			delegation: stakingtypes.Delegation{},
			validator:  stakingtypes.Validator{},
			err:        "empty address string is not allowed",
			ctx:        ctx.WithBlockHeight(2),
		},
		{
			name:       "iterError",
			addr:       addr,
			delegation: stakingtypes.Delegation{},
			validator:  stakingtypes.Validator{},
			ctx:        ctx.WithBlockHeight(3),
			iterError:  errors.New("iter error"),
			err:        "iter error",
		},
		{
			name:       "Getvalidator error",
			addr:       addr,
			delegation: stakingtypes.Delegation{ValidatorAddress: sdk.ValAddress(addr).String()},
			validator:  stakingtypes.Validator{},
			valError:   errors.New("error"),
			err:        "error",
			ctx:        ctx.WithBlockHeight(4),
		},
		{
			name:       "Doesn't have Min requirement",
			addr:       addr,
			delegation: stakingtypes.Delegation{ValidatorAddress: sdk.ValAddress(addr).String(), Shares: math.LegacyOneDec()},
			validator:  stakingtypes.Validator{DelegatorShares: math.LegacyNewDec(10), Status: stakingtypes.Bonded, Tokens: math.OneInt()},
			ctx:        ctx.WithBlockHeight(5),
			hasmin:     false,
		},
		{
			name:       "Has Min requirement",
			addr:       addr,
			delegation: stakingtypes.Delegation{ValidatorAddress: sdk.ValAddress(addr).String(), Shares: math.LegacyOneDec()},
			validator:  stakingtypes.Validator{DelegatorShares: math.LegacyOneDec(), Status: stakingtypes.Bonded, Tokens: math.OneInt()},
			ctx:        ctx.WithBlockHeight(6),
			hasmin:     true,
		},
		{
			name:       "Validator not bonded",
			addr:       addr,
			delegation: stakingtypes.Delegation{ValidatorAddress: sdk.ValAddress(addr).String(), Shares: math.LegacyOneDec()},
			validator:  stakingtypes.Validator{DelegatorShares: math.LegacyOneDec(), Status: stakingtypes.Unbonded, Tokens: math.OneInt()},
			ctx:        ctx.WithBlockHeight(7),
			hasmin:     false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the expectation
			sk.On("IterateDelegatorDelegations", tc.ctx, addr, mock.AnythingOfType("func(types.Delegation) bool")).Return(tc.iterError).Run(func(args mock.Arguments) {
				fn := args.Get(2).(func(stakingtypes.Delegation) bool)
				sk.On("GetValidator", tc.ctx, sdk.ValAddress(tc.addr)).Return(tc.validator, tc.valError)
				fn(tc.delegation)
			})
			has, err := k.HasMin(tc.ctx, addr, math.OneInt())
			if err != nil {
				require.ErrorContains(t, err, tc.err)
			}
			require.Equal(t, has, tc.hasmin)
		})
	}
}

func TestReporterStake(t *testing.T) {
	k, sk, _, _, ctx, _ := setupKeeper(t)

	reporterAddr, selector, noSelectorsReporterAddr, jailedReporterAddr := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya)))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 2)))
	require.NoError(t, k.Reporters.Set(ctx, noSelectorsReporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya)))
	require.NoError(t, k.Reporters.Set(ctx, jailedReporterAddr, types.OracleReporter{Jailed: true}))
	validatorSet := new(mocks.ValidatorSet)
	testCases := []struct {
		name          string
		err           string
		ctx           sdk.Context
		addr          sdk.AccAddress
		stake         math.Int
		maxValidators uint32
	}{
		{
			name: "reporter not found",
			ctx:  ctx.WithBlockHeight(1),
			err:  "not found",
			addr: sample.AccAddressBytes(),
		},
		{
			name: "reporter jailed",
			ctx:  ctx.WithBlockHeight(2),
			err:  "reporter jailed",
			addr: jailedReporterAddr,
		},
		{
			name:  "reporter w/o selectors",
			ctx:   ctx.WithBlockHeight(3),
			addr:  noSelectorsReporterAddr,
			stake: math.ZeroInt(),
		},
		{
			name:          "good reporter, delegator count < max validators",
			ctx:           ctx.WithBlockHeight(4),
			addr:          reporterAddr,
			maxValidators: 3,
			stake:         math.OneInt(),
		},
		{
			name:          "good reporter, delegator count > max validators",
			ctx:           ctx.WithBlockHeight(5),
			addr:          reporterAddr,
			maxValidators: 1,
			stake:         math.OneInt(),
		},
	}
	validator := stakingtypes.Validator{OperatorAddress: sdk.ValAddress(reporterAddr).String(), DelegatorShares: math.LegacyOneDec(), Status: stakingtypes.Bonded, Tokens: math.OneInt()}
	delegation := stakingtypes.Delegation{ValidatorAddress: sdk.ValAddress(reporterAddr).String(), Shares: math.LegacyOneDec()}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the expectation
			if tc.addr.Equals(reporterAddr) {
				sk.On("GetValidatorSet").Return(validatorSet)
				validatorSet.On("MaxValidators", tc.ctx).Return(tc.maxValidators, nil)
				if tc.maxValidators > 1 {
					sk.On("IterateDelegatorDelegations", tc.ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
						fn := args.Get(2).(func(stakingtypes.Delegation) bool)
						sk.On("GetValidator", tc.ctx, sdk.ValAddress(tc.addr)).Return(validator, nil)
						fn(delegation)
					})
				} else {
					validatorSet.On("IterateBondedValidatorsByPower", tc.ctx, mock.AnythingOfType("func(int64, types.ValidatorI) bool")).Return(nil).Run(func(args mock.Arguments) {
						fn := args.Get(1).(func(int64, stakingtypes.ValidatorI) bool)
						sk.On("GetDelegation", tc.ctx, selector, sdk.ValAddress(tc.addr)).Return(delegation, nil)
						fn(1, validator)
					})
					sk.On("IterateDelegatorDelegations", tc.ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
						fn := args.Get(2).(func(stakingtypes.Delegation) bool)
						sk.On("GetValidator", tc.ctx, sdk.ValAddress(tc.addr)).Return(validator, nil)
						fn(delegation)
					})
				}
			}
			stake, err := k.ReporterStake(tc.ctx, tc.addr, []byte{})
			if err != nil {
				require.ErrorContains(t, err, tc.err)
			}
			require.Equal(t, stake, tc.stake)
		})
	}
}

func TestCheckSelectorsDelegations(t *testing.T) {
	k, sk, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()

	testCases := []struct {
		name       string
		addr       sdk.AccAddress
		delegation stakingtypes.Delegation
		validator  stakingtypes.Validator
		iterError  error
		err        string
		ctx        sdk.Context
		amt        math.Int
		count      int64
	}{
		{
			name:       "bad val address",
			addr:       addr,
			delegation: stakingtypes.Delegation{DelegatorAddress: sample.AccAddress(), ValidatorAddress: "", Shares: math.LegacyZeroDec()},
			validator:  stakingtypes.Validator{DelegatorShares: math.LegacyZeroDec()},
			err:        "empty address string is not allowed",
			ctx:        ctx.WithBlockHeight(1),
		},
		{
			name:       "unbonded validator",
			addr:       addr,
			delegation: stakingtypes.Delegation{DelegatorAddress: addr.String(), ValidatorAddress: sdk.ValAddress(addr).String(), Shares: math.LegacyZeroDec()},
			validator:  stakingtypes.Validator{DelegatorShares: math.LegacyZeroDec(), Tokens: math.OneInt()},
			err:        "empty address string is not allowed",
			ctx:        ctx.WithBlockHeight(2),
			count:      1,
			amt:        math.ZeroInt(),
		},
		{
			name:       "bonded validator",
			addr:       addr,
			delegation: stakingtypes.Delegation{DelegatorAddress: addr.String(), ValidatorAddress: sdk.ValAddress(addr).String(), Shares: math.LegacyOneDec()},
			validator:  stakingtypes.Validator{Status: stakingtypes.Bonded, DelegatorShares: math.LegacyOneDec(), Tokens: math.OneInt()},
			err:        "empty address string is not allowed",
			ctx:        ctx.WithBlockHeight(3),
			count:      1,
			amt:        math.OneInt(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the expectation
			sk.On("IterateDelegatorDelegations", tc.ctx, addr, mock.AnythingOfType("func(types.Delegation) bool")).Return(tc.iterError).Run(func(args mock.Arguments) {
				fn := args.Get(2).(func(stakingtypes.Delegation) bool)
				sk.On("GetValidator", tc.ctx, sdk.ValAddress(tc.addr)).Return(tc.validator, nil)
				fn(tc.delegation)
			})
			amt, count, err := k.CheckSelectorsDelegations(tc.ctx, addr)
			if err != nil {
				require.ErrorContains(t, err, tc.err)
			}
			require.Equal(t, amt, tc.amt)
			require.Equal(t, count, tc.count)
		})
	}
}

func TestTotalReporterPower(t *testing.T) {
	k, sk, _, _, ctx, _ := setupKeeper(t)
	valSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(valSet)
	valSet.On("TotalBondedTokens", ctx).Return(math.ZeroInt(), nil)
	power, err := k.TotalReporterPower(ctx)
	require.NoError(t, err)
	require.Equal(t, power, math.ZeroInt())

	ctx = ctx.WithBlockHeight(1)
	valSet.On("TotalBondedTokens", ctx).Return(math.OneInt(), nil)
	power, err = k.TotalReporterPower(ctx)
	require.NoError(t, err)
	require.Equal(t, power, math.OneInt())
}

func TestDelegation(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()
	require.NoError(t, k.Selectors.Set(ctx, addr, types.NewSelection(addr, 2)))
	selection, err := k.Delegation(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, selection, types.NewSelection(addr, 2))
}

func TestReporter(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()
	require.NoError(t, k.Reporters.Set(ctx, addr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya)))
	reporter, err := k.Reporter(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, reporter, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya))
}
