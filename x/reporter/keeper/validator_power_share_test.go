package keeper_test

import (
	"testing"

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

// --- shared fixtures for validator power-share keeper tests ---

// powerIterator is a minimal staking power-store iterator yielding a fixed list
// of validator addresses, so GetBondedValidators scans can be made deterministic
// without a real staking store.
type powerIterator struct {
	values [][]byte
	idx    int
}

func (i *powerIterator) Domain() (start, end []byte) { return nil, nil }
func (i *powerIterator) Valid() bool                 { return i.idx < len(i.values) }
func (i *powerIterator) Next()                       { i.idx++ }
func (i *powerIterator) Key() []byte                 { return nil }
func (i *powerIterator) Value() []byte               { return i.values[i.idx] }
func (i *powerIterator) Error() error                { return nil }
func (i *powerIterator) Close() error                { return nil }

func bondedVal(addr sdk.ValAddress, tokens int64) stakingtypes.Validator {
	return stakingtypes.Validator{
		OperatorAddress:   addr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            math.NewInt(tokens),
		DelegatorShares:   math.NewInt(tokens).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
}

func disableValidatorPowerCap(t *testing.T, k keeper.Keeper, ctx sdk.Context) {
	t.Helper()
	params := types.DefaultParams()
	params.MaxValidatorPowerShare = math.LegacyOneDec()
	require.NoError(t, k.Params.Set(ctx, params))
}

// origin builds a single TokenOriginInfo for the refund-snapshot helpers.
func origin(del sdk.AccAddress, val sdk.ValAddress, amount int64) *types.TokenOriginInfo {
	return &types.TokenOriginInfo{DelegatorAddress: del, ValidatorAddress: val, Amount: math.NewInt(amount)}
}

// setDisputedSnapshot loads a DisputedDelegationAmounts snapshot used by ReturnSlashedTokens.
func setDisputedSnapshot(t *testing.T, k keeper.Keeper, ctx sdk.Context, hashId []byte, total int64, origins ...*types.TokenOriginInfo) {
	t.Helper()
	require.NoError(t, k.DisputedDelegationAmounts.Set(ctx, hashId, types.DelegationsAmounts{TokenOrigins: origins, Total: math.NewInt(total)}))
}

// setFeeSnapshot loads a FeePaidFromStake snapshot used by FeeRefund.
func setFeeSnapshot(t *testing.T, k keeper.Keeper, ctx sdk.Context, hashId []byte, total int64, origins ...*types.TokenOriginInfo) {
	t.Helper()
	require.NoError(t, k.FeePaidFromStake.Set(ctx, hashId, types.DelegationsAmounts{TokenOrigins: origins, Total: math.NewInt(total)}))
}

// mockBondedScan wires the staking mock for a bounded bonded-validator scan:
// TotalBondedTokens, the power-store iterator over the validators, and a
// GetValidator lookup for each.
func mockBondedScan(sk *mocks.StakingKeeper, ctx sdk.Context, totalBonded int64, vals ...stakingtypes.Validator) {
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(totalBonded), nil)
	addresses := make([][]byte, 0, len(vals))
	for _, val := range vals {
		addr, err := sdk.ValAddressFromBech32(val.OperatorAddress)
		if err != nil {
			panic(err)
		}
		addresses = append(addresses, addr)
		sk.On("GetValidator", ctx, addr).Return(val, nil)
	}
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&powerIterator{values: addresses}, nil)
}

// delegateMock wires a single Delegate expectation.
func delegateMock(sk *mocks.StakingKeeper, ctx sdk.Context, del sdk.AccAddress, amt math.Int, tokenSrc stakingtypes.BondStatus, val stakingtypes.Validator) {
	sk.On("Delegate", ctx, del, amt, tokenSrc, val, false).Return(math.LegacyZeroDec(), nil)
}

// --- CheckValidatorPowerShareDelegation ---

func TestCheckValidatorPowerShareDelegation(t *testing.T) {
	testCases := []struct {
		name    string
		tokens  int64
		amount  int64
		wantErr error
	}{
		{"bonded over cap rejects", 29, 2, types.ErrExceedsMaxValidatorPowerShare}, // 29+2=31 vs 30% of 102=30.6
		{"bonded under cap succeeds", 29, 1, nil},                                  // 30 vs 30% of 101=30.3
		{"zero amount succeeds", 30, 0, nil},                                       // short-circuits before cap math
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
	// share >= 1 disables the check, so no TotalBondedTokens call is needed.
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	disableValidatorPowerCap(t, k, ctx)
	val := bondedVal(sdk.ValAddress(sample.AccAddressBytes()), 100)
	require.NoError(t, k.CheckValidatorPowerShareDelegation(ctx, val, math.NewInt(100)))
}

func TestCheckValidatorPowerShareDelegationMissingParams(t *testing.T) {
	// pre-upgrade state with no reporter params disables the check.
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	require.NoError(t, k.Params.Remove(ctx))
	val := bondedVal(sdk.ValAddress(sample.AccAddressBytes()), 100)
	require.NoError(t, k.CheckValidatorPowerShareDelegation(ctx, val, math.NewInt(100)))
}

func TestCheckValidatorPowerShareDelegationNonBonded(t *testing.T) {
	// A non-bonded destination returns nil: this helper only checks immediate
	// active bonded acquisition. ReturnSlashedTokens must redirect not-bonded
	// originals before calling into the cap-checked bonded path.
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	val := bondedVal(valAddr, 100)
	val.Status = stakingtypes.Unbonded
	sk.On("GetValidator", ctx, valAddr).Return(val, nil)
	require.NoError(t, k.CheckValidatorPowerShareDelegation(ctx, val, math.NewInt(100)))
}

// --- PickUnderCapBondedValidator ---

func TestPickUnderCapBondedValidatorScans(t *testing.T) {
	// val1 (29 tokens) is over the cap for a 2-token delegation; val2 (0) is
	// under it. The scan must skip val1 and return val2.
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	val1, val2 := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	mockBondedScan(sk, ctx, 100, bondedVal(val1, 29), bondedVal(val2, 0))
	got, err := k.PickUnderCapBondedValidatorExported(ctx, math.NewInt(2))
	require.NoError(t, err)
	require.Equal(t, val2.String(), got.OperatorAddress)
}

func TestPickUnderCapBondedValidatorAllFail(t *testing.T) {
	// both scanned validators are over the cap; the scan exhausts and returns
	// ErrExceedsMaxValidatorPowerShare rather than panicking.
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	val1, val2 := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	mockBondedScan(sk, ctx, 100, bondedVal(val1, 100), bondedVal(val2, 100))
	_, err := k.PickUnderCapBondedValidatorExported(ctx, math.NewInt(2))
	require.ErrorIs(t, err, types.ErrExceedsMaxValidatorPowerShare)
}

func TestPickUnderCapBondedValidatorEmpty(t *testing.T) {
	// no bonded validators returns an error, not a panic.
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&powerIterator{values: nil}, nil)
	_, err := k.PickUnderCapBondedValidatorExported(ctx, math.NewInt(2))
	require.ErrorIs(t, err, types.ErrExceedsMaxValidatorPowerShare)
}

func TestPickUnderCapBondedValidatorGetBondedValidatorsError(t *testing.T) {
	// a GetBondedValidators/iterator error propagates rather than being masked.
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	sk.On("ValidatorsPowerStoreIterator", ctx).Return((*powerIterator)(nil), stakingtypes.ErrNoValidatorFound)
	_, err := k.PickUnderCapBondedValidatorExported(ctx, math.NewInt(2))
	require.ErrorIs(t, err, stakingtypes.ErrNoValidatorFound)
}

// --- BondedValidatorForDelegation ---

func TestBondedValidatorForDelegationPreservesPreferred(t *testing.T) {
	// preferred bonded validator is under the cap, so it is preserved without scanning.
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	preferred := bondedVal(sdk.ValAddress(sample.AccAddressBytes()), 0)
	sk.On("GetValidator", ctx, sdk.ValAddress(sample.AccAddressBytes())).Return(preferred, nil).Maybe()
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	got, err := k.BondedValidatorForDelegationExported(ctx, preferred, math.NewInt(2))
	require.NoError(t, err)
	require.Equal(t, preferred.OperatorAddress, got.OperatorAddress)
}

func TestBondedValidatorForDelegationFallsBack(t *testing.T) {
	// preferred bonded validator is over the cap; fall back to an under-cap one.
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	prefAddr, otherAddr := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	preferred, other := bondedVal(prefAddr, 100), bondedVal(otherAddr, 0)
	sk.On("GetValidator", ctx, prefAddr).Return(preferred, nil)
	mockBondedScan(sk, ctx, 100, other)
	got, err := k.BondedValidatorForDelegationExported(ctx, preferred, math.NewInt(2))
	require.NoError(t, err)
	require.Equal(t, other.OperatorAddress, got.OperatorAddress)
}

// --- WithdrawTip ---

func TestWithdrawTipValidatorPowerShare(t *testing.T) {
	// Over-cap destination hard-rejects before staking tips: no scan fallback,
	// selector tips remain stored, and neither Delegate nor the bank transfer run.
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

// --- ReturnSlashedTokens scan paths ---

// TestReturnSlashedTokensScansFallback covers both branches that trigger a
// bonded fallback scan: an over-cap bonded original and a missing original. In
// both cases the keeper scans to an under-cap bonded validator and delegates as
// Bonded, returning the full amount through the bonded pool.
func TestReturnSlashedTokensScansFallback(t *testing.T) {
	origAddr, otherAddr := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	other := bondedVal(otherAddr, 0)
	cases := []struct {
		name       string
		origLookup func(sdk.ValAddress) (stakingtypes.Validator, error)
	}{
		{"bonded original over cap scans", func(a sdk.ValAddress) (stakingtypes.Validator, error) { return bondedVal(a, 29), nil }},
		{"missing original scans", func(sdk.ValAddress) (stakingtypes.Validator, error) {
			return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, sk, _, _, _, ctx, _ := setupKeeper(t)
			delAddr := sample.AccAddressBytes()
			setDisputedSnapshot(t, k, ctx, []byte("hashId"), 2, origin(delAddr, origAddr, 2))
			sk.On("GetValidator", ctx, origAddr).Return(tc.origLookup(origAddr))
			mockBondedScan(sk, ctx, 100, other)
			delegateMock(sk, ctx, delAddr, math.NewInt(2), stakingtypes.Bonded, other)
			bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, math.NewInt(2), []byte("hashId"))
			require.NoError(t, err)
			require.Equal(t, math.NewInt(2), bondedReturn)
			require.True(t, unbondedReturn.IsZero())
		})
	}
}

func TestReturnSlashedTokensUnbondedOriginalRefundsUnbonded(t *testing.T) {
	// An existing but not-bonded original validator is refunded to itself with
	// tokenSrc=Unbonded. No bonded fallback scan occurs, the validator cap is
	// intentionally not enforced (no immediate active bonded stake is created),
	// and the amount is returned through the unbonded pool so the dispute caller
	// routes it to NotBondedPoolName. This holds even when every bonded
	// validator would be over the cap.
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	origAddr := sdk.ValAddress(sample.AccAddressBytes())
	setDisputedSnapshot(t, k, ctx, []byte("hashId"), 2, origin(delAddr, origAddr, 2))
	orig := bondedVal(origAddr, 100)
	orig.Status = stakingtypes.Unbonded
	sk.On("GetValidator", ctx, origAddr).Return(orig, nil)
	delegateMock(sk, ctx, delAddr, math.NewInt(2), stakingtypes.Unbonded, orig)

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, math.NewInt(2), []byte("hashId"))
	require.NoError(t, err)
	require.True(t, bondedReturn.IsZero())
	require.Equal(t, math.NewInt(2), unbondedReturn)
	sk.AssertNotCalled(t, "ValidatorsPowerStoreIterator", mock.Anything)
	sk.AssertNotCalled(t, "TotalBondedTokens", mock.Anything)
}

// --- FeeRefund scan paths ---

// TestFeeRefundScansFallback covers FeeRefund branches that trigger a bonded
// fallback scan: an over-cap bonded original, a missing original, and a
// not-bonded original.
func TestFeeRefundScansFallback(t *testing.T) {
	origAddr, otherAddr := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	other := bondedVal(otherAddr, 0)
	cases := []struct {
		name       string
		origLookup func(sdk.ValAddress) (stakingtypes.Validator, error)
	}{
		{"bonded original over cap scans", func(a sdk.ValAddress) (stakingtypes.Validator, error) { return bondedVal(a, 29), nil }},
		{"not-bonded original scans", func(a sdk.ValAddress) (stakingtypes.Validator, error) {
			val := bondedVal(a, 29)
			val.Status = stakingtypes.Unbonded
			return val, nil
		}},
		{"missing original scans", func(sdk.ValAddress) (stakingtypes.Validator, error) {
			return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, sk, _, _, _, ctx, _ := setupKeeper(t)
			delAddr := sample.AccAddressBytes()
			setFeeSnapshot(t, k, ctx, []byte("hashId"), 2, origin(delAddr, origAddr, 2))
			sk.On("GetValidator", ctx, origAddr).Return(tc.origLookup(origAddr))
			mockBondedScan(sk, ctx, 100, other)
			delegateMock(sk, ctx, delAddr, math.NewInt(2), stakingtypes.Bonded, other)
			require.NoError(t, k.FeeRefund(ctx, []byte("hashId"), math.NewInt(2)))
		})
	}
}

// --- AddAmountToStake ---

func TestAddAmountToStakeOverCapScans(t *testing.T) {
	// The account's existing bonded delegation is over the cap, so the keeper
	// scans to an under-cap bonded validator instead.
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	acc := sample.AccAddressBytes()
	origAddr, otherAddr := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	amt := math.NewInt(2)
	orig, other := bondedVal(origAddr, 29), bondedVal(otherAddr, 0)
	sk.On("GetValidator", ctx, origAddr).Return(orig, nil)
	mockBondedScan(sk, ctx, 100, other)
	delegateMock(sk, ctx, acc, amt, stakingtypes.Bonded, other)
	sk.On("IterateDelegatorDelegations", ctx, acc, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		fn(stakingtypes.Delegation{DelegatorAddress: acc.String(), ValidatorAddress: origAddr.String(), Shares: amt.ToLegacyDec()})
	})
	require.NoError(t, k.AddAmountToStake(ctx, acc, amt))
}

func TestAddAmountToStakeNoBondedDelegationScans(t *testing.T) {
	// No bonded delegation: the keeper scans for an under-cap bonded validator.
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	acc := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	amt := math.NewInt(2)
	val := bondedVal(valAddr, 0)
	mockBondedScan(sk, ctx, 100, val)
	delegateMock(sk, ctx, acc, amt, stakingtypes.Bonded, val)
	sk.On("IterateDelegatorDelegations", ctx, acc, mock.Anything).Return(nil)
	require.NoError(t, k.AddAmountToStake(ctx, acc, amt))
}

// TestAddAmountToStakeCallbackErrors covers iterator callback failure handling:
// cap, validator-lookup, and Delegate errors must propagate instead of being
// swallowed while iteration silently succeeds.
func TestAddAmountToStakeCallbackErrors(t *testing.T) {
	amt := math.NewInt(2)
	delCall := func(acc sdk.AccAddress, valAddr sdk.ValAddress) func(mock.Arguments) {
		return func(args mock.Arguments) {
			fn := args.Get(2).(func(stakingtypes.Delegation) bool)
			fn(stakingtypes.Delegation{DelegatorAddress: acc.String(), ValidatorAddress: valAddr.String(), Shares: amt.ToLegacyDec()})
		}
	}

	t.Run("cap error on existing bonded delegation scans and exhausts", func(t *testing.T) {
		k, sk, _, _, _, ctx, _ := setupKeeper(t)
		acc := sample.AccAddressBytes()
		origAddr := sdk.ValAddress(sample.AccAddressBytes())
		orig := bondedVal(origAddr, 29) // over cap for a 2-token add
		sk.On("GetValidator", ctx, origAddr).Return(orig, nil)
		// the scan finds only the over-cap original, so it exhausts.
		sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
		sk.On("ValidatorsPowerStoreIterator", ctx).Return(&powerIterator{values: [][]byte{origAddr}}, nil)
		sk.On("IterateDelegatorDelegations", ctx, acc, mock.Anything).Return(nil).Run(delCall(acc, origAddr))
		err := k.AddAmountToStake(ctx, acc, amt)
		require.ErrorIs(t, err, types.ErrExceedsMaxValidatorPowerShare)
	})

	t.Run("validator lookup error propagates", func(t *testing.T) {
		k, sk, _, _, _, ctx, _ := setupKeeper(t)
		acc := sample.AccAddressBytes()
		valAddr := sdk.ValAddress(sample.AccAddressBytes())
		sk.On("GetValidator", ctx, valAddr).Return(stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound)
		sk.On("IterateDelegatorDelegations", ctx, acc, mock.Anything).Return(nil).Run(delCall(acc, valAddr))
		err := k.AddAmountToStake(ctx, acc, amt)
		require.ErrorIs(t, err, stakingtypes.ErrNoValidatorFound)
	})

	t.Run("Delegate error propagates", func(t *testing.T) {
		k, sk, _, _, _, ctx, _ := setupKeeper(t)
		acc := sample.AccAddressBytes()
		valAddr := sdk.ValAddress(sample.AccAddressBytes())
		val := bondedVal(valAddr, 0) // under cap, so Delegate is reached
		sk.On("GetValidator", ctx, valAddr).Return(val, nil)
		sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
		sk.On("Delegate", ctx, acc, amt, stakingtypes.Bonded, val, false).Return(math.LegacyZeroDec(), stakingtypes.ErrNoValidatorFound)
		sk.On("IterateDelegatorDelegations", ctx, acc, mock.Anything).Return(nil).Run(delCall(acc, valAddr))
		err := k.AddAmountToStake(ctx, acc, amt)
		require.ErrorIs(t, err, stakingtypes.ErrNoValidatorFound)
	})
}

// --- cumulative enforcement ---

// TestReturnSlashedTokensCumulativeEnforcement refunds two origins to the same
// bonded validator. The first preserves the original (under cap); the second,
// re-fetched against the now-larger validator state, exceeds the cap and scans
// to the under-cap fallback. Guards both the per-origin GetValidator re-fetch
// (numerator) and the projected bonded delta (denominator).
func TestReturnSlashedTokensCumulativeEnforcement(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr1, delAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes()
	origAddr, otherAddr := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	setDisputedSnapshot(t, k, ctx, []byte("hashId"), 4,
		origin(delAddr1, origAddr, 2), origin(delAddr2, origAddr, 2))
	orig := bondedVal(origAddr, 28) // first 2-token refund -> 30 (under cap, allowed)
	other := bondedVal(otherAddr, 0)
	// GetValidator is re-fetched per origin; the second reflects the first refund
	// (28 + 2 = 30), so a second 2-token refund -> 32, over cap, scans to fallback.
	sk.On("GetValidator", ctx, origAddr).Return(orig, nil).Once()
	sk.On("GetValidator", ctx, origAddr).Return(bondedVal(origAddr, 30), nil).Once()
	mockBondedScan(sk, ctx, 100, other)
	delegateMock(sk, ctx, delAddr1, math.NewInt(2), stakingtypes.Bonded, orig)
	delegateMock(sk, ctx, delAddr2, math.NewInt(2), stakingtypes.Bonded, other)

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, math.NewInt(4), []byte("hashId"))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(4), bondedReturn)
	require.True(t, unbondedReturn.IsZero())
}

// TestReturnSlashedTokensCumulativeDenominatorProjection guards the denominator
// projection. origin 1 refunds 3 to a validator at 27 (->30, under cap, projected
// delta 3). origin 2 refunds 1 more (re-fetched at 30 ->31). With projected
// denominator 100+3+1=104, 31/104 = 29.8% is under cap so origin 2 preserves the
// original. Without the projection the second origin would see 31/101 = 30.7%,
// be over the cap, and scan to an unmocked fallback — failing this test.
func TestReturnSlashedTokensCumulativeDenominatorProjection(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr1, delAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes()
	origAddr := sdk.ValAddress(sample.AccAddressBytes())
	setDisputedSnapshot(t, k, ctx, []byte("hashId"), 4,
		origin(delAddr1, origAddr, 3), origin(delAddr2, origAddr, 1))
	orig1, orig2 := bondedVal(origAddr, 27), bondedVal(origAddr, 30) // re-fetched after origin 1's 3-token refund
	sk.On("GetValidator", ctx, origAddr).Return(orig1, nil).Once()
	sk.On("GetValidator", ctx, origAddr).Return(orig2, nil).Once()
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	delegateMock(sk, ctx, delAddr1, math.NewInt(3), stakingtypes.Bonded, orig1)
	delegateMock(sk, ctx, delAddr2, math.NewInt(1), stakingtypes.Bonded, orig2)

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, math.NewInt(4), []byte("hashId"))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(4), bondedReturn)
	require.True(t, unbondedReturn.IsZero())
}

// TestReturnSlashedTokensMixedPoolReturn guards that a bonded origin returns
// through the bonded pool and an unbonded original returns through the unbonded
// pool (delegated to the unbonded original with tokenSrc=Unbonded), so the
// dispute caller routes each origin's coins to the matching pool.
func TestReturnSlashedTokensMixedPoolReturn(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	bondedDel, unbondedDel := sample.AccAddressBytes(), sample.AccAddressBytes()
	bondedValAddr, unbondedValAddr := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	setDisputedSnapshot(t, k, ctx, []byte("hashId"), 4,
		origin(bondedDel, bondedValAddr, 2), origin(unbondedDel, unbondedValAddr, 2))
	bondedOrig := bondedVal(bondedValAddr, 0) // under cap
	unbondedOrig := bondedVal(unbondedValAddr, 100)
	unbondedOrig.Status = stakingtypes.Unbonded
	sk.On("GetValidator", ctx, bondedValAddr).Return(bondedOrig, nil)
	sk.On("GetValidator", ctx, unbondedValAddr).Return(unbondedOrig, nil)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	delegateMock(sk, ctx, bondedDel, math.NewInt(2), stakingtypes.Bonded, bondedOrig)
	delegateMock(sk, ctx, unbondedDel, math.NewInt(2), stakingtypes.Unbonded, unbondedOrig)

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, math.NewInt(4), []byte("hashId"))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(2), bondedReturn)
	require.Equal(t, math.NewInt(2), unbondedReturn)
	// no bonded fallback scan is needed for the unbonded original
	sk.AssertNotCalled(t, "ValidatorsPowerStoreIterator", mock.Anything)
}
