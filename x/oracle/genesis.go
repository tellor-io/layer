package oracle

import (
	"context"
	"encoding/hex"
	"slices"
	"strings"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
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

	err = k.QuerySequencer.Set(ctx, genState.QuerySequencer)
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

	queryDataLimit, err := k.QueryDataLimit.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.QueryDataLimit = queryDataLimit.Limit

	querySequencer, err := k.QuerySequencer.Peek(ctx)
	if err != nil {
		panic(err)
	}
	genesis.QuerySequencer = querySequencer

	// export module data we want to migrate over to oracle data file
	exportModuleData(ctx, k)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}

type TipperTotalData struct {
	TipperTotal string `json:"tipper_total"`
	Address     []byte `json:"address"`
	Block       uint64 `json:"block"`
}

type TotalTipsData struct {
	TotalTips string `json:"total_tips"`
	Block     uint64 `json:"block"`
}

type ModuleStateData struct {
	TipperTotal   []TipperTotalData `json:"tipper_total"`
	TotalTips     TotalTipsData     `json:"total_tips"`
	TippedQueries []types.QueryMeta `json:"tipped_queries"`
}

type AggregateStateData struct {
	Aggregate types.Aggregate `json:"aggregate"`
	Timestamp uint64          `json:"timestamp"`
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
				TipperTotal: value.String(),
				Address:     tipperAcc,
				Block:       blockNumber,
			}
			highestBlockNumbers[tipperAccString] = blockNumber
		}

		return false, nil
	})
	if err != nil {
		panic(err)
	}

	err = writer.StartArraySection("tipper_total", false)
	if err != nil {
		panic(err)
	}
	for _, tipperTotal := range tipperTotals {
		err = writer.WriteArrayItem(tipperTotal)
		if err != nil {
			panic(err)
		}
	}

	err = writer.EndArraySection(len(tipperTotals))
	if err != nil {
		panic(err)
	}

	rng := new(collections.Range[uint64]).Descending()
	foundTipTotal := false
	err = k.TotalTips.Walk(ctx, rng, func(key uint64, value math.Int) (bool, error) {
		if !foundTipTotal {
			err = writer.WriteValue("latest_total_tips", TotalTipsData{
				TotalTips: value.String(),
				Block:     key,
			})
			if err != nil {
				panic(err)
			}
		}
		foundTipTotal = true
		return true, nil
	})
	if err != nil {
		panic(err)
	}
	if !foundTipTotal {
		err = writer.WriteValue("latest_total_tips", TotalTipsData{
			TotalTips: math.ZeroInt().String(),
			Block:     1,
		})
		if err != nil {
			panic(err)
		}
	}

	iterQuery, err := k.Query.IterateRaw(ctx, nil, nil, collections.OrderAscending)
	if err != nil {
		panic(err)
	}
	defer iterQuery.Close()
	err = writer.StartArraySection("tipped_queries", true)
	if err != nil {
		panic(err)
	}
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
		if err != nil {
			panic(err)
		}
		numQueries++
	}
	err = writer.EndArraySection(numQueries)
	if err != nil {
		panic(err)
	}

	err = writer.StartArraySection("trbbridge_aggregates", true)
	if err != nil {
		panic(err)
	}
	numTrbBridgeAggs := 0
	err = k.Nonces.Walk(ctx, nil, func(queryId []byte, _ uint64) (bool, error) {
		queryId = slices.Clone(queryId)
		firstAgg, ok, err := firstAggregateForQueryID(ctx, k, queryId)
		if err != nil {
			return true, err
		}
		if !ok {
			return false, nil
		}
		queryType, ok, err := queryTypeFromAggregateReports(ctx, k, firstAgg)
		if err != nil {
			k.Logger(ctx).Error(
				"oracle export: error getting query type from aggregate reports",
				"error", err,
				"query_id", hex.EncodeToString(queryId),
			)
			return true, err
		}
		qtLogged := queryType
		if !ok {
			qtLogged = "<unavailable: no micro report for meta_id/query_id>"
		}
		k.Logger(ctx).Info(
			"oracle export: first aggregate sample query_type",
			"query_id", hex.EncodeToString(queryId),
			"query_type", qtLogged,
		)
		if !ok || !strings.Contains(strings.ToLower(queryType), "trbbridge") {
			return false, nil
		}
		rng := collections.NewPrefixedPairRange[[]byte, uint64](queryId)
		walkErr := k.Aggregates.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.Aggregate) (bool, error) {
			if err := writer.WriteArrayItem(AggregateStateData{
				Aggregate: value,
				Timestamp: key.K2(),
			}); err != nil {
				return true, err
			}
			numTrbBridgeAggs++
			return false, nil
		})
		if walkErr != nil {
			return true, walkErr
		}
		return false, nil
	})
	if err != nil {
		panic(err)
	}
	err = writer.EndArraySection(numTrbBridgeAggs)
	if err != nil {
		panic(err)
	}

	writer.Close()
}

// firstAggregateForQueryID returns the earliest aggregate for queryId by store order (timestamp key ascending).
func firstAggregateForQueryID(ctx context.Context, k keeper.Keeper, queryId []byte) (types.Aggregate, bool, error) {
	rng := collections.NewPrefixedPairRange[[]byte, uint64](queryId)
	var out types.Aggregate
	found := false
	err := k.Aggregates.Walk(ctx, rng, func(_ collections.Pair[[]byte, uint64], value types.Aggregate) (bool, error) {
		out = value
		found = true
		return true, nil
	})
	if err != nil {
		return types.Aggregate{}, false, err
	}
	return out, found, nil
}

// queryTypeFromAggregateReports loads query_type from any micro report tied to the aggregate's (meta_id, query_id).
func queryTypeFromAggregateReports(ctx context.Context, k keeper.Keeper, agg types.Aggregate) (string, bool, error) {
	iter, err := k.Reports.Indexes.IdQueryId.MatchExact(ctx, collections.Join(agg.MetaId, agg.QueryId))
	if err != nil {
		return "", false, err
	}
	defer iter.Close()
	if !iter.Valid() {
		return "", false, nil
	}
	pk, err := iter.PrimaryKey()
	if err != nil {
		return "", false, err
	}
	rep, err := k.Reports.Get(ctx, pk)
	if err != nil {
		return "", false, err
	}
	k.Logger(ctx).Info(
		"oracle export: query type from aggregate reports",
		"query_id", hex.EncodeToString(pk.K1()),
		"report", rep.String(),
	)
	return rep.QueryType, true, nil
}
