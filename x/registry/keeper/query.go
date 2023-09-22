package keeper

import (
	"github.com/tellor-io/layer/x/registry/types"
)

var _ types.QueryServer = Keeper{}
