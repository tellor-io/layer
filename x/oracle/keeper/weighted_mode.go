package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) WeightedMode(ctx sdk.Context, reports []types.MicroReport) {
	if len(reports) == 0 {
		return
	}

	var modeReport types.MicroReport
	var modeReporters []*types.AggregateReporter
	var maxWeight = int64(0)

	for _, r := range reports {
		modeReporters = append(modeReporters, &types.AggregateReporter{Reporter: r.Reporter, Power: r.Power})
		if r.Power > maxWeight {
			maxWeight = r.Power
			modeReport = r
		}
	}

	aggregateReport := types.Aggregate{
		QueryId:           modeReport.QueryId,
		AggregateValue:    modeReport.Value,
		AggregateReporter: modeReport.Reporter,
		ReporterPower:     modeReport.Power,
		Reporters:         modeReporters,
	}

	store := k.AggregateStore(ctx)
	key := []byte(fmt.Sprintf("%s-%d", modeReport.QueryId, ctx.BlockHeight()))
	store.Set(key, k.cdc.MustMarshal(&aggregateReport))

}
