package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
)

// returns a no stake report by query id and timestamp
func (k Keeper) GetNoStakeReportByQueryIdTimestamp(ctx context.Context, queryId []byte, timestamp uint64) (*types.NoStakeMicroReport, error) {
	report, err := k.NoStakeReports.Get(ctx, collections.Join(queryId, timestamp))
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// returns all no stake reports submitted by an address
func (k Keeper) GetNoStakeReportsByReporter(ctx context.Context, reporter string) ([]*types.NoStakeMicroReport, error) {
	iter, err := k.NoStakeReports.Indexes.Reporter.MatchExact(ctx, reporter)
	if err != nil {
		return nil, err
	}
	reports := make([]*types.NoStakeMicroReport, 0)
	for iter.Valid() {
		pk, err := iter.PrimaryKey()
		if err != nil {
			return nil, err
		}
		report, err := k.NoStakeReports.Get(ctx, pk)
		if err != nil {
			return nil, err
		}
		reports = append(reports, &report)
		iter.Next()
	}
	return reports, nil
}
