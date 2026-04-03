package keeper_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestHasMin(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
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
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr, selector, noSelectorsReporterAddr, jailedReporterAddr := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter_moniker")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 2)))
	require.NoError(t, k.Reporters.Set(ctx, noSelectorsReporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter_moniker")))
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
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
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
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
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
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()
	require.NoError(t, k.Selectors.Set(ctx, addr, types.NewSelection(addr, 2)))
	selection, err := k.Delegation(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, selection, types.NewSelection(addr, 2))
}

func TestReporter(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()
	require.NoError(t, k.Reporters.Set(ctx, addr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter_moniker")))
	reporter, err := k.Reporter(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, reporter, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter_moniker"))
}

// Test that ReporterStake uses cached value when no recalc is needed
func TestReporterStake_CacheHit(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	ctx = ctx.WithBlockHeight(10)

	// Set up cached Report entry at block 5
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporterAddr.Bytes(), uint64(5))), types.DelegationsAmounts{
		Total: math.NewInt(1000),
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: reporterAddr, ValidatorAddress: reporterAddr, Amount: math.NewInt(1000)},
		},
	}))

	// Set LastValSetUpdateHeight to block 3 (before cached entry at block 5)
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(3)))

	// Set up reporter (required by ReporterStake)
	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))

	// Ensure no recalc flag is set
	has, err := k.StakeRecalcFlag.Has(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.False(t, has)

	// Call ReporterStake - should return cached value without calling staking keeper
	stake, err := k.ReporterStake(ctx, reporterAddr, []byte("queryid2"))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1000), stake)
}

// Test that ReporterStake recalculates when validator set updated
func TestReporterStake_RecalcOnValSetUpdate(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	ctx = ctx.WithBlockHeight(10)

	// Set up cached Report entry at block 5
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporterAddr.Bytes(), uint64(5))), types.DelegationsAmounts{
		Total: math.NewInt(1000),
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: selector, ValidatorAddress: reporterAddr, Amount: math.NewInt(1000)},
		},
	}))

	// Set LastValSetUpdateHeight to block 7 (after cached entry at block 5)
	// This should trigger recalculation
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(7)))

	// Set up reporter and selector
	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))

	// Set up staking mocks for recalculation
	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", ctx).Return(uint32(10), nil)

	validator := stakingtypes.Validator{
		OperatorAddress: sdk.ValAddress(reporterAddr).String(),
		DelegatorShares: math.LegacyNewDec(2000),
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(2000),
	}
	delegation := stakingtypes.Delegation{
		ValidatorAddress: sdk.ValAddress(reporterAddr).String(),
		Shares:           math.LegacyNewDec(2000),
	}

	sk.On("IterateDelegatorDelegations", ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		sk.On("GetValidator", ctx, sdk.ValAddress(reporterAddr)).Return(validator, nil)
		fn(delegation)
	})

	// Call ReporterStake - should recalculate because valSetUpdateHeight > lastCalcBlock
	stake, err := k.ReporterStake(ctx, reporterAddr, []byte("queryid2"))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(2000), stake) // New calculated value
}

// Test that ReporterStake recalculates when recalc flag is set
func TestReporterStake_RecalcOnFlag(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	ctx = ctx.WithBlockHeight(10)

	// Set up cached Report entry at block 5
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporterAddr.Bytes(), uint64(5))), types.DelegationsAmounts{
		Total: math.NewInt(1000),
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: selector, ValidatorAddress: reporterAddr, Amount: math.NewInt(1000)},
		},
	}))

	// Set LastValSetUpdateHeight to block 3 (before cached entry)
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(3)))

	// Set the recalc flag - this should trigger recalculation despite cache being "fresh"
	require.NoError(t, k.FlagStakeRecalc(ctx, reporterAddr))

	// Set up reporter and selector
	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))

	// Set up staking mocks for recalculation
	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", ctx).Return(uint32(10), nil)

	validator := stakingtypes.Validator{
		OperatorAddress: sdk.ValAddress(reporterAddr).String(),
		DelegatorShares: math.LegacyNewDec(3000),
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(3000),
	}
	delegation := stakingtypes.Delegation{
		ValidatorAddress: sdk.ValAddress(reporterAddr).String(),
		Shares:           math.LegacyNewDec(3000),
	}

	sk.On("IterateDelegatorDelegations", ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		sk.On("GetValidator", ctx, sdk.ValAddress(reporterAddr)).Return(validator, nil)
		fn(delegation)
	})

	// Call ReporterStake - should recalculate because flag is set
	stake, err := k.ReporterStake(ctx, reporterAddr, []byte("queryid2"))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(3000), stake)

	// Flag should be cleared after recalculation
	has, err := k.StakeRecalcFlag.Has(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.False(t, has)
}

// Test first-time reporter (no cached data) triggers recalculation
func TestReporterStake_FirstTimeReporter(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	ctx = ctx.WithBlockHeight(10)

	// No Report entry exists for this reporter
	// Set LastValSetUpdateHeight
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(5)))

	// Set up reporter and selector
	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))

	// Set up staking mocks
	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", ctx).Return(uint32(10), nil)

	validator := stakingtypes.Validator{
		OperatorAddress: sdk.ValAddress(reporterAddr).String(),
		DelegatorShares: math.LegacyNewDec(5000),
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(5000),
	}
	delegation := stakingtypes.Delegation{
		ValidatorAddress: sdk.ValAddress(reporterAddr).String(),
		Shares:           math.LegacyNewDec(5000),
	}

	sk.On("IterateDelegatorDelegations", ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		sk.On("GetValidator", ctx, sdk.ValAddress(reporterAddr)).Return(validator, nil)
		fn(delegation)
	})

	// Call ReporterStake - should calculate fresh since no prior data exists
	stake, err := k.ReporterStake(ctx, reporterAddr, []byte("queryid1"))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(5000), stake)
}

// Test that RecalcAtTime triggers recalculation when lock expires
func TestReporterStake_RecalcOnLockExpiry(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	now := time.Now()
	ctx = ctx.WithBlockHeight(10).WithBlockTime(now)

	// Set up cached Report entry at block 5
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporterAddr.Bytes(), uint64(5))), types.DelegationsAmounts{
		Total: math.NewInt(1000),
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: selector, ValidatorAddress: reporterAddr, Amount: math.NewInt(1000)},
		},
	}))

	// Set LastValSetUpdateHeight before cached entry (no valset trigger)
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(3)))

	// No recalc flag set

	// Set RecalcAtTime to a past time (lock already expired)
	pastLockExpiry := now.Add(-1 * time.Hour).Unix()
	require.NoError(t, k.RecalcAtTime.Set(ctx, reporterAddr.Bytes(), pastLockExpiry))

	// Set up reporter and selector (selector is NOT locked — lock already expired)
	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))

	// Set up staking mocks for recalculation
	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", ctx).Return(uint32(10), nil)

	validator := stakingtypes.Validator{
		OperatorAddress: sdk.ValAddress(reporterAddr).String(),
		DelegatorShares: math.LegacyNewDec(2000),
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(2000),
	}
	delegation := stakingtypes.Delegation{
		ValidatorAddress: sdk.ValAddress(reporterAddr).String(),
		Shares:           math.LegacyNewDec(2000),
	}

	sk.On("IterateDelegatorDelegations", ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		sk.On("GetValidator", ctx, sdk.ValAddress(reporterAddr)).Return(validator, nil)
		fn(delegation)
	})

	// Call ReporterStake — should recalculate because RecalcAtTime expired
	stake, err := k.ReporterStake(ctx, reporterAddr, []byte("queryid2"))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(2000), stake)

	// RecalcAtTime should be cleaned up (no locked selectors)
	_, err = k.RecalcAtTime.Get(ctx, reporterAddr.Bytes())
	require.Error(t, err, "RecalcAtTime should be removed after recalc with no locked selectors")
}

// Test that GetReporterStake sets RecalcAtTime to next lock expiry when locked selectors remain
func TestReporterStake_RecalcAtTimeUpdatedToNextLock(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selectorA := sample.AccAddressBytes() // unlocked
	selectorB := sample.AccAddressBytes() // still locked
	now := time.Now()
	ctx = ctx.WithBlockHeight(10).WithBlockTime(now)

	// Set up reporter
	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))

	// Selector A: unlocked (LockedUntilTime in the past)
	selectionA := types.NewSelection(reporterAddr, 1)
	selectionA.LockedUntilTime = now.Add(-1 * time.Hour)
	require.NoError(t, k.Selectors.Set(ctx, selectorA, selectionA))

	// Selector B: still locked (LockedUntilTime in the future)
	futureLock := now.Add(20 * 24 * time.Hour)
	selectionB := types.NewSelection(reporterAddr, 1)
	selectionB.LockedUntilTime = futureLock
	require.NoError(t, k.Selectors.Set(ctx, selectorB, selectionB))

	// Set up staking mocks for selector A (selector B is skipped because locked)
	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", ctx).Return(uint32(10), nil)

	validator := stakingtypes.Validator{
		OperatorAddress: sdk.ValAddress(reporterAddr).String(),
		DelegatorShares: math.LegacyNewDec(1000),
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(1000),
	}
	delegation := stakingtypes.Delegation{
		ValidatorAddress: sdk.ValAddress(reporterAddr).String(),
		Shares:           math.LegacyNewDec(1000),
	}

	sk.On("IterateDelegatorDelegations", ctx, selectorA, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		sk.On("GetValidator", ctx, sdk.ValAddress(reporterAddr)).Return(validator, nil)
		fn(delegation)
	})

	// Call GetReporterStake directly
	totalTokens, _, _, _, err := k.GetReporterStake(ctx, reporterAddr)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1000), totalTokens) // only selector A's stake

	// RecalcAtTime should be set to selector B's lock expiry
	recalcAt, err := k.RecalcAtTime.Get(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.Equal(t, futureLock.Unix(), recalcAt)
}

// Regression test: PruneOldReports must keep at least one snapshot per
// reporter so that dispute voting can always look up historical power,
// even for reporters who haven't submitted a new value recently.
func TestPruneOldReports_KeepsLastSnapshotPerReporter(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	now := time.Now()
	ctx = ctx.WithBlockTime(now).WithBlockHeight(500)

	// Reporter has only old entries (all below cutoff)
	require.NoError(t, k.Report.Set(ctx,
		collections.Join([]byte("q1"), collections.Join(reporterAddr.Bytes(), uint64(50))),
		types.DelegationsAmounts{Total: math.NewInt(5000)}))
	require.NoError(t, k.Report.Set(ctx,
		collections.Join([]byte("q2"), collections.Join(reporterAddr.Bytes(), uint64(100))),
		types.DelegationsAmounts{Total: math.NewInt(5000)}))

	oracleKeeper := new(mocks.OracleKeeper)
	oracleKeeper.On("GetBlockHeightFromTimestamp", mock.Anything, mock.Anything).Return(uint64(200), nil)
	k.SetOracleKeeper(oracleKeeper)

	// Prune — both entries are below cutoff (200)
	require.NoError(t, k.PruneOldReports(ctx, 100))

	// Block 50 entry should be deleted (older)
	_, err := k.Report.Get(ctx, collections.Join([]byte("q1"), collections.Join(reporterAddr.Bytes(), uint64(50))))
	require.Error(t, err, "older entry should be pruned")

	// Block 100 entry should be kept (most recent, and no newer entries exist)
	_, err = k.Report.Get(ctx, collections.Join([]byte("q2"), collections.Join(reporterAddr.Bytes(), uint64(100))))
	require.NoError(t, err, "most recent entry must be kept when no newer snapshots exist")

	// Dispute voting should still find this reporter's power
	rep, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), 500)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(5000), rep.Total,
		"dispute voting must find nonzero power from the surviving snapshot")
}

// Test that collections.Map.Remove is a no-op when key doesn't exist (no error returned)
func TestCollectionsRemoveNonExistentKey(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	nonExistentAddr := sample.AccAddressBytes()

	// Remove from RecalcAtTime when key was never set — should not error
	err := k.RecalcAtTime.Remove(ctx, nonExistentAddr.Bytes())
	require.NoError(t, err)

	// Remove from StakeRecalcFlag when key was never set — should not error
	err = k.StakeRecalcFlag.Remove(ctx, nonExistentAddr.Bytes())
	require.NoError(t, err)
}
