package reporter

import (
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	err := k.Params.Set(ctx, genState.Params)
	if err != nil {
		panic(err)
	}
	c := ctx.HeaderInfo().Time
	err = k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: &c,
		Amount:     math.ZeroInt(),
	})
	if err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	params, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Params = params

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
