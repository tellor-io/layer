package dispute

import (
	"context"

	"github.com/tellor-io/layer/x/dispute/keeper"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	return k.CheckPrevoteDisputesForExpiration(ctx)
}
