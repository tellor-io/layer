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

func (k Keeper) transfer(ctx sdk.Context, tipper sdk.AccAddress, tip sdk.Coin) (sdk.Coin, error) {
	twoPercent := tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100))
	burnCoin := sdk.NewCoin(tip.Denom, twoPercent)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, tipper, types.ModuleName, sdk.NewCoins(tip)); err != nil {
		return sdk.NewCoin(tip.Denom, math.ZeroInt()), err
	}
	// burn 2% of tip
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(burnCoin)); err != nil {
		return sdk.NewCoin(tip.Denom, math.ZeroInt()), err
	}
	return tip.Sub(burnCoin), nil
}

func (k Keeper) GetQueryTip(ctx sdk.Context, queryId []byte) sdk.Coin {
	tipsSum := sdk.NewCoin(types.DefaultBondDenom, math.ZeroInt())
	rng := collections.NewPrefixedPairRange[[]byte, []byte](queryId) // range all tips for this queryID
	k.Tips.Walk(ctx, rng, func(key collections.Pair[[]byte, []byte], value math.Int) (stop bool, err error) {
		tipsSum = tipsSum.AddAmount(value)
		return false, nil
	})
	return tipsSum
}

func (k Keeper) GetUserTips(ctx context.Context, tipper sdk.AccAddress) types.UserTipTotal {
	it, err := k.Tips.Indexes.Tipper.MatchExact(ctx, tipper.Bytes())
	if err != nil {
		panic(err)
	}

	vals, err := indexes.CollectValues(ctx, k.Tips, it)
	if err != nil {
		panic(err)
	}

	totalTips := sdk.NewCoin(types.DefaultBondDenom, math.ZeroInt())
	for _, tip := range vals {
		totalTips = totalTips.AddAmount(tip)
	}

	return types.UserTipTotal{
		Address: tipper.String(),
		Total:   totalTips,
	}
}

func (k Keeper) GetTotalTips(ctx context.Context) sdk.Coin {
	// TODO: handle this error correctly
	totalTips, err := k.TotalTips.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return sdk.NewCoin(types.DefaultBondDenom, math.ZeroInt())
	}

	return sdk.NewCoin(types.DefaultBondDenom, totalTips)
}
