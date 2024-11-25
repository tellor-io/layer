package keeper

import (
	"context"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
)

func (k Keeper) WeightedMedian(ctx context.Context, reports []types.MicroReport, metaId uint64) (*types.Aggregate, error) {
	var medianReport types.Aggregate
	// values := make(map[string]cosmomath.LegacyDec)

	// for _, r := range reports {
	// 	val, ok := new(big.Int).SetString(r.Value, 16)
	// 	if !ok {
	// 		k.Logger(ctx).Error("WeightedMedian", "error", "failed to parse value")
	// 		return nil, errors.New("failed to parse value")
	// 	}
	// 	values[r.Reporter] = cosmomath.LegacyNewDecFromBigInt(val)
	// }

	// sort.SliceStable(reports, func(i, j int) bool {
	// 	return values[reports[i].Reporter].BigInt().Cmp(values[reports[j].Reporter].BigInt()) < 0
	// })

	// totalReporterPower := cosmomath.LegacyZeroDec()
	// for _, r := range reports {
	// 	totalReporterPower = totalReporterPower.Add(cosmomath.LegacyNewDec(int64(r.Power)))
	// 	medianReport.Reporters = append(medianReport.Reporters, &types.AggregateReporter{Reporter: r.Reporter, Power: r.Power, BlockNumber: r.BlockNumber})
	// }

	// halfTotalPower := totalReporterPower.Quo(cosmomath.LegacyNewDec(2))
	// cumulativePower := cosmomath.LegacyZeroDec()

	// // Find the weighted median
	// for i, s := range reports {
	// 	cumulativePower = cumulativePower.Add(cosmomath.LegacyNewDec(int64(s.Power)))
	// 	if cumulativePower.BigInt().Cmp(halfTotalPower.BigInt()) >= 0 {
	// 		medianReport.ReporterPower = uint64(totalReporterPower.TruncateInt64())
	// 		medianReport.AggregateReporter = s.Reporter
	// 		medianReport.AggregateValue = median.Value
	// 		medianReport.QueryId = s.QueryId
	// 		medianReport.AggregateReportIndex = uint64(i)
	// 		medianReport.MicroHeight = s.BlockNumber
	// 		medianReport.MetaId = metaId
	// 		break
	// 	}
	// }
	median, err := k.Median.Get(ctx, metaId)
	if err != nil {
		// print error
		fmt.Printf("Error getting median value from store: %v\n", err)
	}
	valuesStored, err := k.Values.Get(ctx, collections.Join(metaId, median.Value))
	if err != nil {
		// print error
		fmt.Printf("Error getting median value from store: %v\n", err)
	}
	tPower, err := k.ValuesWeightSum.Get(ctx, metaId)
	if err != nil {
		// print error
		fmt.Printf("Error getting median value from store: %v\n", err)
	}
	medianReport.AggregateValue = median.Value
	medianReport.AggregateReporter = valuesStored.Report.Reporter
	medianReport.MicroHeight = valuesStored.Report.BlockNumber
	medianReport.QueryId = valuesStored.Report.QueryId
	medianReport.MetaId = metaId
	medianReport.ReporterPower = tPower
	return &medianReport, nil
}
