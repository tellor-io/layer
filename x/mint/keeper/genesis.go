package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/mint/types"
)

// InitGenesis initializes the x/mint store with data from the genesis state.
func (k Keeper) InitGenesis(ctx context.Context, ak types.AccountKeeper, gen *types.GenesisState) {
	minter := types.DefaultMinter()
	minter.BondDenom = gen.BondDenom
	err := k.Minter.Set(ctx, minter)
	if err != nil {
		panic(err)
	}
	err = k.InitTbr.Set(ctx, false)
	if err != nil {
		panic(err)
	}
}

// ExportGenesis returns a x/mint GenesisState for the given context.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	minter, err := k.Minter.Get(ctx)
	if err != nil {
		panic(err)
	}
	bondDenom := minter.BondDenom
	return types.NewGenesisState(bondDenom)
}
