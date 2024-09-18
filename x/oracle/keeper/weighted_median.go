package keeper

import (
	"context"
	"errors"
	"math/big"
	"sort"

	cosmomath "cosmossdk.io/math"
	"github.com/tellor-io/layer/x/oracle/types"
)

const FixedPointPrecision = 1e18

func (k Keeper) WeightedMedian(ctx context.Context, reports []types.MicroReport) (*types.Aggregate, error) {
	var medianReport types.Aggregate
	values := make(map[string]cosmomath.Uint)

	for _, r := range reports {
		val, ok := new(big.Int).SetString(r.Value, 16)
		if !ok {
			k.Logger(ctx).Error("WeightedMedian", "error", "failed to parse value")
			return nil, errors.New("failed to parse value")
		}
		values[r.Reporter] = cosmomath.NewUint(val.Uint64())
	}

	sort.SliceStable(reports, func(i, j int) bool {
		return values[reports[i].Reporter].LT(values[reports[j].Reporter])
	})

	totalReporterPower, weightedSum := cosmomath.Uint{}, cosmomath.Uint{}
	for _, r := range reports {
		weightedSum = weightedSum.Add(values[r.Reporter].Mul(cosmomath.NewUint(uint64(r.Power))))
		totalReporterPower = totalReporterPower.Add(cosmomath.NewUint(uint64(r.Power)))
		medianReport.Reporters = append(medianReport.Reporters, &types.AggregateReporter{Reporter: r.Reporter, Power: r.Power, BlockNumber: r.BlockNumber})
	}

	halfTotalPower := totalReporterPower.Quo(cosmomath.NewUint(2))
	cumulativePower := cosmomath.Uint{}

	// Find the weighted median
	for i, s := range reports {
		cumulativePower = cumulativePower.Add(cosmomath.NewUint(uint64(s.Power)))
		if cumulativePower.GTE(halfTotalPower) {
			medianReport.ReporterPower = int64(totalReporterPower.Uint64())
			medianReport.AggregateReporter = s.Reporter
			medianReport.AggregateValue = s.Value
			medianReport.QueryId = s.QueryId
			medianReport.AggregateReportIndex = int64(i)
			medianReport.MicroHeight = s.BlockNumber
			break
		}
	}

	// Calculate the weighted standard deviation
	sumWeightedSquaredDiffs := cosmomath.Uint{}
	weightedMean := weightedSum.Quo(totalReporterPower)

	for _, r := range reports {
		diff := cosmomath.Uint{}
		if values[r.Reporter].GT(weightedMean) {
			diff = values[r.Reporter].Sub(weightedMean)
		} else {
			diff = weightedMean.Sub(values[r.Reporter])
		}
		diffSquared := diff.Mul(diff)
		weightedSquaredDiff := diffSquared.Mul(cosmomath.NewUint(uint64(r.Power)))
		sumWeightedSquaredDiffs = sumWeightedSquaredDiffs.Add(weightedSquaredDiff)
	}

	weightedVariance := sumWeightedSquaredDiffs.Quo(totalReporterPower)
	wstdDeviation := new(big.Int).Sqrt(weightedVariance.BigInt())
	medianReport.StandardDeviation = wstdDeviation.String()

	err := k.SetAggregate(ctx, &medianReport)
	if err != nil {
		return nil, err
	}
	return &medianReport, nil
}
