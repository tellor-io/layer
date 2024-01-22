package oracle

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/keeper"
)

/*
Get previous block height and fetch the queries that were tipped in the previous block.
Then, add the next query from the cycle list to the tipped queries.  Doesn't necessarily mean it has
but that it is eligible to be submitted ie you can commit a report for this query and as result able to submit
a value for this query.
Also, delete the queries that were set in the block before the previous block.
*/
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	previousBlockHeight := ctx.BlockHeight() - 1
	blockTips := k.GetBlockTips(ctx, previousBlockHeight)
	nextQuery := k.RotateQueries(ctx)
	blockTips.Tips[nextQuery] = true
	k.SetBlockTips(ctx, previousBlockHeight, blockTips)
	k.DeletePreviousBlockTips(ctx, previousBlockHeight-1)
}
