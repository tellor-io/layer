package keeper

import (
	"fmt"
	"runtime/debug"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/mint/types"
)

// InitGenesis initializes the x/mint store with data from the genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, ak types.AccountKeeper, gen *types.GenesisState) {
	minter := types.DefaultMinter()
	minter.BondDenom = gen.BondDenom
	k.SetMinter(ctx, minter)

	// mint initial coins
	k.InitialMint(ctx, gen)

	fmt.Println("Initialized x/mint genesis state")
	debug.PrintStack()
}

// ExportGenesis returns a x/mint GenesisState for the given context.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	bondDenom := k.GetMinter(ctx).BondDenom
	return types.NewGenesisState(bondDenom)
}

func (k Keeper) InitialMint(ctx sdk.Context, gen *types.GenesisState) {
	k.bankKeeper.MintCoins(ctx, types.ModuleName, gen.InitialMint)
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, types.MintToTeam, gen.InitialMint); err != nil {
		panic(err)
	}
}
