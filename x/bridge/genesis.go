package bridge

import (
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.Params.Set(ctx, genState.Params); err != nil {
		panic(err)
	}
	if err := k.SnapshotLimit.Set(ctx, types.SnapshotLimit{Limit: genState.SnapshotLimit}); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	var err error
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}
	snapshotLimit, err := k.SnapshotLimit.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.SnapshotLimit = snapshotLimit.Limit

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
