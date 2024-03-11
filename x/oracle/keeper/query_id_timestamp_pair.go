package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) SetQueryIdAndTimestampPairByBlockHeight(ctx sdk.Context, queryId string, timestamp time.Time) {
	k.Logger(ctx).Info("@SetQueryIdAndTimestampPairByBlockHeight", "queryId", queryId, "timestamp", timestamp)
	store := ctx.KVStore(k.storeKey)
	height := ctx.BlockHeight()
	key := types.QueryIdTimestampPairsByBlockHeightKey(height)
	var pairs types.QueryIdTimestampPairsArray
	if bz := store.Get(key); bz != nil {
		k.cdc.MustUnmarshal(bz, &pairs)
	}
	pairs.Pairs = append(pairs.Pairs, &types.QueryIdTimestampPair{QueryId: queryId, Timestamp: timestamp.Unix()})
	store.Set(key, k.cdc.MustMarshal(&pairs))
}

func (k Keeper) GetQueryIdAndTimestampPairsByBlockHeight(ctx sdk.Context, height uint64) types.QueryIdTimestampPairsArray {
	store := ctx.KVStore(k.storeKey)
	key := types.QueryIdTimestampPairsByBlockHeightKey(int64(height))
	var pairs types.QueryIdTimestampPairsArray
	if bz := store.Get(key); bz != nil {
		k.cdc.MustUnmarshal(bz, &pairs)
	}
	return pairs
}
