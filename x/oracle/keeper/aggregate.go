package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
)

// SetAggregatedReport calculates and allocates rewards to reporters based on aggregated reports.
// at a specific blockchain height (to be ran in begin-blocker)
// It retrieves the revealed reports from the reports store, sorts them by query ID, and then
// calculates the aggregate report for each query using either the weighted-median or weighted-mode method.
// Rewards based on the source are then allocated to the reporters.
func (k Keeper) SetAggregatedReport(ctx sdk.Context) (err error) {
	// Get the current block height of the blockchain.
	currentHeight := ctx.BlockHeight()

	// Retrieve the stored reports for the current block height.
	iter, err := k.Reports.Indexes.BlockHeight.MatchExact(ctx, currentHeight)
	if err != nil {
		return err
	}

	kvs, err := indexes.CollectKeyValues(ctx, k.Reports, iter)
	if err != nil {
		return err
	}

	revealedReportsByQueryID := map[string][]types.MicroReport{}
	for _, kv := range kvs {
		// key is queryId, reporter
		qIdStr := hex.EncodeToString(kv.Key.K1())
		if _, ok := revealedReportsByQueryID[qIdStr]; !ok {
			revealedReportsByQueryID[qIdStr] = []types.MicroReport{kv.Value}
		} else {
			revealedReportsByQueryID[qIdStr] = append(revealedReportsByQueryID[qIdStr], kv.Value)
		}
	}

	// Prepare a list to keep track of reporters who are eligible for tbr.
	reportersToPay := make([]*types.AggregateReporter, 0)
	cycleList, err := k.CycleListAsQueryIds(ctx)
	if err != nil {
		return err
	}
	// Process each set of reports based on their aggregation method.
	for queryIdStr, reports := range revealedReportsByQueryID {
		// Handle weighted-median aggregation method.
		var aggrFunc func(ctx sdk.Context, reports []types.MicroReport) (*types.Aggregate, error)
		if reports[0].AggregateMethod == "weighted-median" {
			// Calculate the aggregated report.
			aggrFunc = k.WeightedMedian
		} else {
			// default to weighted-mode aggregation method.
			// Calculate the aggregated report.
			aggrFunc = k.WeightedMode
		}
		report, err := aggrFunc(ctx, reports)
		if err != nil {
			return err
		}

		// Get the tip for this query.
		queryIdBytes, err := hex.DecodeString(queryIdStr)
		if err != nil {
			return err
		}
		tip := k.GetQueryTip(ctx, queryIdBytes)
		// Allocate rewards if there is a tip.
		if !tip.Amount.IsZero() {
			err = k.AllocateRewards(ctx, report.Reporters, tip, true)
			if err != nil {
				return err
			}
		}
		// Add reporters to the tbr payment list.
		if cycleList[strings.ToLower(queryIdStr)] {
			reportersToPay = append(reportersToPay, report.Reporters...)
		}
	}
	// Process time-based rewards for reporters.
	tbr := k.getTimeBasedRewards(ctx)
	// Allocate time-based rewards to all eligible reporters.
	return k.AllocateRewards(ctx, reportersToPay, tbr, false)
}

func (k Keeper) SetAggregate(ctx sdk.Context, report *types.Aggregate) error {
	queryId, err := utils.QueryIDFromString(report.QueryId)
	if err != nil {
		return err
	}
	nonce, err := k.Nonces.Get(ctx, queryId)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	nonce++
	err = k.Nonces.Set(ctx, queryId, nonce)
	if err != nil {
		return err
	}
	report.Nonce = nonce // TODO: do we want to use int64 for nonce? ***Done switch to uint64***

	currentTimestamp := ctx.BlockTime().Unix()

	return k.Aggregates.Set(ctx, collections.Join(queryId, currentTimestamp), *report)
}

// getDataBefore returns the last aggregate before or at the given timestamp for the given query id.
// TODO: add a test for this function.
func (k Keeper) getDataBefore(ctx sdk.Context, queryId []byte, timestamp time.Time) (*types.Aggregate, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).EndInclusive(timestamp.Unix()).Descending()
	var mostRecent *types.Aggregate
	// This should get us the most recent aggregate, as they are walked in descending order
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		if !value.Flagged {
			mostRecent = &value
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		// why panic here? should we return an error instead?
		panic(err)
	}

	if mostRecent == nil {
		return nil, types.ErrNoAvailableReports.Wrapf("no data before timestamp %v available for query id %s", timestamp, hex.EncodeToString(queryId))
	}

	return mostRecent, nil
}

func (k Keeper) GetCurrentValueForQueryId(ctx context.Context, queryId []byte) (*types.Aggregate, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](queryId).Descending()
	var mostRecent *types.Aggregate
	// This should get us the most recent aggregate, as they are walked in descending order
	err := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
		mostRecent = &value
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return mostRecent, nil
}
