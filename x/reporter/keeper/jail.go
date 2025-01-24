package keeper

import (
	"context"
	gomath "math"
	"strconv"
	"time"

	"github.com/tellor-io/layer/x/reporter/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// jail a reporter for a given duration
func (k Keeper) JailReporter(ctx context.Context, reporterAddr sdk.AccAddress, jailDuration uint64) error {
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return err
	}
	if reporter.Jailed {
		return types.ErrReporterJailed.Wrapf("cannot jail already jailed reporter, %v", reporter)
	}
	sdkctx := sdk.UnwrapSDKContext(ctx)
	if jailDuration == gomath.MaxInt64 {
		// Set jailed until to the max time by converting max int64 into seconds and nanoseconds
		reporter.JailedUntil = time.Unix(int64(jailDuration)/1e9, int64(jailDuration)%1e9)
	} else {
		reporter.JailedUntil = sdkctx.BlockTime().Add(time.Second * time.Duration(jailDuration))
	}
	reporter.Jailed = true
	err = k.Reporters.Set(ctx, reporterAddr, reporter)
	if err != nil {
		return err
	}
	sdkctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"jailed_reporter",
			sdk.NewAttribute("reporter", reporterAddr.String()),
			sdk.NewAttribute("duration", strconv.FormatUint(jailDuration, 10)),
		),
	})
	return nil
}

// unjail a reporter
func (k Keeper) UnjailReporter(ctx context.Context, reporterAddr sdk.AccAddress, reporter types.OracleReporter) error {
	if !reporter.Jailed {
		return types.ErrReporterNotJailed.Wrapf("cannot unjail an already unjailed reporter, %v", reporter.Jailed)
	}

	sdkctx := sdk.UnwrapSDKContext(ctx)
	if sdkctx.BlockTime().Before(reporter.JailedUntil) {
		return types.ErrReporterJailed.Wrapf("cannot unjail reporter before jail time is up, %v", reporter.JailedUntil)
	}

	reporter.Jailed = false
	reporter.JailedUntil = time.Time{}
	return k.Reporters.Set(ctx, reporterAddr, reporter)
}

func (k Keeper) UpdateJailedUntilOnFailedDispute(ctx context.Context, reporterAddr sdk.AccAddress) error {
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return err
	}
	if !reporter.Jailed {
		return nil
	}

	sdkctx := sdk.UnwrapSDKContext(ctx)
	reporter.JailedUntil = sdkctx.BlockTime().Add(-1 * time.Second)

	return k.Reporters.Set(ctx, reporterAddr, reporter)
}
