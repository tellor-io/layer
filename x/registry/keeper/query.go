package keeper

import (
	"layer/x/registry/types"
)

var _ types.QueryServer = Keeper{}
