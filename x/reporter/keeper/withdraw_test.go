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
	k, sk, bk, _, ctx, _ := setupKeeper(t)
	fee := math.NewIntWithDecimal(100, 6)
	reporterAddr, selector1, selector2, selector3 := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()

	err := k.FeefromReporterStake(ctx, reporterAddr, fee, []byte("hashId"))
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
	err = k.FeefromReporterStake(ctx, reporterAddr, math.NewIntWithDecimal(100, 6), []byte("hashId"))
	require.NoError(t, err)

	feefromstake, err := k.FeePaidFromStake.Get(ctx, []byte("hashId"))
	require.NoError(t, err)
	expected := tokenShare1.TruncateInt().Add(tokenShare2.TruncateInt()).Add(tokenShare3.TruncateInt())
	require.Equal(t, expected, feefromstake.Total)
}

func TestFeefromReporterStakeMultiplevalidators(t *testing.T) {
	k, sk, bk, _, ctx, _ := setupKeeper(t)
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
	err := k.FeefromReporterStake(ctx, reporterAddr, fee, []byte("hashId"))
	require.NoError(t, err)

	feefromstake, err := k.FeePaidFromStake.Get(ctx, []byte("hashId"))
	require.NoError(t, err)
	expected := fee
	require.Equal(t, expected, feefromstake.Total)

	err = k.FeefromReporterStake(ctx, reporterAddr, fee, []byte("hashId"))
	require.NoError(t, err)

	feefromstake, err = k.FeePaidFromStake.Get(ctx, []byte("hashId"))
	require.NoError(t, err)
	expected = fee.MulRaw(2)
	require.Equal(t, expected, feefromstake.Total)
}

func TestEscrowReporterStake(t *testing.T) {
	k, sk, bk, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	stake := math.NewIntWithDecimal(100, 6)
	require.NoError(t, k.Report.Set(ctx, collections.Join(reporterAddr.Bytes(), uint64(ctx.BlockHeight())), types.DelegationsAmounts{
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
	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, 100, uint64(ctx.BlockHeight()), stake, []byte("hashId")))
}

func TestEscrowReporterStakeUnbondingdelegations(t *testing.T) {
	k, sk, bk, _, ctx, _ := setupKeeper(t)
	reporterAddr, selector2, selector3, valAddr1, valAddr2 := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	stake := math.NewIntWithDecimal(1000, 6)
	require.NoError(t, k.Report.Set(ctx, collections.Join(reporterAddr.Bytes(), uint64(ctx.BlockHeight())), types.DelegationsAmounts{
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
			{CreationHeight: ctx.BlockHeight(), InitialBalance: math.NewInt(500000001), Balance: math.NewInt(500000001)},
		},
	}).Return(nil)

	sk.On("GetValidator", ctx, sdk.ValAddress(valAddr1)).Return(validator1, nil)
	sk.On("GetValidator", ctx, sdk.ValAddress(valAddr2)).Return(validator2, nil)

	sk.On("Unbond", ctx, reporterAddr, sdk.ValAddress(valAddr1), delegation1.Shares.Quo(math.LegacyNewDec(2)).Sub(math.LegacyOneDec())).Return(stake.QuoRaw(2).SubRaw(1), nil)
	sk.On("Unbond", ctx, selector3, sdk.ValAddress(valAddr2), math.LegacyNewDec(500000002)).Return(math.NewInt(500000002), nil)

	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", stake.QuoRaw(2).SubRaw(1)))).Return(nil)
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.NotBondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", stake.QuoRaw(2).SubRaw(1)))).Return(nil)
	bk.On("SendCoinsFromModuleToModule", ctx, stakingtypes.BondedPoolName, "dispute", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(500000002)))).Return(nil)

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

	require.NoError(t, k.EscrowReporterStake(ctx, reporterAddr, 3000, uint64(ctx.BlockHeight()), math.NewIntWithDecimal(1500, 6), []byte("hashId")))
}
