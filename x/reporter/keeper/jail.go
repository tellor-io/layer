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

// lockSelectorRowDispute extends dispute_locked_until only (never locked_until_time).
func (k Keeper) lockSelectorRowDispute(ctx context.Context, delegator sdk.AccAddress, until time.Time) error {
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
	prev := sel.DisputeLockedUntil
	sel.DisputeLockedUntil = maxTime(sel.DisputeLockedUntil, until)
	if err := k.Selectors.Set(ctx, delegator.Bytes(), sel); err != nil {
		return err
	}
	if !sel.DisputeLockedUntil.After(prev) {
		return nil
	}
	if err := k.flagStakeRecalcForUnlockedSelector(ctx, delegator, sel); err != nil {
		return err
	}
	return k.bumpRecalcAtTimeForSelectorLock(ctx, sdk.AccAddress(sel.Reporter), sel.DisputeLockedUntil)
}

func (k Keeper) bumpRecalcAtTimeForSelectorLock(ctx context.Context, reporter sdk.AccAddress, lockUntil time.Time) error {
	lockUnix := lockUntil.Unix()
	existing, err := k.RecalcAtTime.Get(ctx, reporter.Bytes())
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		return k.RecalcAtTime.Set(ctx, reporter.Bytes(), lockUnix)
	}
	if lockUnix > existing {
		return k.RecalcAtTime.Set(ctx, reporter.Bytes(), lockUnix)
	}
	return nil
}

// clearDisputeLock ends an active dispute lock without touching locked_until_time.
func clearDisputeLock(sel *types.Selection, now time.Time) {
	if sel.DisputeLockedUntil.After(now) {
		sel.DisputeLockedUntil = now.Add(-time.Second)
	}
}

// flagStakeRecalcForUnlockedSelector flags reporters that should recompute stake after a
// selector lock ends. With an outgoing pending switch, sel.Reporter is still the
// outgoing reporter (stake is held back until finalize); flag both sides of the handoff.
func (k Keeper) flagStakeRecalcForUnlockedSelector(ctx context.Context, selectorAddr sdk.AccAddress, sel types.Selection) error {
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

// lazyClearSelectorLocksIfExpired flags stake recalc once a dispute lock has expired while
// legacy locked_until_time may still exclude stake. Reporter rows are never auto-unjailed.
func (k Keeper) lazyClearSelectorLocksIfExpired(ctx context.Context, selectorAddr sdk.AccAddress, sel *types.Selection) error {
	if sel.DisputeLockedUntil.IsZero() {
		return nil
	}
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	if types.SelectorStakeLocked(*sel, now) {
		return nil
	}
	return k.flagStakeRecalcForUnlockedSelector(ctx, selectorAddr, *sel)
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
		if err := k.lockSelectorRowDispute(ctx, delegator, until); err != nil {
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
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
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
		clearDisputeLock(&sel, now)
		if err := k.Selectors.Set(ctx, delegator.Bytes(), sel); err != nil {
			return err
		}
		if err := k.flagStakeRecalcForUnlockedSelector(ctx, delegator, sel); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) copyReporterJailToSelection(ctx context.Context, addr sdk.AccAddress, reporter types.OracleReporter) error {
	if !reporter.Jailed {
		return nil
	}
	return k.lockSelectorRowDispute(ctx, addr, reporter.JailedUntil)
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
		reporter.JailedUntil = maxTime(reporter.JailedUntil, until)
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

// thirdPartyUnjailDelay is how long after self-unjail eligibility a third party may unjail.
const thirdPartyUnjailDelay = 7 * 24 * time.Hour

// UnjailReporter clears jail on the reporter row and/or that address's selection row.
// Self-unjail is allowed once jail/dispute locks expire; third-party unjail requires an
// additional thirdPartyUnjailDelay after that time.
func (k Keeper) UnjailReporter(ctx context.Context, callerAddr, reporterAddr sdk.AccAddress) error {
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	selfUnjail := callerAddr.Equals(reporterAddr)

	var reporter types.OracleReporter
	hasReporter := false
	reporterResult, err := k.Reporters.Get(ctx, reporterAddr)
	if err == nil {
		reporter = reporterResult
		hasReporter = true
	} else if !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	var sel types.Selection
	hasSelector := false
	selResult, err := k.Selectors.Get(ctx, reporterAddr.Bytes())
	if err == nil {
		sel = selResult
		hasSelector = true
	} else if !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	earliest := time.Time{}
	if hasReporter && reporter.Jailed {
		earliest = maxTime(earliest, reporter.JailedUntil)
	}
	if hasSelector && sel.DisputeLockedUntil.After(earliest) {
		earliest = sel.DisputeLockedUntil
	}

	if selfUnjail {
		if now.Before(earliest) {
			return types.ErrReporterJailed.Wrapf("cannot unjail before jail time is up, %v", earliest)
		}
	} else if now.Before(earliest.Add(thirdPartyUnjailDelay)) {
		return types.ErrThirdPartyUnjailTooEarly.Wrapf(
			"third-party unjail not allowed until %v (7 days after self-unjail eligibility at %v)",
			earliest.Add(thirdPartyUnjailDelay), earliest,
		)
	}

	hasWork := (hasReporter && reporter.Jailed) || (hasSelector && !sel.DisputeLockedUntil.IsZero())
	if !hasWork {
		return types.ErrReporterNotJailed.Wrapf("cannot unjail an already unjailed reporter")
	}

	if hasReporter && reporter.Jailed {
		reporter.Jailed = false
		if err := k.Reporters.Set(ctx, reporterAddr, reporter); err != nil {
			return err
		}
	}

	if hasSelector && !sel.DisputeLockedUntil.IsZero() {
		clearDisputeLock(&sel, now)
		if err := k.Selectors.Set(ctx, reporterAddr.Bytes(), sel); err != nil {
			return err
		}
		if err := k.flagStakeRecalcForUnlockedSelector(ctx, reporterAddr, sel); err != nil {
			return err
		}
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
