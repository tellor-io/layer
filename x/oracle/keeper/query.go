package keeper

import (
	"github.com/tellor-io/layer/x/oracle/types"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	keeper Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{keeper: keeper}
}
