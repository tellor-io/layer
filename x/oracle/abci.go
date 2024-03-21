package oracle

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	sdkctx := sdk.UnwrapSDKContext(ctx)
	// Rotate through the cycle list and set the current query index
	_ = k.RotateQueries(ctx)
	return k.SetAggregatedReport(sdkctx)
}
