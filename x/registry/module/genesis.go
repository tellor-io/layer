package registry

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

const (
	genQueryTypeSpotPrice     = "spotprice"
	genQueryTypeBridgeDeposit = "trbbridge"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
	k.SetDataSpec(ctx, genQueryTypeSpotPrice, genState.Dataspec)

	// set token bridge spec

	bridgeSpec := types.DataSpec{
		DocumentHash:      "",
		ResponseValueType: "address, string, uint256",
		AbiComponents: []*types.ABIComponent{
			{
				Name:            "toLayer",
				FieldType:       "bool",
				NestedComponent: []*types.ABIComponent{},
			},
			{
				Name:            "depositId",
				FieldType:       "uint256",
				NestedComponent: []*types.ABIComponent{},
			},
		},
		AggregationMethod:  "weighted-mode",
		Registrar:          "genesis",
		ReportBufferWindow: time.Hour * 1,
	}

	k.SetDataSpec(ctx, genQueryTypeBridgeDeposit, bridgeSpec)
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}
	dataspec, err := k.GetSpec(ctx, genQueryTypeSpotPrice)
	if err != nil {
		panic(err)
	}
	genesis := types.DefaultGenesis()
	genesis.Params = params
	genesis.Dataspec = dataspec

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
