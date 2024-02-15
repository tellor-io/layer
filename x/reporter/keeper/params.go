package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/tellor-io/layer/x/reporter/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return params, err
	}
	return params, nil
}

// SetParams set the params
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.Params.Set(ctx, params)
}

func (k Keeper) MinCommissionRate(ctx context.Context) (math.LegacyDec, error) {
	params, err := k.Params.Get(ctx)
	return params.MinCommissionRate, err
}
