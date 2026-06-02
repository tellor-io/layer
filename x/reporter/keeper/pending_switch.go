package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// applyReadyPendingSwitchesForReporter finalizes any pending switches involving
// this reporter that are strictly past unlock_block. It is invoked at the start
// of ReporterStake so MsgSubmitValue triggers handoffs without a BeginBlocker.
// A single ReporterPendingSwitchHead Get is used to short-circuit when nothing
// could be ready at the current height.
func (k Keeper) applyReadyPendingSwitchesForReporter(ctx context.Context, repAddr sdk.AccAddress) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	h := uint64(sdkCtx.BlockHeight())

	head, err := k.ReporterPendingSwitchHeads.Get(ctx, repAddr.Bytes())
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}

	outReady := head.OutgoingCount > 0 && h > head.OutgoingMinUnlock
	inReady := head.IncomingCount > 0 && h > head.IncomingMinUnlock
	if !outReady && !inReady {
		return nil
	}

	if outReady {
		if err := k.applyReadyOutgoingPendingForReporter(ctx, repAddr.Bytes(), h); err != nil {
			return err
		}
	}
	if inReady {
		if err := k.applyReadyIncomingPendingForReporter(ctx, repAddr.Bytes(), h); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) applyReadyOutgoingPendingForReporter(ctx context.Context, from []byte, h uint64) error {
	rng := collections.NewPrefixedPairRange[[]byte, []byte](from)
	iter, err := k.OutgoingPendingSwitches.Iterate(ctx, rng)
	if err != nil {
		return err
	}
	defer iter.Close()

	var keys []collections.Pair[[]byte, []byte]
	for ; iter.Valid(); iter.Next() {
		pk, err := iter.Key()
		if err != nil {
			return err
		}
		val, err := iter.Value()
		if err != nil {
			return err
		}
		if val.UnlockBlock < h {
			keys = append(keys, pk)
		}
	}
	for _, pk := range keys {
		if err := k.finalizePendingSwitch(ctx, pk.K1(), pk.K2()); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) applyReadyIncomingPendingForReporter(ctx context.Context, to []byte, h uint64) error {
	rng := collections.NewPrefixedPairRange[[]byte, []byte](to)
	iter, err := k.IncomingPendingSwitchIdx.Iterate(ctx, rng)
	if err != nil {
		return err
	}
	defer iter.Close()

	var keys []collections.Pair[[]byte, []byte]
	for ; iter.Valid(); iter.Next() {
		pk, err := iter.Key()
		if err != nil {
			return err
		}
		from, err := iter.Value()
		if err != nil {
			return err
		}
		outK := collections.Join(from, pk.K2())
		val, err := k.OutgoingPendingSwitches.Get(ctx, outK)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				continue
			}
			return err
		}
		if val.UnlockBlock < h {
			keys = append(keys, outK)
		}
	}
	for _, outK := range keys {
		if err := k.finalizePendingSwitch(ctx, outK.K1(), outK.K2()); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) finalizePendingSwitch(ctx context.Context, from, selector []byte) error {
	outK := collections.Join(from, selector)
	entry, err := k.OutgoingPendingSwitches.Get(ctx, outK)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}
	to := entry.ToReporter
	inK := collections.Join(to, selector)

	sel, err := k.Selectors.Get(ctx, selector)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return k.removeOutgoingPendingSwitch(ctx, from, selector, to)
		}
		return err
	}
	if !bytes.Equal(sel.Reporter, from) {
		return k.removeOutgoingPendingSwitch(ctx, from, selector, to)
	}

	sel.Reporter = append([]byte(nil), to...)
	sel.SwitchOutLockedUntilBlock = 0
	if err := k.Selectors.Set(ctx, selector, sel); err != nil {
		return err
	}
	if err := k.OutgoingPendingSwitches.Remove(ctx, outK); err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err := k.IncomingPendingSwitchIdx.Remove(ctx, inK); err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err := k.recomputeReporterPendingSwitchHead(ctx, from); err != nil {
		return err
	}
	if err := k.recomputeReporterPendingSwitchHead(ctx, to); err != nil {
		return err
	}
	if err := k.FlagStakeRecalc(ctx, sdk.AccAddress(from)); err != nil {
		return err
	}
	return k.FlagStakeRecalc(ctx, sdk.AccAddress(to))
}

// removeOutgoingPendingSwitch deletes a scheduled switch (from, selector) → oldTo
// and refreshes heads. Does not change Selection.
func (k Keeper) removeOutgoingPendingSwitch(ctx context.Context, from, selector, oldTo []byte) error {
	outK := collections.Join(from, selector)
	inK := collections.Join(oldTo, selector)
	if err := k.OutgoingPendingSwitches.Remove(ctx, outK); err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err := k.IncomingPendingSwitchIdx.Remove(ctx, inK); err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err := k.recomputeReporterPendingSwitchHead(ctx, from); err != nil {
		return err
	}
	return k.recomputeReporterPendingSwitchHead(ctx, oldTo)
}

func (k Keeper) recomputeReporterPendingSwitchHead(ctx context.Context, rep []byte) error {
	var head types.ReporterPendingSwitchHead

	outRng := collections.NewPrefixedPairRange[[]byte, []byte](rep)
	outIter, err := k.OutgoingPendingSwitches.Iterate(ctx, outRng)
	if err != nil {
		return err
	}
	var outMin uint64
	var outCnt uint32
	firstOut := true
	for ; outIter.Valid(); outIter.Next() {
		val, err := outIter.Value()
		if err != nil {
			outIter.Close()
			return err
		}
		outCnt++
		if firstOut || val.UnlockBlock < outMin {
			outMin = val.UnlockBlock
			firstOut = false
		}
	}
	outIter.Close()
	head.OutgoingCount = outCnt
	head.OutgoingMinUnlock = outMin

	inRng := collections.NewPrefixedPairRange[[]byte, []byte](rep)
	inIter, err := k.IncomingPendingSwitchIdx.Iterate(ctx, inRng)
	if err != nil {
		return err
	}
	var inMin uint64
	var inCnt uint32
	firstIn := true
	for ; inIter.Valid(); inIter.Next() {
		pk, err := inIter.Key()
		if err != nil {
			inIter.Close()
			return err
		}
		from, err := inIter.Value()
		if err != nil {
			inIter.Close()
			return err
		}
		ev, err := k.OutgoingPendingSwitches.Get(ctx, collections.Join(from, pk.K2()))
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				continue
			}
			inIter.Close()
			return err
		}
		inCnt++
		if firstIn || ev.UnlockBlock < inMin {
			inMin = ev.UnlockBlock
			firstIn = false
		}
	}
	inIter.Close()
	head.IncomingCount = inCnt
	head.IncomingMinUnlock = inMin

	if head.OutgoingCount == 0 && head.IncomingCount == 0 {
		return k.ReporterPendingSwitchHeads.Remove(ctx, rep)
	}
	return k.ReporterPendingSwitchHeads.Set(ctx, rep, head)
}

func (k Keeper) scheduleReporterSwitch(
	ctx context.Context,
	selectorAddr sdk.AccAddress,
	selection *types.Selection,
	prevReporter, newReporter sdk.AccAddress,
) error {
	if k.oracleKeeper == nil {
		return errors.New("oracle keeper not configured")
	}
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}
	maxP := params.MaxPendingSwitchesPerReporter
	if maxP == 0 {
		maxP = types.DefaultMaxPendingSwitchesPerReporter
	}

	outK := collections.Join(prevReporter.Bytes(), selectorAddr.Bytes())
	existing, err := k.OutgoingPendingSwitches.Get(ctx, outK)
	wasReplace := err == nil
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if wasReplace {
		if bytes.Equal(existing.ToReporter, newReporter.Bytes()) {
			return nil
		}
		if err := k.removeOutgoingPendingSwitch(ctx, prevReporter.Bytes(), selectorAddr.Bytes(), existing.ToReporter); err != nil {
			return err
		}
	}

	if !wasReplace {
		prevHead, _ := k.reporterPendingSwitchHeadOrZero(ctx, prevReporter.Bytes())
		if uint64(prevHead.OutgoingCount) >= maxP {
			return errors.New("outgoing reporter has reached max pending reporter switches")
		}
	}

	newHead, _ := k.reporterPendingSwitchHeadOrZero(ctx, newReporter.Bytes())
	if uint64(newHead.IncomingCount) >= maxP {
		return errors.New("target reporter has reached max pending incoming reporter switches")
	}

	var unlockBlock uint64
	if wasReplace {
		unlockBlock = existing.UnlockBlock
	} else {
		unlockBlock, err = k.oracleKeeper.GetMaxOpenCommitmentForReporter(ctx, prevReporter.Bytes())
		if err != nil {
			return err
		}
	}

	entry := types.PendingSwitchEntry{
		ToReporter:  newReporter.Bytes(),
		UnlockBlock: unlockBlock,
	}
	if err := k.OutgoingPendingSwitches.Set(ctx, outK, entry); err != nil {
		return err
	}
	inK := collections.Join(newReporter.Bytes(), selectorAddr.Bytes())
	if err := k.IncomingPendingSwitchIdx.Set(ctx, inK, prevReporter.Bytes()); err != nil {
		return err
	}

	if err := k.mergeReporterPendingSwitchHeadOutgoingAdd(ctx, prevReporter.Bytes(), unlockBlock); err != nil {
		return err
	}
	if err := k.mergeReporterPendingSwitchHeadIncomingAdd(ctx, newReporter.Bytes(), unlockBlock); err != nil {
		return err
	}

	selection.SwitchOutLockedUntilBlock = unlockBlock
	if err := k.Selectors.Set(ctx, selectorAddr.Bytes(), *selection); err != nil {
		return err
	}
	if wasReplace {
		return k.FlagStakeRecalc(ctx, sdk.AccAddress(existing.ToReporter))
	}
	return nil
}

func (k Keeper) reporterPendingSwitchHeadOrZero(ctx context.Context, rep []byte) (types.ReporterPendingSwitchHead, error) {
	h, err := k.ReporterPendingSwitchHeads.Get(ctx, rep)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.ReporterPendingSwitchHead{}, nil
		}
		return types.ReporterPendingSwitchHead{}, err
	}
	return h, nil
}

func (k Keeper) mergeReporterPendingSwitchHeadOutgoingAdd(ctx context.Context, rep []byte, unlock uint64) error {
	head, err := k.reporterPendingSwitchHeadOrZero(ctx, rep)
	if err != nil {
		return err
	}
	head.OutgoingCount++
	if head.OutgoingCount == 1 || unlock < head.OutgoingMinUnlock {
		head.OutgoingMinUnlock = unlock
	}
	return k.ReporterPendingSwitchHeads.Set(ctx, rep, head)
}

func (k Keeper) mergeReporterPendingSwitchHeadIncomingAdd(ctx context.Context, rep []byte, unlock uint64) error {
	head, err := k.reporterPendingSwitchHeadOrZero(ctx, rep)
	if err != nil {
		return err
	}
	head.IncomingCount++
	if head.IncomingCount == 1 || unlock < head.IncomingMinUnlock {
		head.IncomingMinUnlock = unlock
	}
	return k.ReporterPendingSwitchHeads.Set(ctx, rep, head)
}

// hasOutgoingPendingSwitch returns true if selector has a scheduled switch away
// from repAddr (stake must not count toward that reporter).
func (k Keeper) hasOutgoingPendingSwitch(ctx context.Context, repAddr, selectorAddr []byte) (bool, error) {
	return k.OutgoingPendingSwitches.Has(ctx, collections.Join(repAddr, selectorAddr))
}

// maybeFinalizePendingSwitchForRemoveSelector applies a ready pending switch before
// RemoveSelector proceeds, or rejects removal while the handoff is still locked.
// Finalize uses the same height rule as ReporterStake: unlock_block < current height.
func (k Keeper) maybeFinalizePendingSwitchForRemoveSelector(
	ctx context.Context,
	selectorAddr sdk.AccAddress,
	selector *types.Selection,
	reporter *types.OracleReporter,
) error {
	from := selector.Reporter
	hasPending, err := k.hasOutgoingPendingSwitch(ctx, from, selectorAddr.Bytes())
	if err != nil {
		return err
	}
	if !hasPending {
		return nil
	}
	outK := collections.Join(from, selectorAddr.Bytes())
	entry, err := k.OutgoingPendingSwitches.Get(ctx, outK)
	if err != nil {
		return err
	}
	currentBlock := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	if currentBlock > entry.UnlockBlock {
		if err := k.finalizePendingSwitch(ctx, from, selectorAddr.Bytes()); err != nil {
			return err
		}
		*selector, err = k.Selectors.Get(ctx, selectorAddr.Bytes())
		if err != nil {
			return err
		}
		*reporter, err = k.Reporters.Get(ctx, selector.Reporter)
		return err
	}
	return fmt.Errorf(
		"selector cannot be removed while a reporter switch is pending (switch finalizes after block height %d)",
		entry.UnlockBlock,
	)
}

// pendingSwitchToReporter returns (true, toAddr) if there is a pending switch
// from prevReporter for this selector.
func (k Keeper) pendingSwitchToReporter(ctx context.Context, prevReporter, selectorAddr sdk.AccAddress) (bool, []byte, error) {
	outK := collections.Join(prevReporter.Bytes(), selectorAddr.Bytes())
	e, err := k.OutgoingPendingSwitches.Get(ctx, outK)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, e.ToReporter, nil
}
