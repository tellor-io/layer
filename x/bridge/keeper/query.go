package keeper

import (
	"github.com/tellor-io/layer/x/bridge/types"
)

type Querier struct {
	k Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{k: keeper}
}

var _ types.QueryServer = Querier{}
