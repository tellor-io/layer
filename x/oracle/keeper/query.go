package keeper

import (
	"layer/x/oracle/types"
)

var _ types.QueryServer = Keeper{}
