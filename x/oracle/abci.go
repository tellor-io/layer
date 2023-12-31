package oracle

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	previousBlockHeight := ctx.BlockHeight() - 1
	blockTips := k.GetBlockTips(ctx, previousBlockHeight)

	if len(blockTips.Tips) == 0 {
		nextQuery := k.RotateQueries(ctx)
		blockTips = types.BlockTips{
			Tips: map[string]bool{
				nextQuery: true,
			},
		}
		k.SetBlockTips(ctx, previousBlockHeight, blockTips)
		k.DeletePreviousBlockTips(ctx, previousBlockHeight-1)
	}
}
