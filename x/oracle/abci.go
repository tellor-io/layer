package oracle

import (
	"context"

	// "github.com/tellor-io/layer/utils"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, telemetry.Now(), telemetry.MetricKeyEndBlocker)
	// Rotate through the cycle list and set the current query index
	if err := k.SetAggregatedReport(ctx); err != nil {
		return err
	}
	return k.RotateQueries(ctx)
}
