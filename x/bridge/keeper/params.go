package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams()
}
