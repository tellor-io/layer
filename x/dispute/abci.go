package dispute

import (
	"context"

	"github.com/tellor-io/layer/x/dispute/keeper"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	ids, err := k.CheckPrevoteDisputesForExpiration(ctx)
	if err != nil {
		return err
	}
	return k.ExecuteVotes(ctx, ids)
}
