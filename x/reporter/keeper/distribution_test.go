package keeper_test

import (
	"encoding/binary"
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

func TestSettleReporterZeroTotalCreditsReporter(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()
	reward := math.LegacyNewDec(1_000_000)

	require.NoError(t, k.ReporterPeriodData.Set(ctx, reporter, types.PeriodRewardData{
		Selectors:    nil,
		Total:        math.ZeroInt(),
		RewardAmount: reward,
		Hash:         []byte("empty"),
	}))

	require.NoError(t, k.SettleReporter(ctx, reporter))

	tips, err := k.SelectorTips.Get(ctx, reporter)
	require.NoError(t, err)
	require.True(t, tips.Equal(reward))

	period, err := k.ReporterPeriodData.Get(ctx, reporter)
	require.NoError(t, err)
	require.True(t, period.RewardAmount.IsZero())
}

func TestDistributeQueueItemZeroTotalCreditsReporter(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()
	reward := math.LegacyNewDec(500_000)

	require.NoError(t, k.DistributionQueue.Set(ctx, 0, types.DistributionQueueItem{
		Reporter:     reporter,
		Selectors:    nil,
		Total:        math.ZeroInt(),
		RewardAmount: reward,
	}))
	require.NoError(t, k.DistributionQueueCounter.Set(ctx, types.DistributionQueueCounter{Head: 0, Tail: 1}))

	require.NoError(t, k.ProcessDistributionQueue(ctx, 10))

	tips, err := k.SelectorTips.Get(ctx, reporter)
	require.NoError(t, err)
	require.True(t, tips.Equal(reward))

	counter, err := k.DistributionQueueCounter.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), counter.Head)
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

func feePaidFromStakePayerKey(hashId []byte, payer sdk.AccAddress) []byte {
	const prefix = "fee-paid-from-stake-payer\x00"
	key := make([]byte, 0, len(prefix)+4+len(hashId)+len(payer))
	key = append(key, prefix...)
	key = binary.BigEndian.AppendUint32(key, uint32(len(hashId)))
	key = append(key, hashId...)
	key = append(key, payer...)
	return key
}

func TestReturnFeeExcessFullReturnRemovesTracker(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	payer := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	hashId := []byte("hash")
	key := feePaidFromStakePayerKey(hashId, payer)
	require.NoError(t, k.FeePaidFromStake.Set(ctx, key, types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{origin(payer, valAddr, 100)},
		Total:        math.NewInt(100),
	}))
	validator := stakingtypes.Validator{OperatorAddress: valAddr.String(), Status: stakingtypes.Bonded}
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	sk.On("Delegate", ctx, payer, math.NewInt(100), stakingtypes.Bonded, validator, false).Return(math.LegacyZeroDec(), nil)

	bonded, unbonded, err := k.ReturnFeeExcess(ctx, hashId, payer, math.NewInt(100), math.ZeroInt())
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100), bonded)
	require.True(t, unbonded.IsZero())
	has, hasErr := k.FeePaidFromStake.Has(ctx, key)
	require.NoError(t, hasErr)
	require.False(t, has)
	sk.AssertExpectations(t)
}

func TestReturnFeeExcessPartialKeepRewritesTracker(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	payer := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	hashId := []byte("hash")
	key := feePaidFromStakePayerKey(hashId, payer)
	require.NoError(t, k.FeePaidFromStake.Set(ctx, key, types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{origin(payer, valAddr, 100)},
		Total:        math.NewInt(100),
	}))
	validator := bondedVal(valAddr, 100)
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	sk.On("Delegate", ctx, payer, math.NewInt(40), stakingtypes.Bonded, validator, false).Return(math.LegacyZeroDec(), nil)

	bonded, unbonded, err := k.ReturnFeeExcess(ctx, hashId, payer, math.NewInt(40), math.NewInt(60))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(40), bonded)
	require.True(t, unbonded.IsZero())

	kept, err := k.FeePaidFromStake.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(60), kept.Total)
	require.Len(t, kept.TokenOrigins, 1)
	require.Equal(t, math.NewInt(60), kept.TokenOrigins[0].Amount)
	sk.AssertExpectations(t)
}

func TestReturnFeeExcessPartialKeepSplitsOrigins(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	disablePowerCaps(t, k, ctx)
	payer := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	valB := sdk.ValAddress(sample.AccAddressBytes())
	hashId := []byte("hash")
	key := feePaidFromStakePayerKey(hashId, payer)
	require.NoError(t, k.FeePaidFromStake.Set(ctx, key, types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{origin(payer, valA, 60), origin(payer, valB, 40)},
		Total:        math.NewInt(100),
	}))
	validatorA := bondedVal(valA, 60)
	validatorB := bondedVal(valB, 40)
	sk.On("GetValidator", ctx, valA).Return(validatorA, nil)
	sk.On("GetValidator", ctx, valB).Return(validatorB, nil)
	sk.On("Delegate", ctx, payer, math.NewInt(24), stakingtypes.Bonded, validatorA, false).Return(math.LegacyZeroDec(), nil)
	sk.On("Delegate", ctx, payer, math.NewInt(16), stakingtypes.Bonded, validatorB, false).Return(math.LegacyZeroDec(), nil)

	bonded, unbonded, err := k.ReturnFeeExcess(ctx, hashId, payer, math.NewInt(40), math.NewInt(60))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(40), bonded)
	require.True(t, unbonded.IsZero())

	kept, err := k.FeePaidFromStake.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(60), kept.Total)
	require.Len(t, kept.TokenOrigins, 2)
	require.Equal(t, math.NewInt(36), kept.TokenOrigins[0].Amount)
	require.Equal(t, math.NewInt(24), kept.TokenOrigins[1].Amount)
	originSum := kept.TokenOrigins[0].Amount.Add(kept.TokenOrigins[1].Amount)
	require.Equal(t, kept.Total, originSum)
	sk.AssertExpectations(t)
}

// The ledger is exact, so a split that does not add up to the tracker fails
// instead of being clamped, and nothing is returned or rewritten.
func TestReturnFeeExcessRejectsSplitThatDoesNotMatchTracker(t *testing.T) {
	cases := []struct {
		name    string
		tracked int64
		excess  int64
		kept    int64
	}{
		{name: "short tracker", tracked: 40, excess: 100, kept: 50},
		{name: "short tracker with nothing kept", tracked: 98, excess: 100, kept: 0},
		{name: "surplus tracker with nothing kept", tracked: 102, excess: 100, kept: 0},
		{name: "kept short of the remainder", tracked: 100, excess: 40, kept: 59},
		{name: "nonzero tracker with zero split", tracked: 100, excess: 0, kept: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, _, _, _, _, ctx, _ := setupKeeper(t)
			payer := sample.AccAddressBytes()
			valAddr := sdk.ValAddress(sample.AccAddressBytes())
			hashId := []byte("hash")
			key := feePaidFromStakePayerKey(hashId, payer)
			require.NoError(t, k.FeePaidFromStake.Set(ctx, key, types.DelegationsAmounts{
				TokenOrigins: []*types.TokenOriginInfo{origin(payer, valAddr, tc.tracked)},
				Total:        math.NewInt(tc.tracked),
			}))

			// No staking keeper call is mocked: a mismatch must fail before any.
			bonded, unbonded, err := k.ReturnFeeExcess(ctx, hashId, payer, math.NewInt(tc.excess), math.NewInt(tc.kept))
			require.Error(t, err)
			require.True(t, bonded.IsZero())
			require.True(t, unbonded.IsZero())
			tracker, getErr := k.FeePaidFromStake.Get(ctx, key)
			require.NoError(t, getErr)
			require.Equal(t, math.NewInt(tc.tracked), tracker.Total)
			require.Equal(t, math.NewInt(tc.tracked), tracker.TokenOrigins[0].Amount)
		})
	}
}

func TestReturnFeeExcessZeroSplitRemovesEmptyTracker(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	payer := sample.AccAddressBytes()
	hashId := []byte("hash")
	key := feePaidFromStakePayerKey(hashId, payer)
	require.NoError(t, k.FeePaidFromStake.Set(ctx, key, types.DelegationsAmounts{Total: math.ZeroInt()}))

	bonded, unbonded, err := k.ReturnFeeExcess(ctx, hashId, payer, math.ZeroInt(), math.ZeroInt())
	require.NoError(t, err)
	require.True(t, bonded.IsZero())
	require.True(t, unbonded.IsZero())
	has, hasErr := k.FeePaidFromStake.Has(ctx, key)
	require.NoError(t, hasErr)
	require.False(t, has)
}

func TestReturnFeeExcessNothingExcessKeepsTracker(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	payer := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	hashId := []byte("hash")
	key := feePaidFromStakePayerKey(hashId, payer)
	require.NoError(t, k.FeePaidFromStake.Set(ctx, key, types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{origin(payer, valAddr, 100)},
		Total:        math.NewInt(100),
	}))

	bonded, unbonded, err := k.ReturnFeeExcess(ctx, hashId, payer, math.ZeroInt(), math.NewInt(100))
	require.NoError(t, err)
	require.True(t, bonded.IsZero())
	require.True(t, unbonded.IsZero())
	tracker, getErr := k.FeePaidFromStake.Get(ctx, key)
	require.NoError(t, getErr)
	require.Equal(t, math.NewInt(100), tracker.Total)
}

// The method reads only the payer's own tracker. A legacy hash-id-only tracker
// must not stand in for a missing payer ledger.
func TestReturnFeeExcessMissingPayerTrackerDoesNotUseLegacyTracker(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	payer := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	hashId := []byte("hash")
	require.NoError(t, k.FeePaidFromStake.Set(ctx, hashId, types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{origin(payer, valAddr, 100)},
		Total:        math.NewInt(100),
	}))

	_, _, err := k.ReturnFeeExcess(ctx, hashId, payer, math.NewInt(40), math.NewInt(60))
	require.Error(t, err)
	legacy, getErr := k.FeePaidFromStake.Get(ctx, hashId)
	require.NoError(t, getErr)
	require.Equal(t, math.NewInt(100), legacy.Total)
}

func TestReturnFeeExcessRejectsMalformedSnapshot(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	payer := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	hashId := []byte("hash")
	key := feePaidFromStakePayerKey(hashId, payer)
	require.NoError(t, k.FeePaidFromStake.Set(ctx, key, types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{origin(payer, valAddr, 100)},
		Total:        math.NewInt(90),
	}))

	_, _, err := k.ReturnFeeExcess(ctx, hashId, payer, math.NewInt(10), math.NewInt(80))
	require.ErrorIs(t, err, types.ErrMalformedDelegationSnapshot)
	kept, getErr := k.FeePaidFromStake.Get(ctx, key)
	require.NoError(t, getErr)
	require.Equal(t, math.NewInt(90), kept.Total)
	require.Equal(t, math.NewInt(100), kept.TokenOrigins[0].Amount)
}

func TestReturnFeeExcessUnbondingErrorLeavesTracker(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	params := types.DefaultParams()
	params.MaxValidatorPowerShare = math.LegacyNewDecWithPrec(1, 6)
	params.MaxReporterPowerShare = math.LegacyNewDecWithPrec(1, 6)
	require.NoError(t, k.Params.Set(ctx, params))

	payer := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	hashId := []byte("hash")
	key := feePaidFromStakePayerKey(hashId, payer)
	require.NoError(t, k.FeePaidFromStake.Set(ctx, key, types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{origin(payer, valAddr, 80)},
		Total:        math.NewInt(80),
	}))
	validator := bondedVal(valAddr, 50)
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	sk.On("GetBondedValidatorsByPower", ctx).Return([]stakingtypes.Validator{validator}, nil)
	sk.On("IterateDelegatorDelegations", ctx, payer, mock.Anything).Return(nil)
	sk.On("UnbondingTime", ctx).Return(21*24*time.Hour, nil)
	sk.On("SetUnbondingDelegationEntry", ctx, payer, valAddr, ctx.BlockHeight(), mock.Anything, math.NewInt(30)).Return(stakingtypes.UnbondingDelegation{}, stakingtypes.ErrNoDelegation)

	_, _, err := k.ReturnFeeExcess(ctx, hashId, payer, math.NewInt(30), math.NewInt(50))
	require.ErrorIs(t, err, stakingtypes.ErrNoDelegation)
	kept, getErr := k.FeePaidFromStake.Get(ctx, key)
	require.NoError(t, getErr)
	require.Equal(t, math.NewInt(80), kept.Total)
	require.Equal(t, math.NewInt(80), kept.TokenOrigins[0].Amount)
}

// A full MaxEntries queue does not fail the fee-excess return. Stake over the
// caps goes to a standard unbonding entry on the origin validator: no second
// delegation past the cap and no liquid release.
func TestReturnFeeExcessFullQueueAddsStandardUnbondingEntry(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	params := types.DefaultParams()
	params.MaxValidatorPowerShare = math.LegacyNewDecWithPrec(1, 6)
	params.MaxReporterPowerShare = math.LegacyNewDecWithPrec(1, 6)
	require.NoError(t, k.Params.Set(ctx, params))

	payer := sample.AccAddressBytes()
	originVal := sdk.ValAddress(sample.AccAddressBytes())
	hashId := []byte("hash")
	key := feePaidFromStakePayerKey(hashId, payer)
	total := math.NewInt(80)
	excess := math.NewInt(30)
	kept := math.NewInt(50)
	require.NoError(t, k.FeePaidFromStake.Set(ctx, key, types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{origin(payer, originVal, 80)},
		Total:        total,
	}))

	validator := bondedVal(originVal, 50)
	sk.On("GetValidator", ctx, originVal).Return(validator, nil)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	sk.On("GetBondedValidatorsByPower", ctx).Return([]stakingtypes.Validator{validator}, nil)
	sk.On("IterateDelegatorDelegations", ctx, payer, mock.Anything).Return(nil)
	unbondingMock(sk, ctx, payer, originVal, excess)

	bonded, unbonded, err := k.ReturnFeeExcess(ctx, hashId, payer, excess, kept)
	require.NoError(t, err)
	require.True(t, bonded.IsZero())
	require.Equal(t, excess, unbonded)

	tracker, getErr := k.FeePaidFromStake.Get(ctx, key)
	require.NoError(t, getErr)
	require.Equal(t, kept, tracker.Total)
	require.Len(t, tracker.TokenOrigins, 1)
	require.Equal(t, kept, tracker.TokenOrigins[0].Amount)
	sk.AssertNotCalled(t, "HasMaxUnbondingDelegationEntries", mock.Anything, mock.Anything, mock.Anything)
	sk.AssertNotCalled(t, "Delegate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	sk.AssertExpectations(t)
}
