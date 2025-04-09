package registry

import (
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	genQueryTypeSpotPrice     = "spotprice"
	genQueryTypeBridgeDeposit = "trbbridge"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
	for _, dataspec := range genState.Dataspec {
		if err := k.SetDataSpec(ctx, dataspec.QueryType, dataspec); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	k.Logger(ctx).Info("Exporting genesis from registry module")
	genesis := types.DefaultGenesis()

	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Params = params

	dataspecs, err := k.GetAllDataSpecs(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Dataspec = dataspecs

	// this line is used by starport scaffolding # genesis/module/export
	k.Logger(ctx).Info("Finished exporting from registry module")
	return genesis
}
