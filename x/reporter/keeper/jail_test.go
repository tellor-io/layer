package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
)

func testDisputeHashID(suffix byte) []byte {
	return []byte{'t', 'e', 's', 't', '-', 'd', 'i', 's', 'p', 'u', 't', 'e', '-', suffix}
}

func TestJailReporter(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()
	updatedAt := time.Now().UTC()
	reporter := types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")

	err := k.Reporters.Set(ctx, addr, reporter)
	require.NoError(t, err)

	ctx = ctx.WithBlockTime(updatedAt.Add(time.Second * 10))
	jailedDuration := uint64(100)

	err = k.JailReporter(ctx, addr, jailedDuration, 1, testDisputeHashID('a'))
	require.NoError(t, err)

	ctx = ctx.WithBlockTime(updatedAt.Add(time.Second * 15))
	updatedReporter, err := k.Reporters.Get(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, true, updatedReporter.Jailed)
	require.Equal(t, updatedAt.Add(time.Second*110), updatedReporter.JailedUntil)
}

func TestJailReporterZeroDurationFlagsOnly(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	selectorAddr := sample.AccAddressBytes()
	reportBlock := uint64(5)
	updatedAt := time.Now().UTC()
	ctx = ctx.WithBlockTime(updatedAt)

	reporter := types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")
	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, reporter))
	require.NoError(t, k.Selectors.Set(ctx, selectorAddr, types.NewSelection(reporterAddr, 1)))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporterAddr.Bytes(), reportBlock, []byte("q1")), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: selectorAddr}},
	}))

	require.NoError(t, k.JailReporter(ctx, reporterAddr, 0, reportBlock, testDisputeHashID('a')))

	gotReporter, err := k.Reporters.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.True(t, gotReporter.Jailed)
	require.Equal(t, updatedAt, gotReporter.JailedUntil)

	gotSelector, err := k.Selectors.Get(ctx, selectorAddr)
	require.NoError(t, err)
	// Per-dispute locks persist until as unix seconds; sub-second precision is not preserved.
	require.Equal(t, updatedAt.Truncate(time.Second), gotSelector.DisputeLockedUntil)
	require.True(t, gotSelector.LockedUntilTime.IsZero())

	require.NoError(t, k.UnjailReporter(ctx, reporterAddr, reporterAddr))
	gotReporter, err = k.Reporters.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.False(t, gotReporter.Jailed)
}

func TestUnJailReporter(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()
	jailedAt := time.Now().UTC()
	reporter := types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")
	reporter.Jailed = true
	reporter.JailedUntil = jailedAt.Add(time.Second * 100)
	require.NoError(t, k.Reporters.Set(ctx, addr, reporter))

	ctx = ctx.WithBlockTime(jailedAt.Add(time.Second * 50))
	err := k.UnjailReporter(ctx, addr, addr)
	require.Error(t, err)

	ctx = ctx.WithBlockTime(jailedAt.Add(time.Second * 505))
	err = k.UnjailReporter(ctx, addr, addr)
	require.NoError(t, err)

	updatedReporter, err := k.Reporters.Get(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, false, updatedReporter.Jailed)

	err = k.UnjailReporter(ctx, addr, addr)
	require.Error(t, err)
}

func TestThirdPartyUnjailReporter(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	jailed := sample.AccAddressBytes()
	caller := sample.AccAddressBytes()
	jailedAt := time.Now().UTC()
	reporter := types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")
	reporter.Jailed = true
	reporter.JailedUntil = jailedAt.Add(100 * time.Second)
	require.NoError(t, k.Reporters.Set(ctx, jailed, reporter))

	// Third party cannot unjail before self-eligibility.
	ctx = ctx.WithBlockTime(jailedAt.Add(200 * time.Second))
	err := k.UnjailReporter(ctx, caller, jailed)
	require.ErrorIs(t, err, types.ErrThirdPartyUnjailTooEarly)

	// Third party can unjail 7 days after self-eligibility.
	ctx = ctx.WithBlockTime(jailedAt.Add(100*time.Second + 7*24*time.Hour))
	require.NoError(t, k.UnjailReporter(ctx, caller, jailed))

	got, err := k.Reporters.Get(ctx, jailed)
	require.NoError(t, err)
	require.False(t, got.Jailed)
}

func TestUpdateJailedUntilOnFailedDispute(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	selectorA := sample.AccAddressBytes()
	selectorB := sample.AccAddressBytes()
	reportBlock := uint64(10)
	jailedAt := time.Now().UTC()

	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selectorA, types.NewSelection(reporterAddr, 1)))
	require.NoError(t, k.Selectors.Set(ctx, selectorB, types.NewSelection(reporterAddr, 1)))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporterAddr.Bytes(), reportBlock, []byte("q1")), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: selectorA},
			{DelegatorAddress: selectorB},
		},
	}))

	ctx = ctx.WithBlockTime(jailedAt)
	disputeHash := testDisputeHashID('a')
	require.NoError(t, k.JailReporter(ctx, reporterAddr, 600, reportBlock, disputeHash))

	ctx = ctx.WithBlockTime(jailedAt.Add(time.Second * 50))
	require.NoError(t, k.UpdateJailedUntilOnFailedDispute(ctx, reporterAddr, reportBlock, disputeHash))

	selA, err := k.Selectors.Get(ctx, selectorA)
	require.NoError(t, err)
	require.True(t, selA.DisputeLockedUntil.Before(ctx.BlockTime()))
	require.True(t, selA.LockedUntilTime.IsZero())
	has, err := k.StakeRecalcFlag.Has(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.True(t, has)

	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.Equal(t, jailedAt.Add(time.Second*49), reporter.JailedUntil)
}

func TestFailedDisputePreservesLegacyLockedUntilTime(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	reportBlock := uint64(10)
	now := time.Now().UTC()
	ctx = ctx.WithBlockTime(now)
	legacyLock := now.Add(21 * 24 * time.Hour)

	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.Selection{
		Reporter:        reporterAddr,
		LockedUntilTime: legacyLock,
	}))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporterAddr.Bytes(), reportBlock, []byte("q1")), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: selector}},
	}))

	disputeHash := testDisputeHashID('a')
	require.NoError(t, k.JailReporter(ctx, reporterAddr, 600, reportBlock, disputeHash))
	require.NoError(t, k.UpdateJailedUntilOnFailedDispute(ctx, reporterAddr, reportBlock, disputeHash))

	sel, err := k.Selectors.Get(ctx, selector)
	require.NoError(t, err)
	require.True(t, sel.DisputeLockedUntil.Before(now))
	require.True(t, sel.LockedUntilTime.Equal(legacyLock))
	require.True(t, types.SelectorStakeLocked(sel, now))
}

func TestJailUsesReportByBlockNotReporterIndex(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporterR := sample.AccAddressBytes()
	reporterT := sample.AccAddressBytes()
	selectorA := sample.AccAddressBytes()
	selectorC := sample.AccAddressBytes()
	reportBlock := uint64(5)

	require.NoError(t, k.Reporters.Set(ctx, reporterR, types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "r")))
	require.NoError(t, k.Selectors.Set(ctx, selectorA, types.NewSelection(reporterT, 1)))
	require.NoError(t, k.Selectors.Set(ctx, selectorC, types.NewSelection(reporterR, 1)))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporterR.Bytes(), reportBlock, []byte("q1")), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: selectorA}},
	}))

	require.NoError(t, k.JailReporter(ctx, reporterR, 3600, reportBlock, testDisputeHashID('a')))

	selA, err := k.Selectors.Get(ctx, selectorA)
	require.NoError(t, err)
	require.True(t, selA.DisputeLockedUntil.After(ctx.BlockTime()))
	require.True(t, selA.LockedUntilTime.IsZero())

	selC, err := k.Selectors.Get(ctx, selectorC)
	require.NoError(t, err)
	require.True(t, selC.DisputeLockedUntil.IsZero())
}

func TestJailReporterLocksSnapshotDelegators(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporterR := sample.AccAddressBytes()
	selectorA := sample.AccAddressBytes()
	selectorB := sample.AccAddressBytes()
	reportBlock := uint64(7)

	require.NoError(t, k.Reporters.Set(ctx, reporterR, types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "r")))
	require.NoError(t, k.Selectors.Set(ctx, selectorA, types.NewSelection(reporterR, 1)))
	require.NoError(t, k.Selectors.Set(ctx, selectorB, types.NewSelection(reporterR, 1)))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporterR.Bytes(), reportBlock, []byte("q1")), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: selectorA},
			{DelegatorAddress: selectorB},
		},
	}))

	require.NoError(t, k.JailReporter(ctx, reporterR, 600, reportBlock, testDisputeHashID('a')))

	for _, sel := range [][]byte{selectorA, selectorB} {
		got, err := k.Selectors.Get(ctx, sel)
		require.NoError(t, err)
		require.True(t, got.DisputeLockedUntil.After(ctx.BlockTime()))
		require.True(t, got.LockedUntilTime.IsZero())
	}
}

func TestJailSetsDisputeLockedUntilOnly(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	selector := sample.AccAddressBytes()
	reporter := sample.AccAddressBytes()
	reportBlock := uint64(3)
	legacyLock := ctx.BlockTime().Add(30 * 24 * time.Hour)

	require.NoError(t, k.Selectors.Set(ctx, selector, types.Selection{
		Reporter:        reporter,
		LockedUntilTime: legacyLock,
	}))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporter.Bytes(), reportBlock, []byte("q1")), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: selector}},
	}))

	require.NoError(t, k.JailReporter(ctx, reporter, 3600, reportBlock, testDisputeHashID('a')))

	got, err := k.Selectors.Get(ctx, selector)
	require.NoError(t, err)
	require.True(t, got.DisputeLockedUntil.After(ctx.BlockTime()))
	require.True(t, got.LockedUntilTime.Equal(legacyLock))
}

func TestJailUsesMaxDisputeLockTime(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	selector := sample.AccAddressBytes()
	reporter := sample.AccAddressBytes()
	reportBlock := uint64(3)
	shorter := ctx.BlockTime().Add(30 * time.Minute)
	longer := ctx.BlockTime().Add(2 * time.Hour)
	shortHash := testDisputeHashID('a')
	longHash := testDisputeHashID('b')

	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporter, 1)))
	require.NoError(t, k.SelectorDisputeLocks.Set(ctx, collections.Join(selector.Bytes(), shortHash), shorter.Unix()))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporter.Bytes(), reportBlock, []byte("q1")), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: selector}},
	}))

	require.NoError(t, k.JailReporter(ctx, reporter, 7200, reportBlock, longHash))

	got, err := k.Selectors.Get(ctx, selector)
	require.NoError(t, err)
	require.True(t, got.DisputeLockedUntil.Equal(longer.Truncate(time.Second)))
}

func TestJailDisputeLockSchedulesRecalc(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	selector := sample.AccAddressBytes()
	reporterA := sample.AccAddressBytes()
	reporterB := sample.AccAddressBytes()
	reportBlock := uint64(3)

	require.NoError(t, k.Selectors.Set(ctx, selector, types.Selection{Reporter: reporterA}))
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(reporterA.Bytes(), selector.Bytes()), types.PendingSwitchEntry{
		ToReporter:  reporterB.Bytes(),
		UnlockBlock: uint64(ctx.BlockHeight()) + 100,
	}))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporterA.Bytes(), reportBlock, []byte("q1")), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: selector}},
	}))

	require.NoError(t, k.JailReporter(ctx, reporterA, 3600, reportBlock, testDisputeHashID('a')))

	hasA, err := k.StakeRecalcFlag.Has(ctx, reporterA.Bytes())
	require.NoError(t, err)
	require.True(t, hasA)
	hasB, err := k.StakeRecalcFlag.Has(ctx, reporterB.Bytes())
	require.NoError(t, err)
	require.True(t, hasB)

	recalcAt, err := k.RecalcAtTime.Get(ctx, reporterA.Bytes())
	require.NoError(t, err)
	got, err := k.Selectors.Get(ctx, selector)
	require.NoError(t, err)
	require.Equal(t, got.DisputeLockedUntil.Unix(), recalcAt)
}

func TestLazyClearSelectorLocksIfExpired(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	selector := sample.AccAddressBytes()
	reporter := sample.AccAddressBytes()
	now := time.Now().UTC()
	ctx = ctx.WithBlockTime(now)
	expired := now.Add(-time.Hour)

	require.NoError(t, k.Selectors.Set(ctx, selector, types.Selection{
		Reporter:           reporter,
		DisputeLockedUntil: expired,
		LockedUntilTime:    expired,
	}))

	sel, err := k.GetSelectorForStake(ctx, selector)
	require.NoError(t, err)
	require.Equal(t, expired, sel.DisputeLockedUntil)
	require.Equal(t, expired, sel.LockedUntilTime)
	require.False(t, types.SelectorStakeLocked(sel, ctx.BlockTime()))
	has, err := k.StakeRecalcFlag.Has(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.True(t, has)
}

func TestLazyClearSelectorLocksFlagsRecalcForPendingSwitchTargets(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	selector := sample.AccAddressBytes()
	reporterA := sample.AccAddressBytes()
	reporterB := sample.AccAddressBytes()
	now := time.Now().UTC()
	ctx = ctx.WithBlockTime(now)
	expired := now.Add(-time.Hour)

	require.NoError(t, k.Selectors.Set(ctx, selector, types.Selection{
		Reporter:           reporterA,
		DisputeLockedUntil: expired,
		LockedUntilTime:    expired,
	}))
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(reporterA.Bytes(), selector.Bytes()), types.PendingSwitchEntry{
		ToReporter:  reporterB.Bytes(),
		UnlockBlock: uint64(ctx.BlockHeight()) + 100,
	}))

	_, err := k.GetSelectorForStake(ctx, selector)
	require.NoError(t, err)

	hasA, err := k.StakeRecalcFlag.Has(ctx, reporterA.Bytes())
	require.NoError(t, err)
	require.True(t, hasA)
	hasB, err := k.StakeRecalcFlag.Has(ctx, reporterB.Bytes())
	require.NoError(t, err)
	require.True(t, hasB)
}

func TestLazyClearSelectorLocksSkipsWhileLockedUntilActive(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	selector := sample.AccAddressBytes()
	now := time.Now().UTC()
	ctx = ctx.WithBlockTime(now)

	require.NoError(t, k.Selectors.Set(ctx, selector, types.Selection{
		Reporter:           sample.AccAddressBytes(),
		DisputeLockedUntil: now.Add(-time.Hour),
		LockedUntilTime:    now.Add(time.Hour),
	}))

	sel, err := k.GetSelector(ctx, selector)
	require.NoError(t, err)
	require.True(t, types.SelectorStakeLocked(sel, now))
	has, err := k.StakeRecalcFlag.Has(ctx, sel.Reporter)
	require.NoError(t, err)
	require.False(t, has)
}

func TestUnjailReporterClearsDisputeLockOnly(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()
	until := ctx.BlockTime().Add(time.Hour)
	legacyLock := ctx.BlockTime().Add(21 * 24 * time.Hour)
	disputeHash := testDisputeHashID('a')
	require.NoError(t, k.Reporters.Set(ctx, addr, types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "r")))
	require.NoError(t, k.Selectors.Set(ctx, addr, types.Selection{
		Reporter:           addr,
		DisputeLockedUntil: until,
		LockedUntilTime:    legacyLock,
	}))
	require.NoError(t, k.SelectorDisputeLocks.Set(ctx, collections.Join(addr.Bytes(), disputeHash), until.Unix()))

	ctx = ctx.WithBlockTime(until.Add(time.Second))
	require.NoError(t, k.UnjailReporter(ctx, addr, addr))

	sel, err := k.Selectors.Get(ctx, addr)
	require.NoError(t, err)
	require.True(t, sel.DisputeLockedUntil.IsZero())
	require.True(t, sel.LockedUntilTime.Equal(legacyLock))
	require.True(t, types.SelectorStakeLocked(sel, ctx.BlockTime()))
	_, err = k.SelectorDisputeLocks.Get(ctx, collections.Join(addr.Bytes(), disputeHash))
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestClearOneDisputeLockPreservesOther(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporterAddr := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	reportBlock := uint64(10)
	now := time.Now().UTC()
	ctx = ctx.WithBlockTime(now)
	minorHash := testDisputeHashID('a')
	majorHash := testDisputeHashID('b')

	require.NoError(t, k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1)))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporterAddr.Bytes(), reportBlock, []byte("q1")), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: selector}},
	}))

	require.NoError(t, k.JailReporter(ctx, reporterAddr, 600, reportBlock, minorHash))
	require.NoError(t, k.JailReporter(ctx, reporterAddr, 7200, reportBlock, majorHash))

	sel, err := k.Selectors.Get(ctx, selector)
	require.NoError(t, err)
	majorUntil := now.Add(7200 * time.Second).Truncate(time.Second)
	require.True(t, sel.DisputeLockedUntil.Equal(majorUntil))

	require.NoError(t, k.UpdateJailedUntilOnFailedDispute(ctx, reporterAddr, reportBlock, minorHash))

	sel, err = k.Selectors.Get(ctx, selector)
	require.NoError(t, err)
	require.True(t, sel.DisputeLockedUntil.Equal(majorUntil))
	require.True(t, types.SelectorStakeLocked(sel, now))

	_, err = k.SelectorDisputeLocks.Get(ctx, collections.Join(selector.Bytes(), minorHash))
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = k.SelectorDisputeLocks.Get(ctx, collections.Join(selector.Bytes(), majorHash))
	require.NoError(t, err)

	require.NoError(t, k.UpdateJailedUntilOnFailedDispute(ctx, reporterAddr, reportBlock, majorHash))
	sel, err = k.Selectors.Get(ctx, selector)
	require.NoError(t, err)
	require.True(t, sel.DisputeLockedUntil.IsZero())
	require.False(t, types.SelectorStakeLocked(sel, now))
}
