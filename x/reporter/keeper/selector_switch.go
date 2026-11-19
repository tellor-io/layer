package keeper

import (
	"bytes"
	"context"
	"errors"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) selectorSwitchHistory(ctx context.Context, selector []byte) (types.SelectorReporterSwitches, error) {
	history, err := k.SelectorReporterSwitches.Get(ctx, selector)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.SelectorReporterSwitches{}, nil
		}
		return types.SelectorReporterSwitches{}, err
	}
	return history, nil
}

func (k Keeper) setSelectorSwitchHistory(ctx context.Context, selector []byte, history types.SelectorReporterSwitches) error {
	if len(history.Switches) == 0 {
		return k.SelectorReporterSwitches.Remove(ctx, selector)
	}
	return k.SelectorReporterSwitches.Set(ctx, selector, history)
}

// recordSelectorSwitchBegan prepends an open switch. Selection stays on originalReporter
// until finalization. Replacing the pending target does not call this again.
// A row that already has CancelledHeight set is not the open switch, so the next
// real switch is a new row.
func (k Keeper) recordSelectorSwitchBegan(ctx context.Context, selector, originalReporter []byte) error {
	history, err := k.selectorSwitchHistory(ctx, selector)
	if err != nil {
		return err
	}
	for _, sw := range history.Switches {
		if sw.CancelledHeight != 0 {
			continue
		}
		if sw.CompletedHeight == 0 && bytes.Equal(sw.OriginalReporter, originalReporter) {
			return nil
		}
		break
	}
	entry := types.SelectorReporterSwitch{
		OriginalReporter: append([]byte(nil), originalReporter...),
		BeganHeight:      uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()),
	}
	history.Switches = append([]types.SelectorReporterSwitch{entry}, history.Switches...)
	return k.setSelectorSwitchHistory(ctx, selector, history)
}

// recordSelectorSwitchCompleted closes the open switch away from originalReporter.
// Missing history (a switch scheduled before this index existed) is a no-op.
// A row that already has CancelledHeight set is left alone, so a later finalize
// cannot close the cancelled switch.
func (k Keeper) recordSelectorSwitchCompleted(ctx context.Context, selector, originalReporter []byte) error {
	history, err := k.selectorSwitchHistory(ctx, selector)
	if err != nil {
		return err
	}
	height := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	for i := range history.Switches {
		sw := &history.Switches[i]
		if sw.CancelledHeight != 0 {
			continue
		}
		if sw.CompletedHeight == 0 && bytes.Equal(sw.OriginalReporter, originalReporter) {
			sw.CompletedHeight = height
			return k.setSelectorSwitchHistory(ctx, selector, history)
		}
	}
	return nil
}

// cancelOpenSelectorSwitch marks an open switch that will not move Selection.
// The history row stays, with CancelledHeight set to the current block and
// CompletedHeight still 0, until the same prune that drops a completed switch.
func (k Keeper) cancelOpenSelectorSwitch(ctx context.Context, selector, originalReporter []byte) error {
	history, err := k.selectorSwitchHistory(ctx, selector)
	if err != nil {
		return err
	}
	height := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	for i := range history.Switches {
		sw := &history.Switches[i]
		if sw.CancelledHeight != 0 {
			continue
		}
		if sw.CompletedHeight == 0 && bytes.Equal(sw.OriginalReporter, originalReporter) {
			sw.CancelledHeight = height
			return k.setSelectorSwitchHistory(ctx, selector, history)
		}
	}
	return nil
}

// LatestSelectorSwitch returns the newest switch for selector.
func (k Keeper) LatestSelectorSwitch(ctx context.Context, selector sdk.AccAddress) (types.SelectorReporterSwitch, bool, error) {
	history, err := k.selectorSwitchHistory(ctx, selector.Bytes())
	if err != nil {
		return types.SelectorReporterSwitch{}, false, err
	}
	if len(history.Switches) == 0 {
		return types.SelectorReporterSwitch{}, false, nil
	}
	return history.Switches[0], true, nil
}

// SelectorSwitchesNewestFirst returns the selector's reporter changes, newest first.
func (k Keeper) SelectorSwitchesNewestFirst(ctx context.Context, selector sdk.AccAddress) ([]types.SelectorReporterSwitch, error) {
	history, err := k.selectorSwitchHistory(ctx, selector.Bytes())
	if err != nil {
		return nil, err
	}
	return history.Switches, nil
}

// SelectorSwitchUnfinishedAt reports whether a switch was in progress at blockHeight.
// That is true when a switch began at or before blockHeight and had not completed
// by then: CompletedHeight == 0 or CompletedHeight > blockHeight.
// A row whose CancelledHeight is set and is at or before blockHeight is skipped,
// and the walk continues into older rows. A cancel after blockHeight is still
// unfinished. An open self-demotion is unfinished even though ReporterAtHeight
// still returns the reporter's own address. Switches are newest first.
func (k Keeper) SelectorSwitchUnfinishedAt(ctx context.Context, selector sdk.AccAddress, blockHeight uint64) (bool, error) {
	history, err := k.selectorSwitchHistory(ctx, selector.Bytes())
	if err != nil {
		return false, err
	}
	for _, sw := range history.Switches {
		if sw.BeganHeight > blockHeight {
			continue
		}
		if sw.CancelledHeight != 0 && sw.CancelledHeight <= blockHeight {
			continue
		}
		return sw.CompletedHeight == 0 || sw.CompletedHeight > blockHeight, nil
	}
	return false, nil
}

// ReporterAtHeight returns who selector was reporting for at blockHeight.
// liveAccurate is true when the live selection is already that reporter: no history,
// or the newest switch completed at or before blockHeight. An open switch has not
// moved Selection, so the walk continues into older rows when blockHeight is before
// a prior completion. SelectorSwitchUnfinishedAt reports whether that switch was
// still unfinished.
func (k Keeper) ReporterAtHeight(ctx context.Context, selector sdk.AccAddress, blockHeight uint64) (sdk.AccAddress, bool, error) {
	sel, err := k.Selectors.Get(ctx, selector.Bytes())
	if err != nil {
		return nil, false, err
	}
	live := append(sdk.AccAddress(nil), sel.Reporter...)
	history, err := k.selectorSwitchHistory(ctx, selector.Bytes())
	if err != nil {
		return nil, false, err
	}
	if len(history.Switches) == 0 {
		return live, true, nil
	}
	latest := history.Switches[0]
	if latest.CompletedHeight != 0 && latest.CompletedHeight <= blockHeight {
		return live, true, nil
	}
	current := live
	for _, sw := range history.Switches {
		if sw.CompletedHeight != 0 && sw.CompletedHeight <= blockHeight {
			return current, bytes.Equal(current, live), nil
		}
		current = append(sdk.AccAddress(nil), sw.OriginalReporter...)
	}
	return current, bytes.Equal(current, live), nil
}

// ReporterSelfDemoted reports whether addr's own selection points at someone else.
// An open demotion still selects themselves, and selecting themselves again clears it.
// Switch history is not consulted. A later switch can outlive the self-demotion row
// after prune, and selectors may still point at addr either way.
func (k Keeper) ReporterSelfDemoted(ctx context.Context, addr sdk.AccAddress) (bool, error) {
	sel, err := k.Selectors.Get(ctx, addr.Bytes())
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return !bytes.Equal(sel.Reporter, addr.Bytes()), nil
}

// pruneSwitchHistory drops completed switches that finished at or before pruneHeight
// and cancelled switches whose CancelledHeight is at or before pruneHeight.
// An open switch is kept. For every height >= pruneHeight the walk still resolves,
// because a newer switch's original reporter is who they were from the deleted
// switch's completion until that newer switch completed.
func pruneSwitchHistory(history types.SelectorReporterSwitches, pruneHeight uint64) (types.SelectorReporterSwitches, bool) {
	kept := make([]types.SelectorReporterSwitch, 0, len(history.Switches))
	changed := false
	for _, sw := range history.Switches {
		if sw.CompletedHeight != 0 && sw.CompletedHeight <= pruneHeight {
			changed = true
			continue
		}
		if sw.CancelledHeight != 0 && sw.CancelledHeight <= pruneHeight {
			changed = true
			continue
		}
		kept = append(kept, sw)
	}
	if !changed {
		return history, false
	}
	return types.SelectorReporterSwitches{Switches: kept}, true
}

// PruneSelectorSwitches removes completed and cancelled switch history that no
// longer affects ReporterAtHeight or SelectorSwitchUnfinishedAt for any height
// >= pruneHeight. It visits at most limit selectors per call and resumes on the
// next call, so end block can bound the work.
func (k Keeper) PruneSelectorSwitches(ctx context.Context, pruneHeight uint64, limit int) error {
	if limit <= 0 {
		return nil
	}
	cursor, err := k.SelectorSwitchPruneCursor.Get(ctx)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	rng := new(collections.Range[[]byte])
	if len(cursor) > 0 {
		rng = rng.StartExclusive(cursor)
	}
	iter, err := k.SelectorReporterSwitches.Iterate(ctx, rng)
	if err != nil {
		return err
	}

	type update struct {
		key     []byte
		history types.SelectorReporterSwitches
		remove  bool
	}
	updates := make([]update, 0, limit)
	var last []byte
	for ; iter.Valid() && len(updates) < limit; iter.Next() {
		key, err := iter.Key()
		if err != nil {
			iter.Close()
			return err
		}
		history, err := iter.Value()
		if err != nil {
			iter.Close()
			return err
		}
		pruned, changed := pruneSwitchHistory(history, pruneHeight)
		last = append([]byte(nil), key...)
		if !changed {
			updates = append(updates, update{key: last})
			continue
		}
		updates = append(updates, update{
			key:     last,
			history: pruned,
			remove:  len(pruned.Switches) == 0,
		})
	}
	wrapped := !iter.Valid()
	if err := iter.Close(); err != nil {
		return err
	}

	for _, upd := range updates {
		if upd.remove {
			if err := k.SelectorReporterSwitches.Remove(ctx, upd.key); err != nil {
				return err
			}
			continue
		}
		if len(upd.history.Switches) == 0 {
			continue
		}
		if err := k.SelectorReporterSwitches.Set(ctx, upd.key, upd.history); err != nil {
			return err
		}
	}
	if wrapped || len(last) == 0 {
		return k.SelectorSwitchPruneCursor.Remove(ctx)
	}
	return k.SelectorSwitchPruneCursor.Set(ctx, last)
}
