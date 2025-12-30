package v4

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
)

// MigrateStore performs the v4 migration for the oracle module.
// This migration adds the new LivenessCycles parameter with its default value.
func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	sb := collections.NewSchemaBuilder(storeService)
	paramsItem := collections.NewItem(sb, types.ParamsKeyPrefix(), "params", codec.CollValue[types.Params](cdc))

	// Get existing params
	params, err := paramsItem.Get(ctx)
	if err != nil {
		return err
	}

	if params.LivenessCycles == 0 {
		params.LivenessCycles = types.DefaultLivenessCycles
	}

	// Save updated params
	return paramsItem.Set(ctx, params)
}
