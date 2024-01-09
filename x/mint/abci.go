package mint

import (
	"time"

	cosmosmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/mint/keeper"
	"github.com/tellor-io/layer/x/mint/types"
)

// BeginBlocker updates the inflation rate, annual provisions, and then mints
// the block provision for the current block.
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	currentTime := ctx.BlockTime()
	mintBlockProvision(ctx, k, currentTime)
	setPreviousBlockTime(ctx, k, currentTime)
}

// mintBlockProvision mints the block provision for the current block.
func mintBlockProvision(ctx sdk.Context, k keeper.Keeper, currentTime time.Time) {
	minter := k.GetMinter(ctx)
	if minter.PreviousBlockTime == nil {
		return
	}

	toMintCoin, err := minter.CalculateBlockProvision(currentTime, *minter.PreviousBlockTime)
	if err != nil {
		panic(err)
	}
	toMintCoins := sdk.NewCoins(toMintCoin)
	// mint coins double half going to team and half to oracle
	err = k.MintCoins(ctx, toMintCoins.MulInt(cosmosmath.NewInt(2)))
	if err != nil {
		panic(err)
	}

	err = k.SendCoinsToTeam(ctx, toMintCoins)
	if err != nil {
		panic(err)
	}

	err = k.SendCoinsToOracle(ctx, toMintCoins)
	if err != nil {
		panic(err)
	}

}

func setPreviousBlockTime(ctx sdk.Context, k keeper.Keeper, blockTime time.Time) {
	minter := k.GetMinter(ctx)
	minter.PreviousBlockTime = &blockTime
	k.SetMinter(ctx, minter)
}
