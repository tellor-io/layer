package keeper_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

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

	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := k.ReporterStake(ctx, source, []byte{})
	require.NoError(t, err)

	selAfter, err := k.Selectors.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(selAfter.Reporter, source.Bytes()))
	require.Equal(t, uint64(0), selAfter.SwitchOutLockedUntilBlock)
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
}
