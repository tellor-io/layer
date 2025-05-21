package oracle

import (
	"context"
	"fmt"
	"time"

	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	// "github.com/tellor-io/layer/utils"
	"github.com/cosmos/cosmos-sdk/telemetry"
)

func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, telemetry.Now(), telemetry.MetricKeyEndBlocker)
	k.Logger(ctx).Info(fmt.Sprintf("Start time oracle module end block: %d", time.Now().UnixMilli()))
	// Rotate through the cycle list and set the current query index
	if err := k.SetAggregatedReport(ctx); err != nil {
		return err
	}

	// call claim deposit on oldest aggregate in queue if > 12 hrs old
	if err := k.AutoClaimDeposits(ctx); err != nil {
		return err
	}

	return k.RotateQueries(ctx)
}
