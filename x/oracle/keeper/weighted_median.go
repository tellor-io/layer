package keeper

import (
	"context"
	"errors"
	"math/big"
	"sort"

	"github.com/tellor-io/layer/x/oracle/types"

	cosmomath "cosmossdk.io/math"
)

func (k Keeper) WeightedMedian(ctx context.Context, reports []types.MicroReport, metaId uint64) (*types.Aggregate, error) {
	var medianReport types.Aggregate
	values := make(map[string]cosmomath.LegacyDec)

	for _, r := range reports {
		val, ok := new(big.Int).SetString(r.Value, 16)
		if !ok {
			k.Logger(ctx).Error("WeightedMedian", "error", "failed to parse value")
			return nil, errors.New("failed to parse value")
		}
		values[r.Reporter] = cosmomath.LegacyNewDecFromBigInt(val)
	}

	sort.SliceStable(reports, func(i, j int) bool {
		return values[reports[i].Reporter].BigInt().Cmp(values[reports[j].Reporter].BigInt()) < 0
	})

	totalReporterPower := cosmomath.LegacyZeroDec()
	for _, r := range reports {
		totalReporterPower = totalReporterPower.Add(cosmomath.LegacyNewDec(int64(r.Power)))
	}

	halfTotalPower := totalReporterPower.Quo(cosmomath.LegacyNewDec(2))
	cumulativePower := cosmomath.LegacyZeroDec()

	// Find the weighted median
	for _, s := range reports {
		cumulativePower = cumulativePower.Add(cosmomath.LegacyNewDec(int64(s.Power)))
		if cumulativePower.BigInt().Cmp(halfTotalPower.BigInt()) >= 0 {
			medianReport.AggregatePower = uint64(totalReporterPower.TruncateInt64())
			medianReport.AggregateReporter = s.Reporter
			medianReport.AggregateValue = s.Value
			medianReport.QueryId = s.QueryId
			medianReport.MicroHeight = s.BlockNumber
			medianReport.MetaId = metaId
			break
		}
	}

	return &medianReport, nil
}
