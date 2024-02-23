package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/oracle/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"
)

// SetParams sets the x/oracle module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	for i, query := range params.CycleList {
		query = regtypes.Remove0xPrefix(query)
		params.CycleList[i] = query
	}
	// initially set the cycle index to 0
	if err := k.CycleIndex.Set(ctx, 0); err != nil {
		return err
	}
	return k.Params.Set(ctx, params)
}

// GetParams sets the x/oracle module parameters.
func (k Keeper) GetParams(ctx sdk.Context) (types.Params, error) {
	return k.Params.Get(ctx)
}
