package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/oracle/types"
	rk "github.com/tellor-io/layer/x/registry/keeper"
)

// SetParams sets the x/oracle module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	for i, query := range params.CycleList {
		if rk.Has0xPrefix(query) {
			query = query[2:]
		}
		params.CycleList[i] = query
	}
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKeyPrefix(), bz)

	return nil
}

// GetParams sets the x/oracle module parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKeyPrefix())
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}
