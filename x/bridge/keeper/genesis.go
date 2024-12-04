package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
)

// InitGenesis initializes the x/bridge store with data from the genesis state.
func (k Keeper) InitGenesis(ctx context.Context, gen *types.GenesisState) {
	err := k.SnapshotLimit.Set(ctx, types.SnapshotLimit{Limit: gen.SnapshotLimit})
	if err != nil {
		panic(err)
	}
}

// ExportGenesis returns a x/bridge GenesisState for the given context.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	snapshotLimit, err := k.SnapshotLimit.Get(ctx)
	if err != nil {
		panic(err)
	}
	return &types.GenesisState{
		SnapshotLimit: snapshotLimit.Limit,
	}
}
