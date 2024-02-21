package keeper

import (
	"errors"
	"math"
	"math/big"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) WeightedMedian(ctx sdk.Context, reports []types.MicroReport) (*types.Aggregate, error) {
	var medianReport types.Aggregate
	values := make(map[string]*big.Int)

	for _, r := range reports {
		val, ok := new(big.Int).SetString(r.Value, 16)
		if !ok {
			ctx.Logger().Error("WeightedMedian", "error", "failed to parse value")
			return nil, errors.New("failed to parse value")
		}
		values[r.Reporter] = val
	}

	sort.SliceStable(reports, func(i, j int) bool {
		return values[reports[i].Reporter].Cmp(values[reports[j].Reporter]) < 0

	})

	var totalReporterPower, weightedSum big.Int
	for _, r := range reports {
		weightedSum.Add(&weightedSum, new(big.Int).Mul(values[r.Reporter], big.NewInt(r.Power)))
		totalReporterPower.Add(&totalReporterPower, big.NewInt(r.Power))
		medianReport.Reporters = append(medianReport.Reporters, &types.AggregateReporter{Reporter: r.Reporter, Power: r.Power})
	}

	halfTotalPower := new(big.Int).Div(&totalReporterPower, big.NewInt(2))
	cumulativePower := new(big.Int)

	// Find the weighted median
	for i, s := range reports {
		cumulativePower.Add(cumulativePower, big.NewInt(s.Power))
		if cumulativePower.Cmp(halfTotalPower) >= 0 {
			medianReport.ReporterPower = s.Power
			medianReport.AggregateReporter = s.Reporter
			medianReport.AggregateValue = s.Value
			medianReport.QueryId = s.QueryId
			medianReport.AggregateReportIndex = int64(i)
			break
		}
	}

	// Calculate the weighted standard deviation
	var sumWeightedSquaredDiffs float64
	weightedMean := new(big.Float).Quo(new(big.Float).SetInt(&weightedSum), new(big.Float).SetInt(&totalReporterPower))

	for _, r := range reports {
		valBigFloat := new(big.Float).SetInt(values[r.Reporter])
		diff := new(big.Float).Sub(valBigFloat, weightedMean)
		diffSquared := new(big.Float).Mul(diff, diff)
		weightedSquaredDiff := diffSquared.Mul(diffSquared, new(big.Float).SetInt64(r.Power))
		diffStdDev, _ := weightedSquaredDiff.Float64()
		sumWeightedSquaredDiffs += diffStdDev
	}
	weightedStdDev := math.Sqrt(sumWeightedSquaredDiffs / float64(totalReporterPower.Int64()))
	medianReport.StandardDeviation = weightedStdDev

	k.SetAggregate(ctx, &medianReport)
	return &medianReport, nil
}
