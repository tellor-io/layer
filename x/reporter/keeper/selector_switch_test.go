package keeper

import (
	"bytes"
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func TestSelectorSwitchHistoryNewestFirstAndReporterAtHeight(t *testing.T) {
	k, ctx := newSwitchTestKeeper(t)
	selector := sample.AccAddressBytes()
	r0 := sample.AccAddressBytes()
	r1 := sample.AccAddressBytes()
	r2 := sample.AccAddressBytes()

	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(r0, 1)))

	ctx100 := ctx.WithBlockHeight(100)
	require.NoError(t, k.recordSelectorSwitchBegan(ctx100, selector.Bytes(), r0.Bytes()))
	ctx200 := ctx.WithBlockHeight(200)
	require.NoError(t, k.recordSelectorSwitchCompleted(ctx200, selector.Bytes(), r0.Bytes()))
	require.NoError(t, k.Selectors.Set(ctx200, selector.Bytes(), types.NewSelection(r1, 1)))

	ctx300 := ctx.WithBlockHeight(300)
	require.NoError(t, k.recordSelectorSwitchBegan(ctx300, selector.Bytes(), r1.Bytes()))
	ctx400 := ctx.WithBlockHeight(400)
	require.NoError(t, k.recordSelectorSwitchCompleted(ctx400, selector.Bytes(), r1.Bytes()))
	require.NoError(t, k.Selectors.Set(ctx400, selector.Bytes(), types.NewSelection(r2, 1)))

	switches, err := k.SelectorSwitchesNewestFirst(ctx400, selector)
	require.NoError(t, err)
	require.Len(t, switches, 2)
	require.Equal(t, r1.Bytes(), []byte(switches[0].OriginalReporter))
	require.Equal(t, uint64(300), switches[0].BeganHeight)
	require.Equal(t, uint64(400), switches[0].CompletedHeight)
	require.Equal(t, r0.Bytes(), []byte(switches[1].OriginalReporter))

	latest, ok, err := k.LatestSelectorSwitch(ctx400, selector)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, switches[0], latest)

	// Newest completion is at or before the height: live reporter is accurate.
	got, live, err := k.ReporterAtHeight(ctx400, selector, 400)
	require.NoError(t, err)
	require.True(t, live)
	require.True(t, bytes.Equal(got, r2))

	// Between the two switches they were on r1.
	got, live, err = k.ReporterAtHeight(ctx400, selector, 250)
	require.NoError(t, err)
	require.False(t, live)
	require.True(t, bytes.Equal(got, r1))

	// Before any completion they were on the original reporter.
	got, live, err = k.ReporterAtHeight(ctx400, selector, 50)
	require.NoError(t, err)
	require.False(t, live)
	require.True(t, bytes.Equal(got, r0))

	// An open switch has not moved selection, so a height during that switch
	// still resolves to its original reporter.
	ctx500 := ctx.WithBlockHeight(500)
	require.NoError(t, k.recordSelectorSwitchBegan(ctx500, selector.Bytes(), r2.Bytes()))
	got, live, err = k.ReporterAtHeight(ctx500, selector, 450)
	require.NoError(t, err)
	require.True(t, live)
	require.True(t, bytes.Equal(got, r2))
}

func TestReporterSelfDemotedUsesCompletedSelfSwitch(t *testing.T) {
	k, ctx := newSwitchTestKeeper(t)
	reporter := sample.AccAddressBytes()
	next := sample.AccAddressBytes()

	require.NoError(t, k.Selectors.Set(ctx, reporter.Bytes(), types.NewSelection(reporter, 1)))
	demoted, err := k.ReporterSelfDemoted(ctx, reporter)
	require.NoError(t, err)
	require.False(t, demoted)

	ctx = ctx.WithBlockHeight(10)
	require.NoError(t, k.recordSelectorSwitchBegan(ctx, reporter.Bytes(), reporter.Bytes()))
	// Still selected to themselves while the demotion is open.
	demoted, err = k.ReporterSelfDemoted(ctx, reporter)
	require.NoError(t, err)
	require.False(t, demoted)

	ctx = ctx.WithBlockHeight(20)
	require.NoError(t, k.recordSelectorSwitchCompleted(ctx, reporter.Bytes(), reporter.Bytes()))
	require.NoError(t, k.Selectors.Set(ctx, reporter.Bytes(), types.NewSelection(next, 1)))
	demoted, err = k.ReporterSelfDemoted(ctx, reporter)
	require.NoError(t, err)
	require.True(t, demoted)

	// Becoming a reporter again clears the demoted state even though the history remains.
	require.NoError(t, k.Selectors.Set(ctx, reporter.Bytes(), types.NewSelection(reporter, 1)))
	demoted, err = k.ReporterSelfDemoted(ctx, reporter)
	require.NoError(t, err)
	require.False(t, demoted)
}

func TestReporterSelfDemotedWithNoHistory(t *testing.T) {
	k, ctx := newSwitchTestKeeper(t)
	reporter := sample.AccAddressBytes()
	next := sample.AccAddressBytes()

	// Pointed elsewhere and no switch row: the demotion finished outside the
	// history window, either before the index existed or after prune deleted it.
	require.NoError(t, k.Selectors.Set(ctx, reporter.Bytes(), types.NewSelection(next, 1)))
	demoted, err := k.ReporterSelfDemoted(ctx, reporter)
	require.NoError(t, err)
	require.True(t, demoted)
}

func TestReporterSelfDemotedStaysTrueAfterHistoryPruned(t *testing.T) {
	k, ctx := newSwitchTestKeeper(t)
	reporter := sample.AccAddressBytes()
	next := sample.AccAddressBytes()

	require.NoError(t, k.Selectors.Set(ctx, reporter.Bytes(), types.NewSelection(next, 1)))
	require.NoError(t, k.SelectorReporterSwitches.Set(ctx, reporter.Bytes(), types.SelectorReporterSwitches{
		Switches: []types.SelectorReporterSwitch{{
			OriginalReporter: reporter.Bytes(),
			BeganHeight:      10,
			CompletedHeight:  20,
		}},
	}))

	require.NoError(t, k.PruneSelectorSwitches(ctx, 20, 10))

	switches, err := k.SelectorSwitchesNewestFirst(ctx, reporter)
	require.NoError(t, err)
	require.Empty(t, switches)
	demoted, err := k.ReporterSelfDemoted(ctx, reporter)
	require.NoError(t, err)
	require.True(t, demoted)
}

func TestReporterSelfDemotedWhenLaterSwitchOutlivesSelfSwitch(t *testing.T) {
	k, ctx := newSwitchTestKeeper(t)
	reporter := sample.AccAddressBytes()
	next := sample.AccAddressBytes()
	later := sample.AccAddressBytes()

	require.NoError(t, k.Selectors.Set(ctx, reporter.Bytes(), types.NewSelection(later, 1)))
	require.NoError(t, k.SelectorReporterSwitches.Set(ctx, reporter.Bytes(), types.SelectorReporterSwitches{
		Switches: []types.SelectorReporterSwitch{
			{OriginalReporter: next.Bytes(), BeganHeight: 250, CompletedHeight: 300},
			{OriginalReporter: reporter.Bytes(), BeganHeight: 50, CompletedHeight: 100},
		},
	}))

	require.NoError(t, k.PruneSelectorSwitches(ctx, 200, 10))

	switches, err := k.SelectorSwitchesNewestFirst(ctx, reporter)
	require.NoError(t, err)
	require.Len(t, switches, 1)
	require.Equal(t, next.Bytes(), []byte(switches[0].OriginalReporter))

	demoted, err := k.ReporterSelfDemoted(ctx, reporter)
	require.NoError(t, err)
	require.True(t, demoted)
}

func TestSelectorSwitchUnfinishedAt(t *testing.T) {
	selector := sample.AccAddressBytes()
	original := sample.AccAddressBytes()
	next := sample.AccAddressBytes()

	t.Run("no history", func(t *testing.T) {
		k, ctx := newSwitchTestKeeper(t)
		require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(original, 1)))
		unfinished, err := k.SelectorSwitchUnfinishedAt(ctx, selector, 100)
		require.NoError(t, err)
		require.False(t, unfinished)
	})

	t.Run("open switch including self-demotion", func(t *testing.T) {
		k, ctx := newSwitchTestKeeper(t)
		require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(selector, 1)))
		require.NoError(t, k.SelectorReporterSwitches.Set(ctx, selector.Bytes(), types.SelectorReporterSwitches{
			Switches: []types.SelectorReporterSwitch{{
				OriginalReporter: selector.Bytes(),
				BeganHeight:      100,
				CompletedHeight:  0,
			}},
		}))

		got, _, err := k.ReporterAtHeight(ctx, selector, 150)
		require.NoError(t, err)
		require.True(t, bytes.Equal(got, selector), "an open self-demotion still resolves to self")

		unfinished, err := k.SelectorSwitchUnfinishedAt(ctx, selector, 150)
		require.NoError(t, err)
		require.True(t, unfinished)
		unfinished, err = k.SelectorSwitchUnfinishedAt(ctx, selector, 100)
		require.NoError(t, err)
		require.True(t, unfinished)
		unfinished, err = k.SelectorSwitchUnfinishedAt(ctx, selector, 99)
		require.NoError(t, err)
		require.False(t, unfinished)
	})

	t.Run("completed after the height and already begun", func(t *testing.T) {
		k, ctx := newSwitchTestKeeper(t)
		require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(next, 1)))
		require.NoError(t, k.SelectorReporterSwitches.Set(ctx, selector.Bytes(), types.SelectorReporterSwitches{
			Switches: []types.SelectorReporterSwitch{{
				OriginalReporter: original.Bytes(),
				BeganHeight:      100,
				CompletedHeight:  200,
			}},
		}))

		unfinished, err := k.SelectorSwitchUnfinishedAt(ctx, selector, 150)
		require.NoError(t, err)
		require.True(t, unfinished)
		unfinished, err = k.SelectorSwitchUnfinishedAt(ctx, selector, 200)
		require.NoError(t, err)
		require.False(t, unfinished)
		unfinished, err = k.SelectorSwitchUnfinishedAt(ctx, selector, 201)
		require.NoError(t, err)
		require.False(t, unfinished)
	})

	t.Run("newer switch began after the height", func(t *testing.T) {
		k, ctx := newSwitchTestKeeper(t)
		require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(next, 1)))
		require.NoError(t, k.SelectorReporterSwitches.Set(ctx, selector.Bytes(), types.SelectorReporterSwitches{
			Switches: []types.SelectorReporterSwitch{
				{OriginalReporter: original.Bytes(), BeganHeight: 300, CompletedHeight: 400},
				{OriginalReporter: next.Bytes(), BeganHeight: 100, CompletedHeight: 200},
			},
		}))

		// At 250 the later switch had not begun. The earlier one had finished.
		unfinished, err := k.SelectorSwitchUnfinishedAt(ctx, selector, 250)
		require.NoError(t, err)
		require.False(t, unfinished)

		got, live, err := k.ReporterAtHeight(ctx, selector, 250)
		require.NoError(t, err)
		require.False(t, live)
		require.True(t, bytes.Equal(got, original))
	})
}

func TestReporterAtHeightRoundTripPreservesOpenHeight(t *testing.T) {
	k, ctx := newSwitchTestKeeper(t)
	reporter := sample.AccAddressBytes()
	other := sample.AccAddressBytes()
	require.NoError(t, k.Selectors.Set(ctx, reporter.Bytes(), types.NewSelection(reporter, 1)))
	// Left self at 200-250, then selected self again at 300-400.
	require.NoError(t, k.SelectorReporterSwitches.Set(ctx, reporter.Bytes(), types.SelectorReporterSwitches{
		Switches: []types.SelectorReporterSwitch{
			{OriginalReporter: other.Bytes(), BeganHeight: 300, CompletedHeight: 400},
			{OriginalReporter: reporter.Bytes(), BeganHeight: 200, CompletedHeight: 250},
		},
	}))

	got, live, err := k.ReporterAtHeight(ctx, reporter, 100)
	require.NoError(t, err)
	require.True(t, bytes.Equal(got, reporter))
	require.True(t, live)
	unfinished, err := k.SelectorSwitchUnfinishedAt(ctx, reporter, 100)
	require.NoError(t, err)
	require.False(t, unfinished, "idle self-selection at the open height stays idle after a later round trip")

	got, _, err = k.ReporterAtHeight(ctx, reporter, 220)
	require.NoError(t, err)
	require.True(t, bytes.Equal(got, reporter))
	unfinished, err = k.SelectorSwitchUnfinishedAt(ctx, reporter, 220)
	require.NoError(t, err)
	require.True(t, unfinished, "the demotion was unfinished at 220 even though selection was still self")

	got, live, err = k.ReporterAtHeight(ctx, reporter, 260)
	require.NoError(t, err)
	require.True(t, bytes.Equal(got, other))
	require.False(t, live)
	unfinished, err = k.SelectorSwitchUnfinishedAt(ctx, reporter, 260)
	require.NoError(t, err)
	require.False(t, unfinished)
}

func TestCancelOpenSelectorSwitchDropsOnlyTheOpenRow(t *testing.T) {
	k, ctx := newSwitchTestKeeper(t)
	selector := sample.AccAddressBytes()
	r0 := sample.AccAddressBytes()
	r1 := sample.AccAddressBytes()

	ctx = ctx.WithBlockHeight(5)
	require.NoError(t, k.recordSelectorSwitchBegan(ctx, selector.Bytes(), r0.Bytes()))
	ctx = ctx.WithBlockHeight(6)
	require.NoError(t, k.recordSelectorSwitchCompleted(ctx, selector.Bytes(), r0.Bytes()))
	ctx = ctx.WithBlockHeight(7)
	require.NoError(t, k.recordSelectorSwitchBegan(ctx, selector.Bytes(), r1.Bytes()))

	const cancelHeight int64 = 9
	ctx = ctx.WithBlockHeight(cancelHeight)
	require.NoError(t, k.cancelOpenSelectorSwitch(ctx, selector.Bytes(), r1.Bytes()))
	switches, err := k.SelectorSwitchesNewestFirst(ctx, selector)
	require.NoError(t, err)
	require.Len(t, switches, 2)
	require.Equal(t, r1.Bytes(), []byte(switches[0].OriginalReporter))
	require.Equal(t, uint64(7), switches[0].BeganHeight)
	require.Zero(t, switches[0].CompletedHeight)
	require.Equal(t, uint64(cancelHeight), switches[0].CancelledHeight)
	require.Equal(t, r0.Bytes(), []byte(switches[1].OriginalReporter))
	require.Equal(t, uint64(5), switches[1].BeganHeight)
	require.Equal(t, uint64(6), switches[1].CompletedHeight)
	require.Zero(t, switches[1].CancelledHeight)

	unfinished, err := k.SelectorSwitchUnfinishedAt(ctx, selector, uint64(cancelHeight-1))
	require.NoError(t, err)
	require.True(t, unfinished)
	unfinished, err = k.SelectorSwitchUnfinishedAt(ctx, selector, uint64(cancelHeight))
	require.NoError(t, err)
	require.False(t, unfinished)

	require.NoError(t, k.recordSelectorSwitchCompleted(ctx, selector.Bytes(), r1.Bytes()))
	switches, err = k.SelectorSwitchesNewestFirst(ctx, selector)
	require.NoError(t, err)
	require.Zero(t, switches[0].CompletedHeight)
	require.Equal(t, uint64(cancelHeight), switches[0].CancelledHeight)

	ctx = ctx.WithBlockHeight(12)
	require.NoError(t, k.recordSelectorSwitchBegan(ctx, selector.Bytes(), r1.Bytes()))
	switches, err = k.SelectorSwitchesNewestFirst(ctx, selector)
	require.NoError(t, err)
	require.Len(t, switches, 3)
	require.Equal(t, r1.Bytes(), []byte(switches[0].OriginalReporter))
	require.Equal(t, uint64(12), switches[0].BeganHeight)
	require.Zero(t, switches[0].CompletedHeight)
	require.Zero(t, switches[0].CancelledHeight)
	require.Equal(t, uint64(cancelHeight), switches[1].CancelledHeight)
	require.Zero(t, switches[1].CompletedHeight)
	require.Equal(t, uint64(6), switches[2].CompletedHeight)

	require.NoError(t, k.PruneSelectorSwitches(ctx, uint64(cancelHeight-1), 10))
	switches, err = k.SelectorSwitchesNewestFirst(ctx, selector)
	require.NoError(t, err)
	require.Len(t, switches, 2)
	require.Zero(t, switches[0].CancelledHeight)
	require.Equal(t, uint64(12), switches[0].BeganHeight)
	require.Equal(t, uint64(cancelHeight), switches[1].CancelledHeight)

	require.NoError(t, k.PruneSelectorSwitches(ctx, uint64(cancelHeight), 10))
	switches, err = k.SelectorSwitchesNewestFirst(ctx, selector)
	require.NoError(t, err)
	require.Len(t, switches, 1)
	require.Equal(t, uint64(12), switches[0].BeganHeight)
	require.Zero(t, switches[0].CompletedHeight)
	require.Zero(t, switches[0].CancelledHeight)
}

func TestFinalizePendingSwitchCompletesSelectorHistory(t *testing.T) {
	k, ctx := newSwitchTestKeeper(t)
	ctx = ctx.WithBlockHeight(40)
	from := sample.AccAddressBytes()
	to := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, from.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, to.Bytes(), rep))
	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(from, 1)))
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(from.Bytes(), selector.Bytes()), types.PendingSwitchEntry{
		ToReporter:  to.Bytes(),
		UnlockBlock: 1,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(to.Bytes(), selector.Bytes()), from.Bytes()))
	require.NoError(t, k.recordSelectorSwitchBegan(ctx.WithBlockHeight(15), selector.Bytes(), from.Bytes()))

	require.NoError(t, k.finalizePendingSwitch(ctx, from.Bytes(), selector.Bytes()))

	latest, ok, err := k.LatestSelectorSwitch(ctx, selector)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, from.Bytes(), []byte(latest.OriginalReporter))
	require.Equal(t, uint64(15), latest.BeganHeight)
	require.Equal(t, uint64(40), latest.CompletedHeight)

	sel, err := k.Selectors.Get(ctx, selector.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(sel.Reporter, to.Bytes()))

	got, live, err := k.ReporterAtHeight(ctx, selector, 20)
	require.NoError(t, err)
	require.False(t, live)
	require.True(t, bytes.Equal(got, from))
}

func TestPruneSelectorSwitchesKeepsRowsVotingStillNeeds(t *testing.T) {
	k, ctx := newSwitchTestKeeper(t)
	selector := sample.AccAddressBytes()
	r0 := sample.AccAddressBytes()
	r1 := sample.AccAddressBytes()
	r2 := sample.AccAddressBytes()
	r3 := sample.AccAddressBytes()

	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(r3, 1)))
	require.NoError(t, k.SelectorReporterSwitches.Set(ctx, selector.Bytes(), types.SelectorReporterSwitches{
		Switches: []types.SelectorReporterSwitch{
			{OriginalReporter: r3.Bytes(), BeganHeight: 350, CompletedHeight: 0},
			{OriginalReporter: r2.Bytes(), BeganHeight: 250, CompletedHeight: 300},
			{OriginalReporter: r1.Bytes(), BeganHeight: 150, CompletedHeight: 200},
			{OriginalReporter: r0.Bytes(), BeganHeight: 50, CompletedHeight: 100},
		},
	}))

	require.NoError(t, k.PruneSelectorSwitches(ctx, 200, 10))

	switches, err := k.SelectorSwitchesNewestFirst(ctx, selector)
	require.NoError(t, err)
	require.Len(t, switches, 2)
	require.Zero(t, switches[0].CompletedHeight)
	require.Equal(t, uint64(300), switches[1].CompletedHeight)

	// Height 200 still resolves through the switch that completed at 300.
	got, live, err := k.ReporterAtHeight(ctx, selector, 200)
	require.NoError(t, err)
	require.False(t, live)
	require.Equal(t, []byte(r2), []byte(got))

	got, live, err = k.ReporterAtHeight(ctx, selector, 300)
	require.NoError(t, err)
	require.True(t, live)
	require.Equal(t, []byte(r3), []byte(got))

	require.NoError(t, k.PruneSelectorSwitches(ctx, 400, 10))
	switches, err = k.SelectorSwitchesNewestFirst(ctx, selector)
	require.NoError(t, err)
	require.Len(t, switches, 1)
	require.Zero(t, switches[0].CompletedHeight)
}

func TestPruneSelectorSwitchesResumesAndWraps(t *testing.T) {
	k, ctx := newSwitchTestKeeper(t)
	for _, last := range []byte{1, 2, 3} {
		selector := testSelector(last)
		require.NoError(t, k.SelectorReporterSwitches.Set(ctx, selector, types.SelectorReporterSwitches{
			Switches: []types.SelectorReporterSwitch{{
				OriginalReporter: selector,
				BeganHeight:      1,
				CompletedHeight:  10,
			}},
		}))
	}

	require.NoError(t, k.PruneSelectorSwitches(ctx, 10, 2))
	remaining := 0
	err := k.SelectorReporterSwitches.Walk(ctx, nil, func(_ []byte, _ types.SelectorReporterSwitches) (bool, error) {
		remaining++
		return false, nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, remaining)
	cursor, err := k.SelectorSwitchPruneCursor.Get(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, cursor)

	require.NoError(t, k.PruneSelectorSwitches(ctx, 10, 2))
	err = k.SelectorReporterSwitches.Walk(ctx, nil, func(_ []byte, _ types.SelectorReporterSwitches) (bool, error) {
		t.Fatal("completed history should be gone")
		return true, nil
	})
	require.NoError(t, err)
	hasCursor, err := k.SelectorSwitchPruneCursor.Has(ctx)
	require.NoError(t, err)
	require.False(t, hasCursor)

	// An unchanged open row still consumes a visit. The cursor must move past it
	// or the next batch retries the open selector and skips the one after it.
	first := testSelector(1)
	open := testSelector(2)
	last := testSelector(3)
	for _, selector := range [][]byte{first, last} {
		require.NoError(t, k.SelectorReporterSwitches.Set(ctx, selector, types.SelectorReporterSwitches{
			Switches: []types.SelectorReporterSwitch{{
				OriginalReporter: selector,
				BeganHeight:      1,
				CompletedHeight:  10,
			}},
		}))
	}
	require.NoError(t, k.SelectorReporterSwitches.Set(ctx, open, types.SelectorReporterSwitches{
		Switches: []types.SelectorReporterSwitch{{
			OriginalReporter: open,
			BeganHeight:      1,
			CompletedHeight:  0,
		}},
	}))

	require.NoError(t, k.PruneSelectorSwitches(ctx, 10, 2))
	hasFirst, err := k.SelectorReporterSwitches.Has(ctx, first)
	require.NoError(t, err)
	require.False(t, hasFirst)
	openHistory, err := k.SelectorReporterSwitches.Get(ctx, open)
	require.NoError(t, err)
	require.Len(t, openHistory.Switches, 1)
	require.Zero(t, openHistory.Switches[0].CompletedHeight)
	cursor, err = k.SelectorSwitchPruneCursor.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, open, []byte(cursor))

	require.NoError(t, k.PruneSelectorSwitches(ctx, 10, 2))
	hasLast, err := k.SelectorReporterSwitches.Has(ctx, last)
	require.NoError(t, err)
	require.False(t, hasLast)
	openHistory, err = k.SelectorReporterSwitches.Get(ctx, open)
	require.NoError(t, err)
	require.Len(t, openHistory.Switches, 1)
	require.Zero(t, openHistory.Switches[0].CompletedHeight)
	hasCursor, err = k.SelectorSwitchPruneCursor.Has(ctx)
	require.NoError(t, err)
	require.False(t, hasCursor)
}

func testSelector(last byte) []byte {
	addr := make([]byte, 20)
	addr[19] = last
	return addr
}

func newSwitchTestKeeper(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil, nil, nil, nil,
	)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	require.NoError(t, k.Params.Set(ctx, types.DefaultParams()))
	return k, ctx
}
