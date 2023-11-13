package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) WeightedMode(ctx sdk.Context, reports []types.MicroReport) {
	var modeReport types.Aggregate
	weightSum := make(map[types.MicroReport]int64)

	for _, r := range reports {
		weightSum[r] += r.Power
		modeReport.Reporters = append(modeReport.Reporters,
			&types.AggregateReporter{Reporter: r.Reporter, Power: r.Power})
	}

	var maxWeight int64

	for report, weight := range weightSum {
		if weight > maxWeight {
			maxWeight = weight
			modeReport.QueryId = report.QueryId
			modeReport.AggregateValue = report.Value
			modeReport.AggregateReporter = report.Reporter
			modeReport.ReporterPower = report.Power
		}
	}

	store := k.AggregateStore(ctx)
	store.Set(
		[]byte(fmt.Sprintf("%s-%d", modeReport.QueryId, ctx.BlockHeight())),
		k.cdc.MustMarshal(&modeReport))
}
