package keeper_test

import (
	"errors"
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

func TestReporterStakeSnapshotTotalIsSelectedReporterStake(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockHeight(11)

	reporterAddr := sample.AccAddressBytes()
	otherReporterAddr := sample.AccAddressBytes()
	selectorAddr := sample.AccAddressBytes()
	otherSelectorAddr := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	selectedStake := math.NewInt(100_999_999)
	networkStake := math.NewInt(1_000_000_000)
	queryID := []byte("selected-stake-snapshot")

	reporter := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")
	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, reporter))
	require.NoError(t, k.Reporters.Set(ctx, otherReporterAddr, reporter))
	require.NoError(t, k.Selectors.Set(ctx, selectorAddr, types.NewSelection(reporterAddr, 1)))
	require.NoError(t, k.Selectors.Set(ctx, otherSelectorAddr, types.NewSelection(otherReporterAddr, 1)))

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", ctx).Return(uint32(100), nil).Once()
	validatorSet.On("TotalBondedTokens", ctx).Return(networkStake, nil).Once()

	delegation := stakingtypes.Delegation{
		DelegatorAddress: selectorAddr.String(),
		ValidatorAddress: valAddr.String(),
		Shares:           selectedStake.ToLegacyDec(),
	}
	validator := stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
		Status:          stakingtypes.Bonded,
		Tokens:          selectedStake,
		DelegatorShares: selectedStake.ToLegacyDec(),
	}
	sk.On("IterateDelegatorDelegations", ctx, selectorAddr, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		fn(delegation)
	}).Once()
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil).Once()

	stake, err := k.ReporterStake(ctx, reporterAddr, queryID)
	require.NoError(t, err)
	require.Equal(t, selectedStake, stake)

	snapshot, err := k.ReportByBlock.Get(ctx, collections.Join3(reporterAddr.Bytes(), uint64(ctx.BlockHeight()), queryID))
	require.NoError(t, err)
	require.Equal(t, selectedStake, snapshot.Total)
	require.Len(t, snapshot.TokenOrigins, 1)
	require.Equal(t, selectorAddr.Bytes(), snapshot.TokenOrigins[0].DelegatorAddress)

	totalReporterPower, err := k.TotalReporterPower(ctx)
	require.NoError(t, err)
	require.Equal(t, networkStake, totalReporterPower)
	require.True(t, snapshot.Total.LT(totalReporterPower), "the report snapshot is this reporter's selected stake, not network-wide reporter power")
	sk.AssertExpectations(t)
	validatorSet.AssertExpectations(t)
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

func bondedDelegation(selector sdk.AccAddress, val sdk.ValAddress, amount math.Int) (stakingtypes.Delegation, stakingtypes.Validator) {
	return stakingtypes.Delegation{
			DelegatorAddress: selector.String(),
			ValidatorAddress: val.String(),
			Shares:           amount.ToLegacyDec(),
		}, stakingtypes.Validator{
			OperatorAddress: val.String(),
			Status:          stakingtypes.Bonded,
			Tokens:          amount,
			DelegatorShares: amount.ToLegacyDec(),
		}
}

func countReportByBlockRows(t *testing.T, k keeper.Keeper, ctx sdk.Context) int {
	t.Helper()
	n := 0
	err := k.ReportByBlock.Walk(ctx, nil, func(collections.Triple[[]byte, uint64, []byte], types.DelegationsAmounts) (bool, error) {
		n++
		return false, nil
	})
	require.NoError(t, err)
	return n
}

// TestReporterStakeValidatorMoveUpdatesDisputeSnapshot keeps the reward-period
// hash when selector totals are unchanged, and still stores the validator that
// holds the stake so a later slash does not look up the old one.
func TestReporterStakeValidatorMoveUpdatesDisputeSnapshot(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	valB := sdk.ValAddress(sample.AccAddressBytes())
	amount := math.NewInt(1000)
	queryA := []byte("query-a")
	queryB := []byte("query-b")
	queryC := []byte("query-c")

	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))

	delA, validatorA := bondedDelegation(selector, valA, amount)
	delB, validatorB := bondedDelegation(selector, valB, amount)

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("GetValidator", mock.Anything, valA).Return(validatorA, nil)
	sk.On("GetValidator", mock.Anything, valB).Return(validatorB, nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		switch args.Get(0).(sdk.Context).BlockHeight() {
		case 20, 30:
			fn(delB)
		default:
			fn(delA)
		}
	})

	ctx10 := ctx.WithBlockHeight(10)
	stake, err := k.ReporterStake(ctx10, reporterAddr, queryA)
	require.NoError(t, err)
	require.Equal(t, amount, stake)

	period, err := k.ReporterPeriodData.Get(ctx, reporterAddr)
	require.NoError(t, err)
	hashBefore := append([]byte(nil), period.Hash...)
	period.RewardAmount = math.LegacyNewDec(25)
	require.NoError(t, k.ReporterPeriodData.Set(ctx, reporterAddr, period))
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(1)))

	require.NoError(t, k.FlagStakeRecalc(ctx, reporterAddr))
	ctx20 := ctx.WithBlockHeight(20)
	stake, err = k.ReporterStake(ctx20, reporterAddr, queryB)
	require.NoError(t, err)
	require.Equal(t, amount, stake)

	period, err = k.ReporterPeriodData.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.Equal(t, hashBefore, period.Hash)
	require.True(t, period.RewardAmount.Equal(math.LegacyNewDec(25)), "an unchanged selector total must not queue a new reward period")
	queued, err := k.DistributionQueue.Has(ctx, 0)
	require.NoError(t, err)
	require.False(t, queued)

	moved, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), 20)
	require.NoError(t, err)
	require.Equal(t, amount, moved.Total)
	require.Len(t, moved.TokenOrigins, 1)
	require.Equal(t, valB.Bytes(), moved.TokenOrigins[0].ValidatorAddress)

	original, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), 10)
	require.NoError(t, err)
	require.Equal(t, valA.Bytes(), original.TokenOrigins[0].ValidatorAddress)

	rowsAfterMove := countReportByBlockRows(t, k, ctx)
	require.Equal(t, 2, rowsAfterMove)

	require.NoError(t, k.FlagStakeRecalc(ctx, reporterAddr))
	ctx30 := ctx.WithBlockHeight(30)
	stake, err = k.ReporterStake(ctx30, reporterAddr, queryC)
	require.NoError(t, err)
	require.Equal(t, amount, stake)
	require.Equal(t, rowsAfterMove, countReportByBlockRows(t, k, ctx), "the same origin multiset must not write another snapshot")

	hasNewRow, err := k.ReportByBlock.Has(ctx, collections.Join3(reporterAddr.Bytes(), uint64(30), queryC))
	require.NoError(t, err)
	require.False(t, hasNewRow)
	stillB, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), 30)
	require.NoError(t, err)
	require.Equal(t, valB.Bytes(), stillB.TokenOrigins[0].ValidatorAddress)
}

// TestReporterStakeSameHeightReplacesVisibleSnapshot updates the row the
// dispute lookup actually returns when a lower query id would sort behind an
// older row at the same height.
func TestReporterStakeSameHeightReplacesVisibleSnapshot(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockHeight(15)

	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	valB := sdk.ValAddress(sample.AccAddressBytes())
	amount := math.NewInt(1000)
	highQuery := []byte{0xff}
	lowQuery := []byte{0x01}

	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))

	delA, validatorA := bondedDelegation(selector, valA, amount)
	delB, validatorB := bondedDelegation(selector, valB, amount)
	call := 0

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("GetValidator", mock.Anything, valA).Return(validatorA, nil)
	sk.On("GetValidator", mock.Anything, valB).Return(validatorB, nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		call++
		if call == 1 {
			fn(delA)
			return
		}
		fn(delB)
	})

	stake, err := k.ReporterStake(ctx, reporterAddr, highQuery)
	require.NoError(t, err)
	require.Equal(t, amount, stake)

	period, err := k.ReporterPeriodData.Get(ctx, reporterAddr)
	require.NoError(t, err)
	hashBefore := append([]byte(nil), period.Hash...)
	period.RewardAmount = math.LegacyNewDec(25)
	require.NoError(t, k.ReporterPeriodData.Set(ctx, reporterAddr, period))
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(1)))

	require.NoError(t, k.FlagStakeRecalc(ctx, reporterAddr))
	stake, err = k.ReporterStake(ctx, reporterAddr, lowQuery)
	require.NoError(t, err)
	require.Equal(t, amount, stake)

	period, err = k.ReporterPeriodData.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.Equal(t, hashBefore, period.Hash)
	require.True(t, period.RewardAmount.Equal(math.LegacyNewDec(25)))
	queued, err := k.DistributionQueue.Has(ctx, 0)
	require.NoError(t, err)
	require.False(t, queued)

	visible, err := k.ReportByBlock.Get(ctx, collections.Join3(reporterAddr.Bytes(), uint64(15), highQuery))
	require.NoError(t, err)
	require.Equal(t, valB.Bytes(), visible.TokenOrigins[0].ValidatorAddress)
	hasLow, err := k.ReportByBlock.Has(ctx, collections.Join3(reporterAddr.Bytes(), uint64(15), lowQuery))
	require.NoError(t, err)
	require.False(t, hasLow, "a lower query id must not hide behind the row the lookup returns")
	require.Equal(t, 1, countReportByBlockRows(t, k, ctx))

	loaded, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), 15)
	require.NoError(t, err)
	require.Equal(t, valB.Bytes(), loaded.TokenOrigins[0].ValidatorAddress)

	require.NoError(t, k.FlagStakeRecalc(ctx, reporterAddr))
	stake, err = k.ReporterStake(ctx, reporterAddr, []byte{0x00})
	require.NoError(t, err)
	require.Equal(t, amount, stake)
	require.Equal(t, 1, countReportByBlockRows(t, k, ctx))
}

// TestReporterStakeOriginOrderDoesNotWriteSnapshot treats a reordering of the
// same delegator, validator, and amount triples as unchanged.
func TestReporterStakeOriginOrderDoesNotWriteSnapshot(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	valB := sdk.ValAddress(sample.AccAddressBytes())
	amountA := math.NewInt(400)
	amountB := math.NewInt(600)
	queryID := []byte("ordered-origins")

	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 2)))

	delA, validatorA := bondedDelegation(selector, valA, amountA)
	delB, validatorB := bondedDelegation(selector, valB, amountB)

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("GetValidator", mock.Anything, valA).Return(validatorA, nil)
	sk.On("GetValidator", mock.Anything, valB).Return(validatorB, nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		fn(delA)
		fn(delB)
	})

	ctx10 := ctx.WithBlockHeight(10)
	stake, err := k.ReporterStake(ctx10, reporterAddr, queryID)
	require.NoError(t, err)
	require.Equal(t, amountA.Add(amountB), stake)

	key := collections.Join3(reporterAddr.Bytes(), uint64(10), queryID)
	stored, err := k.ReportByBlock.Get(ctx, key)
	require.NoError(t, err)
	require.Len(t, stored.TokenOrigins, 2)
	stored.TokenOrigins[0], stored.TokenOrigins[1] = stored.TokenOrigins[1], stored.TokenOrigins[0]
	require.NoError(t, k.ReportByBlock.Set(ctx, key, stored))

	period, err := k.ReporterPeriodData.Get(ctx, reporterAddr)
	require.NoError(t, err)
	hashBefore := append([]byte(nil), period.Hash...)
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(1)))
	require.NoError(t, k.FlagStakeRecalc(ctx, reporterAddr))

	ctx20 := ctx.WithBlockHeight(20)
	stake, err = k.ReporterStake(ctx20, reporterAddr, []byte("ordered-origins-again"))
	require.NoError(t, err)
	require.Equal(t, amountA.Add(amountB), stake)

	period, err = k.ReporterPeriodData.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.Equal(t, hashBefore, period.Hash)
	require.Equal(t, 1, countReportByBlockRows(t, k, ctx))

	loaded, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), 20)
	require.NoError(t, err)
	require.Equal(t, stored.TokenOrigins[0].ValidatorAddress, loaded.TokenOrigins[0].ValidatorAddress)
	require.Equal(t, stored.TokenOrigins[1].ValidatorAddress, loaded.TokenOrigins[1].ValidatorAddress)
}

// TestReporterStakeValidatorMoveSlashUsesNewOrigin slashes the validator that
// holds the stake after a same-total move, and leaves the earlier snapshot on A.
func TestReporterStakeValidatorMoveSlashUsesNewOrigin(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	valB := sdk.ValAddress(sample.AccAddressBytes())
	amount := math.NewInt(1000)
	queryA := []byte("query-a")
	queryB := []byte("query-b")
	hashID := []byte("slash-moved")

	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))

	delA, validatorA := bondedDelegation(selector, valA, amount)
	delB, validatorB := bondedDelegation(selector, valB, amount)

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("GetValidator", mock.Anything, valA).Return(validatorA, nil)
	sk.On("GetValidator", mock.Anything, valB).Return(validatorB, nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		if args.Get(0).(sdk.Context).BlockHeight() == 20 {
			fn(delB)
			return
		}
		fn(delA)
	})

	ctx10 := ctx.WithBlockHeight(10)
	stake, err := k.ReporterStake(ctx10, reporterAddr, queryA)
	require.NoError(t, err)
	require.Equal(t, amount, stake)

	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(1)))
	require.NoError(t, k.FlagStakeRecalc(ctx, reporterAddr))

	ctx20 := ctx.WithBlockHeight(20)
	stake, err = k.ReporterStake(ctx20, reporterAddr, queryB)
	require.NoError(t, err)
	require.Equal(t, amount, stake)

	sharesB, err := validatorB.SharesFromTokens(amount)
	require.NoError(t, err)
	sk.On("GetDelegation", ctx20, selector, valB).Return(delB, nil)
	sk.On("Unbond", ctx20, selector, valB, sharesB).Return(amount, nil)
	bk.On("SendCoinsFromModuleToModule", ctx20, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", amount))).Return(nil)

	require.NoError(t, k.EscrowReporterStake(ctx20, reporterAddr, 1, 20, amount, queryB, hashID))

	sk.AssertCalled(t, "Unbond", ctx20, selector, valB, sharesB)
	sk.AssertNotCalled(t, "Unbond", mock.Anything, selector, valA, mock.Anything)
	bk.AssertCalled(t, "SendCoinsFromModuleToModule", ctx20, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", amount)))
	bk.AssertNumberOfCalls(t, "SendCoinsFromModuleToModule", 1)

	stored, err := k.DisputedDelegationAmounts.Get(ctx, hashID)
	require.NoError(t, err)
	require.Equal(t, amount, stored.Total)
	require.Len(t, stored.TokenOrigins, 1)
	require.Equal(t, valB.Bytes(), stored.TokenOrigins[0].ValidatorAddress)
	require.Equal(t, amount, stored.TokenOrigins[0].Amount)

	original, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), 10)
	require.NoError(t, err)
	require.Equal(t, valA.Bytes(), original.TokenOrigins[0].ValidatorAddress)
}

// TestReporterStakeHookMovesSameTotal flags the reporter through the delegation
// hook, then stores validator B without changing the reward-period hash.
func TestReporterStakeHookMovesSameTotal(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	valB := sdk.ValAddress(sample.AccAddressBytes())
	amount := math.NewInt(1000)

	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))

	delA, validatorA := bondedDelegation(selector, valA, amount)
	delB, validatorB := bondedDelegation(selector, valB, amount)

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("GetValidator", mock.Anything, valA).Return(validatorA, nil)
	sk.On("GetValidator", mock.Anything, valB).Return(validatorB, nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		if args.Get(0).(sdk.Context).BlockHeight() == 20 {
			fn(delB)
			return
		}
		fn(delA)
	})

	ctx10 := ctx.WithBlockHeight(10)
	stake, err := k.ReporterStake(ctx10, reporterAddr, []byte("query-a"))
	require.NoError(t, err)
	require.Equal(t, amount, stake)

	period, err := k.ReporterPeriodData.Get(ctx, reporterAddr)
	require.NoError(t, err)
	hashBefore := append([]byte(nil), period.Hash...)
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(1)))
	_, err = k.RecalcAtTime.Get(ctx, reporterAddr.Bytes())
	require.Error(t, err)

	ctx20 := ctx.WithBlockHeight(20)
	require.NoError(t, k.Hooks().AfterDelegationModified(ctx20, selector, valB))
	flagged, err := k.StakeRecalcFlag.Has(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.True(t, flagged)

	stake, err = k.ReporterStake(ctx20, reporterAddr, []byte("query-b"))
	require.NoError(t, err)
	require.Equal(t, amount, stake)

	flagged, err = k.StakeRecalcFlag.Has(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.False(t, flagged, "the delegation hook's flag is what caused the recalculation")

	period, err = k.ReporterPeriodData.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.Equal(t, hashBefore, period.Hash)

	moved, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), 20)
	require.NoError(t, err)
	require.Equal(t, valB.Bytes(), moved.TokenOrigins[0].ValidatorAddress)
}

// TestReporterStakeOneOfTwoSelectorsMoved keeps both origins when only the
// first selector changes validator and the selector totals stay the same.
func TestReporterStakeOneOfTwoSelectorsMoved(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selector1 := sample.AccAddressBytes()
	selector2 := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	valB := sdk.ValAddress(sample.AccAddressBytes())
	valC := sdk.ValAddress(sample.AccAddressBytes())
	amount1 := math.NewInt(400)
	amount2 := math.NewInt(600)

	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector1, types.NewSelection(reporterAddr, 1)))
	require.NoError(t, k.Selectors.Set(ctx, selector2, types.NewSelection(reporterAddr, 1)))

	del1A, validatorA := bondedDelegation(selector1, valA, amount1)
	del1B, validatorB := bondedDelegation(selector1, valB, amount1)
	del2, validatorC := bondedDelegation(selector2, valC, amount2)

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("GetValidator", mock.Anything, valA).Return(validatorA, nil)
	sk.On("GetValidator", mock.Anything, valB).Return(validatorB, nil)
	sk.On("GetValidator", mock.Anything, valC).Return(validatorC, nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, selector1, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		if args.Get(0).(sdk.Context).BlockHeight() == 20 {
			fn(del1B)
			return
		}
		fn(del1A)
	})
	sk.On("IterateDelegatorDelegations", mock.Anything, selector2, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		fn(del2)
	})

	ctx10 := ctx.WithBlockHeight(10)
	stake, err := k.ReporterStake(ctx10, reporterAddr, []byte("two-selectors"))
	require.NoError(t, err)
	require.Equal(t, amount1.Add(amount2), stake)

	period, err := k.ReporterPeriodData.Get(ctx, reporterAddr)
	require.NoError(t, err)
	period.RewardAmount = math.LegacyNewDec(25)
	require.NoError(t, k.ReporterPeriodData.Set(ctx, reporterAddr, period))
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(1)))
	require.NoError(t, k.FlagStakeRecalc(ctx, reporterAddr))

	ctx20 := ctx.WithBlockHeight(20)
	stake, err = k.ReporterStake(ctx20, reporterAddr, []byte("two-selectors-moved"))
	require.NoError(t, err)
	require.Equal(t, amount1.Add(amount2), stake)

	period, err = k.ReporterPeriodData.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.True(t, period.RewardAmount.Equal(math.LegacyNewDec(25)))
	queued, err := k.DistributionQueue.Has(ctx, 0)
	require.NoError(t, err)
	require.False(t, queued)

	moved, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), 20)
	require.NoError(t, err)
	require.Len(t, moved.TokenOrigins, 2)
	require.True(t, hasOrigin(moved.TokenOrigins, selector1, valB, amount1), "moved origin must be validator B")
	require.True(t, hasOrigin(moved.TokenOrigins, selector2, valC, amount2), "unmoved origin must stay in the snapshot")
}

func hasOrigin(origins []*types.TokenOriginInfo, del sdk.AccAddress, val sdk.ValAddress, amount math.Int) bool {
	for _, origin := range origins {
		if origin == nil {
			continue
		}
		if sdk.AccAddress(origin.DelegatorAddress).Equals(del) &&
			sdk.ValAddress(origin.ValidatorAddress).Equals(val) &&
			origin.Amount.Equal(amount) {
			return true
		}
	}
	return false
}

// TestReporterStakeLegacyReportRecalcInsertsReportByBlock reads a snapshot that
// exists only in the legacy Report collection, writes validator B into
// ReportByBlock, and leaves the legacy row for the old height.
func TestReporterStakeLegacyReportRecalcInsertsReportByBlock(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	valB := sdk.ValAddress(sample.AccAddressBytes())
	amount := math.NewInt(1000)
	legacyQuery := []byte("legacy-query")
	newQuery := []byte("recalc-query")

	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))
	require.NoError(t, k.Report.Set(ctx, collections.Join(legacyQuery, collections.Join(reporterAddr.Bytes(), uint64(10))), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{
			DelegatorAddress: selector.Bytes(),
			ValidatorAddress: valA.Bytes(),
			Amount:           amount,
		}},
		Total: amount,
	}))
	require.Equal(t, 0, countReportByBlockRows(t, k, ctx))

	delB, validatorB := bondedDelegation(selector, valB, amount)

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("GetValidator", mock.Anything, valB).Return(validatorB, nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		fn(delB)
	})

	ctx20 := ctx.WithBlockHeight(20)
	stake, err := k.ReporterStake(ctx20, reporterAddr, newQuery)
	require.NoError(t, err)
	require.Equal(t, amount, stake)

	fromNew, err := k.ReportByBlock.Get(ctx, collections.Join3(reporterAddr.Bytes(), uint64(20), newQuery))
	require.NoError(t, err)
	require.Equal(t, valB.Bytes(), fromNew.TokenOrigins[0].ValidatorAddress)
	loaded, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), 20)
	require.NoError(t, err)
	require.Equal(t, valB.Bytes(), loaded.TokenOrigins[0].ValidatorAddress)
	require.Equal(t, 1, countReportByBlockRows(t, k, ctx))

	legacy, err := k.Report.Get(ctx, collections.Join(legacyQuery, collections.Join(reporterAddr.Bytes(), uint64(10))))
	require.NoError(t, err)
	require.Equal(t, valA.Bytes(), legacy.TokenOrigins[0].ValidatorAddress)
	oldLoaded, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), 10)
	require.NoError(t, err)
	require.Equal(t, valA.Bytes(), oldLoaded.TokenOrigins[0].ValidatorAddress)

	rowsAfterInsert := countReportByBlockRows(t, k, ctx)
	ctx30 := ctx.WithBlockHeight(30)
	stake, err = k.ReporterStake(ctx30, reporterAddr, []byte("recalc-again"))
	require.NoError(t, err)
	require.Equal(t, amount, stake)
	require.Equal(t, rowsAfterInsert, countReportByBlockRows(t, k, ctx))
	hasAgain, err := k.ReportByBlock.Has(ctx, collections.Join3(reporterAddr.Bytes(), uint64(30), []byte("recalc-again")))
	require.NoError(t, err)
	require.False(t, hasAgain)
}

// TestReporterStakeCacheHitIgnoresSameTotalMove returns the cached validator
// when nothing has flagged a recalculation, even if live stake sits on B.
func TestReporterStakeCacheHitIgnoresSameTotalMove(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)

	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	valB := sdk.ValAddress(sample.AccAddressBytes())
	amount := math.NewInt(1000)
	ctx = ctx.WithBlockHeight(20)

	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporterAddr.Bytes(), uint64(10), []byte("cached")), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{
			DelegatorAddress: selector.Bytes(),
			ValidatorAddress: valA.Bytes(),
			Amount:           amount,
		}},
		Total: amount,
	}))
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(3)))
	flagged, err := k.StakeRecalcFlag.Has(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.False(t, flagged)
	_, err = k.RecalcAtTime.Get(ctx, reporterAddr.Bytes())
	require.Error(t, err)

	delB, validatorB := bondedDelegation(selector, valB, amount)
	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("GetValidator", mock.Anything, valB).Return(validatorB, nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		fn(delB)
	})

	rowsBefore := countReportByBlockRows(t, k, ctx)
	require.Equal(t, 1, rowsBefore)

	stake, err := k.ReporterStake(ctx, reporterAddr, []byte("cache-hit"))
	require.NoError(t, err)
	require.Equal(t, amount, stake)

	loaded, err := k.GetDelegationsAmount(ctx, reporterAddr.Bytes(), uint64(ctx.BlockHeight()))
	require.NoError(t, err)
	require.Equal(t, valA.Bytes(), loaded.TokenOrigins[0].ValidatorAddress)
	require.Equal(t, rowsBefore, countReportByBlockRows(t, k, ctx))
	sk.AssertNotCalled(t, "IterateDelegatorDelegations", mock.Anything, selector, mock.Anything)
}
