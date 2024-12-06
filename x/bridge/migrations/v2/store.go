package v2

import (
	"context"
	"encoding/json"

	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/core/store"
)

type SnapshotLimit struct {
	Limit uint64 `protobuf:"varint,1,opt,name=limit,proto3"`
}

func MigrateStoreFromV1ToV2(ctx context.Context, storeService store.KVStoreService) error {
	kvStore := storeService.OpenKVStore(ctx)

	limit := bridgetypes.SnapshotLimit{Limit: 1000}
	data, err := json.Marshal(limit)
	if err != nil {
		return err
	}

	key := []byte("SnapshotLimit")
	err = kvStore.Set(key, data)
	if err != nil {
		return err
	}

	return nil
}
