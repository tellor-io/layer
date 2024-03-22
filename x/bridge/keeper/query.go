package keeper

import (
	"github.com/tellor-io/layer/x/bridge/types"
)

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

var _ types.QueryServer = Querier{}
