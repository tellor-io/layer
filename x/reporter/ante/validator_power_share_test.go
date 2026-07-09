package ante

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// setupPowerCapDecorator builds a decorator over a fresh keeper with the tracker
// set to trackerTotal and TotalBondedTokens mocked to the same value. Most
// validator-power-share ante tests start from this 100/100 baseline.
func setupPowerCapDecorator(t *testing.T, trackerTotal int64) (k keeper.Keeper, sk *mocks.StakingKeeper, ctx sdk.Context, decorator TrackStakeChangesDecorator) {
	t.Helper()
	k, sk, _, _, _, ctx, _ = keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator = NewTrackStakeChangesDecorator(k, sk)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: math.NewInt(trackerTotal)}))
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(trackerTotal), nil)
	return
}

func disableValidatorPowerCap(t *testing.T, k keeper.Keeper, ctx sdk.Context) {
	t.Helper()
	params := types.DefaultParams()
	params.MaxValidatorPowerShare = math.LegacyOneDec()
	require.NoError(t, k.Params.Set(ctx, params))
}

// mockRedelegatePair wires two 25-token bonded validators (src, dst), the power
// store over them, and a 25-token delegation del->src. Shared by the redelegate,
// disabled, and authz cap tests.
func mockRedelegatePair(sk *mocks.StakingKeeper, ctx sdk.Context) (srcVal, dstVal sdk.ValAddress, del sdk.AccAddress) {
	srcVal, dstVal = sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	mockValidator(sk, ctx, validator(srcVal, stakingtypes.Bonded, math.NewInt(25)))
	mockValidator(sk, ctx, validator(dstVal, stakingtypes.Bonded, math.NewInt(25)))
	mockPowerStore(sk, ctx, 2, dstVal, srcVal)
	del = sample.AccAddressBytes()
	mockDelegation(sk, ctx, del, srcVal, math.NewInt(25))
	sk.On("GetAllDelegatorDelegations", ctx, del).Return([]stakingtypes.Delegation{delegation(del, srcVal, math.NewInt(25))}, nil)
	return
}

func TestActiveSetProjectionBondedDelta(t *testing.T) {
	changes := activeSetChanges{
		entering: []prospectiveValidator{{postTokens: math.NewInt(15)}},
		leaving:  []prospectiveValidator{{postTokens: math.NewInt(4)}},
	}

	projection := newActiveSetProjection(changes)

	require.Equal(t, changes, projection.changes)
	require.Equal(t, math.NewInt(11), projection.bondedDelta)
}

func TestValidatorPowerShareRedelegate(t *testing.T) {
	testCases := []struct {
		name    string
		amount  int64
		wantErr error
	}{
		{"bonded-to-bonded over cap rejects", 6, types.ErrExceedsMaxValidatorPowerShare}, // dst 25+6=31 vs 30% of 100=30
		{"bonded-to-bonded exactly at cap succeeds", 5, nil},                             // dst 25+5=30 == cap
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, sk, ctx, decorator := setupPowerCapDecorator(t, 100)
			srcVal, dstVal, del := mockRedelegatePair(sk, ctx)
			tx := buildTx(t, &stakingtypes.MsgBeginRedelegate{
				DelegatorAddress:    del.String(),
				ValidatorSrcAddress: srcVal.String(),
				ValidatorDstAddress: dstVal.String(),
				Amount:              coin(tc.amount),
			})
			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatorPowerShareDelegate(t *testing.T) {
	// v0 starts at 29/100 (under the cap); a 2-token delegate makes it 31/102,
	// and 30% of 102 = 30.6, so 31 is strictly over. The complementary validator
	// holds the remaining 71 and is not touched, so it is not an acquisition candidate.
	_, sk, ctx, decorator := setupPowerCapDecorator(t, 100)
	dstVal := sdk.ValAddress(sample.AccAddressBytes())
	otherVal := sdk.ValAddress(sample.AccAddressBytes())
	mockValidator(sk, ctx, validator(dstVal, stakingtypes.Bonded, math.NewInt(29)))
	mockValidator(sk, ctx, validator(otherVal, stakingtypes.Bonded, math.NewInt(71)))
	mockPowerStore(sk, ctx, 2, otherVal, dstVal)
	del := sample.AccAddressBytes()
	sk.On("GetAllDelegatorDelegations", ctx, del).Return([]stakingtypes.Delegation{}, nil)
	mockIterateDelegations(sk, ctx, del, []stakingtypes.Delegation{})

	tx := buildTx(t, &stakingtypes.MsgDelegate{
		DelegatorAddress: del.String(),
		ValidatorAddress: dstVal.String(),
		Amount:           coin(2),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.ErrorIs(t, err, types.ErrExceedsMaxValidatorPowerShare)
}

func TestValidatorPowerShareCancelUnbonding(t *testing.T) {
	// Canceling unbonding to a bonded validator adds active stake. dst 29 + 2
	// = 31 > 30% of 102 = 30.6, under the 5% threshold.
	_, sk, ctx, decorator := setupPowerCapDecorator(t, 100)
	dstVal := sdk.ValAddress(sample.AccAddressBytes())
	otherVal := sdk.ValAddress(sample.AccAddressBytes())
	mockValidator(sk, ctx, validator(dstVal, stakingtypes.Bonded, math.NewInt(29)))
	mockValidator(sk, ctx, validator(otherVal, stakingtypes.Bonded, math.NewInt(71)))
	mockPowerStore(sk, ctx, 2, otherVal, dstVal)
	del := sample.AccAddressBytes()
	mockIterateDelegations(sk, ctx, del, []stakingtypes.Delegation{})

	tx := buildTx(t, &stakingtypes.MsgCancelUnbondingDelegation{
		DelegatorAddress: del.String(),
		ValidatorAddress: dstVal.String(),
		Amount:           coin(2),
		CreationHeight:   1,
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.ErrorIs(t, err, types.ErrExceedsMaxValidatorPowerShare)
}

func TestValidatorPowerShareCreateValidatorBlockedByDelegatorCap(t *testing.T) {
	// For a pure MsgCreateValidator the self-delegator's bonded stake equals
	// the new validator's tokens, and checkDelegatorStakeShares (30%, strict >)
	// runs before checkValidatorPowerShares in finalizeStakeChanges. So any
	// self-delegation that would trip the validator cap trips the delegator cap
	// first; the validator cap is subsumed for single-self-delegation creation.
	// This test documents that ordering: a 50-token entrant into a not-full
	// active set (MaxValidators=2, one bonded validator at 100) is blocked by
	// the delegator cap, not the validator cap.
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: math.NewInt(10_000)}))
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)

	bondedVal := sdk.ValAddress(sample.AccAddressBytes())
	newVal := sdk.ValAddress(sample.AccAddressBytes())
	mockValidator(sk, ctx, validator(bondedVal, stakingtypes.Bonded, math.NewInt(100)))
	mockPowerStore(sk, ctx, 2, bondedVal)
	mockIterateDelegations(sk, ctx, sdk.AccAddress(newVal), []stakingtypes.Delegation{})

	tx := buildTx(t, &stakingtypes.MsgCreateValidator{
		ValidatorAddress:  newVal.String(),
		MinSelfDelegation: math.OneInt(),
		Value:             coin(50),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestValidatorPowerShareOverCapSheddingAllowed(t *testing.T) {
	// An already-over-cap validator (40/100) only undelegates, which is an
	// acquisition-only cap's allowed path: after < before, never rejected.
	_, sk, ctx, decorator := setupPowerCapDecorator(t, 100)

	overVal := sdk.ValAddress(sample.AccAddressBytes())
	mockValidator(sk, ctx, validator(overVal, stakingtypes.Bonded, math.NewInt(40)))
	mockPowerStore(sk, ctx, 1, overVal)
	del := sample.AccAddressBytes()
	mockDelegation(sk, ctx, del, overVal, math.NewInt(40))

	tx := buildTx(t, &stakingtypes.MsgUndelegate{
		DelegatorAddress: del.String(),
		ValidatorAddress: overVal.String(),
		Amount:           coin(1),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.NoError(t, err)
}

func TestValidatorPowerSharePassiveDrift(t *testing.T) {
	// v0 is already over the cap (40/100) but is not touched by this tx; only
	// another validator is undelegated. v0's token balance does not increase, so
	// passive denominator drift must not be rejected by the validator cap.
	_, sk, ctx, decorator := setupPowerCapDecorator(t, 100)

	overVal := sdk.ValAddress(sample.AccAddressBytes())
	otherVal := sdk.ValAddress(sample.AccAddressBytes())
	mockValidator(sk, ctx, validator(overVal, stakingtypes.Bonded, math.NewInt(40)))
	mockValidator(sk, ctx, validator(otherVal, stakingtypes.Bonded, math.NewInt(60)))
	mockPowerStore(sk, ctx, 2, otherVal, overVal)
	del := sample.AccAddressBytes()
	mockDelegation(sk, ctx, del, otherVal, math.NewInt(60))

	tx := buildTx(t, &stakingtypes.MsgUndelegate{
		DelegatorAddress: del.String(),
		ValidatorAddress: otherVal.String(),
		Amount:           coin(1),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.NoError(t, err)
}

func TestValidatorPowerShareActiveSetReplacement(t *testing.T) {
	// Two bonded validators at 100 each (total 200). An unbonded candidate at
	// 96 is split across two delegators (48 each) so the per-delegator 30% cap
	// does not fire first. Undelegating 5 from one bonded validator lets the
	// 96-token candidate enter the active set; 30% of ~196 = 58.8, so 96 is over
	// the validator cap and the entrant (before=0, after=96) is rejected.
	_, sk, ctx, decorator := setupPowerCapDecorator(t, 200)

	leavingVal := sdk.ValAddress(sample.AccAddressBytes())
	steadyVal := sdk.ValAddress(sample.AccAddressBytes())
	candidateVal := sdk.ValAddress(sample.AccAddressBytes())
	mockValidator(sk, ctx, validator(leavingVal, stakingtypes.Bonded, math.NewInt(100)))
	mockValidator(sk, ctx, validator(steadyVal, stakingtypes.Bonded, math.NewInt(100)))
	mockValidator(sk, ctx, validator(candidateVal, stakingtypes.Unbonded, math.NewInt(96)))
	mockPowerStore(sk, ctx, 2, steadyVal, leavingVal, candidateVal)

	leavingDel := sample.AccAddressBytes()
	mockDelegation(sk, ctx, leavingDel, leavingVal, math.NewInt(100))
	sk.On("GetValidatorDelegations", ctx, leavingVal).Return([]stakingtypes.Delegation{delegation(leavingDel, leavingVal, math.NewInt(100))}, nil)

	candDels := []sdk.AccAddress{sample.AccAddressBytes(), sample.AccAddressBytes()}
	candDelegations := []stakingtypes.Delegation{
		delegation(candDels[0], candidateVal, math.NewInt(48)),
		delegation(candDels[1], candidateVal, math.NewInt(48)),
	}
	sk.On("GetValidatorDelegations", ctx, candidateVal).Return(candDelegations, nil)
	for _, d := range candDels {
		mockIterateDelegations(sk, ctx, d, candDelegations)
	}

	tx := buildTx(t, &stakingtypes.MsgUndelegate{
		DelegatorAddress: leavingDel.String(),
		ValidatorAddress: leavingVal.String(),
		Amount:           coin(5),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.ErrorIs(t, err, types.ErrExceedsMaxValidatorPowerShare)
}

func TestValidatorPowerShareDelegateToJailed(t *testing.T) {
	// Delegating to a jailed validator does not unjail it; the active-set
	// projection skips jailed validators, so the jailed validator is never
	// treated as an entrant and the delegate must not be rejected by the cap.
	testCases := []struct {
		name         string
		status       stakingtypes.BondStatus
		tokens       int64
		amount       int64
		mockDelegate bool
	}{
		{
			name:   "unbonded jailed validator stays inactive",
			status: stakingtypes.Unbonded,
			tokens: 50,
			amount: 1,
		},
		{
			name:         "bonded jailed validator stays inactive",
			status:       stakingtypes.Bonded,
			tokens:       29,
			amount:       2,
			mockDelegate: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, sk, ctx, decorator := setupPowerCapDecorator(t, 100)

			bondedVal := sdk.ValAddress(sample.AccAddressBytes())
			jailedVal := sdk.ValAddress(sample.AccAddressBytes())
			mockValidator(sk, ctx, validator(bondedVal, stakingtypes.Bonded, math.NewInt(100)))
			jailed := validator(jailedVal, tc.status, math.NewInt(tc.tokens))
			jailed.Jailed = true
			mockValidator(sk, ctx, jailed)
			mockPowerStore(sk, ctx, 1, bondedVal)
			del := sample.AccAddressBytes()
			sk.On("GetAllDelegatorDelegations", ctx, del).Return([]stakingtypes.Delegation{}, nil)
			if tc.mockDelegate {
				mockIterateDelegations(sk, ctx, del, []stakingtypes.Delegation{})
			}

			tx := buildTx(t, &stakingtypes.MsgDelegate{
				DelegatorAddress: del.String(),
				ValidatorAddress: jailedVal.String(),
				Amount:           coin(tc.amount),
			})

			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			require.NoError(t, err)
		})
	}
}

func TestRedelegateFromJailedBondedSourceDoesNotProjectLeaver(t *testing.T) {
	k, sk, ctx, decorator := setupPowerCapDecorator(t, 100)
	disableValidatorPowerCap(t, k, ctx)

	dstVal := sdk.ValAddress(sample.AccAddressBytes())
	srcVal := sdk.ValAddress(sample.AccAddressBytes())
	mockValidator(sk, ctx, validator(dstVal, stakingtypes.Bonded, math.NewInt(71)))
	jailed := validator(srcVal, stakingtypes.Bonded, math.NewInt(29))
	jailed.Jailed = true
	mockValidator(sk, ctx, jailed)
	mockPowerStore(sk, ctx, 1, dstVal)

	del := sample.AccAddressBytes()
	srcDelegations := []stakingtypes.Delegation{delegation(del, srcVal, math.NewInt(29))}
	sk.On("GetAllDelegatorDelegations", ctx, del).Return(srcDelegations, nil)
	mockDelegation(sk, ctx, del, srcVal, math.NewInt(29))
	sk.On("GetValidatorDelegations", ctx, srcVal).Return(srcDelegations, nil)

	tx := buildTx(t, &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    del.String(),
		ValidatorSrcAddress: srcVal.String(),
		ValidatorDstAddress: dstVal.String(),
		Amount:              coin(1),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.NoError(t, err)
}

func TestValidatorPowerShareUnjail(t *testing.T) {
	// Two bonded validators at 100 each (total 200); a jailed unbonded validator
	// would enter the active set when unjailed (top 2). 30% of (200 + entering -
	// 100 leaving) = 75: a 150-token entrant (150 > 75) is rejected; a 50-token
	// entrant is below the top set and the ante must not reject.
	testCases := []struct {
		name    string
		tokens  int64
		wantErr error
	}{
		{"jailed over cap rejects", 150, types.ErrExceedsMaxValidatorPowerShare},
		{"jailed below top set succeeds", 50, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, sk, ctx, decorator := setupPowerCapDecorator(t, 200)
			bondedA := sdk.ValAddress(sample.AccAddressBytes())
			bondedB := sdk.ValAddress(sample.AccAddressBytes())
			jailedVal := sdk.ValAddress(sample.AccAddressBytes())
			mockValidator(sk, ctx, validator(bondedA, stakingtypes.Bonded, math.NewInt(100)))
			mockValidator(sk, ctx, validator(bondedB, stakingtypes.Bonded, math.NewInt(100)))
			jailed := validator(jailedVal, stakingtypes.Unbonded, math.NewInt(tc.tokens))
			jailed.Jailed = true
			mockValidator(sk, ctx, jailed)
			mockPowerStore(sk, ctx, 2, bondedA, bondedB)
			tx := buildTx(t, &slashingtypes.MsgUnjail{ValidatorAddr: jailedVal.String()})
			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatorPowerShareUnjailBonded(t *testing.T) {
	// SDK jail removes a validator from the power index without changing Status,
	// so a jailed validator can still report IsBonded()==true (the EndBlocker
	// that would move it to Unbonding has not run yet). Such a validator is NOT
	// in the active set, so unjailing it is an acquisition from 0 and must be
	// cap-checked. Before the preTxActive fix, before==after==Tokens and an
	// over-cap re-entry bypassed the cap.
	testCases := []struct {
		name    string
		tokens  int64
		total   int64
		wantErr error
	}{
		{
			// jailed bonded validator with 150 of 350 total bonded re-enters:
			// 150 > 30% of 350 = 105, so reject.
			name:    "jailed bonded over cap rejects on unjail",
			tokens:  150,
			total:   350,
			wantErr: types.ErrExceedsMaxValidatorPowerShare,
		},
		{
			// jailed bonded validator with 50 of 250 total bonded re-enters:
			// 50 < 30% of 250 = 75, so allow.
			name:   "jailed bonded under cap succeeds on unjail",
			tokens: 50,
			total:  250,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, sk, ctx, decorator := setupPowerCapDecorator(t, tc.total)

			bondedA := sdk.ValAddress(sample.AccAddressBytes())
			bondedB := sdk.ValAddress(sample.AccAddressBytes())
			jailedVal := sdk.ValAddress(sample.AccAddressBytes())
			mockValidator(sk, ctx, validator(bondedA, stakingtypes.Bonded, math.NewInt(100)))
			mockValidator(sk, ctx, validator(bondedB, stakingtypes.Bonded, math.NewInt(100)))
			jailed := validator(jailedVal, stakingtypes.Bonded, math.NewInt(tc.tokens))
			jailed.Jailed = true
			mockValidator(sk, ctx, jailed)
			// MaxValidators=3 so the unjailed validator re-enters without
			// displacing another bonded validator; the jailed validator is absent
			// from the power store (SDK jail removes it).
			mockPowerStore(sk, ctx, 3, bondedA, bondedB)

			tx := buildTx(t, &slashingtypes.MsgUnjail{
				ValidatorAddr: jailedVal.String(),
			})

			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatorPowerShareDisabled(t *testing.T) {
	k, sk, ctx, decorator := setupPowerCapDecorator(t, 100)
	disableValidatorPowerCap(t, k, ctx)
	srcVal, dstVal, del := mockRedelegatePair(sk, ctx)

	// 31/100 would be over the cap when enabled; with share >= 1 it is allowed.
	tx := buildTx(t, &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    del.String(),
		ValidatorSrcAddress: srcVal.String(),
		ValidatorDstAddress: dstVal.String(),
		Amount:              coin(6),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.NoError(t, err)
}

func TestValidatorPowerShareAuthz(t *testing.T) {
	// Authz-wrapped redelegate must not bypass the validator cap.
	_, sk, ctx, decorator := setupPowerCapDecorator(t, 100)
	srcVal, dstVal, del := mockRedelegatePair(sk, ctx)

	tx := buildTx(t, &authz.MsgExec{
		Grantee: sample.AccAddressBytes().String(),
		Msgs: []*codectypes.Any{
			mustAny(&stakingtypes.MsgBeginRedelegate{
				DelegatorAddress:    del.String(),
				ValidatorSrcAddress: srcVal.String(),
				ValidatorDstAddress: dstVal.String(),
				Amount:              coin(6),
			}),
		},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.ErrorIs(t, err, types.ErrExceedsMaxValidatorPowerShare)
}

func TestValidatorPowerShareAuthzNestingLimit(t *testing.T) {
	// Authz nesting continues to respect MaxNestedMsgCount even when the inner
	// message is a stake-change message covered by the validator cap.
	_, _, ctx, decorator := setupPowerCapDecorator(t, 0)

	srcVal := sdk.ValAddress(sample.AccAddressBytes())
	dstVal := sdk.ValAddress(sample.AccAddressBytes())
	tx := buildTx(t, nestedExec(MaxNestedMsgCount, &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    sample.AccAddressBytes().String(),
		ValidatorSrcAddress: srcVal.String(),
		ValidatorDstAddress: dstVal.String(),
		Amount:              coin(1),
	}))

	_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
	require.ErrorContains(t, err, "nested message count exceeds the maximum allowed")
}
