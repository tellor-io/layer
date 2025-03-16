package mint

import (
	"context"
	"time"

	"github.com/tellor-io/layer/x/mint/keeper"
	"github.com/tellor-io/layer/x/mint/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlocker updates the inflation rate, annual provisions, and then mints
// the block provision for the current block.
func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	minter, err := k.Minter.Get(ctx)
	if err != nil {
		return err
	}
	if !minter.Initialized {
		return nil
	}

	currentTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	if currentTime.IsZero() {
		// return on invalid block time
		return nil
	}

	if err := MintBlockProvision(ctx, k, currentTime, minter); err != nil {
		return err
	}

	return SetPreviousBlockTime(ctx, k, currentTime)
}

// MintBlockProvision mints the block provision for the current block.
func MintBlockProvision(ctx context.Context, k keeper.Keeper, currentTime time.Time, minter types.Minter) error {
	if minter.PreviousBlockTime == nil {
		return nil
	}
	toMintCoin, err := minter.CalculateBlockProvision(currentTime, *minter.PreviousBlockTime)
	if err != nil {
		return err
	}
	toMintCoins := sdk.NewCoins(toMintCoin)
	// mint tbr coins
	err = k.MintCoins(ctx, toMintCoins)
	if err != nil {
		return err
	}

	return k.SendInflationaryRewards(ctx, toMintCoins)
}

func SetPreviousBlockTime(ctx context.Context, k keeper.Keeper, blockTime time.Time) error {
	minter, err := k.Minter.Get(ctx)
	if err != nil {
		return err
	}
	minter.PreviousBlockTime = &blockTime
	return k.Minter.Set(ctx, minter)
}
