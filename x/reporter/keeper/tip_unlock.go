package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/tellor-io/layer/lib/metrics"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ProcessMatureTipUnlocks pays out up to maxItems mature tip unlocks (FIFO by
// completion time, then unlock_id). Remaining mature entries are processed in
// later blocks — same bounded-queue pattern as ProcessDistributionQueue.
func (k Keeper) ProcessMatureTipUnlocks(ctx context.Context, maxItems int) error {
	if maxItems <= 0 {
		return nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	endUnix := sdkCtx.BlockTime().Unix()
	rng := collections.NewPrefixUntilPairRange[int64, uint64](endUnix)

	var toProcess []collections.Pair[int64, uint64]
	err := k.TipUnlockQueue.Walk(ctx, rng, func(key collections.Pair[int64, uint64], _ []byte) (stop bool, err error) {
		toProcess = append(toProcess, key)
		return len(toProcess) >= maxItems, nil
	})
	if err != nil {
		return err
	}

	for _, queueKey := range toProcess {
		unlockID := queueKey.K2()
		selectorBytes, err := k.TipUnlockQueue.Get(ctx, queueKey)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				continue
			}
			return err
		}
		selector := sdk.AccAddress(selectorBytes)

		entry, err := k.TipUnlocks.Get(ctx, collections.Join(selector.Bytes(), unlockID))
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				// Queue row without TipUnlocks entry — drop the stale queue key.
				_ = k.TipUnlockQueue.Remove(ctx, queueKey)
				continue
			}
			return err
		}

		coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, entry.Amount))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.TipsUnlockPool, selector, coins); err != nil {
			return err
		}

		if err := k.TipUnlocks.Remove(ctx, collections.Join(selector.Bytes(), unlockID)); err != nil {
			return err
		}
		if err := k.TipUnlockQueue.Remove(ctx, queueKey); err != nil {
			return err
		}

		sdkCtx.EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				"tip_unlock_completed",
				sdk.NewAttribute("selector", selector.String()),
				sdk.NewAttribute("unlock_id", strconv.FormatUint(unlockID, 10)),
				sdk.NewAttribute("amount", entry.Amount.String()),
			),
		})
		telemetry.IncrCounterWithLabels(
			[]string{"tip_unlock_completed_amount"},
			float32(entry.Amount.Int64()),
			[]metrics.Label{{Name: "chain_id", Value: sdkCtx.ChainID()}, {Name: "selector", Value: hex.EncodeToString(selector.Bytes())}},
		)
	}

	return nil
}

// clearSelectorTipsAmount reduces SelectorTips matching WithdrawTip dust handling.
// shares are the full LegacyDec tip balance before withdraw.
func (k Keeper) clearSelectorTipsAmount(ctx context.Context, selector sdk.AccAddress, shares math.LegacyDec) error {
	remainder := shares.Sub(shares.TruncateDec())
	if remainder.IsZero() {
		return k.SelectorTips.Remove(ctx, selector)
	}
	return k.SelectorTips.Set(ctx, selector, remainder)
}

// startTipUnlock creates TipUnlocks + TipUnlockQueue entries and returns the new unlock_id.
func (k Keeper) startTipUnlock(ctx context.Context, selector sdk.AccAddress, amount math.Int, completionTime time.Time) (uint64, error) {
	unlockID, err := k.TipUnlockID.Next(ctx)
	if err != nil {
		return 0, err
	}

	entry := types.TipUnlockEntry{
		Amount:         amount,
		CompletionTime: completionTime,
	}
	if err := k.TipUnlocks.Set(ctx, collections.Join(selector.Bytes(), unlockID), entry); err != nil {
		return 0, err
	}
	if err := k.TipUnlockQueue.Set(ctx, collections.Join(completionTime.Unix(), unlockID), selector.Bytes()); err != nil {
		return 0, err
	}
	return unlockID, nil
}

// removeTipUnlock deletes TipUnlocks and TipUnlockQueue for (selector, unlockID).
func (k Keeper) removeTipUnlock(ctx context.Context, selector sdk.AccAddress, unlockID uint64, completionUnix int64) error {
	if err := k.TipUnlocks.Remove(ctx, collections.Join(selector.Bytes(), unlockID)); err != nil {
		return err
	}
	return k.TipUnlockQueue.Remove(ctx, collections.Join(completionUnix, unlockID))
}

// creditSelectorTips adds amount (as LegacyDec) back onto SelectorTips.
func (k Keeper) creditSelectorTips(ctx context.Context, selector sdk.AccAddress, amount math.Int) error {
	oldTips, err := k.SelectorTips.Get(ctx, selector)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		oldTips = math.LegacyZeroDec()
	}
	return k.SelectorTips.Set(ctx, selector, oldTips.Add(math.LegacyNewDecFromInt(amount)))
}

// settleSelectorReporter settles the selector's reporter period if the selector exists.
func (k Keeper) settleSelectorReporter(ctx context.Context, selector sdk.AccAddress) error {
	selection, err := k.Selectors.Get(ctx, selector)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}
	return k.SettleReporter(ctx, selection.Reporter)
}
