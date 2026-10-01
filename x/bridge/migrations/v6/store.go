package v6

import (
	"bytes"
	"context"

	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
)

// lastConsensusBridge is the subset of x/bridge/keeper.Keeper used by this
// backfill. Declared here so this package does not import keeper (cycle with
// keeper/migrations.go).
type lastConsensusBridge interface {
	GetPowerThreshold(ctx context.Context) (uint64, error)
	SetLastConsensusTimestamp(ctx context.Context, queryId []byte, timestamp uint64) error
}

// MigrateStore backfills LastConsensusTimestampByQueryId from the latest
// consensus aggregate per queryId (current PowerThreshold, flagged included).
func MigrateStore(ctx context.Context, bridgeKeeper lastConsensusBridge, oracleKeeper oraclekeeper.Keeper) error {
	threshold, err := bridgeKeeper.GetPowerThreshold(ctx)
	if err != nil {
		return err
	}

	return forEachAggregateQueryId(ctx, oracleKeeper, func(queryId []byte) error {
		ts, found, err := latestConsensusTimestamp(ctx, oracleKeeper, queryId, threshold)
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		return bridgeKeeper.SetLastConsensusTimestamp(ctx, queryId, ts)
	})
}

func forEachAggregateQueryId(ctx context.Context, oracleKeeper oraclekeeper.Keeper, fn func(queryId []byte) error) error {
	var ranger collections.Ranger[collections.Pair[[]byte, uint64]]
	for {
		var queryId []byte
		err := oracleKeeper.Aggregates.Walk(ctx, ranger, func(key collections.Pair[[]byte, uint64], _ oracletypes.Aggregate) (bool, error) {
			queryId = bytes.Clone(key.K1())
			return true, nil
		})
		if err != nil {
			return err
		}
		if queryId == nil {
			return nil
		}
		if err := fn(queryId); err != nil {
			return err
		}
		ranger = new(collections.Range[collections.Pair[[]byte, uint64]]).
			StartExclusive(collections.Join(queryId, ^uint64(0)))
	}
}

func latestConsensusTimestamp(ctx context.Context, oracleKeeper oraclekeeper.Keeper, queryId []byte, threshold uint64) (uint64, bool, error) {
	rng := collections.NewPrefixedPairRange[[]byte, uint64](queryId).Descending()
	var foundTs uint64
	var found bool
	err := oracleKeeper.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value oracletypes.Aggregate) (bool, error) {
		if value.AggregatePower >= threshold {
			foundTs = key.K2()
			found = true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return 0, false, err
	}
	return foundTs, found, nil
}
