package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) WeightedMode(ctx context.Context, reports []types.MicroReport, metaId uint64) (*types.Aggregate, error) {
	if len(reports) == 0 {
		return nil, types.ErrNoReportsToAggregate.Wrapf("can't aggregate empty reports")
	}

	var modeReport types.MicroReport
	var totalReporterPower uint64

	var maxFrequency uint64
	var maxWeight uint64
	// populate frequency map
	frequencyMap := make(map[string]uint64)
	for _, r := range reports {
		frequencyMap[r.Value] += r.Power
		totalReporterPower += r.Power
		if frequencyMap[r.Value] > maxFrequency {
			maxFrequency = frequencyMap[r.Value]
			if r.Power > maxWeight {
				maxWeight = r.Power
				modeReport = r
			}
		}
	}

	aggregateReport := types.Aggregate{
		QueryId:           modeReport.QueryId,
		AggregateValue:    modeReport.Value,
		AggregateReporter: modeReport.Reporter,
		AggregatePower:    totalReporterPower,
		MicroHeight:       modeReport.BlockNumber,
		MetaId:            metaId,
	}

	return &aggregateReport, nil
}
