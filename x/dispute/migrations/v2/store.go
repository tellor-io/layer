package v2

import (
	"context"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/x/dispute/types"
)

func MigrateStore(ctx context.Context, disputesMap *collections.IndexedMap[uint64, types.Dispute, types.DisputesIndex]) error {
	// The keys are just the uint64 dispute IDs encoded
	for id := uint64(2); id <= 3; id++ {
		dispute, err := disputesMap.Get(ctx, id)
		if err != nil {
			return err
		}

		dispute.Open = false
		err = disputesMap.Set(ctx, id, dispute)
		if err != nil {
			return err
		}
	}

	return nil
}
