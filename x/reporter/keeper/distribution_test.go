package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestDivvyingTips(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	height := uint64(10)
	val1Address := sample.AccAddressBytes()
	vals := simtestutil.ConvertAddrsToValAddrs([]sdk.AccAddress{val1Address})
	val1 := vals[0]
	addr := sample.AccAddressBytes()
	addr2 := sample.AccAddressBytes()
	reporter1 := types.NewReporter(math.LegacyZeroDec(), math.OneInt(), "reporter_moniker")
	ctx = ctx.WithBlockHeight(int64(height))

	err := k.Reporters.Set(ctx, addr, reporter1)
	require.NoError(t, err)

	tokenOrigin1 := &types.TokenOriginInfo{
		DelegatorAddress: addr.Bytes(),
		ValidatorAddress: val1.Bytes(),
		Amount:           math.NewInt(1000 * 1e6),
	}

	tokenOrigin2 := &types.TokenOriginInfo{
		DelegatorAddress: addr2.Bytes(),
		ValidatorAddress: val1.Bytes(),
		Amount:           math.NewInt(1000 * 1e6),
	}
	tokenOrigins := []*types.TokenOriginInfo{tokenOrigin1, tokenOrigin2}
	total := tokenOrigin1.Amount.Add(tokenOrigin2.Amount)

	delegationAmounts := types.DelegationsAmounts{TokenOrigins: tokenOrigins, Total: total}

	err = k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(addr.Bytes(), height)), delegationAmounts)
	require.NoError(t, err)
}

func TestReturnSlashedTokens(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)

	delAddr1, delAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes()
	val1Address, val2Address := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	tokenOrigin1 := &types.TokenOriginInfo{
		DelegatorAddress: delAddr1,
		ValidatorAddress: val1Address,
		Amount:           math.NewIntWithDecimal(1000, 6),
	}

	tokenOrigin2 := &types.TokenOriginInfo{
		DelegatorAddress: delAddr2,
		ValidatorAddress: val2Address,
		Amount:           math.NewIntWithDecimal(1000, 6),
	}
	err := k.DisputedDelegationAmounts.Set(ctx, []byte("hashId"), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{tokenOrigin1, tokenOrigin2}, Total: math.NewIntWithDecimal(2000, 6),
	},
	)
	require.NoError(t, err)
	validator1 := stakingtypes.Validator{OperatorAddress: val1Address.String(), Status: stakingtypes.Bonded}
	validator2 := stakingtypes.Validator{OperatorAddress: val2Address.String(), Status: stakingtypes.Bonded}
	sk.On("GetValidator", ctx, val1Address).Return(validator1, nil)
	sk.On("GetValidator", ctx, val2Address).Return(validator2, nil)
	sk.On("Delegate", ctx, delAddr1, tokenOrigin1.Amount, stakingtypes.Bonded, validator1, false).Return(math.LegacyZeroDec(), nil)
	sk.On("Delegate", ctx, delAddr2, tokenOrigin2.Amount, stakingtypes.Bonded, validator2, false).Return(math.LegacyZeroDec(), nil)

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), math.ZeroInt())
	require.NoError(t, err)
	require.Equal(t, math.NewIntWithDecimal(2000, 6), bondedReturn)
	require.True(t, unbondedReturn.IsZero())
}

// TestReturnSlashedTokensExtraReturnScaled guards the AGAINST/winning-purse path:
// extraReturn (reporter-win fee upside) is distributed proportionally across the
// collected principal origins, and snapshot.Total remains the denominator. With a
// single origin equal to Total there is no truncation, so the bonded return is
// exactly Total + extraReturn.
func TestReturnSlashedTokensExtraReturnScaled(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	delAddr := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	principal := math.NewInt(1000)
	extra := math.NewInt(200)
	require.NoError(t, k.DisputedDelegationAmounts.Set(ctx, []byte("hashId"), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: delAddr, ValidatorAddress: valAddr, Amount: principal},
		},
		Total: principal,
	}))
	validator := stakingtypes.Validator{OperatorAddress: valAddr.String(), Status: stakingtypes.Bonded}
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	sk.On("Delegate", ctx, delAddr, principal.Add(extra), stakingtypes.Bonded, validator, false).Return(math.LegacyZeroDec(), nil)

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), extra)
	require.NoError(t, err)
	require.Equal(t, principal.Add(extra), bondedReturn)
	require.True(t, unbondedReturn.IsZero())
}

func TestReturnSlashedTokensExtraReturnDustDelegatedToOrigin(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	delAddr1, delAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes()
	valAddr1, valAddr2 := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	extra := math.OneInt()
	require.NoError(t, k.DisputedDelegationAmounts.Set(ctx, []byte("hashId"), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: delAddr1, ValidatorAddress: valAddr1, Amount: math.OneInt()},
			{DelegatorAddress: delAddr2, ValidatorAddress: valAddr2, Amount: math.NewInt(2)},
		},
		Total: math.NewInt(3),
	}))
	validator1 := stakingtypes.Validator{OperatorAddress: valAddr1.String(), Status: stakingtypes.Bonded}
	validator2 := stakingtypes.Validator{OperatorAddress: valAddr2.String(), Status: stakingtypes.Bonded}
	sk.On("GetValidator", ctx, valAddr1).Return(validator1, nil).Once()
	sk.On("GetValidator", ctx, valAddr2).Return(validator2, nil).Once()
	sk.On("Delegate", ctx, delAddr1, math.OneInt(), stakingtypes.Bonded, validator1, false).Return(math.LegacyZeroDec(), nil).Once()
	sk.On("Delegate", ctx, delAddr2, math.NewInt(3), stakingtypes.Bonded, validator2, false).Return(math.LegacyZeroDec(), nil).Once()

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), extra)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(4), bondedReturn)
	require.True(t, unbondedReturn.IsZero())
	sk.AssertExpectations(t)
}

// TestReturnSlashedTokensExtraReturnWithoutPrincipal guards the zero-collected
// edge: with no principal origins to distribute across, a positive extraReturn
// is refused with a distinct sentinel error (so the P1-2 guard pages operators
// via dispute_execution_failed rather than silently deferring). The snapshot is
// NOT removed on this error path so the retry under P1-2 can roll it back.
func TestReturnSlashedTokensExtraReturnWithoutPrincipal(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	require.NoError(t, k.DisputedDelegationAmounts.Set(ctx, []byte("hashId"), types.DelegationsAmounts{
		TokenOrigins: nil,
		Total:        math.ZeroInt(),
	}))

	_, _, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), math.NewInt(100))
	require.ErrorIs(t, err, types.ErrReturnExtraWithoutPrincipal)

	// snapshot must still be present (error path does not remove it)
	_, err = k.DisputedDelegationAmounts.Get(ctx, []byte("hashId"))
	require.NoError(t, err)

	// zero principal + zero extra returns success and removes the empty snapshot
	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), math.ZeroInt())
	require.NoError(t, err)
	require.True(t, bondedReturn.IsZero())
	require.True(t, unbondedReturn.IsZero())
	_, err = k.DisputedDelegationAmounts.Get(ctx, []byte("hashId"))
	require.ErrorIs(t, err, collections.ErrNotFound)
}

// TestReturnSlashedTokensRoutesToUnbondingWhenNoBondedDestination verifies the
// escape hatch in PR3: when the original validator is gone and every bonded
// fallback is over the cap, the origin's tokens are placed in the unbonding
// queue against the original validator address rather than deferring execution.
func TestReturnSlashedTokensRoutesToUnbondingWhenNoBondedDestination(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	require.NoError(t, k.DisputedDelegationAmounts.Set(ctx, []byte("hashId"), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: delAddr, ValidatorAddress: valAddr, Amount: math.NewInt(10)},
		},
		Total: math.NewInt(10),
	}))

	// original validator no longer exists -> cannot preserve-to-original
	sk.On("GetValidator", ctx, valAddr).Return(stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound).Once()

	// all bonded fallbacks are over the validator cap (40/100 and 50/100 both > 30%)
	overCap1 := bondedVal(sdk.ValAddress(sample.AccAddressBytes()), 40)
	overCap2 := bondedVal(sdk.ValAddress(sample.AccAddressBytes()), 50)
	mockBondedScan(sk, ctx, 100, overCap1, overCap2)
	noDelegations(sk, ctx, delAddr)

	// unbonding queue escape hatch
	sk.On("UnbondingTime", ctx).Return(21*24*time.Hour, nil).Once()
	sk.On("SetUnbondingDelegationEntry", ctx, delAddr, valAddr, ctx.BlockHeight(), mock.Anything, math.NewInt(10)).Return(stakingtypes.UnbondingDelegation{}, nil).Once()
	sk.On("InsertUBDQueue", ctx, stakingtypes.UnbondingDelegation{}, mock.Anything).Return(nil).Once()

	bondedReturn, unbondedReturn, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), math.ZeroInt())
	require.NoError(t, err)
	require.True(t, bondedReturn.IsZero())
	require.Equal(t, math.NewInt(10), unbondedReturn)

	// snapshot removed after successful return
	_, err = k.DisputedDelegationAmounts.Get(ctx, []byte("hashId"))
	require.ErrorIs(t, err, collections.ErrNotFound)
}

// TestReturnSlashedTokensPositiveTotalWithoutOrigins rejects malformed snapshots
// where Total is positive but there are no origins to distribute across.
func TestReturnSlashedTokensPositiveTotalWithoutOrigins(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	require.NoError(t, k.DisputedDelegationAmounts.Set(ctx, []byte("hashId"), types.DelegationsAmounts{
		TokenOrigins: nil,
		Total:        math.NewInt(100),
	}))

	_, _, err := k.ReturnSlashedTokens(ctx, []byte("hashId"), math.ZeroInt())
	require.ErrorIs(t, err, types.ErrMalformedDelegationSnapshot)

	_, err = k.DisputedDelegationAmounts.Get(ctx, []byte("hashId"))
	require.NoError(t, err)
}

func TestFeeRefund(t *testing.T) {
	// set fee refund
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	delAddr1, delAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes()
	valAddr1, valAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes()
	tokenOrigin1 := &types.TokenOriginInfo{
		DelegatorAddress: delAddr1,
		ValidatorAddress: valAddr1,
		Amount:           math.NewInt(1000 * 1e6),
	}

	tokenOrigin2 := &types.TokenOriginInfo{
		DelegatorAddress: delAddr2,
		ValidatorAddress: valAddr2,
		Amount:           math.NewInt(1000 * 1e6),
	}
	err := k.FeePaidFromStake.Set(ctx, []byte("hashId"), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{tokenOrigin1, tokenOrigin2},
		Total:        math.NewInt(2000 * 1e6),
	})
	amtminusburn := math.NewInt(1800 * 1e6)
	sharesperrefund := math.NewInt(900 * 1e6)
	require.NoError(t, err)
	validator1 := stakingtypes.Validator{OperatorAddress: valAddr1.String(), Status: stakingtypes.Bonded}
	validator2 := stakingtypes.Validator{OperatorAddress: valAddr2.String(), Status: stakingtypes.Bonded}
	sk.On("GetValidator", ctx, sdk.ValAddress(valAddr1)).Return(validator1, nil)
	sk.On("GetValidator", ctx, sdk.ValAddress(valAddr2)).Return(validator2, nil)
	sk.On("Delegate", ctx, delAddr1, sharesperrefund, stakingtypes.Bonded, validator1, false).Return(math.LegacyZeroDec(), nil)
	sk.On("Delegate", ctx, delAddr2, sharesperrefund, stakingtypes.Bonded, validator2, false).Return(math.LegacyZeroDec(), nil)
	bondedReturn, unbondedReturn, err := k.FeeRefund(ctx, []byte("hashId"), delAddr1, amtminusburn)
	require.NoError(t, err)
	require.Equal(t, amtminusburn, bondedReturn)
	require.True(t, unbondedReturn.IsZero())
}

func TestFeeRefundAssignsRoundingDust(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	delAddr1, delAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes()
	valAddr1, valAddr2 := sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	require.NoError(t, k.FeePaidFromStake.Set(ctx, []byte("hashId"), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: delAddr1, ValidatorAddress: valAddr1, Amount: math.OneInt()},
			{DelegatorAddress: delAddr2, ValidatorAddress: valAddr2, Amount: math.OneInt()},
		},
		Total: math.NewInt(2),
	}))

	validator1 := stakingtypes.Validator{OperatorAddress: valAddr1.String(), Status: stakingtypes.Bonded}
	sk.On("GetValidator", ctx, valAddr1).Return(validator1, nil)
	sk.On("Delegate", ctx, delAddr1, math.OneInt(), stakingtypes.Bonded, validator1, false).Return(math.LegacyZeroDec(), nil)

	bondedReturn, unbondedReturn, err := k.FeeRefund(ctx, []byte("hashId"), delAddr1, math.OneInt())
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), bondedReturn)
	require.True(t, unbondedReturn.IsZero())
	sk.AssertExpectations(t)
}

func TestFeeRefundRejectsMalformedPositiveTotalWithoutOrigins(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	hashId := []byte("hashId")
	payer := sample.AccAddressBytes()
	require.NoError(t, k.FeePaidFromStake.Set(ctx, hashId, types.DelegationsAmounts{
		Total: math.OneInt(),
	}))

	_, _, err := k.FeeRefund(ctx, hashId, payer, math.OneInt())
	require.ErrorIs(t, err, types.ErrMalformedDelegationSnapshot)
	has, hasErr := k.FeePaidFromStake.Has(ctx, hashId)
	require.NoError(t, hasErr)
	require.True(t, has)
}

func TestFeeRefundRejectsMalformedOriginTotalMismatch(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	hashId := []byte("hashId")
	payer := sample.AccAddressBytes()
	require.NoError(t, k.FeePaidFromStake.Set(ctx, hashId, types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: payer,
				ValidatorAddress: sdk.ValAddress(sample.AccAddressBytes()),
				Amount:           math.NewInt(1000),
			},
		},
		Total: math.NewInt(999),
	}))

	_, _, err := k.FeeRefund(ctx, hashId, payer, math.NewInt(999))
	require.ErrorIs(t, err, types.ErrMalformedDelegationSnapshot)
	has, hasErr := k.FeePaidFromStake.Has(ctx, hashId)
	require.NoError(t, hasErr)
	require.True(t, has)
}

func TestAddAmountToStake(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)

	acc := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	amt := math.NewInt(1000 * 1e6)

	validator := stakingtypes.Validator{OperatorAddress: valAddr.String(), Status: stakingtypes.Bonded}

	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	sk.On("Delegate", ctx, acc, amt, stakingtypes.Bonded, validator, false).Return(math.LegacyZeroDec(), nil)
	// the account has an existing bonded delegation, so winnings go to that
	// validator and the fallback scan is not reached.
	sk.On("IterateDelegatorDelegations", ctx, acc, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		fn(stakingtypes.Delegation{
			DelegatorAddress: acc.String(),
			ValidatorAddress: valAddr.String(),
			Shares:           amt.ToLegacyDec(),
		})
	})

	bondedReturn, unbondedReturn, err := k.AddAmountToStake(ctx, acc, amt)
	require.NoError(t, err)
	require.Equal(t, amt, bondedReturn)
	require.True(t, unbondedReturn.IsZero())
}
