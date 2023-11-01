package keeper

import (
	"github.com/tellor-io/layer/x/dispute/types"
)

var _ types.QueryServer = Keeper{}
