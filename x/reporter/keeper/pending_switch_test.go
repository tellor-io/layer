package keeper_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const (
	// openSwitchBegan is inside the seeded open row. Canceling later keeps that
	// row, so a height before the cancel still sees the switch as unfinished.
	openSwitchBegan = uint64(40)
	olderSwitchDone = uint64(20)
)

func seedOpenSwitchWithOlderRow(t *testing.T, k keeper.Keeper, ctx sdk.Context, selector, openOriginal, olderOriginal []byte) {
	t.Helper()
	require.NoError(t, k.SelectorReporterSwitches.Set(ctx, selector, types.SelectorReporterSwitches{
		Switches: []types.SelectorReporterSwitch{
			{OriginalReporter: append([]byte(nil), openOriginal...), BeganHeight: openSwitchBegan, CompletedHeight: 0},
			{OriginalReporter: append([]byte(nil), olderOriginal...), BeganHeight: 10, CompletedHeight: olderSwitchDone},
		},
	}))
}

func requireOpenSwitchAborted(t *testing.T, k keeper.Keeper, ctx sdk.Context, selector, olderOriginal sdk.AccAddress) {
	t.Helper()
	history, err := k.SelectorReporterSwitches.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.Len(t, history.Switches, 2)
	require.Equal(t, openSwitchBegan, history.Switches[0].BeganHeight)
	require.Zero(t, history.Switches[0].CompletedHeight)
	require.Equal(t, uint64(ctx.BlockHeight()), history.Switches[0].CancelledHeight)
	require.Equal(t, olderOriginal.Bytes(), history.Switches[1].OriginalReporter)
	require.Equal(t, olderSwitchDone, history.Switches[1].CompletedHeight)
	require.Zero(t, history.Switches[1].CancelledHeight)
	unfinished, err := k.SelectorSwitchUnfinishedAt(ctx, selector, openSwitchBegan+10)
	require.NoError(t, err)
	require.True(t, unfinished)
	unfinished, err = k.SelectorSwitchUnfinishedAt(ctx, selector, uint64(ctx.BlockHeight()))
	require.NoError(t, err)
	require.False(t, unfinished)
}

func mockSelectorBonded(sk *mocks.StakingKeeper, selector sdk.AccAddress) {
	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet).Maybe()
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil).Maybe()
	val := stakingtypes.Validator{
		OperatorAddress: sdk.ValAddress(selector).String(),
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(1_000_000),
		DelegatorShares: math.LegacyNewDec(1_000),
	}
	sk.On("GetValidator", mock.Anything, sdk.ValAddress(selector)).Return(val, nil).Maybe()
	sk.On("IterateDelegatorDelegations", mock.Anything, selector, mock.AnythingOfType("func(types.Delegation) bool")).
		Return(nil).
		Run(func(args mock.Arguments) {
			fn := args.Get(2).(func(stakingtypes.Delegation) bool)
			fn(stakingtypes.Delegation{
				DelegatorAddress: selector.String(),
				ValidatorAddress: sdk.ValAddress(selector).String(),
				Shares:           math.LegacyNewDec(1000),
			})
		}).Maybe()
}

func TestFinalizePendingSwitchCleansOrphanWhenSelectorMissing(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(100)

	outgoing, incoming, selector := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, outgoing.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, incoming.Bytes(), rep))

	outPK := collections.Join(outgoing.Bytes(), selector.Bytes())
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, outPK, types.PendingSwitchEntry{
		ToReporter:  incoming.Bytes(),
		UnlockBlock: 1,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(incoming.Bytes(), selector.Bytes()), outgoing.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, outgoing.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 1,
	}))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, incoming.Bytes(), types.ReporterPendingSwitchHead{
		IncomingCount:     1,
		IncomingMinUnlock: 1,
	}))
	older := sample.AccAddressBytes()
	seedOpenSwitchWithOlderRow(t, k, ctx, selector.Bytes(), outgoing.Bytes(), older.Bytes())

	_, err := k.ReporterStake(ctx, incoming, []byte{})
	require.NoError(t, err)

	hasOut, err := k.OutgoingPendingSwitches.Has(ctx, outPK)
	require.NoError(t, err)
	require.False(t, hasOut)

	hasIn, err := k.IncomingPendingSwitchIdx.Has(ctx, collections.Join(incoming.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasIn)

	_, err = k.ReporterPendingSwitchHeads.Get(ctx, outgoing.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = k.ReporterPendingSwitchHeads.Get(ctx, incoming.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)

	_, err = k.Selectors.Get(ctx, selector.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound, "missing selector must not be created on the abandoned target")
	requireOpenSwitchAborted(t, k, ctx, selector, older)
}

func TestGetReporterStakeFinalizesReadyPendingSwitches(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(100)

	outgoing, incoming, selector := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, outgoing.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, incoming.Bytes(), rep))

	outPK := collections.Join(outgoing.Bytes(), selector.Bytes())
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, outPK, types.PendingSwitchEntry{
		ToReporter:  incoming.Bytes(),
		UnlockBlock: 1,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(incoming.Bytes(), selector.Bytes()), outgoing.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, outgoing.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 1,
	}))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, incoming.Bytes(), types.ReporterPendingSwitchHead{
		IncomingCount:     1,
		IncomingMinUnlock: 1,
	}))

	_, _, _, _, err := k.GetReporterStake(ctx, incoming)
	require.NoError(t, err)

	hasOut, err := k.OutgoingPendingSwitches.Has(ctx, outPK)
	require.NoError(t, err)
	require.False(t, hasOut)

	hasIn, err := k.IncomingPendingSwitchIdx.Has(ctx, collections.Join(incoming.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasIn)
}

func TestGetReporterStakeRejectsPendingSelfDemotion(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(10)

	reporter, target := sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, reporter.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, target.Bytes(), rep))
	require.NoError(t, k.Selectors.Set(ctx, reporter.Bytes(), types.NewSelection(reporter, 1)))

	// Self-demotion pending but not yet unlocked — reporting must fail closed.
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(reporter.Bytes(), reporter.Bytes()), types.PendingSwitchEntry{
		ToReporter:  target.Bytes(),
		UnlockBlock: 100,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(target.Bytes(), reporter.Bytes()), reporter.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, reporter.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 100,
	}))

	_, _, _, _, err := k.GetReporterStake(ctx, reporter)
	require.ErrorIs(t, err, types.ErrReporterSelfDemoting)

	_, err = k.ReporterStake(ctx, reporter, []byte("qid"))
	require.ErrorIs(t, err, types.ErrReporterSelfDemoting)

	// Period data must not be rewritten by the failed stake path.
	_, err = k.ReporterPeriodData.Get(ctx, reporter)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestGetReporterStakeDoesNotFinalizeOwnReadySelfDemotion(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(100)

	reporter, target := sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, reporter.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, target.Bytes(), rep))
	require.NoError(t, k.Selectors.Set(ctx, reporter.Bytes(), types.NewSelection(reporter, 1)))

	// Self-demotion is already past unlock — mutating stake on the demoter must fail
	// fast without applying (which would remove the row then not-found the Get).
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(reporter.Bytes(), reporter.Bytes()), types.PendingSwitchEntry{
		ToReporter:  target.Bytes(),
		UnlockBlock: 1,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(target.Bytes(), reporter.Bytes()), reporter.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, reporter.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 1,
	}))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, target.Bytes(), types.ReporterPendingSwitchHead{
		IncomingCount:     1,
		IncomingMinUnlock: 1,
	}))

	_, _, _, _, err := k.GetReporterStake(ctx, reporter)
	require.ErrorIs(t, err, types.ErrReporterSelfDemoting)

	_, err = k.ReporterStake(ctx, reporter, []byte("qid"))
	require.ErrorIs(t, err, types.ErrReporterSelfDemoting)

	has, err := k.Reporters.Has(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.True(t, has, "demoter row must remain until the target mutates stake")

	hasOut, err := k.OutgoingPendingSwitches.Has(ctx, collections.Join(reporter.Bytes(), reporter.Bytes()))
	require.NoError(t, err)
	require.True(t, hasOut)
}

func TestFinalizePendingSwitchDropsWhenTargetSelfDemoting(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(100)

	source, demoting, dest, selector := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, source.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, demoting.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, dest.Bytes(), rep))
	require.NoError(t, k.Selectors.Set(ctx, demoting.Bytes(), types.NewSelection(demoting, 1)))
	sel := types.NewSelection(source, 1)
	sel.SwitchOutLockedUntilBlock = 50
	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), sel))

	// Selector scheduled source → demoting while demoting was healthy; unlock is ready.
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(source.Bytes(), selector.Bytes()), types.PendingSwitchEntry{
		ToReporter:  demoting.Bytes(),
		UnlockBlock: 50,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(demoting.Bytes(), selector.Bytes()), source.Bytes()))

	// Demoting reporter then scheduled self-demotion (not yet applied).
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(demoting.Bytes(), demoting.Bytes()), types.PendingSwitchEntry{
		ToReporter:  dest.Bytes(),
		UnlockBlock: 200,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(dest.Bytes(), demoting.Bytes()), demoting.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, source.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 50,
	}))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, demoting.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 200,
		IncomingCount:     1,
		IncomingMinUnlock: 50,
	}))
	older := sample.AccAddressBytes()
	seedOpenSwitchWithOlderRow(t, k, ctx, selector.Bytes(), source.Bytes(), older.Bytes())

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := k.ReporterStake(ctx, source, []byte{})
	require.NoError(t, err)

	selAfter, err := k.Selectors.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(selAfter.Reporter, source.Bytes()), "selection must stay on source, not land on demoting reporter")
	require.Equal(t, uint64(0), selAfter.SwitchOutLockedUntilBlock)

	hasOut, err := k.OutgoingPendingSwitches.Has(ctx, collections.Join(source.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasOut)
	hasIn, err := k.IncomingPendingSwitchIdx.Has(ctx, collections.Join(demoting.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasIn)

	hasDemoting, err := k.Reporters.Has(ctx, demoting.Bytes())
	require.NoError(t, err)
	require.True(t, hasDemoting)
	require.False(t, bytes.Equal(selAfter.Reporter, demoting.Bytes()))
	requireOpenSwitchAborted(t, k, ctx, selector, older)
}

func TestFinalizePendingSwitchClearsLockWhenTargetGone(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(100)

	source, selector, missing := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, source.Bytes(), rep))
	sel := types.NewSelection(source, 1)
	sel.SwitchOutLockedUntilBlock = 50
	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), sel))

	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(source.Bytes(), selector.Bytes()), types.PendingSwitchEntry{
		ToReporter:  missing.Bytes(),
		UnlockBlock: 50,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(missing.Bytes(), selector.Bytes()), source.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, source.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 50,
	}))
	older := sample.AccAddressBytes()
	seedOpenSwitchWithOlderRow(t, k, ctx, selector.Bytes(), source.Bytes(), older.Bytes())

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := k.ReporterStake(ctx, source, []byte{})
	require.NoError(t, err)

	selAfter, err := k.Selectors.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(selAfter.Reporter, source.Bytes()))
	require.False(t, bytes.Equal(selAfter.Reporter, missing.Bytes()))
	require.Equal(t, uint64(0), selAfter.SwitchOutLockedUntilBlock)
	requireOpenSwitchAborted(t, k, ctx, selector, older)
}

func TestSelfDemotionCancelsIncomingPendingSwitches(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(100)

	demoting, target, source, selector := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, demoting.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, target.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, source.Bytes(), rep))
	require.NoError(t, k.Selectors.Set(ctx, demoting.Bytes(), types.NewSelection(demoting, 1)))
	sel := types.NewSelection(source, 1)
	sel.SwitchOutLockedUntilBlock = 50
	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), sel))
	require.NoError(t, k.ReporterPeriodData.Set(ctx, demoting, types.PeriodRewardData{
		Total:        math.ZeroInt(),
		RewardAmount: math.LegacyZeroDec(),
		Hash:         []byte("stale"),
	}))

	// Selector is mid-switch from source → demoting (not yet finalized).
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(source.Bytes(), selector.Bytes()), types.PendingSwitchEntry{
		ToReporter:  demoting.Bytes(),
		UnlockBlock: 50,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(demoting.Bytes(), selector.Bytes()), source.Bytes()))

	// Demoting reporter self-demotes to target (ready to finalize).
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(demoting.Bytes(), demoting.Bytes()), types.PendingSwitchEntry{
		ToReporter:  target.Bytes(),
		UnlockBlock: 1,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(target.Bytes(), demoting.Bytes()), demoting.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, demoting.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 1,
		IncomingCount:     1,
		IncomingMinUnlock: 50,
	}))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, target.Bytes(), types.ReporterPendingSwitchHead{
		IncomingCount:     1,
		IncomingMinUnlock: 1,
	}))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, source.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 50,
	}))
	older := sample.AccAddressBytes()
	prior := sample.AccAddressBytes()
	seedOpenSwitchWithOlderRow(t, k, ctx, selector.Bytes(), source.Bytes(), older.Bytes())
	// The demotion itself is open. An older completed row must survive the cancel of incoming switches.
	require.NoError(t, k.SelectorReporterSwitches.Set(ctx, demoting.Bytes(), types.SelectorReporterSwitches{
		Switches: []types.SelectorReporterSwitch{
			{OriginalReporter: demoting.Bytes(), BeganHeight: 30, CompletedHeight: 0},
			{OriginalReporter: prior.Bytes(), BeganHeight: 10, CompletedHeight: olderSwitchDone},
		},
	}))

	agent := sample.AccAddressBytes()
	require.NoError(t, k.TwoBitAgentToReporter.Set(ctx, agent.Bytes(), demoting.Bytes()))
	require.NoError(t, k.TwoBitReporterToAgent.Set(ctx, demoting.Bytes(), agent.Bytes()))
	pendingAgent := sample.AccAddressBytes()
	require.NoError(t, k.SetTwoBitReporter(ctx, demoting, pendingAgent))

	// After finalize, demoting becomes a selector of target — stake iteration needs staking mocks.
	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, demoting, mock.Anything).Return(nil)

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	_, err := k.ReporterStake(ctx, target, []byte{})
	require.NoError(t, err)

	// Incoming join canceled: selector stays on source, lock cleared, pending gone.
	selAfter, err := k.Selectors.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(selAfter.Reporter, source.Bytes()))
	require.Equal(t, uint64(0), selAfter.SwitchOutLockedUntilBlock)

	hasOut, err := k.OutgoingPendingSwitches.Has(ctx, collections.Join(source.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasOut)
	hasIn, err := k.IncomingPendingSwitchIdx.Has(ctx, collections.Join(demoting.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasIn)

	hasDemoting, err := k.Reporters.Has(ctx, demoting.Bytes())
	require.NoError(t, err)
	require.False(t, hasDemoting)

	_, err = k.ReporterPeriodData.Get(ctx, demoting)
	require.ErrorIs(t, err, collections.ErrNotFound)

	hasAgent, err := k.TwoBitAgentToReporter.Has(ctx, agent.Bytes())
	require.NoError(t, err)
	require.False(t, hasAgent)
	hasTwoBit, err := k.TwoBitReporterToAgent.Has(ctx, demoting.Bytes())
	require.NoError(t, err)
	require.False(t, hasTwoBit)
	_, err = k.TwoBitPendingRequests.Get(ctx, demoting.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)

	found := false
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type != "pending_switch_canceled_self_demotion" {
			continue
		}
		attrs := map[string]string{}
		for _, a := range ev.Attributes {
			attrs[a.Key] = a.Value
		}
		require.Equal(t, selector.String(), attrs["selector"])
		require.Equal(t, source.String(), attrs["from_reporter"])
		require.Equal(t, demoting.String(), attrs["canceled_to_reporter"])
		require.Equal(t, "50", attrs["unlock_block"])
		found = true
	}
	require.True(t, found, "expected pending_switch_canceled_self_demotion event")

	require.False(t, bytes.Equal(selAfter.Reporter, demoting.Bytes()))
	requireOpenSwitchAborted(t, k, ctx, selector, older)

	demotingHistory, err := k.SelectorReporterSwitches.Get(ctx, demoting.Bytes())
	require.NoError(t, err)
	require.Len(t, demotingHistory.Switches, 2)
	require.Equal(t, demoting.Bytes(), demotingHistory.Switches[0].OriginalReporter)
	require.Equal(t, uint64(100), demotingHistory.Switches[0].CompletedHeight)
	require.Equal(t, prior.Bytes(), demotingHistory.Switches[1].OriginalReporter)
	require.Equal(t, olderSwitchDone, demotingHistory.Switches[1].CompletedHeight)
}

func TestFinalizePendingSwitchCancelsHistoryWhenSelectionReporterChanged(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(100)

	from, target, elsewhere, selector := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	older := sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, from.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, target.Bytes(), rep))
	// Selection already points somewhere other than the pending switch's from reporter.
	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(elsewhere, 1)))

	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(from.Bytes(), selector.Bytes()), types.PendingSwitchEntry{
		ToReporter:  target.Bytes(),
		UnlockBlock: 1,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(target.Bytes(), selector.Bytes()), from.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, from.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 1,
	}))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, target.Bytes(), types.ReporterPendingSwitchHead{
		IncomingCount:     1,
		IncomingMinUnlock: 1,
	}))
	seedOpenSwitchWithOlderRow(t, k, ctx, selector.Bytes(), from.Bytes(), older.Bytes())

	_, err := k.ReporterStake(ctx, from, []byte{})
	require.NoError(t, err)

	selAfter, err := k.Selectors.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(selAfter.Reporter, elsewhere.Bytes()))
	require.False(t, bytes.Equal(selAfter.Reporter, target.Bytes()))

	hasOut, err := k.OutgoingPendingSwitches.Has(ctx, collections.Join(from.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasOut)
	hasIn, err := k.IncomingPendingSwitchIdx.Has(ctx, collections.Join(target.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasIn)
	requireOpenSwitchAborted(t, k, ctx, selector, older)
}

func TestSwitchReporterScheduleAndFinalizeWritesHistory(t *testing.T) {
	k, sk, _, _, _, ms, ctx := setupMsgServer(t)
	scheduleHeight := int64(20)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(scheduleHeight)

	from, to, selector := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, from.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, to.Bytes(), rep))
	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(from, 1)))
	mockSelectorBonded(sk, selector)

	_, err := ms.SwitchReporter(ctx, &types.MsgSwitchReporter{
		SelectorAddress: selector.String(),
		ReporterAddress: to.String(),
	})
	require.NoError(t, err)

	selPending, err := k.Selectors.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(selPending.Reporter, from.Bytes()))

	finalizeHeight := scheduleHeight + 1
	ctx = ctx.WithBlockHeight(finalizeHeight)
	_, err = k.ReporterStake(ctx, to, []byte{})
	require.NoError(t, err)

	history, err := k.SelectorReporterSwitches.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.Len(t, history.Switches, 1)
	require.Equal(t, from.Bytes(), history.Switches[0].OriginalReporter)
	require.Equal(t, uint64(scheduleHeight), history.Switches[0].BeganHeight)
	require.Equal(t, uint64(finalizeHeight), history.Switches[0].CompletedHeight)

	selFinal, err := k.Selectors.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(selFinal.Reporter, to.Bytes()))

	got, live, err := k.ReporterAtHeight(ctx, selector, uint64(scheduleHeight))
	require.NoError(t, err)
	require.False(t, live)
	require.True(t, bytes.Equal(got, from))

	got, live, err = k.ReporterAtHeight(ctx, selector, uint64(finalizeHeight))
	require.NoError(t, err)
	require.True(t, live)
	require.True(t, bytes.Equal(got, to))
}

func TestFinalizePendingSwitchWithoutHistoryLeavesNoRow(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(100)

	from, to, selector := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, from.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, to.Bytes(), rep))
	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(from, 1)))
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(from.Bytes(), selector.Bytes()), types.PendingSwitchEntry{
		ToReporter:  to.Bytes(),
		UnlockBlock: 1,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(to.Bytes(), selector.Bytes()), from.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, from.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 1,
	}))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, to.Bytes(), types.ReporterPendingSwitchHead{
		IncomingCount:     1,
		IncomingMinUnlock: 1,
	}))
	mockSelectorBonded(sk, selector)

	_, err := k.ReporterStake(ctx, to, []byte{})
	require.NoError(t, err)

	selAfter, err := k.Selectors.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(selAfter.Reporter, to.Bytes()))

	hasOut, err := k.OutgoingPendingSwitches.Has(ctx, collections.Join(from.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasOut)
	hasIn, err := k.IncomingPendingSwitchIdx.Has(ctx, collections.Join(to.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasIn)

	hasHistory, err := k.SelectorReporterSwitches.Has(ctx, selector.Bytes())
	require.NoError(t, err)
	require.False(t, hasHistory)
}

func TestNormalPendingSwitchKeepsTwoBitRegistration(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(100)

	outgoing, incoming, selector := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	outgoingAgent, incomingAgent := sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, outgoing.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, incoming.Bytes(), rep))
	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(outgoing.Bytes(), 1)))
	require.NoError(t, k.SetTwoBitMapping(ctx, outgoing.Bytes(), outgoingAgent.Bytes()))
	require.NoError(t, k.SetTwoBitMapping(ctx, incoming.Bytes(), incomingAgent.Bytes()))

	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(outgoing.Bytes(), selector.Bytes()), types.PendingSwitchEntry{
		ToReporter:  incoming.Bytes(),
		UnlockBlock: 1,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(incoming.Bytes(), selector.Bytes()), outgoing.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, outgoing.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 1,
	}))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, incoming.Bytes(), types.ReporterPendingSwitchHead{
		IncomingCount:     1,
		IncomingMinUnlock: 1,
	}))

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := k.ReporterStake(ctx, incoming, []byte("qid"))
	require.NoError(t, err)

	selAfter, err := k.Selectors.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(incoming.Bytes(), selAfter.Reporter))

	hasOutgoing, err := k.Reporters.Has(ctx, outgoing.Bytes())
	require.NoError(t, err)
	require.True(t, hasOutgoing)
	hasIncoming, err := k.Reporters.Has(ctx, incoming.Bytes())
	require.NoError(t, err)
	require.True(t, hasIncoming)

	got, err := k.TwoBitReporter(ctx, outgoingAgent)
	require.NoError(t, err)
	require.True(t, bytes.Equal(outgoing.Bytes(), got.Bytes()))
	got, err = k.TwoBitReporter(ctx, incomingAgent)
	require.NoError(t, err)
	require.True(t, bytes.Equal(incoming.Bytes(), got.Bytes()))

	stored, err := k.TwoBitReporterToAgent.Get(ctx, outgoing.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(outgoingAgent.Bytes(), stored))
	stored, err = k.TwoBitReporterToAgent.Get(ctx, incoming.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(incomingAgent.Bytes(), stored))
}
