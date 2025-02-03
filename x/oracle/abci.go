package oracle

import (
	"context"

	// "github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
)

func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	// Rotate through the cycle list and set the current query index
	if err := k.SetAggregatedReport(ctx); err != nil {
		return err
	}
	return k.RotateQueries(ctx)
}
