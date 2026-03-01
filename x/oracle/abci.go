package oracle

import (
	"context"
	"time"

	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	// "github.com/tellor-io/layer/utils"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, telemetry.Now(), telemetry.MetricKeyEndBlocker)
	endBlockStart := time.Now()

	// Rotate through the cycle list and set the current query index
	t0 := time.Now()
	if err := k.SetAggregatedReport(ctx); err != nil {
		return err
	}
	setAggDur := time.Since(t0)

	// call claim deposit on oldest aggregate in queue if > 12 hrs old
	t1 := time.Now()
	if err := k.AutoClaimDeposits(ctx); err != nil {
		return err
	}
	autoClaimDur := time.Since(t1)

	t2 := time.Now()
	if err := k.RotateQueries(ctx); err != nil {
		return err
	}
	rotateDur := time.Since(t2)

	t3 := time.Now()
	err := k.RemoveOldReports(ctx)
	removeReportsDur := time.Since(t3)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.Logger(ctx).Info("oracle EndBlocker",
		"height", sdkCtx.BlockHeight(),
		"SetAggregatedReport_ms", setAggDur.Milliseconds(),
		"AutoClaimDeposits_ms", autoClaimDur.Milliseconds(),
		"RotateQueries_ms", rotateDur.Milliseconds(),
		"RemoveOldReports_ms", removeReportsDur.Milliseconds(),
		"total_ms", time.Since(endBlockStart).Milliseconds(),
	)

	return err
}
