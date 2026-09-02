package v5

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/runtime"
)

// MigrateStore deletes the MaxBatchSize collections.Item (prefix 34).
// Delete is a no-op if governance never set the value.
func MigrateStore(ctx context.Context, storeService store.KVStoreService) error {
	kv := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	kv.Delete(types.DeprecatedMaxBatchSizePrefix)
	return nil
}
