package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
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

func (k Keeper) GetUserTips(ctx context.Context, tipper sdk.AccAddress) (types.UserTipTotal, error) {
	it, err := k.Tips.Indexes.Tipper.MatchExact(ctx, tipper.Bytes())
	if err != nil {
		return types.UserTipTotal{}, err
	}

	vals, err := indexes.CollectValues(ctx, k.Tips, it)
	if err != nil {
		return types.UserTipTotal{}, err
	}

	totalTips := math.ZeroInt()
	for _, tip := range vals {
		totalTips = totalTips.Add(tip)
	}

	return types.UserTipTotal{
		Address: tipper.String(),
		Total:   totalTips,
	}, nil
}

func (k Keeper) GetTotalTips(ctx context.Context) (math.Int, error) {
	totalTips, err := k.TotalTips.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return math.ZeroInt(), nil
		}
		return math.Int{}, err
	}

	return totalTips, nil
}

// Add to overall total tips, used for dispute voting calculation
func (k Keeper) AddtoTotalTips(ctx context.Context, tip math.Int) error {
	totalTips, err := k.TotalTips.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return k.TotalTips.Set(ctx, tip)
		} else {
			return err
		}
	}

	totalTips = totalTips.Add(tip)
	return k.TotalTips.Set(ctx, totalTips)
}
