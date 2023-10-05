package keeper

import (
	"github.com/tellor-io/layer/x/oracle/types"
)

var _ types.QueryServer = Keeper{}
