package oracle

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	err := k.SetParams(ctx, genState.Params)
	if err != nil {
		panic(err)
	}
	err = k.GenesisCycleList(ctx, genState.Cyclelist)
	if err != nil {
		panic(err)
	}
	err = k.SetQueryDataLimit(ctx, genState.QueryDataLimit)
	if err != nil {
		panic(err)
	}
	// initialize TipperTotals from genesis state
	for _, data := range genState.TipperTotal {
		err = k.TipperTotal.Set(ctx, collections.Join(data.TipperAddr, data.BlockHeight), data.TipAmount)
		if err != nil {
			panic(err)
		}
	}

	// initialize total tips from genesis state
	for _, data := range genState.TotalTips {
		err = k.TotalTips.Set(ctx, data.BlockHeight, data.TipAmount)
		if err != nil {
			panic(err)
		}
	}

	// initialize Nonces from genesis state
	for _, data := range genState.Nonces {
		err = k.Nonces.Set(ctx, data.QueryId, data.Nonce)
		if err != nil {
			panic(err)
		}
	}

	// initialize reports from genesis state
	for _, data := range genState.Reports {
		repAcc, err := sdk.AccAddressFromBech32(data.Reporter)
		if err != nil {
			panic(err)
		}
		err = k.Reports.Set(ctx, collections.Join3(data.QueryId, repAcc.Bytes(), data.MetaId), *data)
		if err != nil {
			panic(err)
		}
	}

	// initialize querys from genesis state
	for _, data := range genState.Query {
		queryId := utils.QueryIDFromData(data.QueryData)
		err = k.Query.Set(ctx, collections.Join(queryId, data.Id), *data)
		if err != nil {
			panic(err)
		}
	}

	// initialize aggregates from genesis state
	for _, data := range genState.Aggregates {
		err = k.Aggregates.Set(ctx, collections.Join(data.Aggregate.QueryId, data.Timestamp), *data.Aggregate)
		if err != nil {
			panic(err)
		}
	}

	//initialize Values from genesis state
	for _, data := range genState.Values {
		err = k.Values.Set(ctx, collections.Join(data.MetaId, data.ValueString), *data.Value)
		if err != nil {
			panic(err)
		}
	}

	// initialize AggregateValue from genesis state
	for _, data := range genState.AggregateValue {
		err = k.AggregateValue.Set(ctx, data.MetaId, *data.RunningAggregate)
		if err != nil {
			panic(err)
		}
	}

	// initialize ValuesWeightSum from genesis state
	for _, data := range genState.ValuesWeightSum {
		err = k.ValuesWeightSum.Set(ctx, data.MetaId, data.TotalWeight)
		if err != nil {
			panic(err)
		}
	}

	//initialize ValuesWeightedMode from genesis state
	for _, data := range genState.ValuesWeightedMode {
		err = k.ValuesWeightedMode.Set(ctx, collections.Join(data.MetaId, data.Value), data.TotalPower)
		if err != nil {
			panic(err)
		}
	}

}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx context.Context, k keeper.Keeper) *types.GenesisState {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()
	days_ago := sdkCtx.BlockTime().Add(-21 * 24 * time.Hour) // 21 days ago
	genesis := types.DefaultGenesis()
	// get params
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Params = params

	// get querydata limit
	queryDataLimit, err := k.QueryDataLimit.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.QueryDataLimit = queryDataLimit.Limit

	// get cyclelist
	cyclelist, err := k.GetCyclelist(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Cyclelist = cyclelist

	//export any reports in the keeper
	iterReports, err := k.Reports.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	reports := make([]*types.MicroReport, 0)
	for ; iterReports.Valid(); iterReports.Next() {
		report, err := iterReports.Value()
		if err != nil {
			panic(err)
		}
		if report.Timestamp.After(days_ago) {
			reports = append(reports, &report)
		}
	}
	genesis.Reports = reports

	iterTipperTotal, err := k.TipperTotal.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	tipperTotals := make([]*types.TipperTotalStateEntry, 0)
	for ; iterTipperTotal.Valid(); iterTipperTotal.Next() {
		keys, err := iterTipperTotal.Key()
		if err != nil {
			panic(err)
		}
		tipperAddr := keys.K1()
		blockheight := keys.K2()
		if (uint64(currentBlockHeight)-1134000) > blockheight && currentBlockHeight > 1134000 {
			continue
		}
		tipAmount, err := iterTipperTotal.Value()
		if err != nil {
			panic(err)
		}
		tipperTotals = append(tipperTotals, &types.TipperTotalStateEntry{TipperAddr: tipperAddr, BlockHeight: blockheight, TipAmount: tipAmount})
	}
	genesis.TipperTotal = tipperTotals

	iterTotalTips, err := k.TotalTips.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	totalTips := make([]*types.TotalTipsStateEntry, 0)
	for ; iterTotalTips.Valid(); iterTotalTips.Next() {
		blockheight, err := iterTotalTips.Key()
		if err != nil {
			panic(err)
		}

		if (uint64(currentBlockHeight)-1134000) > blockheight && currentBlockHeight > 1134000 {
			continue
		}

		tipTotal, err := iterTotalTips.Value()
		if err != nil {
			panic(err)
		}
		totalTips = append(totalTips, &types.TotalTipsStateEntry{BlockHeight: blockheight, TipAmount: tipTotal})
	}
	genesis.TotalTips = totalTips

	iterNonces, err := k.Nonces.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	nonces := make([]*types.NoncesStateEntry, 0)
	for ; iterNonces.Valid(); iterNonces.Next() {
		queryId, err := iterNonces.Key()
		if err != nil {
			panic(err)
		}
		nonce, err := iterNonces.Value()
		if err != nil {
			panic(err)
		}
		nonces = append(nonces, &types.NoncesStateEntry{QueryId: queryId, Nonce: nonce})
	}
	genesis.Nonces = nonces

	// querySequencerValue, err := k.QuerySequencer.Peek(ctx)
	// if err != nil {
	// 	panic(err)
	// }
	// genesis.QuerySequencer = querySequencerValue

	// cyclelistSequencerValue, err := k.CyclelistSequencer.Peek(ctx)
	// if err != nil {
	// 	panic(err)
	// }
	// genesis.CyclelistSequence = cyclelistSequencerValue

	iterQuery, err := k.Query.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	queries := make([]*types.QueryMeta, 0)
	values := make([]*types.ValuesStateEntry, 0)
	aggValues := make([]*types.AggregateValueStateEntry, 0)
	valueSums := make([]*types.ValuesWeightSumStateEntry, 0)
	for ; iterQuery.Valid(); iterQuery.Next() {
		query, err := iterQuery.Value()
		if err != nil {
			panic(err)
		}
		if (uint64(currentBlockHeight)-1134000) > query.Expiration && query.Amount.IsZero() && currentBlockHeight > 1134000 {
			continue
		}
		rng := collections.NewPrefixedPairRange[uint64, string](query.Id)
		err = k.Values.Walk(ctx, rng, func(key collections.Pair[uint64, string], value types.Value) (stop bool, err error) {
			values = append(values, &types.ValuesStateEntry{MetaId: key.K1(), ValueString: key.K2(), Value: &value})
			return false, nil
		})
		if err != nil {
			panic(err)
		}

		aggValue, err := k.AggregateValue.Get(ctx, query.Id)
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				panic(err)
			}
		}
		aggValues = append(aggValues, &types.AggregateValueStateEntry{MetaId: query.Id, RunningAggregate: &aggValue})

		valueSum, err := k.ValuesWeightSum.Get(ctx, query.Id)
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				panic(err)
			}
		}
		valueSums = append(valueSums, &types.ValuesWeightSumStateEntry{MetaId: query.Id, TotalWeight: valueSum})

		queries = append(queries, &query)
	}
	genesis.Query = queries
	genesis.Values = values
	genesis.AggregateValue = aggValues
	genesis.ValuesWeightSum = valueSums

	iterAggregates, err := k.Aggregates.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	olderThan21Days := days_ago.UnixMilli()
	aggregates := make([]*types.AggregateStateEntry, 0)
	for ; iterAggregates.Valid(); iterAggregates.Next() {
		keys, err := iterAggregates.Key()
		if err != nil {
			panic(err)
		}
		timestamp := keys.K2()
		if timestamp < uint64(olderThan21Days) {
			fmt.Println("timestamp", timestamp, "olderThan21Days", olderThan21Days)
			continue
		}
		aggregate, err := iterAggregates.Value()
		if err != nil {
			panic(err)
		}
		aggregates = append(aggregates, &types.AggregateStateEntry{Timestamp: timestamp, Aggregate: &aggregate})
	}
	genesis.Aggregates = aggregates

	iterValuesMode, err := k.ValuesWeightedMode.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	valueModes := make([]*types.ValuesWeightedModeStateEntry, 0)
	for ; iterValuesMode.Valid(); iterValuesMode.Next() {
		keys, err := iterValuesMode.Key()
		if err != nil {
			panic(err)
		}
		metaId := keys.K1()
		value := keys.K2()
		totalPower, err := iterValuesMode.Value()
		if err != nil {
			panic(err)
		}
		valueModes = append(valueModes, &types.ValuesWeightedModeStateEntry{MetaId: metaId, Value: value, TotalPower: totalPower})
	}
	genesis.ValuesWeightedMode = valueModes

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
