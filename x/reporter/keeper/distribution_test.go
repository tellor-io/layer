package keeper_test

import (
	"testing"

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
	k, _, _, _, ctx, _ := setupKeeper(t)
	height := uint64(10)
	val1Address := sample.AccAddressBytes()
	vals := simtestutil.ConvertAddrsToValAddrs([]sdk.AccAddress{val1Address})
	val1 := vals[0]
	addr := sample.AccAddressBytes()
	addr2 := sample.AccAddressBytes()
	reporter1 := types.NewReporter(math.LegacyZeroDec(), math.OneInt())
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
	k, sk, _, _, ctx, _ := setupKeeper(t)

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
	_, err = k.ReturnSlashedTokens(ctx, math.NewIntWithDecimal(2000, 6), []byte("hashId"))
	require.NoError(t, err)
}

func TestFeeRefund(t *testing.T) {
	// set fee refund
	k, sk, _, _, ctx, _ := setupKeeper(t)
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
	require.NoError(t, k.FeeRefund(ctx, []byte("hashId"), amtminusburn))
}

func TestGetBondedValidators(t *testing.T) {
	k, sk, _, _, ctx, kvstore := setupKeeper(t)

	valAddr := sdk.ValAddress(sample.AccAddressBytes())

	store := kvstore.OpenKVStore(ctx)
	require.NoError(t, store.Set([]byte("p_reporter"), valAddr))
	iterator, err := store.Iterator(nil, nil)
	require.NoError(t, err)

	validator := stakingtypes.Validator{OperatorAddress: valAddr.String(), Status: stakingtypes.Unbonding}

	sk.On("ValidatorsPowerStoreIterator", ctx).Return(iterator, nil)
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)

	vals, err := k.GetBondedValidators(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 0, len(vals))

	ctx = ctx.WithBlockHeight(1)
	valAddr2 := sdk.ValAddress(sample.AccAddressBytes())
	require.NoError(t, store.Set([]byte("bonded"), valAddr2))
	iterator, err = store.Iterator(nil, nil)
	require.NoError(t, err)

	sk.On("ValidatorsPowerStoreIterator", ctx).Return(iterator, nil)
	validator.Status = stakingtypes.Bonded
	sk.On("GetValidator", ctx, valAddr2).Return(validator, nil)

	vals, err = k.GetBondedValidators(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, len(vals))
}

func TestAddAmountToStake(t *testing.T) {
	k, sk, _, _, ctx, kvstore := setupKeeper(t)

	acc := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	amt := math.NewInt(1000 * 1e6)

	store := kvstore.OpenKVStore(ctx)
	require.NoError(t, store.Set([]byte("key"), valAddr))
	iterator, err := store.Iterator(nil, nil)
	require.NoError(t, err)

	validator := stakingtypes.Validator{OperatorAddress: valAddr.String(), Status: stakingtypes.Bonded}

	sk.On("ValidatorsPowerStoreIterator", ctx).Return(iterator, nil)
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	sk.On("Delegate", ctx, acc, amt, stakingtypes.Bonded, validator, false).Return(math.LegacyZeroDec(), nil)

	err = k.AddAmountToStake(ctx, acc, amt)
	require.NoError(t, err)
}
