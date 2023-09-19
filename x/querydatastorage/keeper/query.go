package keeper

import (
	"layer/x/querydatastorage/types"
)

var _ types.QueryServer = Keeper{}
