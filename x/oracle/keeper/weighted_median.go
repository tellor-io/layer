package keeper

import (
	"fmt"
	"math"
	"math/big"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) WeightedMedian(ctx sdk.Context, reports []types.MicroReport) {
	var medianReport types.Aggregate
	sort.SliceStable(reports, func(i, j int) bool {
		bi := new(big.Int)
		value1, _ := bi.SetString(reports[i].Value, 16)
		value2, _ := bi.SetString(reports[j].Value, 16)
		return value1.Cmp(value2) < 0
	})
	totalReporterPower := int64(0)
	for _, s := range reports {
		totalReporterPower += s.Power
		medianReport.Reporters = append(medianReport.Reporters, &types.AggregateReporter{Reporter: s.Reporter, Power: s.Power})
	}
	halfTotalPower := totalReporterPower / 2

	// Find the weighted median.
	var cumulativePower int64
	for _, s := range reports {
		cumulativePower += s.Power
		if cumulativePower >= halfTotalPower {
			medianReport.ReporterPower = s.Power
			medianReport.AggregateReporter = s.Reporter
			medianReport.AggregateValue = s.Value
			medianReport.QueryId = s.QueryId
			break
		}
	}
	// Calculate weighted mean
	var weightedSum int64
	for _, s := range reports {
		bi := new(big.Int)
		val, _ := bi.SetString(s.Value, 16)
		weightedSum += s.Power * val.Int64()
	}
	weightedMean := float64(weightedSum) / float64(totalReporterPower)

	// Calculate the weighted standard deviation
	var sumWeightedSquaredDiffs float64
	for _, s := range reports {
		bi := new(big.Int)
		val, _ := bi.SetString(s.Value, 16)
		diff := float64(val.Uint64()) - weightedMean
		weightedSquaredDiff := float64(s.Power) * diff * diff
		sumWeightedSquaredDiffs += weightedSquaredDiff
	}
	weightedStdDev := math.Sqrt(sumWeightedSquaredDiffs / float64(totalReporterPower))
	medianReport.StandardDeviation = weightedStdDev
	store := k.AggregateStore(ctx)
	store.Set([]byte(fmt.Sprintf("%s-%d", medianReport.QueryId, ctx.BlockHeight())), k.cdc.MustMarshal(&medianReport))

}
