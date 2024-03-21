package keeper

import (
	"context"

	"cosmossdk.io/math"
)

func (k Keeper) MinCommissionRate(ctx context.Context) (math.LegacyDec, error) {
	params, err := k.Params.Get(ctx)
	return params.MinCommissionRate, err
}
