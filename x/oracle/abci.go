package oracle

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/keeper"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	return k.SetAggregatedReport(ctx)
}

func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	// Rotate through the cycle list and set the current query index
	return k.RotateQueries(ctx)
}
