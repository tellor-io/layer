package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func bondedVal(addr sdk.ValAddress, tokens int64) stakingtypes.Validator {
	return stakingtypes.Validator{
		OperatorAddress:   addr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            math.NewInt(tokens),
		DelegatorShares:   math.NewInt(tokens).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
}

func invalidExRateVal(addr sdk.ValAddress, status stakingtypes.BondStatus) stakingtypes.Validator {
	return stakingtypes.Validator{
		OperatorAddress:   addr.String(),
		Status:            status,
		Tokens:            math.ZeroInt(),
		DelegatorShares:   math.LegacyOneDec(),
		MinSelfDelegation: math.OneInt(),
	}
}

func disablePowerCaps(t *testing.T, k keeper.Keeper, ctx sdk.Context) {
	t.Helper()
	params := types.DefaultParams()
	params.MaxValidatorPowerShare = math.LegacyOneDec()
	params.MaxReporterPowerShare = math.LegacyOneDec()
	require.NoError(t, k.Params.Set(ctx, params))
}

func origin(del sdk.AccAddress, val sdk.ValAddress, amount int64) *types.TokenOriginInfo {
	return &types.TokenOriginInfo{DelegatorAddress: del, ValidatorAddress: val, Amount: math.NewInt(amount)}
}

func setDisputedSnapshot(t *testing.T, k keeper.Keeper, ctx sdk.Context, hashId []byte, total int64, origins ...*types.TokenOriginInfo) {
	t.Helper()
	require.NoError(t, k.DisputedDelegationAmounts.Set(ctx, hashId, types.DelegationsAmounts{TokenOrigins: origins, Total: math.NewInt(total)}))
}

func setFeeSnapshot(t *testing.T, k keeper.Keeper, ctx sdk.Context, hashId []byte, total int64, origins ...*types.TokenOriginInfo) {
	t.Helper()
	require.NoError(t, k.FeePaidFromStake.Set(ctx, hashId, types.DelegationsAmounts{TokenOrigins: origins, Total: math.NewInt(total)}))
}

func mockBondedScan(sk *mocks.StakingKeeper, ctx sdk.Context, totalBonded int64, vals ...stakingtypes.Validator) {
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(totalBonded), nil)
	sk.On("GetBondedValidatorsByPower", ctx).Return(vals, nil)
}

func delegateMock(sk *mocks.StakingKeeper, ctx sdk.Context, del sdk.AccAddress, amt math.Int, tokenSrc stakingtypes.BondStatus, val stakingtypes.Validator) {
	sk.On("Delegate", ctx, del, amt, tokenSrc, val, false).Return(math.LegacyZeroDec(), nil)
}

func noDelegations(sk *mocks.StakingKeeper, ctx sdk.Context, del sdk.AccAddress) {
	sk.On("IterateDelegatorDelegations", ctx, del, mock.Anything).Return(nil)
}

func withBondedDelegation(sk *mocks.StakingKeeper, ctx sdk.Context, acc sdk.AccAddress, valAddr sdk.ValAddress, shares math.Int) {
	sk.On("IterateDelegatorDelegations", ctx, acc, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		fn(stakingtypes.Delegation{DelegatorAddress: acc.String(), ValidatorAddress: valAddr.String(), Shares: shares.ToLegacyDec()})
	})
}

func unbondingMock(sk *mocks.StakingKeeper, ctx sdk.Context, del sdk.AccAddress, val sdk.ValAddress, amt math.Int) {
	sk.On("UnbondingTime", ctx).Return(21*24*time.Hour, nil)
	sk.On("SetUnbondingDelegationEntry", ctx, del, val, ctx.BlockHeight(), mock.Anything, amt).Return(stakingtypes.UnbondingDelegation{}, nil)
	sk.On("InsertUBDQueue", ctx, stakingtypes.UnbondingDelegation{}, mock.Anything).Return(nil)
}

func userTxUnbondingMock(sk *mocks.StakingKeeper, ctx sdk.Context, del sdk.AccAddress, val sdk.ValAddress, amt math.Int) {
	sk.On("HasMaxUnbondingDelegationEntries", ctx, del, val).Return(false, nil)
	unbondingMock(sk, ctx, del, val, amt)
}

func loadDisputedSnapshot(t *testing.T, k keeper.Keeper, ctx sdk.Context, hashId []byte, del sdk.AccAddress, orig sdk.ValAddress) {
	t.Helper()
	setDisputedSnapshot(t, k, ctx, hashId, 2, origin(del, orig, 2))
}

func loadFeeSnapshot(t *testing.T, k keeper.Keeper, ctx sdk.Context, hashId []byte, del sdk.AccAddress, orig sdk.ValAddress) {
	t.Helper()
	setFeeSnapshot(t, k, ctx, hashId, 2, origin(del, orig, 2))
}

func setupCapSplitBond(t *testing.T, sk *mocks.StakingKeeper, ctx sdk.Context, del sdk.AccAddress, val sdk.ValAddress) stakingtypes.Validator {
	t.Helper()
	valObj := bondedVal(val, 29)
	sk.On("GetValidator", ctx, val).Return(valObj, nil)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	delegateMock(sk, ctx, del, math.OneInt(), stakingtypes.Bonded, valObj)
	return valObj
}

func setupCapSplitUnbond(t *testing.T, sk *mocks.StakingKeeper, ctx sdk.Context, del sdk.AccAddress, ubdKey sdk.ValAddress) {
	t.Helper()
	userTxUnbondingMock(sk, ctx, del, ubdKey, math.OneInt())
}

type poolReturnFn func(k keeper.Keeper, ctx sdk.Context, hashId []byte) (bonded, unbonded math.Int, err error)

func runScanFallbackFullBond(
	t *testing.T,
	loadSnapshot func(t *testing.T, k keeper.Keeper, ctx sdk.Context, hashId []byte, del sdk.AccAddress, orig sdk.ValAddress),
	run poolReturnFn,
	origLookup func(sdk.ValAddress) (stakingtypes.Validator, error),
) {
	t.Helper()
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	origAddr, otherAddr := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	other := bondedVal(otherAddr, 0)
	hashId := []byte("hashId")
	loadSnapshot(t, k, ctx, hashId, delAddr, origAddr)
	sk.On("GetValidator", ctx, origAddr).Return(origLookup(origAddr))
	mockBondedScan(sk, ctx, 100, other)
	noDelegations(sk, ctx, delAddr)
	delegateMock(sk, ctx, delAddr, math.NewInt(2), stakingtypes.Bonded, other)
	bondedReturn, unbondedReturn, err := run(k, ctx, hashId)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(2), bondedReturn)
	require.True(t, unbondedReturn.IsZero())
}

func TestCheckValidatorPowerShareDelegation(t *testing.T) {
	testCases := []struct {
		name    string
		tokens  int64
		amount  int64
		wantErr error
	}{
		{"bonded over cap rejects", 29, 2, types.ErrExceedsMaxValidatorPowerShare},
		{"bonded under cap succeeds", 29, 1, nil},
		{"zero amount succeeds", 30, 0, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k, sk, _, _, _, ctx, _ := setupKeeper(t)
			valAddr := sdk.ValAddress(sample.AccAddressBytes())
			val := bondedVal(valAddr, tc.tokens)
			sk.On("GetValidator", ctx, valAddr).Return(val, nil)
			sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
			err := k.CheckValidatorPowerShareDelegation(ctx, val, math.NewInt(tc.amount))
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCheckValidatorPowerShareDelegationDisabled(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	val := bondedVal(sdk.ValAddress(sample.AccAddressBytes()), 100)
	require.NoError(t, k.CheckValidatorPowerShareDelegation(ctx, val, math.NewInt(100)))
}

func TestCheckValidatorPowerShareDelegationMissingParams(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	require.NoError(t, k.Params.Remove(ctx))
	val := bondedVal(sdk.ValAddress(sample.AccAddressBytes()), 100)
	require.NoError(t, k.CheckValidatorPowerShareDelegation(ctx, val, math.NewInt(100)))
}

func TestCheckValidatorPowerShareDelegationNonBonded(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	val := bondedVal(valAddr, 100)
	val.Status = stakingtypes.Unbonded
	sk.On("GetValidator", ctx, valAddr).Return(val, nil)
	require.NoError(t, k.CheckValidatorPowerShareDelegation(ctx, val, math.NewInt(100)))
}

func TestMaxBondableAmountValidatorCap(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	val := bondedVal(valAddr, 29)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	noDelegations(sk, ctx, delAddr)

	headroom, err := k.MaxBondableAmountExported(ctx, delAddr, val, types.DefaultMaxValidatorPowerShare, types.DefaultMaxReporterPowerShare, math.ZeroInt())
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), headroom)
}

func TestMaxBondableAmountDisabledCaps(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	delAddr := sample.AccAddressBytes()
	val := bondedVal(sdk.ValAddress(sample.AccAddressBytes()), 100)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)

	headroom, err := k.MaxBondableAmountExported(ctx, delAddr, val, math.LegacyOneDec(), math.LegacyOneDec(), math.ZeroInt())
	require.NoError(t, err)
	require.True(t, headroom.GT(math.NewInt(1_000_000)))
}

func TestReturnSlashedTokensScansLowestPowerFirst(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	missingAddr := sdk.ValAddress(sample.AccAddressBytes())
	vals := make([]stakingtypes.Validator, 41)
	bigAddr := sdk.ValAddress(sample.AccAddressBytes())
	vals[0] = bondedVal(bigAddr, 400)
	var smallAddr sdk.ValAddress
	for i := 1; i < len(vals); i++ {
		smallAddr = sdk.ValAddress(sample.AccAddressBytes())
		vals[i] = bondedVal(smallAddr, 15)
	}
	setDisputedSnapshot(t, k, ctx, []byte("hashId"), 2, origin(delAddr, missingAddr, 2))
	sk.On("GetValidator", ctx, missingAddr).Return(stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound)
	mockBondedScan(sk, ctx, 1000, vals...)
	noDelegations(sk, ctx, delAddr)
	delegateMock(sk, ctx, delAddr, math.NewInt(2), stakingtypes.Bonded, vals[len(vals)-1])

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), math.ZeroInt())
	require.NoError(t, err)
	require.Equal(t, math.NewInt(2), bondedReturn)
	require.True(t, unbondedReturn.IsZero())
}

func TestWithdrawTipValidatorPowerShare(t *testing.T) {
	k, sk, bk, _, ak, msg, ctx := setupMsgServer(t)
	selector := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	require.NoError(t, k.SelectorTips.Set(ctx, selector, math.LegacyNewDec(2)))
	val := bondedVal(valAddr, 29)
	sk.On("GetValidator", ctx, valAddr).Return(val, nil)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)

	_, err := msg.WithdrawTip(ctx, &types.MsgWithdrawTip{
		SelectorAddress: selector.String(), ValidatorAddress: valAddr.String(),
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxValidatorPowerShare)
	sk.AssertNotCalled(t, "Delegate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	bk.AssertNotCalled(t, "DelegateCoinsFromAccountToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	ak.AssertNotCalled(t, "GetModuleAddress", mock.Anything)
	tips, err := k.SelectorTips.Get(ctx, selector)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(2), tips)
}

func TestBondHeadroomAndUnbondOverflow(t *testing.T) {
	cases := []struct {
		name         string
		setup        func(t *testing.T) (k keeper.Keeper, sk *mocks.StakingKeeper, ctx sdk.Context, run poolReturnFn)
		assertNoScan bool
	}{
		{
			name: "ReturnSlashedTokens over-cap original",
			setup: func(t *testing.T) (keeper.Keeper, *mocks.StakingKeeper, sdk.Context, poolReturnFn) {
				t.Helper()
				k, sk, _, _, _, ctx, _ := setupKeeper(t)
				delAddr := sample.AccAddressBytes()
				origAddr := sdk.ValAddress(sample.AccAddressBytes())
				hashId := []byte("hashId")
				setDisputedSnapshot(t, k, ctx, hashId, 2, origin(delAddr, origAddr, 2))
				setupCapSplitBond(t, sk, ctx, delAddr, origAddr)
				setupCapSplitUnbond(t, sk, ctx, delAddr, origAddr)
				noDelegations(sk, ctx, delAddr)
				return k, sk, ctx, func(k keeper.Keeper, ctx sdk.Context, hashId []byte) (math.Int, math.Int, error) {
					return k.ReturnSlashedTokens(ctx, hashId, math.ZeroInt())
				}
			},
			assertNoScan: true,
		},
		{
			name: "ReturnSlashedTokens missing original fallback",
			setup: func(t *testing.T) (keeper.Keeper, *mocks.StakingKeeper, sdk.Context, poolReturnFn) {
				t.Helper()
				k, sk, _, _, _, ctx, _ := setupKeeper(t)
				delAddr := sample.AccAddressBytes()
				origAddr, fallbackAddr := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
				hashId := []byte("hashId")
				setDisputedSnapshot(t, k, ctx, hashId, 2, origin(delAddr, origAddr, 2))
				fallback := bondedVal(fallbackAddr, 29)
				sk.On("GetValidator", ctx, origAddr).Return(stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound)
				mockBondedScan(sk, ctx, 100, fallback)
				noDelegations(sk, ctx, delAddr)
				delegateMock(sk, ctx, delAddr, math.OneInt(), stakingtypes.Bonded, fallback)
				userTxUnbondingMock(sk, ctx, delAddr, origAddr, math.OneInt())
				return k, sk, ctx, func(k keeper.Keeper, ctx sdk.Context, hashId []byte) (math.Int, math.Int, error) {
					return k.ReturnSlashedTokens(ctx, hashId, math.ZeroInt())
				}
			},
		},
		{
			name: "FeeRefund over-cap original",
			setup: func(t *testing.T) (keeper.Keeper, *mocks.StakingKeeper, sdk.Context, poolReturnFn) {
				t.Helper()
				k, sk, _, _, _, ctx, _ := setupKeeper(t)
				delAddr := sample.AccAddressBytes()
				origAddr := sdk.ValAddress(sample.AccAddressBytes())
				hashId := []byte("hashId")
				setFeeSnapshot(t, k, ctx, hashId, 2, origin(delAddr, origAddr, 2))
				setupCapSplitBond(t, sk, ctx, delAddr, origAddr)
				setupCapSplitUnbond(t, sk, ctx, delAddr, origAddr)
				noDelegations(sk, ctx, delAddr)
				return k, sk, ctx, func(k keeper.Keeper, ctx sdk.Context, hashId []byte) (math.Int, math.Int, error) {
					return k.FeeRefund(ctx, hashId, math.NewInt(2))
				}
			},
		},
		{
			name: "AddAmountToStake preferred delegation",
			setup: func(t *testing.T) (keeper.Keeper, *mocks.StakingKeeper, sdk.Context, poolReturnFn) {
				t.Helper()
				k, sk, _, _, _, ctx, _ := setupKeeper(t)
				acc := sample.AccAddressBytes()
				origAddr := sdk.ValAddress(sample.AccAddressBytes())
				amt := math.NewInt(2)
				setupCapSplitBond(t, sk, ctx, acc, origAddr)
				setupCapSplitUnbond(t, sk, ctx, acc, origAddr)
				withBondedDelegation(sk, ctx, acc, origAddr, amt)
				return k, sk, ctx, func(k keeper.Keeper, ctx sdk.Context, _ []byte) (math.Int, math.Int, error) {
					return k.AddAmountToStake(ctx, acc, amt)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, sk, ctx, run := tc.setup(t)
			bondedReturn, unbondedReturn, err := run(k, ctx, []byte("hashId"))
			require.NoError(t, err)
			require.Equal(t, math.OneInt(), bondedReturn)
			require.Equal(t, math.OneInt(), unbondedReturn)
			if tc.assertNoScan {
				sk.AssertNotCalled(t, "GetBondedValidatorsByPower", mock.Anything)
			}
		})
	}
}

func TestScanFallbackFullBond(t *testing.T) {
	missingOrig := func(sdk.ValAddress) (stakingtypes.Validator, error) {
		return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
	}
	cases := []struct {
		name       string
		load       func(t *testing.T, k keeper.Keeper, ctx sdk.Context, hashId []byte, del sdk.AccAddress, orig sdk.ValAddress)
		run        poolReturnFn
		origLookup func(sdk.ValAddress) (stakingtypes.Validator, error)
	}{
		{
			name: "ReturnSlashedTokens missing original",
			load: loadDisputedSnapshot,
			run: func(k keeper.Keeper, ctx sdk.Context, hashId []byte) (math.Int, math.Int, error) {
				return k.ReturnSlashedTokens(ctx, hashId, math.ZeroInt())
			},
			origLookup: missingOrig,
		},
		{
			name: "ReturnSlashedTokens invalid unbonded original",
			load: loadDisputedSnapshot,
			run: func(k keeper.Keeper, ctx sdk.Context, hashId []byte) (math.Int, math.Int, error) {
				return k.ReturnSlashedTokens(ctx, hashId, math.ZeroInt())
			},
			origLookup: func(a sdk.ValAddress) (stakingtypes.Validator, error) {
				return invalidExRateVal(a, stakingtypes.Unbonded), nil
			},
		},
		{
			name: "FeeRefund bonded original at cap",
			load: loadFeeSnapshot,
			run: func(k keeper.Keeper, ctx sdk.Context, hashId []byte) (math.Int, math.Int, error) {
				return k.FeeRefund(ctx, hashId, math.NewInt(2))
			},
			origLookup: func(a sdk.ValAddress) (stakingtypes.Validator, error) { return bondedVal(a, 30), nil },
		},
		{
			name: "FeeRefund not-bonded original",
			load: loadFeeSnapshot,
			run: func(k keeper.Keeper, ctx sdk.Context, hashId []byte) (math.Int, math.Int, error) {
				return k.FeeRefund(ctx, hashId, math.NewInt(2))
			},
			origLookup: func(a sdk.ValAddress) (stakingtypes.Validator, error) {
				val := bondedVal(a, 29)
				val.Status = stakingtypes.Unbonded
				return val, nil
			},
		},
		{
			name: "FeeRefund missing original",
			load: loadFeeSnapshot,
			run: func(k keeper.Keeper, ctx sdk.Context, hashId []byte) (math.Int, math.Int, error) {
				return k.FeeRefund(ctx, hashId, math.NewInt(2))
			},
			origLookup: missingOrig,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runScanFallbackFullBond(t, tc.load, tc.run, tc.origLookup)
		})
	}
}

func TestReturnSlashedTokensUnbondedOriginalRefundsUnbonded(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	origAddr := sdk.ValAddress(sample.AccAddressBytes())
	setDisputedSnapshot(t, k, ctx, []byte("hashId"), 2, origin(delAddr, origAddr, 2))
	orig := bondedVal(origAddr, 100)
	orig.Status = stakingtypes.Unbonded
	sk.On("GetValidator", ctx, origAddr).Return(orig, nil)
	delegateMock(sk, ctx, delAddr, math.NewInt(2), stakingtypes.Unbonded, orig)

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), math.ZeroInt())
	require.NoError(t, err)
	require.True(t, bondedReturn.IsZero())
	require.Equal(t, math.NewInt(2), unbondedReturn)
	sk.AssertNotCalled(t, "GetBondedValidatorsByPower", mock.Anything)
	sk.AssertNotCalled(t, "TotalBondedTokens", mock.Anything)
}

func TestUserTxOverflowHonorsMaxUnbondingEntries(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T) error
	}{
		{
			name: "FeeRefund",
			run: func(t *testing.T) error {
				t.Helper()
				k, sk, _, _, _, ctx, _ := setupKeeper(t)
				delAddr := sample.AccAddressBytes()
				origAddr := sdk.ValAddress(sample.AccAddressBytes())
				hashId := []byte("hashId")
				setFeeSnapshot(t, k, ctx, hashId, 2, origin(delAddr, origAddr, 2))
				setupCapSplitBond(t, sk, ctx, delAddr, origAddr)
				noDelegations(sk, ctx, delAddr)
				sk.On("HasMaxUnbondingDelegationEntries", ctx, delAddr, origAddr).Return(true, nil)
				_, _, err := k.FeeRefund(ctx, hashId, math.NewInt(2))
				return err
			},
		},
		{
			name: "AddAmountToStake",
			run: func(t *testing.T) error {
				t.Helper()
				k, sk, _, _, _, ctx, _ := setupKeeper(t)
				acc := sample.AccAddressBytes()
				origAddr := sdk.ValAddress(sample.AccAddressBytes())
				amt := math.NewInt(2)
				setupCapSplitBond(t, sk, ctx, acc, origAddr)
				sk.On("HasMaxUnbondingDelegationEntries", ctx, acc, origAddr).Return(true, nil)
				withBondedDelegation(sk, ctx, acc, origAddr, amt)
				_, _, err := k.AddAmountToStake(ctx, acc, amt)
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorIs(t, tc.run(t), stakingtypes.ErrMaxUnbondingDelegationEntries)
		})
	}
}

func TestAddAmountToStakeScanFallback(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T) (k keeper.Keeper, ctx sdk.Context, acc sdk.AccAddress, amt math.Int)
	}{
		{
			name: "over-cap bonded delegation scans",
			setup: func(t *testing.T) (keeper.Keeper, sdk.Context, sdk.AccAddress, math.Int) {
				t.Helper()
				k, sk, _, _, _, ctx, _ := setupKeeper(t)
				acc := sample.AccAddressBytes()
				origAddr, otherAddr := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
				amt := math.NewInt(2)
				orig, other := bondedVal(origAddr, 30), bondedVal(otherAddr, 0)
				sk.On("GetValidator", ctx, origAddr).Return(orig, nil)
				mockBondedScan(sk, ctx, 100, other)
				delegateMock(sk, ctx, acc, amt, stakingtypes.Bonded, other)
				withBondedDelegation(sk, ctx, acc, origAddr, amt)
				return k, ctx, acc, amt
			},
		},
		{
			name: "no bonded delegation scans",
			setup: func(t *testing.T) (keeper.Keeper, sdk.Context, sdk.AccAddress, math.Int) {
				t.Helper()
				k, sk, _, _, _, ctx, _ := setupKeeper(t)
				acc := sample.AccAddressBytes()
				valAddr := sdk.ValAddress(sample.AccAddressBytes())
				amt := math.NewInt(2)
				val := bondedVal(valAddr, 0)
				mockBondedScan(sk, ctx, 100, val)
				delegateMock(sk, ctx, acc, amt, stakingtypes.Bonded, val)
				noDelegations(sk, ctx, acc)
				return k, ctx, acc, amt
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx, acc, amt := tc.setup(t)
			bondedReturn, unbondedReturn, err := k.AddAmountToStake(ctx, acc, amt)
			require.NoError(t, err)
			require.Equal(t, amt, bondedReturn)
			require.True(t, unbondedReturn.IsZero())
		})
	}
}

func TestAddAmountToStakeCallbackErrors(t *testing.T) {
	amt := math.NewInt(2)
	delCall := func(acc sdk.AccAddress, valAddr sdk.ValAddress) func(mock.Arguments) {
		return func(args mock.Arguments) {
			fn := args.Get(2).(func(stakingtypes.Delegation) bool)
			fn(stakingtypes.Delegation{DelegatorAddress: acc.String(), ValidatorAddress: valAddr.String(), Shares: amt.ToLegacyDec()})
		}
	}

	t.Run("unbonding overflow error propagates", func(t *testing.T) {
		k, sk, _, _, _, ctx, _ := setupKeeper(t)
		acc := sample.AccAddressBytes()
		origAddr := sdk.ValAddress(sample.AccAddressBytes())
		orig := bondedVal(origAddr, 29)
		sk.On("GetValidator", ctx, origAddr).Return(orig, nil)
		sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
		sk.On("Delegate", ctx, acc, math.OneInt(), stakingtypes.Bonded, orig, false).Return(math.LegacyZeroDec(), nil)
		sk.On("HasMaxUnbondingDelegationEntries", ctx, acc, origAddr).Return(false, nil)
		sk.On("UnbondingTime", ctx).Return(time.Duration(0), stakingtypes.ErrNoValidatorFound)
		sk.On("IterateDelegatorDelegations", ctx, acc, mock.Anything).Return(nil).Run(delCall(acc, origAddr))
		_, _, err := k.AddAmountToStake(ctx, acc, amt)
		require.ErrorIs(t, err, stakingtypes.ErrNoValidatorFound)
	})

	t.Run("validator lookup error propagates", func(t *testing.T) {
		k, sk, _, _, _, ctx, _ := setupKeeper(t)
		acc := sample.AccAddressBytes()
		valAddr := sdk.ValAddress(sample.AccAddressBytes())
		sk.On("GetValidator", ctx, valAddr).Return(stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound)
		sk.On("IterateDelegatorDelegations", ctx, acc, mock.Anything).Return(nil).Run(delCall(acc, valAddr))
		_, _, err := k.AddAmountToStake(ctx, acc, amt)
		require.ErrorIs(t, err, stakingtypes.ErrNoValidatorFound)
	})

	t.Run("Delegate error propagates", func(t *testing.T) {
		k, sk, _, _, _, ctx, _ := setupKeeper(t)
		acc := sample.AccAddressBytes()
		valAddr := sdk.ValAddress(sample.AccAddressBytes())
		val := bondedVal(valAddr, 1)
		sk.On("GetValidator", ctx, valAddr).Return(val, nil)
		sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
		sk.On("Delegate", ctx, acc, amt, stakingtypes.Bonded, val, false).Return(math.LegacyZeroDec(), stakingtypes.ErrNoValidatorFound)
		sk.On("IterateDelegatorDelegations", ctx, acc, mock.Anything).Return(nil).Run(delCall(acc, valAddr))
		_, _, err := k.AddAmountToStake(ctx, acc, amt)
		require.ErrorIs(t, err, stakingtypes.ErrNoValidatorFound)
	})
}

func TestReturnSlashedTokensCumulativeRestoration(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr1, delAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes()
	origAddr := sdk.ValAddress(sample.AccAddressBytes())
	setDisputedSnapshot(t, k, ctx, []byte("hashId"), 4,
		origin(delAddr1, origAddr, 2), origin(delAddr2, origAddr, 2))
	orig := bondedVal(origAddr, 28)
	sk.On("GetValidator", ctx, origAddr).Return(orig, nil).Once()
	sk.On("GetValidator", ctx, origAddr).Return(bondedVal(origAddr, 30), nil).Once()
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	noDelegations(sk, ctx, delAddr1)
	noDelegations(sk, ctx, delAddr2)
	sk.On("GetBondedValidatorsByPower", ctx).Return([]stakingtypes.Validator{bondedVal(origAddr, 30)}, nil)
	delegateMock(sk, ctx, delAddr1, math.NewInt(2), stakingtypes.Bonded, orig)
	unbondingMock(sk, ctx, delAddr2, origAddr, math.NewInt(2))

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), math.ZeroInt())
	require.NoError(t, err)
	require.Equal(t, math.NewInt(2), bondedReturn)
	require.Equal(t, math.NewInt(2), unbondedReturn)
}

func TestReturnSlashedTokensCumulativeDenominatorProjection(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr1, delAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes()
	origAddr, missingAddr, fallbackAddr := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	setDisputedSnapshot(t, k, ctx, []byte("hashId"), 4,
		origin(delAddr1, origAddr, 3), origin(delAddr2, missingAddr, 1))
	orig, fallback := bondedVal(origAddr, 27), bondedVal(fallbackAddr, 30)
	sk.On("GetValidator", ctx, origAddr).Return(orig, nil)
	sk.On("GetValidator", ctx, missingAddr).Return(stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound)
	mockBondedScan(sk, ctx, 100, fallback)
	noDelegations(sk, ctx, delAddr1)
	noDelegations(sk, ctx, delAddr2)
	delegateMock(sk, ctx, delAddr1, math.NewInt(3), stakingtypes.Bonded, orig)
	delegateMock(sk, ctx, delAddr2, math.NewInt(1), stakingtypes.Bonded, fallback)

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), math.ZeroInt())
	require.NoError(t, err)
	require.Equal(t, math.NewInt(4), bondedReturn)
	require.True(t, unbondedReturn.IsZero())
}

func TestReturnSlashedTokensMixedPoolReturn(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	bondedDel, unbondedDel := sample.AccAddressBytes(), sample.AccAddressBytes()
	bondedValAddr, unbondedValAddr := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	setDisputedSnapshot(t, k, ctx, []byte("hashId"), 4,
		origin(bondedDel, bondedValAddr, 2), origin(unbondedDel, unbondedValAddr, 2))
	bondedOrig := bondedVal(bondedValAddr, 0)
	unbondedOrig := bondedVal(unbondedValAddr, 100)
	unbondedOrig.Status = stakingtypes.Unbonded
	sk.On("GetValidator", ctx, bondedValAddr).Return(bondedOrig, nil)
	sk.On("GetValidator", ctx, unbondedValAddr).Return(unbondedOrig, nil)
	delegateMock(sk, ctx, bondedDel, math.NewInt(2), stakingtypes.Bonded, bondedOrig)
	delegateMock(sk, ctx, unbondedDel, math.NewInt(2), stakingtypes.Unbonded, unbondedOrig)

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), math.ZeroInt())
	require.NoError(t, err)
	require.Equal(t, math.NewInt(2), bondedReturn)
	require.Equal(t, math.NewInt(2), unbondedReturn)
	sk.AssertNotCalled(t, "GetBondedValidatorsByPower", mock.Anything)
}
