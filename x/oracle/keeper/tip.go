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

func (k Keeper) GetUserTips(ctx context.Context, tipper sdk.AccAddress) (types.UserTipTotal, error) {
	// it, err := k.Tips.Indexes.Tipper.MatchExact(ctx, tipper.Bytes())
	// if err != nil {
	// 	return types.UserTipTotal{}, err
	// }

	// vals, err := indexes.CollectValues(ctx, k.Tips, it)
	// if err != nil {
	// 	return types.UserTipTotal{}, err
	// }

	// totalTips := math.ZeroInt()
	// for _, tip := range vals {
	// 	totalTips = totalTips.Add(tip)
	// }
	tip, err := k.GetTipsAtBlockForTipper(ctx, sdk.UnwrapSDKContext(ctx).BlockHeight(), tipper)
	if err != nil {
		return types.UserTipTotal{}, err
	}
	return types.UserTipTotal{
		Address: tipper.String(),
		Total:   tip,
	}, nil
}

func (k Keeper) GetTotalTips(ctx context.Context) (math.Int, error) {
	return k.GetTotalTipsAtBlock(ctx, sdk.UnwrapSDKContext(ctx).BlockHeight())
}

// get tips at block
func (k Keeper) GetTipsAtBlockForTipper(ctx context.Context, blockNumber int64, tipper sdk.AccAddress) (math.Int, error) {
	totalTips := math.ZeroInt()
	rng := collections.NewPrefixedPairRange[[]byte, int64](tipper).EndInclusive(blockNumber)
	k.TipperTotal.Walk(ctx, rng, func(k collections.Pair[[]byte, int64], v math.Int) (stop bool, err error) {
		totalTips = v
		return true, nil
	})

	return totalTips, nil
}

func (k Keeper) GetTotalTipsAtBlock(ctx context.Context, blockNumber int64) (math.Int, error) {
	totalTips := math.ZeroInt()
	rng := new(collections.Range[int64]).EndInclusive(blockNumber)
	iter, err := k.TipperTotal.Indexes.BlockNumber.Iterate(ctx, rng)
	if err != nil {
		return math.Int{}, err
	}

	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		pk, err := iter.PrimaryKey()
		if err != nil {
			return math.Int{}, err
		}
		amt, err := k.TipperTotal.Get(ctx, pk)
		if err != nil {
			return math.Int{}, err
		}
		totalTips = totalTips.Add(amt)
	}
	return totalTips, nil
}
