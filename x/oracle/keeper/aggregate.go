package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) SetAggregatedReport(ctx sdk.Context) {
	reportsStore := k.ReportsStore(ctx)
	currentHeight := ctx.BlockHeight()

	bz := reportsStore.Get(types.NumKey(currentHeight))
	var revealedReports types.Reports
	k.cdc.Unmarshal(bz, &revealedReports)

	reportMapping := make(map[string][]types.MicroReport)

	// sort by query id
	for _, s := range revealedReports.MicroReports {
		reportMapping[s.QueryId] = append(reportMapping[s.QueryId], *s)
	}

	for _, reports := range reportMapping {
		if reports[0].AggregateMethod == "weighted-median" {
			k.WeightedMedian(ctx, reports)
		}
		if reports[0].AggregateMethod == "weighted-mode" {
			k.WeightedMode(ctx, reports)
		}
	}
}
