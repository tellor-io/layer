package keeper

import (
	"github.com/tellor-io/layer/x/reporter/types"
)

type Querier struct {
	Keeper
}

var _ types.QueryServer = Querier{}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}
