package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) transfer(ctx context.Context, tipper sdk.AccAddress, tip sdk.Coin) (sdk.Coin, error) {
	twoPercent := tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100))
	burnCoin := sdk.NewCoin(tip.Denom, twoPercent)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, tipper, types.ModuleName, sdk.NewCoins(tip)); err != nil {
		return sdk.Coin{}, err
	}
	// burn 2% of tip
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(burnCoin)); err != nil {
		return sdk.Coin{}, err
	}
	return tip.Sub(burnCoin), nil
}

func (k Keeper) GetQueryTip(ctx context.Context, queryId []byte) (math.Int, error) {
	tip, err := k.Query.Get(ctx, queryId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return math.ZeroInt(), nil
		} else {
			return math.Int{}, err
		}
	}
	return tip.Amount, nil
}

func (k Keeper) GetUserTips(ctx context.Context, tipper sdk.AccAddress) (math.Int, error) {
	tip, err := k.GetTipsAtBlockForTipper(ctx, sdk.UnwrapSDKContext(ctx).BlockHeight(), tipper)
	if err != nil {
		return math.Int{}, err
	}
	return tip, nil
}

// get tips at block
func (k Keeper) GetTipsAtBlockForTipper(ctx context.Context, blockNumber int64, tipper sdk.AccAddress) (math.Int, error) {
	totalTips := math.ZeroInt()
	rng := collections.NewPrefixedPairRange[[]byte, int64](tipper).EndInclusive(blockNumber).Descending()
	err := k.TipperTotal.Walk(ctx, rng, func(k collections.Pair[[]byte, int64], v math.Int) (stop bool, err error) {
		totalTips = v
		return true, nil
	})
	if err != nil {
		return math.Int{}, err
	}

	return totalTips, nil
}

func (k Keeper) GetTotalTipsAtBlock(ctx context.Context, blockNumber int64) (math.Int, error) {
	totalTips := math.ZeroInt()
	rng := new(collections.Range[int64]).EndInclusive(blockNumber).Descending()
	err := k.TotalTips.Walk(ctx, rng, func(_ int64, total math.Int) (stop bool, err error) {
		totalTips = total
		return true, nil
	})
	if err != nil {
		return math.Int{}, err
	}
	return totalTips, nil
}

func (k Keeper) AddtoTotalTips(ctx context.Context, amt math.Int) error {
	totalTips, err := k.GetTotalTips(ctx)
	if err != nil {
		return err
	}
	totalTips = totalTips.Add(amt)
	return k.TotalTips.Set(ctx, sdk.UnwrapSDKContext(ctx).BlockHeight(), totalTips)
}

func (k Keeper) GetTotalTips(ctx context.Context) (math.Int, error) {
	totalTips, err := k.GetTotalTipsAtBlock(ctx, sdk.UnwrapSDKContext(ctx).BlockHeight())
	if err != nil {
		return math.Int{}, err
	}
	return totalTips, nil
}

func (k Keeper) AddToTipperTotal(ctx context.Context, tipper sdk.AccAddress, amt math.Int) error {
	totalTips, err := k.GetUserTips(ctx, tipper)
	if err != nil {
		return err
	}
	totalTips = totalTips.Add(amt)
	return k.TipperTotal.Set(ctx, collections.Join(tipper.Bytes(), sdk.UnwrapSDKContext(ctx).BlockHeight()), totalTips)
}
