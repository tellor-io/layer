package keeper

import (
	"context"

	"layer/x/registry/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetDataSpec(goCtx context.Context, req *types.QueryGetDataSpecRequest) (*types.QueryGetDataSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	dataSpecBytes := k.Spec(ctx, req.QueryType)
	var dataSpec types.DataSpec
	k.cdc.Unmarshal(dataSpecBytes, &dataSpec)

	return &types.QueryGetDataSpecResponse{Spec: &dataSpec}, nil
}

func (k Keeper) Spec(ctx sdk.Context, queryType string) []byte {

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecRegistryKey))

	return store.Get([]byte(queryType))
}
