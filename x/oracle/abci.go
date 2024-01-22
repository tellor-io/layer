package oracle

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/keeper"
)

func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	// Rotate through the cycle list and set the current query index
	k.RotateQueries(ctx)
}
