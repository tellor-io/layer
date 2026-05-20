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

func TestJailReporter(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()
	updatedAt := time.Now().UTC()
	reporter := types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")

	err := k.Reporters.Set(ctx, addr, reporter)
	require.NoError(t, err)

	ctx = ctx.WithBlockTime(updatedAt.Add(time.Second * 10))
	jailedDuration := uint64(100)

	err = k.JailReporter(ctx, addr, jailedDuration, 1)
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

	require.NoError(t, k.JailReporter(ctx, reporterAddr, 0, reportBlock))

	gotReporter, err := k.Reporters.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.True(t, gotReporter.Jailed)
	require.Equal(t, updatedAt, gotReporter.JailedUntil)

	gotSelector, err := k.Selectors.Get(ctx, selectorAddr)
	require.NoError(t, err)
	require.True(t, gotSelector.Jailed)
	require.Equal(t, updatedAt, gotSelector.JailedUntil)
	require.Equal(t, updatedAt, gotSelector.LockedUntilTime)

	require.NoError(t, k.UnjailReporter(ctx, reporterAddr))
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
	err := k.UnjailReporter(ctx, addr)
	require.Error(t, err)

	ctx = ctx.WithBlockTime(jailedAt.Add(time.Second * 505))
	err = k.UnjailReporter(ctx, addr)
	require.NoError(t, err)

	updatedReporter, err := k.Reporters.Get(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, false, updatedReporter.Jailed)

	err = k.UnjailReporter(ctx, addr)
	require.Error(t, err)
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
	require.NoError(t, k.JailReporter(ctx, reporterAddr, 600, reportBlock))

	ctx = ctx.WithBlockTime(jailedAt.Add(time.Second * 50))
	require.NoError(t, k.UpdateJailedUntilOnFailedDispute(ctx, reporterAddr, reportBlock))

	selA, err := k.Selectors.Get(ctx, selectorA)
	require.NoError(t, err)
	require.False(t, selA.Jailed)
	require.True(t, selA.LockedUntilTime.Before(ctx.BlockTime()))
	has, err := k.StakeRecalcFlag.Has(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.True(t, has)

	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.Equal(t, jailedAt.Add(time.Second*49), reporter.JailedUntil)
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

	require.NoError(t, k.JailReporter(ctx, reporterR, 3600, reportBlock))

	selA, err := k.Selectors.Get(ctx, selectorA)
	require.NoError(t, err)
	require.True(t, selA.Jailed)
	require.True(t, selA.LockedUntilTime.After(ctx.BlockTime()))

	selC, err := k.Selectors.Get(ctx, selectorC)
	require.NoError(t, err)
	require.False(t, selC.Jailed)
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

	require.NoError(t, k.JailReporter(ctx, reporterR, 600, reportBlock))

	for _, sel := range [][]byte{selectorA, selectorB} {
		got, err := k.Selectors.Get(ctx, sel)
		require.NoError(t, err)
		require.True(t, got.Jailed)
		require.True(t, got.LockedUntilTime.After(ctx.BlockTime()))
	}
}

func TestJailUsesMaxLockTime(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	selector := sample.AccAddressBytes()
	reporter := sample.AccAddressBytes()
	reportBlock := uint64(3)
	shorter := ctx.BlockTime().Add(30 * time.Minute)
	longer := ctx.BlockTime().Add(2 * time.Hour)

	require.NoError(t, k.Selectors.Set(ctx, selector, types.Selection{
		Reporter:        reporter,
		LockedUntilTime: shorter,
		Jailed:          true,
		JailedUntil:     shorter,
	}))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporter.Bytes(), reportBlock, []byte("q1")), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: selector}},
	}))

	require.NoError(t, k.JailReporter(ctx, reporter, 7200, reportBlock))

	got, err := k.Selectors.Get(ctx, selector)
	require.NoError(t, err)
	require.True(t, got.LockedUntilTime.Equal(longer))
	require.True(t, got.JailedUntil.Equal(longer))
}

func TestLazyUnjailSelectorIfExpired(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	selector := sample.AccAddressBytes()
	reporter := sample.AccAddressBytes()
	now := time.Now().UTC()
	ctx = ctx.WithBlockTime(now)
	expired := now.Add(-time.Hour)

	require.NoError(t, k.Selectors.Set(ctx, selector, types.Selection{
		Reporter:        reporter,
		Jailed:          true,
		JailedUntil:     expired,
		LockedUntilTime: expired,
	}))

	sel, err := k.GetSelectorForStake(ctx, selector)
	require.NoError(t, err)
	require.False(t, sel.Jailed)
	require.Equal(t, expired, sel.JailedUntil)
	require.Equal(t, expired, sel.LockedUntilTime)
	require.False(t, types.SelectorStakeLocked(sel, ctx.BlockTime()))
	has, err := k.StakeRecalcFlag.Has(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.True(t, has)
}

func TestLazyUnjailSelectorFlagsRecalcForPendingSwitchTargets(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	selector := sample.AccAddressBytes()
	reporterA := sample.AccAddressBytes()
	reporterB := sample.AccAddressBytes()
	now := time.Now().UTC()
	ctx = ctx.WithBlockTime(now)
	expired := now.Add(-time.Hour)

	require.NoError(t, k.Selectors.Set(ctx, selector, types.Selection{
		Reporter:        reporterA,
		Jailed:          true,
		JailedUntil:     expired,
		LockedUntilTime: expired,
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

func TestLazyUnjailSelectorSkipsWhileLockedUntilActive(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	selector := sample.AccAddressBytes()
	now := time.Now().UTC()
	ctx = ctx.WithBlockTime(now)

	require.NoError(t, k.Selectors.Set(ctx, selector, types.Selection{
		Reporter:        sample.AccAddressBytes(),
		Jailed:          true,
		JailedUntil:     now.Add(-time.Hour),
		LockedUntilTime: now.Add(time.Hour),
	}))

	sel, err := k.GetSelector(ctx, selector)
	require.NoError(t, err)
	require.True(t, sel.Jailed)
	require.True(t, types.SelectorStakeLocked(sel, now))
}

func TestUnjailReporterClearsSelection(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()
	until := ctx.BlockTime().Add(time.Hour)
	require.NoError(t, k.Reporters.Set(ctx, addr, types.NewReporter(types.DefaultMinCommissionRate, math.OneInt(), "r")))
	require.NoError(t, k.Selectors.Set(ctx, addr, types.Selection{
		Reporter:        addr,
		Jailed:          true,
		JailedUntil:     until,
		LockedUntilTime: until,
	}))

	ctx = ctx.WithBlockTime(until.Add(time.Second))
	require.NoError(t, k.UnjailReporter(ctx, addr))

	sel, err := k.Selectors.Get(ctx, addr)
	require.NoError(t, err)
	require.False(t, sel.Jailed)
	require.True(t, sel.LockedUntilTime.Before(ctx.BlockTime()))
	has, err := k.StakeRecalcFlag.Has(ctx, addr.Bytes())
	require.NoError(t, err)
	require.True(t, has)
}
