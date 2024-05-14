package oracle

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/keeper"
)

func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	// Rotate through the cycle list and set the current query index
	err := k.RotateQueries(ctx)
	if err != nil {
		return err
	}

	return k.SetAggregatedReport(ctx)
}
