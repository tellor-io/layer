package keeper

import (
	"github.com/tellor-io/layer/x/bridge/types"
)

var _ types.QueryServer = Keeper{}
