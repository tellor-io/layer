package v6

import (
	"context"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
)

// MigrateStore backfills CurrentCycleListQuery from the pre-v6 getter:
// GetCyclelist()[CyclelistSequencer.Peek()], wrapping to q[0] when Peek is OOB.
// The sequencer is left unchanged. Does not walk the Query store.
func MigrateStore(ctx context.Context, storeService store.KVStoreService) error {
	sb := collections.NewSchemaBuilder(storeService)
	cyclelist := collections.NewMap(sb, types.CyclelistPrefix, "cyclelist", collections.BytesKey, collections.BytesValue)
	cyclelistSequencer := collections.NewSequence(sb, types.CycleSeqPrefix, "cycle_sequencer")
	current := collections.NewItem(sb, types.CurrentCycleListQueryPrefix, "current_cycle_list_query", collections.BytesValue)

	idx, err := cyclelistSequencer.Peek(ctx)
	if err != nil {
		return err
	}

	iter, err := cyclelist.Iterate(ctx, nil)
	if err != nil {
		return err
	}
	q, err := iter.Values()
	if err != nil {
		iter.Close()
		return err
	}
	if err := iter.Close(); err != nil {
		return err
	}
	if len(q) == 0 {
		return fmt.Errorf("cycle list is empty")
	}

	data := q[0]
	if idx < uint64(len(q)) {
		data = q[idx]
	}
	return current.Set(ctx, data)
}
