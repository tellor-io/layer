package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/mint/types"
)

// InitGenesis initializes the x/mint store with data from the genesis state.
func (k Keeper) InitGenesis(ctx context.Context, ak types.AccountKeeper, gen *types.GenesisState) {
	minter := types.DefaultMinter()
	minter.BondDenom = gen.BondDenom
	minter.Initialized = gen.Initialized
	minter.PreviousBlockTime = gen.PreviousBlockTime
	err := k.Minter.Set(ctx, minter)
	if err != nil {
		panic(err)
	}
	k.accountKeeper.GetModuleAccount(ctx, types.TimeBasedRewards)
}

// ExportGenesis returns a x/mint GenesisState for the given context.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	k.Logger(ctx).Info("Exporting genesis from mint module")
	minter, err := k.Minter.Get(ctx)
	if err != nil {
		panic(err)
	}
	bondDenom := minter.BondDenom
	initialized := minter.Initialized
	previousBlockTime := minter.PreviousBlockTime
	k.Logger(ctx).Info("Finished exporting from mint module")
	return types.NewGenesisState(bondDenom, initialized, previousBlockTime)
}
