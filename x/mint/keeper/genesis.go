package keeper

import (
	"github.com/tellor-io/layer/x/mint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the x/mint store with data from the genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, ak types.AccountKeeper, gen *types.GenesisState) {
	minter := types.DefaultMinter()
	minter.BondDenom = gen.BondDenom
	k.SetMinter(ctx, minter)
}

// ExportGenesis returns a x/mint GenesisState for the given context.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	bondDenom := k.GetMinter(ctx).BondDenom
	return types.NewGenesisState(bondDenom)
}
