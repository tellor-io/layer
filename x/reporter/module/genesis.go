package reporter

import (
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	err := k.Params.Set(ctx, genState.Params)
	if err != nil {
		panic(err)
	}
	c := ctx.BlockTime()
	err = k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: &c,
		Amount:     math.ZeroInt(),
	})
	if err != nil {
		panic(err)
	}

	// loop through any reporters found in the genesis file and add them to state
	for _, data := range genState.Reporters {
		err = k.Reporters.Set(ctx, data.ReporterAddress, *data.Reporter)
		if err != nil {
			panic(err)
		}
	}

	// loop through selector tips found in genesis file and add them to state
	for _, data := range genState.SelectorTips {
		err = k.SelectorTips.Set(ctx, data.SelectorAddress, data.Tips)
		if err != nil {
			panic(err)
		}
	}

	// loop through selectors found in genesis file and add them to state
	for _, data := range genState.Selectors {
		err = k.Selectors.Set(ctx, data.SelectorAddress, *data.Selection)
		if err != nil {
			panic(err)
		}
	}

	// loop through DisputedDelegationAmounts found in the genesis file and add them to state
	for _, data := range genState.DisputedDelegationAmounts {
		err = k.DisputedDelegationAmounts.Set(ctx, data.HashId, *data.DelegationAmount)
		if err != nil {
			panic(err)
		}
	}

	// loop through FeePaidFromStake found in the genesis and add to state
	for _, data := range genState.FeePaidFromStake {
		err = k.FeePaidFromStake.Set(ctx, data.HashId, *data.DelegationAmount)
		if err != nil {
			panic(err)
		}
	}

	// loop through any Report data found in genesis and add to state
	for _, data := range genState.Report {
		err = k.Report.Set(ctx, collections.Join(data.QueryId, collections.Join(data.ReporterAddress, data.BlockHeight)), *data.DelegationAmount)
		if err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	params, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Params = params

	iterReporters, err := k.Reporters.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	reporters := make([]*types.ReporterStateEntry, 0)
	for ; iterReporters.Valid(); iterReporters.Next() {
		reporter_addr, err := iterReporters.Key()
		if err != nil {
			panic(err)
		}

		reporter, err := iterReporters.Value()
		if err != nil {
			panic(err)
		}
		reporters = append(reporters, &types.ReporterStateEntry{ReporterAddress: reporter_addr, Reporter: &reporter})
	}
	genesis.Reporters = reporters

	iterSelectorTips, err := k.SelectorTips.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	selectorTips := make([]*types.SelectorTipsStateEntry, 0)
	for ; iterReporters.Valid(); iterReporters.Next() {
		selector_addr, err := iterSelectorTips.Key()
		if err != nil {
			panic(err)
		}

		tips, err := iterSelectorTips.Value()
		if err != nil {
			panic(err)
		}
		selectorTips = append(selectorTips, &types.SelectorTipsStateEntry{SelectorAddress: selector_addr, Tips: tips})
	}
	genesis.SelectorTips = selectorTips

	iterSelectors, err := k.Selectors.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	selectors := make([]*types.SelectorsStateEntry, 0)
	for ; iterSelectors.Valid(); iterSelectors.Next() {
		selector_addr, err := iterSelectors.Key()
		if err != nil {
			panic(err)
		}

		selection, err := iterSelectors.Value()
		if err != nil {
			panic(err)
		}
		selectors = append(selectors, &types.SelectorsStateEntry{SelectorAddress: selector_addr, Selection: &selection})
	}
	genesis.Selectors = selectors

	iterDisputedDelAmt, err := k.DisputedDelegationAmounts.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	disputedDelAmts := make([]*types.DisputedDelegationAmountStateEntry, 0)
	for ; iterDisputedDelAmt.Valid(); iterDisputedDelAmt.Next() {
		disputeHashId, err := iterDisputedDelAmt.Key()
		if err != nil {
			panic(err)
		}

		delAmount, err := iterDisputedDelAmt.Value()
		if err != nil {
			panic(err)
		}
		disputedDelAmts = append(disputedDelAmts, &types.DisputedDelegationAmountStateEntry{HashId: disputeHashId, DelegationAmount: &delAmount})
	}
	genesis.DisputedDelegationAmounts = disputedDelAmts

	iterFeePaidFromStake, err := k.FeePaidFromStake.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	feePaidFromStake := make([]*types.FeePaidFromStakeStateEntry, 0)
	for ; iterFeePaidFromStake.Valid(); iterFeePaidFromStake.Next() {
		disputeHashId, err := iterFeePaidFromStake.Key()
		if err != nil {
			panic(err)
		}

		delAmount, err := iterFeePaidFromStake.Value()
		if err != nil {
			panic(err)
		}
		feePaidFromStake = append(feePaidFromStake, &types.FeePaidFromStakeStateEntry{HashId: disputeHashId, DelegationAmount: &delAmount})
	}
	genesis.FeePaidFromStake = feePaidFromStake

	iterReport, err := k.Report.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	reports := make([]*types.ReportStateEntry, 0)
	currentBlockHeight := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	for ; iterReport.Valid(); iterReport.Next() {
		keys, err := iterReport.Key()
		if err != nil {
			panic(err)
		}

		queryId := keys.K1()
		repAddr := keys.K2().K1()
		blockheight := keys.K2().K2()
		if (uint64(currentBlockHeight) - 1134000) > blockheight {
			continue
		}

		delAmount, err := iterReport.Value()
		if err != nil {
			panic(err)
		}
		reports = append(reports, &types.ReportStateEntry{QueryId: queryId, ReporterAddress: repAddr, BlockHeight: blockheight, DelegationAmount: &delAmount})
	}
	genesis.Report = reports

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
