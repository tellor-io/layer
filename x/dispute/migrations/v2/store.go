package v2

import (
	"context"
	"fmt"

	disputetypes "github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
)

func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	disputeStore := prefix.NewStore(store, disputetypes.DisputesPrefix)

	// The keys are just the uint64 dispute IDs encoded
	for id := uint64(2); id <= 3; id++ {
		key := make([]byte, 8)
		_, err := collections.Uint64Key.Encode(key, id)
		if err != nil {
			return err
		}
		if disputeBytes := disputeStore.Get(key); disputeBytes != nil {
			var dispute disputetypes.Dispute
			if err := cdc.Unmarshal(disputeBytes, &dispute); err != nil {
				return err
			}
			dispute.Open = false
			newData, err := cdc.Marshal(&dispute)
			if err != nil {
				return err
			}
			disputeStore.Set(key, newData)
		} else {
			return fmt.Errorf("dispute %d not found", id)
		}
	}

	return nil
}
