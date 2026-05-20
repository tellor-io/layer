package keeper

import (
	"context"
	"errors"
	gomath "math"
	"sort"
	"strconv"
	"time"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func (k Keeper) jailUntil(ctx context.Context, jailDuration uint64) (time.Time, error) {
	sdkctx := sdk.UnwrapSDKContext(ctx)
	if jailDuration == uint64(gomath.MaxInt64) {
		return time.Unix(int64(jailDuration)/1e9, int64(jailDuration)%1e9), nil
	}
	return sdkctx.BlockTime().Add(time.Second * time.Duration(jailDuration)), nil
}

func (k Keeper) lockSelectorRow(ctx context.Context, delegator sdk.AccAddress, until time.Time) error {
	sel, err := k.Selectors.Get(ctx, delegator.Bytes())
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	if until.Before(now) {
		until = now
	}
	sel.LockedUntilTime = maxTime(sel.LockedUntilTime, until)
	sel.JailedUntil = maxTime(sel.JailedUntil, until)
	sel.Jailed = true
	return k.Selectors.Set(ctx, delegator.Bytes(), sel)
}

// flagStakeRecalcForUnjailedSelector flags reporters that should recompute stake after a
// selector's dispute lock ends. With an outgoing pending switch, sel.Reporter is still the
// outgoing reporter (stake is held back until finalize); flag both sides of the handoff.
func (k Keeper) flagStakeRecalcForUnjailedSelector(ctx context.Context, selectorAddr sdk.AccAddress, sel types.Selection) error {
	hasOutgoing, err := k.hasOutgoingPendingSwitch(ctx, sel.Reporter, selectorAddr.Bytes())
	if err != nil {
		return err
	}
	if hasOutgoing {
		entry, err := k.OutgoingPendingSwitches.Get(ctx, collections.Join(sel.Reporter, selectorAddr.Bytes()))
		if err != nil {
			return err
		}
		if err := k.FlagStakeRecalc(ctx, sdk.AccAddress(sel.Reporter)); err != nil {
			return err
		}
		return k.FlagStakeRecalc(ctx, sdk.AccAddress(entry.ToReporter))
	}
	return k.FlagStakeRecalc(ctx, sdk.AccAddress(sel.Reporter))
}

// unjailSelectorRow ends an active dispute lock on a selector without writing zero
// timestamps (invalid for state encoding). Historical JailedUntil is retained.
func (k Keeper) unjailSelectorRow(ctx context.Context, selectorAddr sdk.AccAddress, sel *types.Selection, clearActiveLockUntil bool) error {
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	sel.Jailed = false
	if clearActiveLockUntil && sel.LockedUntilTime.After(now) {
		sel.LockedUntilTime = now.Add(-time.Second)
	}
	if err := k.Selectors.Set(ctx, selectorAddr.Bytes(), *sel); err != nil {
		return err
	}
	return k.flagStakeRecalcForUnjailedSelector(ctx, selectorAddr, *sel)
}

// lazyUnjailSelectorIfExpired clears selector dispute-jail once the sentence has ended so
// stake counts again on the next report. Reporter rows are never auto-unjailed.
func (k Keeper) lazyUnjailSelectorIfExpired(ctx context.Context, selectorAddr sdk.AccAddress, sel *types.Selection) error {
	if !sel.Jailed {
		return nil
	}
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	if types.SelectorStakeLocked(*sel, now) {
		return nil
	}
	return k.unjailSelectorRow(ctx, selectorAddr, sel, false)
}

func (k Keeper) jailSelectorsFromReportSnapshot(
	ctx context.Context,
	reporter []byte,
	reportBlockNumber uint64,
	until time.Time,
) error {
	snap, err := k.GetDelegationsAmount(ctx, reporter, reportBlockNumber)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{})
	delegators := make([]string, 0)
	for _, origin := range snap.TokenOrigins {
		delegator := sdk.AccAddress(origin.DelegatorAddress)
		key := delegator.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		delegators = append(delegators, key)
	}
	sort.Strings(delegators)
	sdkctx := sdk.UnwrapSDKContext(ctx)
	for _, key := range delegators {
		delegator := sdk.MustAccAddressFromBech32(key)
		if err := k.lockSelectorRow(ctx, delegator, until); err != nil {
			return err
		}
		sdkctx.EventManager().EmitEvent(sdk.NewEvent(
			"jailed_selector",
			sdk.NewAttribute("selector", key),
			sdk.NewAttribute("until", until.Format(time.RFC3339)),
		))
	}
	return nil
}

func (k Keeper) clearSelectorLocksFromReportSnapshot(
	ctx context.Context,
	reporter []byte,
	reportBlockNumber uint64,
) error {
	snap, err := k.GetDelegationsAmount(ctx, reporter, reportBlockNumber)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, origin := range snap.TokenOrigins {
		delegator := sdk.AccAddress(origin.DelegatorAddress)
		if _, ok := seen[delegator.String()]; ok {
			continue
		}
		seen[delegator.String()] = struct{}{}
		sel, err := k.Selectors.Get(ctx, delegator.Bytes())
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				continue
			}
			return err
		}
		if err := k.unjailSelectorRow(ctx, delegator, &sel, true); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) copyReporterJailToSelection(ctx context.Context, addr sdk.AccAddress, reporter types.OracleReporter) error {
	if !reporter.Jailed {
		return nil
	}
	return k.lockSelectorRow(ctx, addr, reporter.JailedUntil)
}

// JailReporter jails the reporter row (if present) and every selector in the report snapshot.
// Warning disputes use jailDuration 0: until is block time, so the reporter is jailed but may
// unjail immediately; JailedUntil is only bumped when it is already before block time.
func (k Keeper) JailReporter(ctx context.Context, reporterAddr sdk.AccAddress, jailDuration, reportBlockNumber uint64) error {
	until, err := k.jailUntil(ctx, jailDuration)
	if err != nil {
		return err
	}
	sdkctx := sdk.UnwrapSDKContext(ctx)
	now := sdkctx.BlockTime()
	if until.Before(now) {
		until = now
	}

	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err == nil {
		wasJailed := reporter.Jailed
		reporter.Jailed = true
		if reporter.JailedUntil.Before(now) {
			reporter.JailedUntil = until
		}
		if err := k.Reporters.Set(ctx, reporterAddr, reporter); err != nil {
			return err
		}
		if !wasJailed {
			sdkctx.EventManager().EmitEvent(sdk.NewEvent(
				"jailed_reporter",
				sdk.NewAttribute("reporter", reporterAddr.String()),
				sdk.NewAttribute("duration", strconv.FormatUint(jailDuration, 10)),
			))
		}
	} else if !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	return k.jailSelectorsFromReportSnapshot(ctx, reporterAddr.Bytes(), reportBlockNumber, until)
}

// UnjailReporter clears jail on the reporter row and/or that address's selection row.
func (k Keeper) UnjailReporter(ctx context.Context, reporterAddr sdk.AccAddress) error {
	sdkctx := sdk.UnwrapSDKContext(ctx)
	now := sdkctx.BlockTime()
	unjailed := false

	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err == nil {
		if reporter.Jailed {
			if now.Before(reporter.JailedUntil) {
				return types.ErrReporterJailed.Wrapf("cannot unjail reporter before jail time is up, %v", reporter.JailedUntil)
			}
			reporter.Jailed = false
			if err := k.Reporters.Set(ctx, reporterAddr, reporter); err != nil {
				return err
			}
			unjailed = true
		}
	} else if !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	sel, err := k.Selectors.Get(ctx, reporterAddr.Bytes())
	if err == nil {
		if sel.Jailed {
			if now.Before(sel.JailedUntil) {
				return types.ErrReporterJailed.Wrapf("cannot unjail selector before jail time is up, %v", sel.JailedUntil)
			}
			if err := k.unjailSelectorRow(ctx, reporterAddr, &sel, true); err != nil {
				return err
			}
			unjailed = true
		}
	} else if !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	if !unjailed {
		return types.ErrReporterNotJailed.Wrapf("cannot unjail an already unjailed reporter")
	}
	return nil
}

func (k Keeper) UpdateJailedUntilOnFailedDispute(ctx context.Context, reporterAddr sdk.AccAddress, reportBlockNumber uint64) error {
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err == nil && reporter.Jailed {
		sdkctx := sdk.UnwrapSDKContext(ctx)
		reporter.JailedUntil = sdkctx.BlockTime().Add(-1 * time.Second)
		if err := k.Reporters.Set(ctx, reporterAddr, reporter); err != nil {
			return err
		}
	} else if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	return k.clearSelectorLocksFromReportSnapshot(ctx, reporterAddr.Bytes(), reportBlockNumber)
}
