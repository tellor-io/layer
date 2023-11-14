package keeper

import (
	"encoding/hex"
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) transfer(ctx sdk.Context, tipper sdk.AccAddress, tip sdk.Coin) (sdk.Coin, error) {
	twoPercent := tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100))
	burnCoin := sdk.NewCoin(tip.Denom, twoPercent)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, tipper, types.ModuleName, sdk.NewCoins(tip)); err != nil {
		return sdk.NewCoin(tip.Denom, sdk.ZeroInt()), err
	}
	// burn 2% of tip
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(burnCoin)); err != nil {
		return sdk.NewCoin(tip.Denom, sdk.ZeroInt()), err
	}
	return tip.Sub(burnCoin), nil
}

func (k Keeper) SetTip(ctx sdk.Context, tipper sdk.AccAddress, queryData string, tip sdk.Coin) {
	tipStore := k.TipStore(ctx)
	k.SetQueryTips(ctx, tipStore, queryData, tip)
	k.SetTipperTipsForQuery(ctx, tipStore, tipper.String(), queryData, tip)
	k.SetTipperTotalTips(ctx, tipStore, tipper, tip)
}

func (k Keeper) SetQueryTips(ctx sdk.Context, tipStore storetypes.KVStore, queryData string, tip sdk.Coin) {
	tips, queryId := k.GetQueryTips(ctx, tipStore, queryData)
	tips.Amount = tips.Amount.Add(tip)
	tips.TotalTips = tips.TotalTips.Add(tip)
	tipStore.Set(queryId, k.cdc.MustMarshal(&tips))
}

func (k Keeper) SetTipperTipsForQuery(ctx sdk.Context, tipStore sdk.KVStore, tipper, queryData string, tip sdk.Coin) {
	tips := k.GetUserQueryTips(ctx, tipStore, tipper, queryData)
	tips.Total = tips.Total.Add(tip)
	tipStore.Set(k.TipperKey(tipper, queryData), k.cdc.MustMarshal(&tips))

}

func (k Keeper) SetTipperTotalTips(ctx sdk.Context, tipStore sdk.KVStore, tipper sdk.AccAddress, tip sdk.Coin) {
	tips := k.GetUserTips(ctx, tipStore, tipper)
	tips.Total = tips.Total.Add(tip)
	tipStore.Set(tipper, k.cdc.MustMarshal(&tips))
}

func (k Keeper) GetQueryTips(ctx sdk.Context, tipStore sdk.KVStore, queryData string) (types.Tips, []byte) {
	// decode query data hex string to bytes
	queryDataBytes, err := hex.DecodeString(queryData)
	if err != nil {
		panic(err)
	}
	queryId := HashQueryData(queryDataBytes)
	bz := tipStore.Get(queryId)
	if bz == nil {
		return types.Tips{
			QueryData: queryData,
			Amount:    sdk.NewCoin(sdk.DefaultBondDenom, sdk.ZeroInt()),
			TotalTips: sdk.NewCoin(sdk.DefaultBondDenom, sdk.ZeroInt()),
		}, queryId
	}
	var tips types.Tips
	k.cdc.Unmarshal(bz, &tips)
	return tips, queryId
}

func (k Keeper) GetUserTips(ctx sdk.Context, tipStore sdk.KVStore, tipper sdk.AccAddress) types.UserTipTotal {
	bz := tipStore.Get(tipper)
	if bz == nil {
		return types.UserTipTotal{
			Address: tipper.String(),
			Total:   sdk.NewCoin(sdk.DefaultBondDenom, sdk.ZeroInt()),
		}
	}
	var tips types.UserTipTotal
	k.cdc.Unmarshal(bz, &tips)
	return tips
}

func (k Keeper) GetUserQueryTips(ctx sdk.Context, tipStore sdk.KVStore, tipper, queryData string) (tips types.UserTipTotal) {
	bz := tipStore.Get(k.TipperKey(tipper, queryData))
	if bz == nil {
		return types.UserTipTotal{
			Address: tipper,
			Total:   sdk.NewCoin(sdk.DefaultBondDenom, sdk.ZeroInt()),
		}
	}
	k.cdc.Unmarshal(bz, &tips)
	return
}

func (k Keeper) TipperKey(tipper, queryData string) []byte {
	return types.KeyPrefix(fmt.Sprintf("%s:%s", tipper, queryData))
}
