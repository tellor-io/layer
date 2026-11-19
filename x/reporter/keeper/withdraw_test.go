package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportermocks "github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func mockIterateDelegatorRedelegations(
	sk *reportermocks.StakingKeeper,
	ctx sdk.Context,
	delegator sdk.AccAddress,
	redelegations []stakingtypes.Redelegation,
) {
	sk.On("IterateDelegatorRedelegations", ctx, delegator, mock.AnythingOfType("func(types.Redelegation) bool")).
		Run(func(args mock.Arguments) {
			callback := args.Get(2).(func(stakingtypes.Redelegation) bool)
			for _, redelegation := range redelegations {
				if callback(redelegation) {
					return
				}
			}
		}).
		Return(nil).
		Once()
}

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

	feefromstake, err := k.FeePaidFromStakeByPayer(ctx, []byte("hashId"), reporterAddr)
	require.NoError(t, err)
	expected := tokenShare1.TruncateInt().Add(tokenShare2.TruncateInt()).Add(tokenShare3.TruncateInt())
	require.Equal(t, expected, feefromstake.Total)
}

func TestFeefromReporterStakeFullyCoveredRecordsActualUnbondAmount(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	fee := math.NewInt(1000)
	escrowedAmt := math.NewInt(999)

	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))

	delegation := stakingtypes.Delegation{
		DelegatorAddress: selector.String(),
		ValidatorAddress: valAddr.String(),
		Shares:           math.LegacyNewDec(2000),
	}
	validator := stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(2000),
		DelegatorShares: math.LegacyNewDec(2000),
	}
	sharesToUnbond, err := validator.SharesFromTokens(fee)
	require.NoError(t, err)

	sk.On("IterateDelegatorDelegations", ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		fn(delegation)
	})
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	sk.On("Unbond", ctx, selector, valAddr, sharesToUnbond).Return(escrowedAmt, nil)
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", escrowedAmt))).Return(nil)

	require.NoError(t, k.FeefromReporterStake(ctx, reporterAddr, fee, []byte("hashId"), true))

	tracker, err := k.FeePaidFromStakeByPayer(ctx, []byte("hashId"), reporterAddr)
	require.NoError(t, err)
	require.Equal(t, escrowedAmt, tracker.Total)
	require.Len(t, tracker.TokenOrigins, 1)
	require.Equal(t, escrowedAmt, tracker.TokenOrigins[0].Amount)

	sum := math.ZeroInt()
	for _, origin := range tracker.TokenOrigins {
		sum = sum.Add(origin.Amount)
	}
	require.Equal(t, tracker.Total, sum)
	sk.AssertExpectations(t)
	bk.AssertExpectations(t)
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

	feefromstake, err := k.FeePaidFromStakeByPayer(ctx, []byte("hashId"), reporterAddr)
	require.NoError(t, err)
	expected := fee
	require.Equal(t, expected, feefromstake.Total)

	err = k.FeefromReporterStake(ctx, reporterAddr, fee, []byte("hashId"), true)
	require.NoError(t, err)

	feefromstake, err = k.FeePaidFromStakeByPayer(ctx, []byte("hashId"), reporterAddr)
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
		Total: stake.MulRaw(3),
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

	mockIterateDelegatorRedelegations(sk, ctx, selector3, []stakingtypes.Redelegation{
		{
			DelegatorAddress:    selector3.String(),
			ValidatorSrcAddress: sdk.ValAddress(valAddr1).String(),
			ValidatorDstAddress: sdk.ValAddress(valAddr2).String(),
			Entries: []stakingtypes.RedelegationEntry{
				{CreationHeight: ctx.BlockHeight(), InitialBalance: stake, SharesDst: math.LegacyDec(stake)},
			},
		},
	})

	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, 3000, uint64(ctx.BlockHeight()), math.NewIntWithDecimal(1500, 6), []byte{}, []byte("hashId")))
}

// TestEscrowReporterStakePartialCollection asserts that a positive slash which
// collects less than requested stores the coins actually taken.
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
	mockIterateDelegatorRedelegations(sk, ctx, reporterAddr, []stakingtypes.Redelegation{
		{
			DelegatorAddress:    reporterAddr.String(),
			ValidatorSrcAddress: sdk.ValAddress(valAddr1).String(),
			ValidatorDstAddress: sdk.ValAddress(valAddr2).String(),
			Entries: []stakingtypes.RedelegationEntry{
				{CreationHeight: ctx.BlockHeight(), InitialBalance: amt, SharesDst: math.LegacyDec(amt)},
			},
		},
	})

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

	err := k.EscrowReporterStake(ctx, reporterAddr, 1, uint64(ctx.BlockHeight()), amt, []byte{}, []byte("hashId"))
	require.NoError(t, err)

	stored, err := k.DisputedDelegationAmounts.Get(ctx, []byte("hashId"))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(400), stored.Total)
	require.True(t, stored.Total.LT(amt))
	require.Len(t, stored.TokenOrigins, 1)
	require.Equal(t, math.NewInt(400), stored.TokenOrigins[0].Amount)
	require.Equal(t, sdk.ValAddress(valAddr2).Bytes(), stored.TokenOrigins[0].ValidatorAddress)
}

// TestEscrowReporterStakeOneLoyaShortfallStoresCollected asserts that a large
// slash which comes back one loya short still stores that smaller total.
func TestEscrowReporterStakeOneLoyaShortfallStoresCollected(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	requested := sdk.DefaultPowerReduction.MulRaw(1_000_000)
	collected := requested.Sub(math.OneInt())
	hashID := []byte("one-loya-shortfall")

	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(reporterAddr.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{
			DelegatorAddress: reporterAddr.Bytes(),
			ValidatorAddress: valAddr.Bytes(),
			Amount:           requested,
		}},
		Total: requested,
	}))

	validator := stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
		Status:          stakingtypes.Bonded,
		Tokens:          requested,
		DelegatorShares: requested.ToLegacyDec(),
	}
	delegation := stakingtypes.Delegation{
		DelegatorAddress: reporterAddr.String(),
		ValidatorAddress: valAddr.String(),
		Shares:           requested.ToLegacyDec(),
	}
	shares, err := validator.SharesFromTokens(requested)
	require.NoError(t, err)
	sk.On("GetDelegation", ctx, reporterAddr, valAddr).Return(delegation, nil)
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	sk.On("Unbond", ctx, reporterAddr, valAddr, shares).Return(collected, nil)
	sk.On("GetUnbondingDelegation", ctx, reporterAddr, valAddr).Return(stakingtypes.UnbondingDelegation{}, stakingtypes.ErrNoUnbondingDelegation)
	sk.On("IterateDelegatorRedelegations", ctx, reporterAddr, mock.AnythingOfType("func(types.Redelegation) bool")).Return(nil)
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", collected))).Return(nil)

	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, 1, uint64(ctx.BlockHeight()), requested, []byte{}, hashID))

	stored, err := k.DisputedDelegationAmounts.Get(ctx, hashID)
	require.NoError(t, err)
	require.Equal(t, collected, stored.Total)
	require.True(t, stored.Total.Equal(requested.Sub(math.OneInt())))
	require.Len(t, stored.TokenOrigins, 1)
	require.Equal(t, collected, stored.TokenOrigins[0].Amount)
	require.Equal(t, valAddr.Bytes(), stored.TokenOrigins[0].ValidatorAddress)
}

func TestEscrowReporterStakeZeroAmountStoresEmptyCollection(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(reporterAddr.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{
		Total: math.ZeroInt(),
	}))

	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, 0, uint64(ctx.BlockHeight()), math.ZeroInt(), []byte{}, []byte("hash-zero")))

	stored, err := k.DisputedDelegationAmounts.Get(ctx, []byte("hash-zero"))
	require.NoError(t, err)
	require.True(t, stored.Total.IsZero())
	require.Empty(t, stored.TokenOrigins)
}

// TestEscrowReporterStakeZeroCollected asserts a positive slash against a
// snapshot validator that has no delegation, unbonding entry, or redelegation
// fails and stores nothing.
func TestEscrowReporterStakeZeroCollected(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	origin := math.NewInt(1_000)
	hashID := []byte("zero-collected")

	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(reporterAddr.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{
			DelegatorAddress: reporterAddr.Bytes(),
			ValidatorAddress: valA.Bytes(),
			Amount:           origin,
		}},
		Total: origin,
	}))

	sk.On("GetDelegation", ctx, reporterAddr, valA).Return(stakingtypes.Delegation{}, stakingtypes.ErrNoDelegation)
	sk.On("GetUnbondingDelegation", ctx, reporterAddr, valA).Return(stakingtypes.UnbondingDelegation{}, stakingtypes.ErrNoUnbondingDelegation)
	sk.On("IterateDelegatorRedelegations", ctx, reporterAddr, mock.AnythingOfType("func(types.Redelegation) bool")).Return(nil)

	err := k.EscrowReporterStake(ctx, reporterAddr, 1, uint64(ctx.BlockHeight()), origin, []byte{}, hashID)
	require.ErrorIs(t, err, types.ErrReporterSlashShortfall)

	has, err := k.DisputedDelegationAmounts.Has(ctx, hashID)
	require.NoError(t, err)
	require.False(t, has)
}

// TestEscrowReporterStakePartialOriginStoresCollected asserts that collecting the
// full allocation from one origin and nothing from the next stores only the
// coins taken from the first origin.
func TestEscrowReporterStakePartialOriginStoresCollected(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	valB := sdk.ValAddress(sample.AccAddressBytes())
	first := math.NewInt(600)
	second := math.NewInt(400)
	requested := first.Add(second)
	hashID := []byte("partial-origin-shortfall")

	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(reporterAddr.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: reporterAddr.Bytes(), ValidatorAddress: valA.Bytes(), Amount: first},
			{DelegatorAddress: reporterAddr.Bytes(), ValidatorAddress: valB.Bytes(), Amount: second},
		},
		Total: requested,
	}))

	validatorA := stakingtypes.Validator{
		OperatorAddress: valA.String(),
		Status:          stakingtypes.Bonded,
		Tokens:          first,
		DelegatorShares: first.ToLegacyDec(),
	}
	delegationA := stakingtypes.Delegation{
		DelegatorAddress: reporterAddr.String(),
		ValidatorAddress: valA.String(),
		Shares:           first.ToLegacyDec(),
	}
	sharesA, err := validatorA.SharesFromTokens(first)
	require.NoError(t, err)
	sk.On("GetDelegation", ctx, reporterAddr, valA).Return(delegationA, nil)
	sk.On("GetValidator", ctx, valA).Return(validatorA, nil)
	sk.On("Unbond", ctx, reporterAddr, valA, sharesA).Return(first, nil)
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", first))).Return(nil)

	sk.On("GetDelegation", ctx, reporterAddr, valB).Return(stakingtypes.Delegation{}, stakingtypes.ErrNoDelegation)
	sk.On("GetUnbondingDelegation", ctx, reporterAddr, valB).Return(stakingtypes.UnbondingDelegation{}, stakingtypes.ErrNoUnbondingDelegation)
	sk.On("IterateDelegatorRedelegations", ctx, reporterAddr, mock.AnythingOfType("func(types.Redelegation) bool")).Return(nil)

	err = k.EscrowReporterStake(ctx, reporterAddr, 1, uint64(ctx.BlockHeight()), requested, []byte{}, hashID)
	require.NoError(t, err)

	stored, err := k.DisputedDelegationAmounts.Get(ctx, hashID)
	require.NoError(t, err)
	require.Equal(t, first, stored.Total)
	require.True(t, stored.Total.LT(requested))
	require.Len(t, stored.TokenOrigins, 1)
	require.Equal(t, first, stored.TokenOrigins[0].Amount)
	require.Equal(t, valA.Bytes(), stored.TokenOrigins[0].ValidatorAddress)
	sk.AssertCalled(t, "Unbond", ctx, reporterAddr, valA, sharesA)
	sk.AssertNotCalled(t, "Unbond", mock.Anything, reporterAddr, valB, mock.Anything)
}

// TestEscrowReporterStakeZeroAmountWithPositiveOrigin stores a zero total for
// a real origin without reading or moving that delegation.
func TestEscrowReporterStakeZeroAmountWithPositiveOrigin(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	valA := sdk.ValAddress(sample.AccAddressBytes())
	origin := math.NewInt(1_000)
	hashID := []byte("hash-zero-origin")

	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("q"), collections.Join(reporterAddr.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{
			DelegatorAddress: reporterAddr.Bytes(),
			ValidatorAddress: valA.Bytes(),
			Amount:           origin,
		}},
		Total: origin,
	}))

	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, 1, uint64(ctx.BlockHeight()), math.ZeroInt(), []byte("q"), hashID))

	stored, err := k.DisputedDelegationAmounts.Get(ctx, hashID)
	require.NoError(t, err)
	require.True(t, stored.Total.IsZero())
	require.Empty(t, stored.TokenOrigins)
	sk.AssertNotCalled(t, "GetDelegation", mock.Anything, mock.Anything, mock.Anything)
	sk.AssertNotCalled(t, "Unbond", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	bk.AssertNotCalled(t, "SendCoinsFromModuleToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestEscrowReporterStakeNeverCollectsMoreThanRequestedFromTruncatedPower(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	selector1 := sample.AccAddressBytes()
	selector2 := sample.AccAddressBytes()
	valAddr1 := sdk.ValAddress(sample.AccAddressBytes())
	valAddr2 := sdk.ValAddress(sample.AccAddressBytes())
	queryID := []byte("truncated-power")
	hashID := []byte("truncated-power-hash")

	// ReporterStake stores all selected stake, while MicroReport.Power truncates it
	// to whole PowerReduction units. Allocation must use the exact snapshot total,
	// not the reconstructed amount from truncated report power.
	origin1Amount := math.NewInt(100_500_000)
	origin2Amount := math.NewInt(499_999)
	reportTotal := origin1Amount.Add(origin2Amount)
	reportPower := uint64(100)
	requested := sdk.DefaultPowerReduction.MulRaw(int64(reportPower))
	require.Equal(t, math.NewInt(100_999_999), reportTotal)
	require.Equal(t, math.NewInt(100_000_000), requested)
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporterAddr.Bytes(), uint64(ctx.BlockHeight()), queryID), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: selector1.Bytes(), ValidatorAddress: valAddr1.Bytes(), Amount: origin1Amount},
			{DelegatorAddress: selector2.Bytes(), ValidatorAddress: valAddr2.Bytes(), Amount: origin2Amount},
		},
		Total: reportTotal,
	}))

	expectedOrigin1 := origin1Amount.Mul(requested).Quo(reportTotal)
	expectedOrigin2 := requested.Sub(expectedOrigin1)
	require.True(t, expectedOrigin1.IsPositive())
	require.True(t, expectedOrigin2.IsPositive())
	require.Equal(t, requested, expectedOrigin1.Add(expectedOrigin2))

	validator1 := stakingtypes.Validator{
		OperatorAddress: valAddr1.String(),
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(200_000_000),
		DelegatorShares: math.NewInt(200_000_000).ToLegacyDec(),
	}
	delegation1 := stakingtypes.Delegation{
		DelegatorAddress: selector1.String(),
		ValidatorAddress: valAddr1.String(),
		Shares:           math.NewInt(150_000_000).ToLegacyDec(),
	}
	sharesToUnbond1, err := validator1.SharesFromTokens(expectedOrigin1)
	require.NoError(t, err)
	sk.On("GetDelegation", ctx, selector1, valAddr1).Return(delegation1, nil).Once()
	sk.On("GetValidator", ctx, valAddr1).Return(validator1, nil).Once()
	sk.On("Unbond", ctx, selector1, valAddr1, sharesToUnbond1).Return(expectedOrigin1, nil).Once()
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", expectedOrigin1))).Return(nil).Once()

	validator2 := stakingtypes.Validator{
		OperatorAddress: valAddr2.String(),
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(1_000_000),
		DelegatorShares: math.NewInt(1_000_000).ToLegacyDec(),
	}
	delegation2 := stakingtypes.Delegation{
		DelegatorAddress: selector2.String(),
		ValidatorAddress: valAddr2.String(),
		Shares:           math.NewInt(600_000).ToLegacyDec(),
	}
	sharesToUnbond2, err := validator2.SharesFromTokens(expectedOrigin2)
	require.NoError(t, err)
	sk.On("GetDelegation", ctx, selector2, valAddr2).Return(delegation2, nil).Once()
	sk.On("GetValidator", ctx, valAddr2).Return(validator2, nil).Once()
	sk.On("Unbond", ctx, selector2, valAddr2, sharesToUnbond2).Return(expectedOrigin2, nil).Once()
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", expectedOrigin2))).Return(nil).Once()

	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, reportPower, uint64(ctx.BlockHeight()), requested, queryID, hashID))

	stored, err := k.DisputedDelegationAmounts.Get(ctx, hashID)
	require.NoError(t, err)
	require.Equal(t, requested, stored.Total)
	require.Len(t, stored.TokenOrigins, 2)
	allocated := math.ZeroInt()
	for _, origin := range stored.TokenOrigins {
		require.False(t, origin.Amount.IsNegative())
		allocated = allocated.Add(origin.Amount)
	}
	require.Equal(t, requested, allocated)
	require.False(t, stored.Total.GT(requested), "escrowed %s, exceeding requested slash %s", stored.Total, requested)
	sk.AssertExpectations(t)
	bk.AssertExpectations(t)
}

func TestEscrowReporterStakeMeasuredShortfallChasesSameValidatorUnbonding(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	originAmount := math.NewInt(1_000_000_000)
	owed := math.NewInt(1000)
	bondedCollected := math.NewInt(999)
	unbondingDust := math.OneInt()

	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(reporterAddr.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: reporterAddr, ValidatorAddress: valAddr, Amount: originAmount},
		},
		Total: originAmount,
	}))

	validator := stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
		Tokens:          math.NewInt(2000),
		DelegatorShares: math.LegacyNewDec(2000),
		Status:          stakingtypes.Bonded,
	}
	delegation := stakingtypes.Delegation{
		DelegatorAddress: reporterAddr.String(),
		ValidatorAddress: valAddr.String(),
		Shares:           math.LegacyNewDec(2000),
	}
	sharesToUnbond, err := validator.SharesFromTokens(owed)
	require.NoError(t, err)

	ubd := stakingtypes.UnbondingDelegation{
		DelegatorAddress: reporterAddr.String(),
		ValidatorAddress: valAddr.String(),
		Entries: []stakingtypes.UnbondingDelegationEntry{
			{CreationHeight: ctx.BlockHeight(), InitialBalance: unbondingDust, Balance: unbondingDust},
		},
	}
	emptyUBD := ubd
	emptyUBD.Entries = nil

	sk.On("GetDelegation", ctx, reporterAddr, valAddr).Return(delegation, nil)
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	sk.On("Unbond", ctx, reporterAddr, valAddr, sharesToUnbond).Return(bondedCollected, nil)
	sk.On("GetUnbondingDelegation", ctx, reporterAddr, valAddr).Return(ubd, nil)
	sk.On("RemoveUnbondingDelegation", ctx, emptyUBD).Return(nil)

	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", bondedCollected))).Return(nil)
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.NotBondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", unbondingDust))).Return(nil)

	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, 1, uint64(ctx.BlockHeight()), owed, []byte{}, []byte("hashId")))

	stored, err := k.DisputedDelegationAmounts.Get(ctx, []byte("hashId"))
	require.NoError(t, err)
	require.Equal(t, owed, stored.Total)
	sum := math.ZeroInt()
	for _, origin := range stored.TokenOrigins {
		sum = sum.Add(origin.Amount)
	}
	require.Equal(t, stored.Total, sum)
	sk.AssertExpectations(t)
	bk.AssertExpectations(t)
}

func TestEscrowReporterStakeCollectsFromLastWideFanOutDestination(t *testing.T) {
	k, sk, bk, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	delegatorAddr := sample.AccAddressBytes()
	src := sdk.ValAddress(sample.AccAddressBytes())
	queryID := []byte("wide-fan-out")
	hashID := []byte("wide-fan-out-hash")
	requested := math.NewInt(1000)

	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporterAddr.Bytes(), uint64(ctx.BlockHeight()), queryID), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{
			DelegatorAddress: delegatorAddr.Bytes(),
			ValidatorAddress: src.Bytes(),
			Amount:           requested,
		}},
		Total: requested,
	}))

	// The source and the first eight direct destinations have no remaining
	// delegation or unbonding balance. The ninth destination holds the full
	// amount and must still be attempted.
	sk.On("GetDelegation", ctx, delegatorAddr, src).Return(stakingtypes.Delegation{}, stakingtypes.ErrNoDelegation).Once()
	sk.On("GetUnbondingDelegation", ctx, delegatorAddr, src).Return(stakingtypes.UnbondingDelegation{}, stakingtypes.ErrNoUnbondingDelegation).Once()

	const directDestinations = 9
	destinations := make([]sdk.ValAddress, directDestinations)
	redelegations := make([]stakingtypes.Redelegation, 0, directDestinations)
	for i := range destinations {
		destinations[i] = sdk.ValAddress(sample.AccAddressBytes())
		redelegations = append(redelegations, stakingtypes.Redelegation{
			DelegatorAddress:    delegatorAddr.String(),
			ValidatorSrcAddress: src.String(),
			ValidatorDstAddress: destinations[i].String(),
			Entries: []stakingtypes.RedelegationEntry{{
				CreationHeight: ctx.BlockHeight(),
				InitialBalance: requested,
				SharesDst:      requested.ToLegacyDec(),
			}},
		})
		if i < len(destinations)-1 {
			sk.On("GetDelegation", ctx, delegatorAddr, destinations[i]).Return(stakingtypes.Delegation{}, stakingtypes.ErrNoDelegation).Once()
			sk.On("GetUnbondingDelegation", ctx, delegatorAddr, destinations[i]).Return(stakingtypes.UnbondingDelegation{}, stakingtypes.ErrNoUnbondingDelegation).Once()
		}
	}
	mockIterateDelegatorRedelegations(sk, ctx, delegatorAddr, redelegations)

	last := destinations[len(destinations)-1]
	validator := stakingtypes.Validator{
		OperatorAddress: last.String(),
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(2000),
		DelegatorShares: math.LegacyNewDec(2000),
	}
	delegation := stakingtypes.Delegation{
		DelegatorAddress: delegatorAddr.String(),
		ValidatorAddress: last.String(),
		Shares:           math.LegacyNewDec(1000),
	}
	sharesToUnbond, err := validator.SharesFromTokens(requested)
	require.NoError(t, err)
	sk.On("GetDelegation", ctx, delegatorAddr, last).Return(delegation, nil).Once()
	sk.On("GetValidator", ctx, last).Return(validator, nil).Once()
	sk.On("Unbond", ctx, delegatorAddr, last, sharesToUnbond).Return(requested, nil).Once()
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", requested))).Return(nil).Once()

	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, 1, uint64(ctx.BlockHeight()), requested, queryID, hashID))

	stored, err := k.DisputedDelegationAmounts.Get(ctx, hashID)
	require.NoError(t, err)
	require.Equal(t, requested, stored.Total)
	require.Len(t, stored.TokenOrigins, 1)
	require.Equal(t, last.Bytes(), stored.TokenOrigins[0].ValidatorAddress)
	sk.AssertExpectations(t)
	bk.AssertExpectations(t)
}

// TestGetRedelegationPathMultiHop verifies that getRedelegationPath follows a
// chain of redelegations. With a val1 -> val2 -> val3 chain, it must return
// every hop destination in order so callers can collect stake parked at any of
// them, not just the terminal validator.
func TestGetRedelegationPathMultiHop(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	val1 := sdk.ValAddress(sample.AccAddressBytes())
	val2 := sdk.ValAddress(sample.AccAddressBytes())
	val3 := sdk.ValAddress(sample.AccAddressBytes())

	amt := math.NewInt(1000)
	mockIterateDelegatorRedelegations(sk, ctx, delAddr, []stakingtypes.Redelegation{
		{
			DelegatorAddress:    delAddr.String(),
			ValidatorSrcAddress: val1.String(),
			ValidatorDstAddress: val2.String(),
			Entries:             []stakingtypes.RedelegationEntry{{CreationHeight: ctx.BlockHeight(), InitialBalance: amt, SharesDst: amt.ToLegacyDec()}},
		},
		{
			DelegatorAddress:    delAddr.String(),
			ValidatorSrcAddress: val2.String(),
			ValidatorDstAddress: val3.String(),
			Entries:             []stakingtypes.RedelegationEntry{{CreationHeight: ctx.BlockHeight(), InitialBalance: amt, SharesDst: amt.ToLegacyDec()}},
		},
	})

	got, err := k.GetRedelegationPathExported(ctx, delAddr, val1)
	require.NoError(t, err)
	require.Equal(t, []sdk.ValAddress{val2, val3}, got)
}

func TestGetRedelegationPathUnrelatedDelegatorsDoNotInflateExaminedRecordGas(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	otherDelegator := sample.AccAddressBytes()
	src := sdk.ValAddress(sample.AccAddressBytes())
	dst := sdk.ValAddress(sample.AccAddressBytes())
	targetRecords := []stakingtypes.Redelegation{{
		DelegatorAddress:    delAddr.String(),
		ValidatorSrcAddress: src.String(),
		ValidatorDstAddress: dst.String(),
	}}
	const unrelatedRecordCount = 25
	unrelatedRecords := make([]stakingtypes.Redelegation, unrelatedRecordCount)
	for i := range unrelatedRecords {
		unrelatedRecords[i] = stakingtypes.Redelegation{
			DelegatorAddress:    otherDelegator.String(),
			ValidatorSrcAddress: sdk.ValAddress(sample.AccAddressBytes()).String(),
			ValidatorDstAddress: sdk.ValAddress(sample.AccAddressBytes()).String(),
		}
	}
	recordsByDelegator := map[string][]stakingtypes.Redelegation{
		delAddr.String():        targetRecords,
		otherDelegator.String(): unrelatedRecords,
	}
	sk.On("IterateDelegatorRedelegations", ctx, mock.Anything, mock.AnythingOfType("func(types.Redelegation) bool")).
		Run(func(args mock.Arguments) {
			delegator := args.Get(1).(sdk.AccAddress)
			callback := args.Get(2).(func(stakingtypes.Redelegation) bool)
			for _, redelegation := range recordsByDelegator[delegator.String()] {
				if callback(redelegation) {
					return
				}
			}
		}).
		Return(nil).
		Once()

	gasBefore := ctx.GasMeter().GasConsumed()
	got, err := k.GetRedelegationPathExported(ctx, delAddr, src)
	require.NoError(t, err)
	require.Equal(t, []sdk.ValAddress{dst}, got)
	gasUsed := ctx.GasMeter().GasConsumed() - gasBefore
	require.Equal(t, reporterkeeper.RedelegationRecordGas, gasUsed)
	require.NotEqual(t, reporterkeeper.RedelegationRecordGas*(1+unrelatedRecordCount), gasUsed)
	sk.AssertExpectations(t)
}

func TestGetRedelegationPathReturnsAllDirectDestinationsInIteratorOrder(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	src := sdk.ValAddress(sample.AccAddressBytes())
	const directDestinations = 7
	destinations := make([]sdk.ValAddress, directDestinations)
	redelegations := make([]stakingtypes.Redelegation, 0, len(destinations))
	for i := range destinations {
		destinations[i] = sdk.ValAddress(sample.AccAddressBytes())
		redelegations = append(redelegations, stakingtypes.Redelegation{
			DelegatorAddress:    delAddr.String(),
			ValidatorSrcAddress: src.String(),
			ValidatorDstAddress: destinations[i].String(),
			Entries:             []stakingtypes.RedelegationEntry{{CreationHeight: ctx.BlockHeight(), InitialBalance: math.OneInt(), SharesDst: math.LegacyOneDec()}},
		})
	}
	mockIterateDelegatorRedelegations(sk, ctx, delAddr, redelegations)

	got, err := k.GetRedelegationPathExported(ctx, delAddr, src)
	require.NoError(t, err)
	require.Equal(t, destinations, got)
}

func TestGetRedelegationPathReturnsAllWideFanOutDestinationsAndChargesRecords(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	src := sdk.ValAddress(sample.AccAddressBytes())
	const directDestinations = 9
	destinations := make([]sdk.ValAddress, directDestinations)
	redelegations := make([]stakingtypes.Redelegation, 0, len(destinations))
	for i := range destinations {
		destinations[i] = sdk.ValAddress(sample.AccAddressBytes())
		redelegations = append(redelegations, stakingtypes.Redelegation{
			DelegatorAddress:    delAddr.String(),
			ValidatorSrcAddress: src.String(),
			ValidatorDstAddress: destinations[i].String(),
			Entries:             []stakingtypes.RedelegationEntry{{CreationHeight: ctx.BlockHeight(), InitialBalance: math.OneInt(), SharesDst: math.LegacyOneDec()}},
		})
	}
	mockIterateDelegatorRedelegations(sk, ctx, delAddr, redelegations)

	gasBefore := ctx.GasMeter().GasConsumed()
	got, err := k.GetRedelegationPathExported(ctx, delAddr, src)
	require.NoError(t, err)
	require.Equal(t, destinations, got)
	gasUsed := ctx.GasMeter().GasConsumed() - gasBefore
	require.Equal(t, reporterkeeper.RedelegationRecordGas*directDestinations, gasUsed)
	sk.AssertExpectations(t)
}

// TestGetRedelegationPathSkipsCycle verifies that a cycle is deduped instead
// of treated as an escrow-fatal error.
func TestGetRedelegationPathSkipsCycle(t *testing.T) {
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
	redelegations := append(red(val1, val2), red(val2, val1)...)
	mockIterateDelegatorRedelegations(sk, ctx, delAddr, redelegations)

	got, err := k.GetRedelegationPathExported(ctx, delAddr, val1)
	require.NoError(t, err)
	require.Equal(t, []sdk.ValAddress{val2}, got)
}

// TestGetRedelegationPathFollowsLongFiniteChain verifies traversal does not
// silently omit a reachable destination based on an internal depth limit.
func TestGetRedelegationPathFollowsLongFiniteChain(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	delAddr := sample.AccAddressBytes()
	amt := math.NewInt(1000)

	vals := make([]sdk.ValAddress, 9)
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
	redelegations := make([]stakingtypes.Redelegation, 0, len(vals)-1)
	for i := 0; i < len(vals)-1; i++ {
		redelegations = append(redelegations, red(vals[i], vals[i+1])...)
	}
	mockIterateDelegatorRedelegations(sk, ctx, delAddr, redelegations)

	got, err := k.GetRedelegationPathExported(ctx, delAddr, vals[0])
	require.NoError(t, err)
	require.Equal(t, vals[1:], got)
}
