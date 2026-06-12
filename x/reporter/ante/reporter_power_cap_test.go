package ante

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// noopNext lets tests run the decorator in isolation.
func noopNext(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	return ctx, nil
}

func TestReporterPowerCapSelect(t *testing.T) {
	testCases := []struct {
		name          string
		selectorStake math.Int
		err           error
	}{
		{
			// reporter 20 + selector 9 = 29 < 30% of 100
			name:          "allows select below the cap",
			selectorStake: math.NewInt(9),
			err:           nil,
		},
		{
			// reporter 20 + selector 10 = 30, reaching 30% of 100 is rejected
			name:          "blocks select reaching the cap",
			selectorStake: math.NewInt(10),
			err:           types.ErrExceedsMaxReporterPower,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
			ctx = ctx.WithBlockHeight(1)
			decorator := NewTrackStakeChangesDecorator(k, sk)

			valAddr := sdk.ValAddress(sample.AccAddressBytes())
			val := validator(valAddr, stakingtypes.Bonded, math.NewInt(1000))
			reporterAddr := sample.AccAddressBytes()
			selectorAddr := sample.AccAddressBytes()
			require.NoError(t, k.Selectors.Set(ctx, reporterAddr.Bytes(), types.NewSelection(reporterAddr.Bytes(), 1)))

			mockValidator(sk, ctx, val)
			sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
			mockIterateDelegations(sk, ctx, reporterAddr, []stakingtypes.Delegation{delegation(reporterAddr, valAddr, math.NewInt(20))})
			mockIterateDelegations(sk, ctx, selectorAddr, []stakingtypes.Delegation{delegation(selectorAddr, valAddr, tc.selectorStake)})

			tx := buildTx(t, &types.MsgSelectReporter{
				SelectorAddress: selectorAddr.String(),
				ReporterAddress: reporterAddr.String(),
			})

			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReporterPowerCapCreateReporter(t *testing.T) {
	testCases := []struct {
		name         string
		creatorStake math.Int
		err          error
	}{
		{
			name:         "allows create below the cap",
			creatorStake: math.NewInt(29),
			err:          nil,
		},
		{
			name:         "blocks create reaching the cap",
			creatorStake: math.NewInt(30),
			err:          types.ErrExceedsMaxReporterPower,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
			ctx = ctx.WithBlockHeight(1)
			decorator := NewTrackStakeChangesDecorator(k, sk)

			valAddr := sdk.ValAddress(sample.AccAddressBytes())
			val := validator(valAddr, stakingtypes.Bonded, math.NewInt(1000))
			creatorAddr := sample.AccAddressBytes()

			mockValidator(sk, ctx, val)
			sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
			mockIterateDelegations(sk, ctx, creatorAddr, []stakingtypes.Delegation{delegation(creatorAddr, valAddr, tc.creatorStake)})

			tx := buildTx(t, &types.MsgCreateReporter{
				ReporterAddress:   creatorAddr.String(),
				CommissionRate:    math.LegacyZeroDec(),
				MinTokensRequired: math.OneInt(),
				Moniker:           "moniker",
			})

			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReporterPowerCapSwitch(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)

	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	val := validator(valAddr, stakingtypes.Bonded, math.NewInt(1000))
	fromReporter := sample.AccAddressBytes()
	toReporter := sample.AccAddressBytes()
	selectorAddr := sample.AccAddressBytes()
	require.NoError(t, k.Selectors.Set(ctx, toReporter.Bytes(), types.NewSelection(toReporter.Bytes(), 1)))
	require.NoError(t, k.Selectors.Set(ctx, selectorAddr.Bytes(), types.NewSelection(fromReporter.Bytes(), 1)))

	mockValidator(sk, ctx, val)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	mockIterateDelegations(sk, ctx, toReporter, []stakingtypes.Delegation{delegation(toReporter, valAddr, math.NewInt(20))})
	mockIterateDelegations(sk, ctx, selectorAddr, []stakingtypes.Delegation{delegation(selectorAddr, valAddr, math.NewInt(10))})

	// destination reporter 20 + switching selector 10 reaches 30% of 100
	tx := buildTx(t, &types.MsgSwitchReporter{
		SelectorAddress: selectorAddr.String(),
		ReporterAddress: toReporter.String(),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.ErrorIs(t, err, types.ErrExceedsMaxReporterPower)
}

func TestReporterPowerCapSwitchAlreadyPending(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)

	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	val := validator(valAddr, stakingtypes.Bonded, math.NewInt(1000))
	fromReporter := sample.AccAddressBytes()
	toReporter := sample.AccAddressBytes()
	selectorAddr := sample.AccAddressBytes()
	require.NoError(t, k.Selectors.Set(ctx, toReporter.Bytes(), types.NewSelection(toReporter.Bytes(), 1)))
	require.NoError(t, k.Selectors.Set(ctx, selectorAddr.Bytes(), types.NewSelection(fromReporter.Bytes(), 1)))
	// the switch is already scheduled, so its stake is already booked against
	// the destination's potential stake and a re-send must not double count
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(fromReporter.Bytes(), selectorAddr.Bytes()), types.PendingSwitchEntry{
		ToReporter:  toReporter.Bytes(),
		UnlockBlock: 100,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(toReporter.Bytes(), selectorAddr.Bytes()), fromReporter.Bytes()))

	mockValidator(sk, ctx, val)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	mockIterateDelegations(sk, ctx, toReporter, []stakingtypes.Delegation{delegation(toReporter, valAddr, math.NewInt(20))})
	mockIterateDelegations(sk, ctx, selectorAddr, []stakingtypes.Delegation{delegation(selectorAddr, valAddr, math.NewInt(10))})

	// the handler treats this as a no-op, so the ante must not block it even
	// though destination potential stake (20 + 10 pending) is at the cap
	tx := buildTx(t, &types.MsgSwitchReporter{
		SelectorAddress: selectorAddr.String(),
		ReporterAddress: toReporter.String(),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.NoError(t, err)
}

func TestReporterPowerCapDelegateBySelector(t *testing.T) {
	testCases := []struct {
		name        string
		delegateAmt math.Int
		err         error
	}{
		{
			// reporter potential 29 + 1 = 30 vs 30% of 101 = 30.3
			name:        "allows delegate keeping reporter below cap",
			delegateAmt: math.OneInt(),
			err:         nil,
		},
		{
			// reporter potential 29 + 2 = 31 vs 30% of 102 = 30.6
			name:        "blocks delegate pushing reporter to cap",
			delegateAmt: math.NewInt(2),
			err:         types.ErrExceedsMaxReporterPower,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
			ctx = ctx.WithBlockHeight(1)
			decorator := NewTrackStakeChangesDecorator(k, sk)

			valAddr := sdk.ValAddress(sample.AccAddressBytes())
			val := validator(valAddr, stakingtypes.Bonded, math.NewInt(1000))
			reporterAddr := sample.AccAddressBytes()
			selectorAddr := sample.AccAddressBytes()
			require.NoError(t, k.Selectors.Set(ctx, reporterAddr.Bytes(), types.NewSelection(reporterAddr.Bytes(), 1)))
			require.NoError(t, k.Selectors.Set(ctx, selectorAddr.Bytes(), types.NewSelection(reporterAddr.Bytes(), 1)))

			selectorDelegations := []stakingtypes.Delegation{delegation(selectorAddr, valAddr, math.NewInt(9))}
			mockValidator(sk, ctx, val)
			sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
			sk.On("GetAllDelegatorDelegations", ctx, selectorAddr).Return(selectorDelegations, nil)
			mockIterateDelegations(sk, ctx, reporterAddr, []stakingtypes.Delegation{delegation(reporterAddr, valAddr, math.NewInt(20))})
			mockIterateDelegations(sk, ctx, selectorAddr, selectorDelegations)

			tx := buildTx(t, &stakingtypes.MsgDelegate{
				DelegatorAddress: selectorAddr.String(),
				ValidatorAddress: valAddr.String(),
				Amount:           sdk.Coin{Denom: "loya", Amount: tc.delegateAmt},
			})

			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReporterPowerCapSelectPlusDelegate(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)

	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	val := validator(valAddr, stakingtypes.Bonded, math.NewInt(1000))
	reporterAddr := sample.AccAddressBytes()
	selectorAddr := sample.AccAddressBytes()
	require.NoError(t, k.Selectors.Set(ctx, reporterAddr.Bytes(), types.NewSelection(reporterAddr.Bytes(), 1)))

	selectorDelegations := []stakingtypes.Delegation{delegation(selectorAddr, valAddr, math.NewInt(5))}
	mockValidator(sk, ctx, val)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	sk.On("GetAllDelegatorDelegations", ctx, selectorAddr).Return(selectorDelegations, nil)
	mockIterateDelegations(sk, ctx, reporterAddr, []stakingtypes.Delegation{delegation(reporterAddr, valAddr, math.NewInt(20))})
	mockIterateDelegations(sk, ctx, selectorAddr, selectorDelegations)

	// The selector joins with 5 bonded and delegates 10 more in the same tx.
	// The reporter's projected stake is 20 + (5 + 10) = 35, which reaches 30%
	// of the projected total of 110. Without folding the same-tx delegation
	// into the joiner's contribution this would pass at 25 / 110.
	tx := buildTx(t,
		&types.MsgSelectReporter{
			SelectorAddress: selectorAddr.String(),
			ReporterAddress: reporterAddr.String(),
		},
		&stakingtypes.MsgDelegate{
			DelegatorAddress: selectorAddr.String(),
			ValidatorAddress: valAddr.String(),
			Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(10)},
		},
	)

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.ErrorIs(t, err, types.ErrExceedsMaxReporterPower)
}

func TestReporterPowerCapDisabled(t *testing.T) {
	testCases := []struct {
		name  string
		share math.LegacyDec
	}{
		{
			name:  "share of one disables the check",
			share: math.LegacyOneDec(),
		},
		{
			name:  "nil share (pre-migration) disables the check",
			share: math.LegacyDec{},
		},
		{
			name:  "zero share disables the check",
			share: math.LegacyZeroDec(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
			ctx = ctx.WithBlockHeight(1)
			decorator := NewTrackStakeChangesDecorator(k, sk)
			params := types.DefaultParams()
			params.MaxReporterPowerShare = tc.share
			require.NoError(t, k.Params.Set(ctx, params))

			valAddr := sdk.ValAddress(sample.AccAddressBytes())
			val := validator(valAddr, stakingtypes.Bonded, math.NewInt(1000))
			reporterAddr := sample.AccAddressBytes()
			selectorAddr := sample.AccAddressBytes()
			require.NoError(t, k.Selectors.Set(ctx, reporterAddr.Bytes(), types.NewSelection(reporterAddr.Bytes(), 1)))

			mockValidator(sk, ctx, val)
			sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
			mockIterateDelegations(sk, ctx, reporterAddr, []stakingtypes.Delegation{delegation(reporterAddr, valAddr, math.NewInt(40))})
			mockIterateDelegations(sk, ctx, selectorAddr, []stakingtypes.Delegation{delegation(selectorAddr, valAddr, math.NewInt(20))})

			// 40 + 20 = 60% of total bonded would be far over an active cap
			tx := buildTx(t, &types.MsgSelectReporter{
				SelectorAddress: selectorAddr.String(),
				ReporterAddress: reporterAddr.String(),
			})

			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			require.NoError(t, err)
		})
	}
}

func TestReporterPowerCapUndelegateNotBlocked(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)

	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	val := validator(valAddr, stakingtypes.Bonded, math.NewInt(1000))
	reporterAddr := sample.AccAddressBytes()
	// the reporter is already over the cap (40% of total bonded) but only sheds
	// stake, which must always be allowed
	require.NoError(t, k.Selectors.Set(ctx, reporterAddr.Bytes(), types.NewSelection(reporterAddr.Bytes(), 1)))

	mockValidator(sk, ctx, val)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	mockDelegation(sk, ctx, reporterAddr, valAddr, math.NewInt(40))
	mockPowerStore(sk, ctx, 1, valAddr)
	mockIterateDelegations(sk, ctx, reporterAddr, []stakingtypes.Delegation{delegation(reporterAddr, valAddr, math.NewInt(40))})

	tx := buildTx(t, &stakingtypes.MsgUndelegate{
		DelegatorAddress: reporterAddr.String(),
		ValidatorAddress: valAddr.String(),
		Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.NoError(t, err)
}
