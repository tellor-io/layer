package keeper

import (
	"encoding/hex"
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) transfer(ctx sdk.Context, tipper sdk.AccAddress, tip sdk.Coin) error {
	twoPercent := tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100))
	burnCoin := sdk.NewCoin(tip.Denom, twoPercent)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, tipper, types.ModuleName, sdk.NewCoins(tip)); err != nil {
		return err
	}
	// burn 2% of tip
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(burnCoin)); err != nil {
		return err
	}
	return nil
}

func (k Keeper) SetTip(ctx sdk.Context, tipper sdk.AccAddress, queryData string, tip sdk.Coin) {
	tipStore := k.TipStore(ctx)
	k.SetQueryTips(ctx, tipStore, queryData, tip)
	k.SetTipperTipsForQuery(ctx, tipStore, tipper.String(), queryData, tip)
}

func (k Keeper) SetQueryTips(ctx sdk.Context, tipStore storetypes.KVStore, queryData string, tip sdk.Coin) {
	tips, queryId := k.GetQueryTips(ctx, tipStore, queryData)
	tips.Amount = tips.Amount.Add(tip)
	tips.TotalTips = tips.TotalTips.Add(tips.Amount)
	tipStore.Set(queryId, k.cdc.MustMarshal(&tips))
}

func (k Keeper) SetTipperTipsForQuery(ctx sdk.Context, tipStore sdk.KVStore, tipper, queryData string, tip sdk.Coin) {
	tips := k.GetUserTips(ctx, tipStore, tipper, queryData)
	tips.TotalTipped = tips.TotalTipped.Add(tip)
	tipStore.Set(k.TipperKey(tipper, queryData), k.cdc.MustMarshal(&tips))

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

func (k Keeper) GetUserTips(ctx sdk.Context, tipStore sdk.KVStore, tipper, queryData string) (tips types.TipsByTipper) {
	bz := tipStore.Get(k.TipperKey(tipper, queryData))
	if bz == nil {
		return types.TipsByTipper{
			Tipper:      tipper,
			TotalTipped: sdk.NewCoin(sdk.DefaultBondDenom, sdk.ZeroInt()),
		}
	}
	k.cdc.Unmarshal(bz, &tips)
	return
}

func (k Keeper) TipperKey(tipper, queryData string) []byte {
	return types.KeyPrefix(fmt.Sprintf("%s:%s", tipper, queryData))
}
