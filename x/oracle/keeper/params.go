package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"
)

// SetParams sets the x/oracle module parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	return k.Params.Set(ctx, params)
}

// GetParams sets the x/oracle module parameters.
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	return k.Params.Get(ctx)
}
