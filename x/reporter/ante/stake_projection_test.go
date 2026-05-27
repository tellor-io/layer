package ante

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/encoding"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestShareCapReplacement(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(200)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: currentTotal}))

	attackerAddr := sample.AccAddressBytes()
	otherAddr := sample.AccAddressBytes()
	leavingValAddr := sdk.ValAddress(sample.AccAddressBytes())
	steadyValAddr := sdk.ValAddress(sample.AccAddressBytes())
	candidateValAddr := sdk.ValAddress(sample.AccAddressBytes())
	leavingVal := validator(leavingValAddr, stakingtypes.Bonded, math.NewInt(100))
	steadyVal := validator(steadyValAddr, stakingtypes.Bonded, math.NewInt(100))
	candidateVal := validator(candidateValAddr, stakingtypes.Unbonded, math.NewInt(96))
	candidateDelegations := []stakingtypes.Delegation{delegation(attackerAddr, candidateValAddr, math.NewInt(96))}
	leavingDelegations := []stakingtypes.Delegation{delegation(otherAddr, leavingValAddr, math.NewInt(100))}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	mockValidator(sk, ctx, leavingVal)
	mockValidator(sk, ctx, steadyVal)
	mockValidator(sk, ctx, candidateVal)
	mockPowerStore(sk, ctx, 2, leavingValAddr, steadyValAddr, candidateValAddr)
	mockDelegation(sk, ctx, otherAddr, leavingValAddr, math.NewInt(100))
	sk.On("GetValidatorDelegations", ctx, candidateValAddr).Return(candidateDelegations, nil)
	sk.On("GetValidatorDelegations", ctx, leavingValAddr).Return(leavingDelegations, nil)
	mockIterateDelegations(sk, ctx, attackerAddr, candidateDelegations)

	// The tx only removes 5 active tokens, which is allowed by the 5% rule.
	// That drop lets a 96-token inactive candidate enter the active set, so the
	// candidate delegator's prospective 96 / 196 bonded share must be rejected.
	tx := buildTx(t, &stakingtypes.MsgUndelegate{
		DelegatorAddress: otherAddr.String(),
		ValidatorAddress: leavingValAddr.String(),
		Amount:           coin(5),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestActiveSetDelegationExpansionConsumesGas(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(200)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: currentTotal}))
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(ActiveSetDelegationCheckGas * 2))

	leavingDelegator := sample.AccAddressBytes()
	candidateDelegators := []sdk.AccAddress{
		sample.AccAddressBytes(),
		sample.AccAddressBytes(),
		sample.AccAddressBytes(),
	}
	leavingValAddr := sdk.ValAddress(sample.AccAddressBytes())
	steadyValAddr := sdk.ValAddress(sample.AccAddressBytes())
	candidateValAddr := sdk.ValAddress(sample.AccAddressBytes())
	leavingVal := validator(leavingValAddr, stakingtypes.Bonded, math.NewInt(100))
	steadyVal := validator(steadyValAddr, stakingtypes.Bonded, math.NewInt(100))
	candidateVal := validator(candidateValAddr, stakingtypes.Unbonded, math.NewInt(96))
	candidateDelegations := make([]stakingtypes.Delegation, 0, len(candidateDelegators))
	for _, delegator := range candidateDelegators {
		candidateDelegations = append(candidateDelegations, delegation(delegator, candidateValAddr, math.NewInt(32)))
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	mockValidator(sk, ctx, leavingVal)
	mockValidator(sk, ctx, steadyVal)
	mockValidator(sk, ctx, candidateVal)
	mockPowerStore(sk, ctx, 2, leavingValAddr, steadyValAddr, candidateValAddr)
	mockDelegation(sk, ctx, leavingDelegator, leavingValAddr, math.NewInt(100))
	sk.On("GetValidatorDelegations", ctx, candidateValAddr).Return(candidateDelegations, nil)

	tx := buildTx(t, &stakingtypes.MsgUndelegate{
		DelegatorAddress: leavingDelegator.String(),
		ValidatorAddress: leavingValAddr.String(),
		Amount:           coin(5),
	})

	require.Panics(t, func() {
		_, _ = decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
			return ctx, nil
		})
	})
}

func TestFivePercentReplacement(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: currentTotal}))

	otherAddr := sample.AccAddressBytes()
	leavingValAddr := valAddress(2)
	candidateValAddr := valAddress(1)
	leavingVal := validator(leavingValAddr, stakingtypes.Bonded, math.NewInt(100))
	candidateVal := validator(candidateValAddr, stakingtypes.Unbonded, math.NewInt(90))

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	mockValidator(sk, ctx, leavingVal)
	mockValidator(sk, ctx, candidateVal)
	mockDelegation(sk, ctx, otherAddr, leavingValAddr, math.NewInt(100))
	mockPowerStoreWithReduction(sk, ctx, 1, math.NewInt(10), leavingValAddr, candidateValAddr)
	sk.On("GetValidatorDelegations", ctx, candidateValAddr).Return([]stakingtypes.Delegation{}, nil)
	sk.On("GetValidatorDelegations", ctx, leavingValAddr).Return([]stakingtypes.Delegation{
		delegation(otherAddr, leavingValAddr, math.NewInt(100)),
	}, nil)

	// With a power reduction of 10, the 95-token outgoing validator and the
	// 90-token candidate tie at consensus power 9. The candidate address sorts
	// first, so the final bonded total is 90 and the full replacement exceeds
	// the allowed 5% decrease even though the explicit undelegate is only 5.
	tx := buildTx(t, &stakingtypes.MsgUndelegate{
		DelegatorAddress: otherAddr.String(),
		ValidatorAddress: leavingValAddr.String(),
		Amount:           coin(5),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorContains(t, err, "total stake decrease exceeds the allowed 5% threshold within a twelve-hour period")
}

func TestCreateValidatorInactive(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: currentTotal}))

	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	newValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedVal := validator(bondedValAddr, stakingtypes.Bonded, currentTotal)

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	mockValidator(sk, ctx, bondedVal)
	mockPowerStore(sk, ctx, 1, bondedValAddr)

	// Creating a validator records a candidate, but max validators is already
	// full and the existing bonded validator has more stake. The candidate stays
	// inactive, so neither the 5% rule nor the share cap should run on it.
	tx := buildTx(t, &stakingtypes.MsgCreateValidator{
		ValidatorAddress:  newValAddr.String(),
		MinSelfDelegation: math.OneInt(),
		Value:             coin(10),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.NoError(t, err)
}

func TestCreateValidatorActive(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: currentTotal}))

	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	newValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedVal := validator(bondedValAddr, stakingtypes.Bonded, currentTotal)

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	mockValidator(sk, ctx, bondedVal)
	mockPowerStore(sk, ctx, 2, bondedValAddr)

	// With room for a second active validator, the new 6-token validator would
	// enter the bonded set. That raises total bonded stake by more than 5%.
	tx := buildTx(t, &stakingtypes.MsgCreateValidator{
		ValidatorAddress:  newValAddr.String(),
		MinSelfDelegation: math.OneInt(),
		Value:             coin(6),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorContains(t, err, "total stake increase exceeds the allowed 5% threshold within a twelve-hour period")
}

func TestShareCapDecimals(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(101)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: currentTotal}))

	delAddr := sample.AccAddressBytes()
	valAAddr := sdk.ValAddress(sample.AccAddressBytes())
	valBAddr := sdk.ValAddress(sample.AccAddressBytes())
	valA := validator(valAAddr, stakingtypes.Bonded, math.NewInt(100))
	valA.DelegatorShares = math.NewInt(1000).ToLegacyDec()
	valB := validator(valBAddr, stakingtypes.Bonded, math.OneInt())
	delegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valAAddr.String(),
			Shares:           math.NewInt(297).ToLegacyDec(),
		},
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return(delegations, nil)
	mockValidator(sk, ctx, valA)
	mockValidator(sk, ctx, valB)
	mockIterateDelegations(sk, ctx, delAddr, delegations)

	// The existing delegation is worth 29.7 tokens, not 29. Truncating would
	// allow the new delegate, but the precise final share is 30.7 / 102.
	tx := buildTx(t, &stakingtypes.MsgDelegate{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valBAddr.String(),
		Amount:           coin(1),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestShareCapNewBonded(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	delegatorAddr := sample.AccAddressBytes()
	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedValidator := stakingtypes.Validator{
		OperatorAddress:   bondedValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            currentTotal,
		DelegatorShares:   currentTotal.ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	bondedDelegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: delegatorAddr.String(),
			ValidatorAddress: bondedValAddr.String(),
			Shares:           math.NewInt(29).ToLegacyDec(),
		},
	}
	candidateValAddr := sdk.ValAddress(sample.AccAddressBytes())
	candidateValidator := stakingtypes.Validator{
		OperatorAddress:   candidateValAddr.String(),
		Status:            stakingtypes.Unbonded,
		Tokens:            math.ZeroInt(),
		DelegatorShares:   math.LegacyZeroDec(),
		MinSelfDelegation: math.OneInt(),
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, delegatorAddr).Return(bondedDelegations, nil)
	sk.On("GetValidator", ctx, candidateValAddr).Return(candidateValidator, nil)
	sk.On("GetValidator", ctx, bondedValAddr).Return(bondedValidator, nil)
	sk.On("MaxValidators", ctx).Return(uint32(2), nil)
	sk.On("PowerReduction", ctx).Return(math.OneInt())
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&validatorPowerIterator{values: [][]byte{bondedValAddr}}, nil)
	sk.On("GetValidatorDelegations", ctx, candidateValAddr).Return([]stakingtypes.Delegation{}, nil)
	sk.On("IterateDelegatorDelegations", ctx, delegatorAddr, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		for _, delegation := range bondedDelegations {
			if fn(delegation) {
				return
			}
		}
	})

	s := encoding.GetTestEncodingCfg()
	txBuilder := client.Context{}.WithTxConfig(s.TxConfig).TxConfig.NewTxBuilder()
	// The candidate starts unbonded, but this delegate would make it enter the
	// bonded set. The cap is checked against that prospective bonded state, so
	// the delegator's 31 / 102 final share is rejected.
	require.NoError(t, txBuilder.SetMsgs(&stakingtypes.MsgDelegate{
		DelegatorAddress: delegatorAddr.String(),
		ValidatorAddress: candidateValAddr.String(),
		Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(2)},
	}))

	_, err := decorator.AnteHandle(ctx, txBuilder.GetTx(), false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestShareCapExistingStake(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	attackerAddr := sample.AccAddressBytes()
	touchAddr := sample.AccAddressBytes()
	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	candidateValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedValidator := stakingtypes.Validator{
		OperatorAddress:   bondedValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            currentTotal,
		DelegatorShares:   currentTotal.ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	candidateValidator := stakingtypes.Validator{
		OperatorAddress:   candidateValAddr.String(),
		Status:            stakingtypes.Unbonded,
		Tokens:            math.NewInt(2),
		DelegatorShares:   math.NewInt(2).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	attackerDelegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: attackerAddr.String(),
			ValidatorAddress: bondedValAddr.String(),
			Shares:           math.NewInt(29).ToLegacyDec(),
		},
		{
			DelegatorAddress: attackerAddr.String(),
			ValidatorAddress: candidateValAddr.String(),
			Shares:           math.NewInt(2).ToLegacyDec(),
		},
	}
	candidateDelegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: attackerAddr.String(),
			ValidatorAddress: candidateValAddr.String(),
			Shares:           math.NewInt(2).ToLegacyDec(),
		},
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, touchAddr).Return([]stakingtypes.Delegation{}, nil)
	sk.On("GetValidator", ctx, bondedValAddr).Return(bondedValidator, nil)
	sk.On("GetValidator", ctx, candidateValAddr).Return(candidateValidator, nil)
	sk.On("MaxValidators", ctx).Return(uint32(2), nil)
	sk.On("PowerReduction", ctx).Return(math.OneInt())
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&validatorPowerIterator{values: [][]byte{bondedValAddr}}, nil)
	sk.On("GetValidatorDelegations", ctx, candidateValAddr).Return(candidateDelegations, nil)
	mockIterateDelegations(sk, ctx, attackerAddr, attackerDelegations)
	mockIterateDelegations(sk, ctx, touchAddr, []stakingtypes.Delegation{})

	// The attacker already has 29 bonded tokens and 2 inactive tokens on the
	// candidate. A separate 1-token delegate makes the candidate bonded, so the
	// attacker's final active stake becomes 31 / 103 and must fail.
	tx := buildTx(t, &stakingtypes.MsgDelegate{
		DelegatorAddress: touchAddr.String(),
		ValidatorAddress: candidateValAddr.String(),
		Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestShareCapStillUnbonded(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	delAddr := sample.AccAddressBytes()
	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	candidateValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedValidator := stakingtypes.Validator{
		OperatorAddress:   bondedValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            currentTotal,
		DelegatorShares:   currentTotal.ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	candidateValidator := stakingtypes.Validator{
		OperatorAddress:   candidateValAddr.String(),
		Status:            stakingtypes.Unbonded,
		Tokens:            math.NewInt(29),
		DelegatorShares:   math.NewInt(29).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}

	sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return([]stakingtypes.Delegation{}, nil)
	sk.On("GetValidator", ctx, candidateValAddr).Return(candidateValidator, nil)
	sk.On("GetValidator", ctx, bondedValAddr).Return(bondedValidator, nil)
	sk.On("MaxValidators", ctx).Return(uint32(1), nil)
	sk.On("PowerReduction", ctx).Return(math.OneInt())
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&validatorPowerIterator{values: [][]byte{bondedValAddr}}, nil)

	// The candidate grows from 29 to 30 tokens, but max validators is 1 and the
	// bonded validator still has 100. Since the candidate remains inactive,
	// the delegate does not increase bonded stake and should not trip the cap.
	tx := buildTx(t, &stakingtypes.MsgDelegate{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: candidateValAddr.String(),
		Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.NoError(t, err)
}

func TestFivePercentNewBonded(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	delegatorAddr := sample.AccAddressBytes()
	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedValidator := stakingtypes.Validator{
		OperatorAddress:   bondedValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            currentTotal,
		DelegatorShares:   currentTotal.ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	candidateValAddr := sdk.ValAddress(sample.AccAddressBytes())
	candidateValidator := stakingtypes.Validator{
		OperatorAddress:   candidateValAddr.String(),
		Status:            stakingtypes.Unbonded,
		Tokens:            math.NewInt(5),
		DelegatorShares:   math.NewInt(5).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, delegatorAddr).Return([]stakingtypes.Delegation{}, nil)
	sk.On("GetValidator", ctx, candidateValAddr).Return(candidateValidator, nil)
	sk.On("GetValidator", ctx, bondedValAddr).Return(bondedValidator, nil)
	sk.On("MaxValidators", ctx).Return(uint32(2), nil)
	sk.On("PowerReduction", ctx).Return(math.OneInt())
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&validatorPowerIterator{values: [][]byte{bondedValAddr}}, nil)

	s := encoding.GetTestEncodingCfg()
	txBuilder := client.Context{}.WithTxConfig(s.TxConfig).TxConfig.NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(&stakingtypes.MsgDelegate{
		DelegatorAddress: delegatorAddr.String(),
		ValidatorAddress: candidateValAddr.String(),
		Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
	}))

	_, err := decorator.AnteHandle(ctx, txBuilder.GetTx(), false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorContains(t, err, "total stake increase exceeds the allowed 5% threshold within a twelve-hour period")
}
