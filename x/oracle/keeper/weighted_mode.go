package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) WeightedMode(ctx sdk.Context, reports []types.MicroReport) (*types.Aggregate, error) {
	if len(reports) == 0 {
		return nil, types.ErrNoReportsToAggregate.Wrapf("can't aggregate empty reports")
	}

	var modeReport types.MicroReport
	var modeReporters []*types.AggregateReporter
	var maxWeight = int64(0)
	var maxFrequency = 0
	var mode string
	frequencyMap := make(map[string]int)

	// populate frequency map
	for _, r := range reports {
		modeReporters = append(modeReporters, &types.AggregateReporter{Reporter: r.Reporter, Power: r.Power})
		entries := r.Power
		for i := int64(0); i < entries; i++ {
			frequencyMap[r.Value]++
		}
	}

	// find the max frequency
	for value, frequency := range frequencyMap {
		if frequency > maxFrequency {
			maxFrequency = frequency
			mode = value
		}
	}

	// set mode report from most powerful reporter who submitted mode value
	var modeReportIndex int64
	for i, r := range reports {
		if mode == r.Value {
			if r.Power > maxWeight {
				maxWeight = r.Power
				modeReport = r
				modeReportIndex = int64(i)
			}
		}

	}

	aggregateReport := types.Aggregate{
		QueryId:              modeReport.QueryId,
		AggregateValue:       modeReport.Value,
		AggregateReporter:    modeReport.Reporter,
		ReporterPower:        modeReport.Power,
		Reporters:            modeReporters,
		AggregateReportIndex: modeReportIndex,
	}

	err := k.SetAggregate(ctx, &aggregateReport)
	if err != nil {
		return nil, err
	}
	return &aggregateReport, nil
}
