package keeper

import (
	"time"

	"github.com/tellor-io/layer/x/reporter/types"
)

func selectorStakeLocked(sel types.Selection, now time.Time) bool {
	return types.SelectorStakeLocked(sel, now)
}
