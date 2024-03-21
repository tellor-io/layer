package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"

	rtypes "github.com/tellor-io/layer/x/registry/types"
)

var _ types.RegistryHooks = Hooks{}

// Hooks wrapper struct for oracle keeper
type Hooks struct {
	k Keeper
}

// Return the oracle hooks
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

func (h Hooks) AfterDataSpecUpdated(ctx context.Context, querytype string, dataspec rtypes.DataSpec) error {
	return h.k.UpdateQuery(ctx, querytype, dataspec.ReportBufferWindow)
}
