package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestFeefromReporterStake(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	fee := math.NewIntWithDecimal(100, 6)
	reporterAddr, selector1, selector2, selector3 := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()

	err := k.FeefromReporterStake(ctx, reporterAddr, fee, []byte("hashId"), true)
	require.ErrorContains(t, err, "insufficient stake to pay fee")

	require.NoError(t, k.Selectors.Set(ctx, selector1, types.NewSelection(reporterAddr, 1)))
	require.NoError(t, k.Selectors.Set(ctx, selector2, types.NewSelection(reporterAddr, 1)))
	require.NoError(t, k.Selectors.Set(ctx, selector3, types.NewSelection(reporterAddr, 1)))

	delegations := []stakingtypes.Delegation{
		{DelegatorAddress: selector1.String(), ValidatorAddress: sdk.ValAddress(reporterAddr).String(), Shares: math.LegacyNewDecWithPrec(100, 6)},
		{DelegatorAddress: selector2.String(), ValidatorAddress: sdk.ValAddress(reporterAddr).String(), Shares: math.LegacyNewDecWithPrec(200, 6)},
		{DelegatorAddress: selector3.String(), ValidatorAddress: sdk.ValAddress(reporterAddr).String(), Shares: math.LegacyNewDecWithPrec(300, 6)},
	}

	validator := stakingtypes.Validator{
		OperatorAddress: sdk.ValAddress(reporterAddr).String(),
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewIntWithDecimal(1000, 6),
		DelegatorShares: math.LegacyNewDecWithPrec(600, 6),
	}
	tokenShare1 := validator.TokensFromShares(delegations[0].Shares).Quo(math.NewIntWithDecimal(1000, 6).ToLegacyDec()).Mul(fee.ToLegacyDec())
	tokenShare2 := validator.TokensFromShares(delegations[1].Shares).Quo(math.NewIntWithDecimal(1000, 6).ToLegacyDec()).Mul(fee.ToLegacyDec())
	tokenShare3 := validator.TokensFromShares(delegations[2].Shares).Quo(math.NewIntWithDecimal(1000, 6).ToLegacyDec()).Mul(fee.ToLegacyDec())

	sk.On("IterateDelegatorDelegations", ctx, selector1, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		sk.On("GetValidator", ctx, sdk.ValAddress(reporterAddr)).Return(validator, nil)
		fn(delegations[0])
	})
	sk.On("IterateDelegatorDelegations", ctx, selector2, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		sk.On("GetValidator", ctx, sdk.ValAddress(reporterAddr)).Return(validator, nil)
		fn(delegations[1])
	})
	sk.On("IterateDelegatorDelegations", ctx, selector3, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		sk.On("GetValidator", ctx, sdk.ValAddress(reporterAddr)).Return(validator, nil)
		fn(delegations[2])
	})
	feeShare1, err := validator.SharesFromTokens(tokenShare1.TruncateInt())
	require.NoError(t, err)
	feeShare2, err := validator.SharesFromTokens(tokenShare2.TruncateInt())
	require.NoError(t, err)
	feeShare3, err := validator.SharesFromTokens(tokenShare3.TruncateInt())

	require.NoError(t, err)
	sk.On("Unbond", ctx, selector1, sdk.ValAddress(reporterAddr), feeShare1).Return(tokenShare1.TruncateInt(), nil)
	sk.On("Unbond", ctx, selector2, sdk.ValAddress(reporterAddr), feeShare2).Return(tokenShare2.TruncateInt(), nil)
	sk.On("Unbond", ctx, selector3, sdk.ValAddress(reporterAddr), feeShare3).Return(tokenShare3.TruncateInt(), nil)

	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(99_999_999)))).Return(nil)
	err = k.FeefromReporterStake(ctx, reporterAddr, math.NewIntWithDecimal(100, 6), []byte("hashId"), true)
	require.NoError(t, err)

	feefromstake, err := k.FeePaidFromStake.Get(ctx, []byte("hashId"))
	require.NoError(t, err)
	expected := tokenShare1.TruncateInt().Add(tokenShare2.TruncateInt()).Add(tokenShare3.TruncateInt())
	require.Equal(t, expected, feefromstake.Total)
}

func TestFeefromReporterStakeMultiplevalidators(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	fee := math.NewIntWithDecimal(100, 6)
	reporterAddr, selector := sample.AccAddressBytes(), sample.AccAddressBytes()

	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 2)))
	// share with validator less than fee
	delegations1 := []stakingtypes.Delegation{
		{DelegatorAddress: reporterAddr.String(), ValidatorAddress: sdk.ValAddress(reporterAddr).String(), Shares: math.LegacyNewDecWithPrec(100, 6)},
		{DelegatorAddress: selector.String(), ValidatorAddress: sdk.ValAddress(selector).String(), Shares: math.LegacyNewDecWithPrec(100, 6)},
	}

	validator1 := stakingtypes.Validator{Tokens: math.NewIntWithDecimal(50, 6), DelegatorShares: math.LegacyNewDecWithPrec(100, 6), Status: stakingtypes.Bonded}
	validator2 := stakingtypes.Validator{Tokens: math.NewIntWithDecimal(50, 6), DelegatorShares: math.LegacyNewDecWithPrec(100, 6), Status: stakingtypes.Bonded}
	validators := []stakingtypes.Validator{validator1, validator2}
	sk.On("IterateDelegatorDelegations", ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		for i, del := range delegations1 {
			valAddr, _ := sdk.ValAddressFromBech32(del.ValidatorAddress)
			sk.On("GetValidator", ctx, valAddr).Return(validators[i], nil)
			sk.On("Unbond", ctx, selector, valAddr, del.Shares).Return(fee.QuoRaw(2), nil)
			fn(del)
		}
	})

	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", fee))).Return(nil)
	err := k.FeefromReporterStake(ctx, reporterAddr, fee, []byte("hashId"), true)
	require.NoError(t, err)

	feefromstake, err := k.FeePaidFromStake.Get(ctx, []byte("hashId"))
	require.NoError(t, err)
	expected := fee
	require.Equal(t, expected, feefromstake.Total)

	err = k.FeefromReporterStake(ctx, reporterAddr, fee, []byte("hashId"), true)
	require.NoError(t, err)

	feefromstake, err = k.FeePaidFromStake.Get(ctx, []byte("hashId"))
	require.NoError(t, err)
	expected = fee.MulRaw(2)
	require.Equal(t, expected, feefromstake.Total)
}

func TestEscrowReporterStake(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	stake := math.NewIntWithDecimal(100, 6)
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(reporterAddr.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: reporterAddr, ValidatorAddress: sdk.ValAddress(reporterAddr), Amount: stake},
		},
		Total: stake,
	}))
	delegation := stakingtypes.Delegation{DelegatorAddress: reporterAddr.String(), ValidatorAddress: sdk.ValAddress(reporterAddr).String(), Shares: math.LegacyNewDecWithPrec(100, 6)}
	validator := stakingtypes.Validator{Tokens: math.NewIntWithDecimal(1000, 6), DelegatorShares: math.LegacyNewDecWithPrec(100, 6), Status: stakingtypes.Bonded}
	sk.On("GetDelegation", ctx, reporterAddr, sdk.ValAddress(reporterAddr)).Return(delegation, nil)
	sk.On("GetValidator", ctx, sdk.ValAddress(reporterAddr)).Return(validator, nil)
	delTokens, err := validator.SharesFromTokens(stake)
	require.NoError(t, err)
	sk.On("Unbond", ctx, reporterAddr, sdk.ValAddress(reporterAddr), delTokens).Return(stake, nil)
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", stake))).Return(nil)
	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, 100, uint64(ctx.BlockHeight()), stake, []byte{}, []byte("hashId")))
}

func TestEscrowReporterStakeUnbondingdelegations(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	reporterAddr, selector2, selector3, valAddr1, valAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	stake := math.NewIntWithDecimal(1000, 6)
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(reporterAddr.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: reporterAddr, ValidatorAddress: sdk.ValAddress(valAddr1), Amount: stake},
			{DelegatorAddress: selector2, ValidatorAddress: sdk.ValAddress(valAddr1), Amount: stake},
			{DelegatorAddress: selector3, ValidatorAddress: sdk.ValAddress(valAddr1), Amount: stake},
		},
		Total: stake,
	}))
	validator1 := stakingtypes.Validator{Tokens: math.NewIntWithDecimal(1_000, 6), DelegatorShares: math.NewIntWithDecimal(1_000, 6).ToLegacyDec(), Status: stakingtypes.Bonded}
	validator2 := stakingtypes.Validator{Tokens: math.NewIntWithDecimal(1_000, 6), DelegatorShares: math.NewIntWithDecimal(1_000, 6).ToLegacyDec(), Status: stakingtypes.Bonded}
	delegation1 := stakingtypes.Delegation{DelegatorAddress: reporterAddr.String(), ValidatorAddress: sdk.ValAddress(valAddr1).String(), Shares: math.NewIntWithDecimal(1_000, 6).ToLegacyDec()}
	sk.On("GetDelegation", ctx, reporterAddr, sdk.ValAddress(valAddr1)).Return(delegation1, nil)
	delegation2 := stakingtypes.Delegation{DelegatorAddress: selector2.String(), ValidatorAddress: sdk.ValAddress(valAddr1).String(), Shares: math.LegacyZeroDec()}
	sk.On("GetDelegation", ctx, selector2, sdk.ValAddress(valAddr1)).Return(delegation2, nil)
	delegation3 := stakingtypes.Delegation{DelegatorAddress: selector3.String(), ValidatorAddress: sdk.ValAddress(valAddr2).String(), Shares: math.NewIntWithDecimal(1_000, 6).ToLegacyDec()}
	sk.On("GetDelegation", ctx, selector3, sdk.ValAddress(valAddr1)).Return(stakingtypes.Delegation{}, stakingtypes.ErrNoDelegation)
	sk.On("GetDelegation", ctx, selector3, sdk.ValAddress(valAddr2)).Return(delegation3, nil)

	sk.On("GetUnbondingDelegation", ctx, selector2, sdk.ValAddress(valAddr1)).Return(stakingtypes.UnbondingDelegation{
		DelegatorAddress: selector2.String(),
		ValidatorAddress: sdk.ValAddress(valAddr1).String(),
		Entries: []stakingtypes.UnbondingDelegationEntry{
			{CreationHeight: ctx.BlockHeight(), InitialBalance: stake, Balance: stake},
		},
	}, nil)
	sk.On("GetUnbondingDelegation", ctx, selector3, sdk.ValAddress(valAddr1)).Return(stakingtypes.UnbondingDelegation{}, stakingtypes.ErrNoUnbondingDelegation)
	sk.On("SetUnbondingDelegation", ctx, stakingtypes.UnbondingDelegation{
		DelegatorAddress: selector2.String(),
		ValidatorAddress: sdk.ValAddress(valAddr1).String(),
		Entries: []stakingtypes.UnbondingDelegationEntry{
			{CreationHeight: ctx.BlockHeight(), InitialBalance: math.NewInt(500000000), Balance: math.NewInt(500000000)},
		},
	}).Return(nil)

	sk.On("GetValidator", ctx, sdk.ValAddress(valAddr1)).Return(validator1, nil)
	sk.On("GetValidator", ctx, sdk.ValAddress(valAddr2)).Return(validator2, nil)

	sk.On("Unbond", ctx, reporterAddr, sdk.ValAddress(valAddr1), delegation1.Shares.Quo(math.LegacyNewDec(2))).Return(stake.QuoRaw(2), nil)
	sk.On("Unbond", ctx, selector3, sdk.ValAddress(valAddr2), math.LegacyNewDec(500000000)).Return(math.NewInt(500000000), nil)

	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", stake.QuoRaw(2)))).Return(nil)
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.NotBondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", stake.QuoRaw(2)))).Return(nil)
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(500000000)))).Return(nil)

	sk.On("GetRedelegationsFromSrcValidator", ctx, sdk.ValAddress(valAddr1)).Return([]stakingtypes.Redelegation{
		{
			DelegatorAddress:    selector3.String(),
			ValidatorSrcAddress: sdk.ValAddress(valAddr1).String(),
			ValidatorDstAddress: sdk.ValAddress(valAddr2).String(),
			Entries: []stakingtypes.RedelegationEntry{
				{CreationHeight: ctx.BlockHeight(), InitialBalance: stake, SharesDst: math.LegacyDec(stake)},
			},
		},
	}, nil)
	// the multi-hop chase checks for further redelegations from the destination
	// validator and stops when there are none.
	sk.On("GetRedelegationsFromSrcValidator", ctx, sdk.ValAddress(valAddr2)).Return([]stakingtypes.Redelegation{}, nil)

	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, 3000, uint64(ctx.BlockHeight()), math.NewIntWithDecimal(1500, 6), []byte{}, []byte("hashId")))
}

// TestEscrowReporterStakePartialCollection asserts that when the redelegation
// chase collects less than the intended slash, DisputedDelegationAmounts.Total
// equals the coins actually escrowed (sum of appended origin amounts) and is
// strictly less than `amt`. ReturnSlashedTokens relies on Total == coins-held so
// it never scales an under-collected snapshot up into dispute-fee funds.
func TestEscrowReporterStakePartialCollection(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	valAddr1, valAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes()
	// power=1 -> totalTokens = PowerReduction (1e9). origin.Amount == totalTokens
	// so the single selector's proportional share == amt (1000), leftover 0.
	originAmount := math.NewInt(1_000_000_000)
	amt := math.NewInt(1000)
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(reporterAddr.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: reporterAddr, ValidatorAddress: sdk.ValAddress(valAddr1), Amount: originAmount},
		},
		Total: originAmount,
	}))

	// first undelegate on valAddr1: no live delegation and no unbonding -> full
	// `remaining` (1000), nothing collected on the first hop.
	sk.On("GetDelegation", ctx, reporterAddr, sdk.ValAddress(valAddr1)).Return(stakingtypes.Delegation{}, stakingtypes.ErrNoDelegation)
	sk.On("GetUnbondingDelegation", ctx, reporterAddr, sdk.ValAddress(valAddr1)).Return(stakingtypes.UnbondingDelegation{}, stakingtypes.ErrNoUnbondingDelegation)

	// the selector redelegated valAddr1 -> valAddr2; chase lands on valAddr2.
	sk.On("GetRedelegationsFromSrcValidator", ctx, sdk.ValAddress(valAddr1)).Return([]stakingtypes.Redelegation{
		{
			DelegatorAddress:    reporterAddr.String(),
			ValidatorSrcAddress: sdk.ValAddress(valAddr1).String(),
			ValidatorDstAddress: sdk.ValAddress(valAddr2).String(),
			Entries: []stakingtypes.RedelegationEntry{
				{CreationHeight: ctx.BlockHeight(), InitialBalance: amt, SharesDst: math.LegacyDec(amt)},
			},
		},
	}, nil)
	// the multi-hop chase verifies there are no further redelegations from valAddr2.
	sk.On("GetRedelegationsFromSrcValidator", ctx, sdk.ValAddress(valAddr2)).Return([]stakingtypes.Redelegation{}, nil)

	// second undelegate on valAddr2: a partial delegation covers only 400 of the
	// 1000 remaining, and there is no unbonding entry to cover the rest -> the
	// chase collects 400 and returns a 600 shortfall.
	validator2 := stakingtypes.Validator{Tokens: math.NewInt(1000), DelegatorShares: math.LegacyNewDec(1000), Status: stakingtypes.Bonded}
	sk.On("GetDelegation", ctx, reporterAddr, sdk.ValAddress(valAddr2)).Return(stakingtypes.Delegation{
		DelegatorAddress: reporterAddr.String(), ValidatorAddress: sdk.ValAddress(valAddr2).String(), Shares: math.LegacyNewDec(400),
	}, nil)
	sk.On("GetValidator", ctx, sdk.ValAddress(valAddr2)).Return(validator2, nil)
	sk.On("Unbond", ctx, reporterAddr, sdk.ValAddress(valAddr2), math.LegacyNewDec(400)).Return(math.NewInt(400), nil)
	sk.On("GetUnbondingDelegation", ctx, reporterAddr, sdk.ValAddress(valAddr2)).Return(stakingtypes.UnbondingDelegation{}, stakingtypes.ErrNoUnbondingDelegation)
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(400)))).Return(nil)

	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, 1, uint64(ctx.BlockHeight()), amt, []byte{}, []byte("hashId")))

	stored, err := k.DisputedDelegationAmounts.Get(ctx, []byte("hashId"))
	require.NoError(t, err)
	require.True(t, stored.Total.LT(amt), "Total (%s) must be less than intended slash amt (%s)", stored.Total, amt)
	require.Equal(t, math.NewInt(400), stored.Total)
	sum := math.ZeroInt()
	for _, o := range stored.TokenOrigins {
		sum = sum.Add(o.Amount)
	}
	require.Equal(t, stored.Total, sum, "Total must equal sum of appended origin amounts")
}

// TestGetDstValidatorMultiHop verifies that getDstValidator follows a chain of
// redelegations. With a val1 -> val2 -> val3 chain, it must return val3.
func TestGetDstValidatorMultiHop(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	val1 := sdk.ValAddress(sample.AccAddressBytes())
	val2 := sdk.ValAddress(sample.AccAddressBytes())
	val3 := sdk.ValAddress(sample.AccAddressBytes())

	amt := math.NewInt(1000)
	sk.On("GetRedelegationsFromSrcValidator", ctx, val1).Return([]stakingtypes.Redelegation{
		{
			DelegatorAddress:    delAddr.String(),
			ValidatorSrcAddress: val1.String(),
			ValidatorDstAddress: val2.String(),
			Entries:             []stakingtypes.RedelegationEntry{{CreationHeight: ctx.BlockHeight(), InitialBalance: amt, SharesDst: amt.ToLegacyDec()}},
		},
	}, nil)
	sk.On("GetRedelegationsFromSrcValidator", ctx, val2).Return([]stakingtypes.Redelegation{
		{
			DelegatorAddress:    delAddr.String(),
			ValidatorSrcAddress: val2.String(),
			ValidatorDstAddress: val3.String(),
			Entries:             []stakingtypes.RedelegationEntry{{CreationHeight: ctx.BlockHeight(), InitialBalance: amt, SharesDst: amt.ToLegacyDec()}},
		},
	}, nil)
	sk.On("GetRedelegationsFromSrcValidator", ctx, val3).Return([]stakingtypes.Redelegation{}, nil)

	got, err := k.GetDstValidatorExported(ctx, delAddr, val1)
	require.NoError(t, err)
	require.Equal(t, val3, got)
}

// TestGetDstValidatorCycleDetected verifies that a redelegation cycle is rejected.
func TestGetDstValidatorCycleDetected(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	val1 := sdk.ValAddress(sample.AccAddressBytes())
	val2 := sdk.ValAddress(sample.AccAddressBytes())
	amt := math.NewInt(1000)

	red := func(src, dst sdk.ValAddress) []stakingtypes.Redelegation {
		return []stakingtypes.Redelegation{{
			DelegatorAddress:    delAddr.String(),
			ValidatorSrcAddress: src.String(),
			ValidatorDstAddress: dst.String(),
			Entries:             []stakingtypes.RedelegationEntry{{CreationHeight: ctx.BlockHeight(), InitialBalance: amt, SharesDst: amt.ToLegacyDec()}},
		}}
	}
	sk.On("GetRedelegationsFromSrcValidator", ctx, val1).Return(red(val1, val2), nil)
	sk.On("GetRedelegationsFromSrcValidator", ctx, val2).Return(red(val2, val1), nil)

	_, err := k.GetDstValidatorExported(ctx, delAddr, val1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cycle")
}

// TestGetDstValidatorHopLimitExceeded verifies the hop bound rejects chains longer
// than maxRedelegationHops lookups (7).
func TestGetDstValidatorHopLimitExceeded(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	amt := math.NewInt(1000)

	vals := make([]sdk.ValAddress, 8)
	for i := range vals {
		vals[i] = sdk.ValAddress(sample.AccAddressBytes())
	}

	red := func(src, dst sdk.ValAddress) []stakingtypes.Redelegation {
		return []stakingtypes.Redelegation{{
			DelegatorAddress:    delAddr.String(),
			ValidatorSrcAddress: src.String(),
			ValidatorDstAddress: dst.String(),
			Entries:             []stakingtypes.RedelegationEntry{{CreationHeight: ctx.BlockHeight(), InitialBalance: amt, SharesDst: amt.ToLegacyDec()}},
		}}
	}
	for i := 0; i < len(vals)-1; i++ {
		sk.On("GetRedelegationsFromSrcValidator", ctx, vals[i]).Return(red(vals[i], vals[i+1]), nil)
	}

	_, err := k.GetDstValidatorExported(ctx, delAddr, vals[0])
	require.Error(t, err)
	require.Contains(t, err.Error(), "hop limit exceeded")
}
