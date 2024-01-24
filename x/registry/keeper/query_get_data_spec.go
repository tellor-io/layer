package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
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
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.SpecRegistryKey)

	return store.Get([]byte(queryType))
}
