package oracle

import (
	"context"
	"encoding/hex"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
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
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx context.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Params = params

	// get cyclelist
	cyclelist, err := k.GetCyclelist(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Cyclelist = cyclelist

	// export module data we want to migrate over to oracle data file
	exportModuleData(ctx, k)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}

type TipperTotalData struct {
	TipperTotal math.Int
	Address     []byte
}

type TotalTipsData struct {
	TotalTips math.Int
	Block     uint64
}

type ModuleStateData struct {
	TipperTotal []TipperTotalData
	TotalTips   TotalTipsData
}

func exportModuleData(ctx context.Context, k keeper.Keeper) {
	writer, err := utils.NewModuleStateWriter("oracle_module_state.json")
	if err != nil {
		panic(err)
	}

	tipperTotals := make(map[string]TipperTotalData)
	highestBlockNumbers := make(map[string]uint64)

	// Iterate over the TipperTotal map
	err = k.TipperTotal.Walk(ctx, nil, func(key collections.Pair[[]byte, uint64], value math.Int) (bool, error) {
		tipperAcc := key.K1()
		blockNumber := key.K2()

		tipperAccString := hex.EncodeToString(tipperAcc)

		// Check if this is the highest block number for this reporter
		if highestBlock, exists := highestBlockNumbers[tipperAccString]; !exists || blockNumber > highestBlock {
			tipperTotals[tipperAccString] = TipperTotalData{
				TipperTotal: value,
				Address:     tipperAcc,
			}
			highestBlockNumbers[tipperAccString] = blockNumber
		}

		return false, nil
	})

	writer.StartArraySection("tipper_total", false)
	for _, tipperTotal := range tipperTotals {
		err = writer.WriteArrayItem(tipperTotal)
		if err != nil {
			panic(err)
		}
	}
	writer.EndArraySection(len(tipperTotals))
	if err != nil {
		panic(err)
	}

	rng := new(collections.Range[uint64]).Descending()
	err = k.TotalTips.Walk(ctx, rng, func(key uint64, value math.Int) (bool, error) {
		writer.WriteValue("latest_total_tips", TotalTipsData{
			TotalTips: value,
			Block:     key,
		})
		return false, nil
	})
	if err != nil {
		panic(err)
	}

	iterQuery, err := k.Query.IterateRaw(ctx, nil, nil, collections.OrderAscending)
	if err != nil {
		panic(err)
	}
	err = writer.StartArraySection("tipped_queries", true)
	numQueries := 0
	for ; iterQuery.Valid(); iterQuery.Next() {
		query, err := iterQuery.Value()
		if err != nil {
			panic(err)
		}
		if query.Amount.IsZero() {
			continue
		}
		err = writer.WriteArrayItem(query)
		numQueries++
	}
	writer.EndArraySection(numQueries)
	if err != nil {
		panic(err)
	}

}
