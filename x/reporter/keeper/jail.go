package keeper

import (
	"context"
	"strconv"
	"time"

	"github.com/tellor-io/layer/x/reporter/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// send a reporter to jail
func (k Keeper) JailReporter(ctx context.Context, reporterAddr sdk.AccAddress, jailDuration int64) error {
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return err
	}
	if reporter.Jailed {
		return types.ErrReporterJailed.Wrapf("cannot jail already jailed reporter, %v", reporter)
	}
	sdkctx := sdk.UnwrapSDKContext(ctx)
	reporter.JailedUntil = sdkctx.HeaderInfo().Time.Add(time.Second * time.Duration(jailDuration))
	reporter.Jailed = true
	err = k.Reporters.Set(ctx, reporterAddr, reporter)
	if err != nil {
		return err
	}
	sdkctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"jailed_reporter",
			sdk.NewAttribute("reporter", reporterAddr.String()),
			sdk.NewAttribute("duration", strconv.FormatInt(jailDuration, 10)),
		),
	})
	return nil
}

// remove a reporter from jail
func (k Keeper) UnjailReporter(ctx context.Context, reporterAddr sdk.AccAddress, reporter types.OracleReporter) error {
	if !reporter.Jailed {
		return types.ErrReporterNotJailed.Wrapf("cannot unjail already unjailed reporter, %v", reporter.Jailed)
	}

	sdkctx := sdk.UnwrapSDKContext(ctx)
	if sdkctx.HeaderInfo().Time.Before(reporter.JailedUntil) {
		return types.ErrReporterJailed.Wrapf("cannot unjail reporter before jail time is up, %v", reporter.JailedUntil)
	}

	reporter.Jailed = false
	return k.Reporters.Set(ctx, reporterAddr, reporter)
}
