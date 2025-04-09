package oracle_test

import (
	"bytes"
	"fmt"
	"sort"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/oracle"
	"github.com/tellor-io/layer/x/oracle/types"
)

func TestGenesis(t *testing.T) {
	k, _, _, _, _, _, ctx := keepertest.OracleKeeper(t)
	require := require.New(t)
	require.NotNil(k)
	require.NotNil(ctx)

	// init genesis with expected start values
	genesisState := types.GenesisState{
		Params:             types.DefaultParams(),
		Cyclelist:          types.InitialCycleList(),
		QueryDataLimit:     types.DefaultGenesis().QueryDataLimit,
		Reports:            []*types.MicroReport{},
		TipperTotal:        []*types.TipperTotalStateEntry{},
		TotalTips:          []*types.TotalTipsStateEntry{},
		Nonces:             []*types.NoncesStateEntry{},
		Query:              []*types.QueryMeta{},
		Aggregates:         []*types.AggregateStateEntry{},
		Values:             []*types.ValuesStateEntry{},
		AggregateValue:     []*types.AggregateValueStateEntry{},
		ValuesWeightSum:    []*types.ValuesWeightSumStateEntry{},
		ValuesWeightedMode: []*types.ValuesWeightedModeStateEntry{},
	}
	// init genesis
	oracle.InitGenesis(ctx, k, genesisState)
	// export genesis
	got := oracle.ExportGenesis(ctx, k)

	// sort cyclelist so order doesnt matter for comparison
	sortByteSlices := func(slices [][]byte) {
		sort.Slice(slices, func(i, j int) bool {
			return bytes.Compare(slices[i], slices[j]) < 0
		})
	}
	sortByteSlices(genesisState.Cyclelist)
	sortByteSlices(got.Cyclelist)

	require.Equal(genesisState.Params, got.Params)
	require.Equal(genesisState.Cyclelist, got.Cyclelist)
	require.Equal(genesisState.QueryDataLimit, got.QueryDataLimit)
	require.Equal(genesisState.Reports, got.Reports)
	require.Equal(genesisState.TipperTotal, got.TipperTotal)
	require.Equal(genesisState.TotalTips, got.TotalTips)
	require.Equal(genesisState.Nonces, got.Nonces)
	require.Equal(genesisState.Query, got.Query)
	require.Equal(genesisState.Aggregates, got.Aggregates)
	require.Equal(genesisState.Values, got.Values)
	require.Equal(genesisState.AggregateValue, got.AggregateValue)
	require.Equal(genesisState.ValuesWeightSum, got.ValuesWeightSum)
	require.Equal(genesisState.ValuesWeightedMode, got.ValuesWeightedMode)
	require.NotNil(got)

	now := time.Now()
	// add data to every field
	got.AggregateValue = append(got.AggregateValue, &types.AggregateValueStateEntry{
		MetaId: uint64(1),
		RunningAggregate: &types.RunningAggregate{
			Value:           "1",
			CrossoverWeight: uint64(1),
		},
	})
	got.Aggregates = append(got.Aggregates, &types.AggregateStateEntry{
		Aggregate: &types.Aggregate{
			QueryId:           []byte("query1"),
			AggregateValue:    "1",
			AggregatePower:    uint64(1),
			Flagged:           false,
			AggregateReporter: sample.AccAddressBytes().String(),
			Index:             uint64(1),
			Height:            uint64(1),
			MicroHeight:       uint64(1),
			MetaId:            uint64(1),
		},
		Timestamp: uint64(now.UnixMilli()),
	})
	got.Cyclelist = append(got.Cyclelist, []byte("BonusCyclelistItem"))
	got.Nonces = append(got.Nonces, &types.NoncesStateEntry{
		Nonce:   uint64(1),
		QueryId: []byte("queryId"),
	})
	got.Query = append(got.Query, &types.QueryMeta{
		Id:                      uint64(1),
		Amount:                  math.NewInt(1),
		Expiration:              uint64(1),
		QueryData:               []byte("queryData"),
		RegistrySpecBlockWindow: uint64(1),
		HasRevealedReports:      false,
		QueryType:               "queryType",
		CycleList:               false,
	})
	got.QueryDataLimit = uint64(10000000)
	got.Reports = append(got.Reports, &types.MicroReport{
		Reporter:        sample.AccAddressBytes().String(),
		Power:           uint64(1),
		QueryType:       "SpotPrice",
		QueryId:         []byte("queryId"),
		AggregateMethod: "weighted-median",
		Value:           "1",
		MetaId:          uint64(1),
		BlockNumber:     uint64(1),
		Timestamp:       now,
	})
	got.TotalTips = append(got.TotalTips, &types.TotalTipsStateEntry{
		BlockHeight: uint64(1),
		TipAmount:   math.NewInt(1),
	})
	got.TipperTotal = append(got.TipperTotal, &types.TipperTotalStateEntry{
		TipperAddr:  sample.AccAddressBytes(),
		TipAmount:   math.NewInt(1),
		BlockHeight: uint64(1),
	})
	got.Values = append(got.Values, &types.ValuesStateEntry{
		Value: &types.Value{
			CrossoverWeight: uint64(1),
			MicroReport: &types.MicroReport{
				Reporter:        sample.AccAddressBytes().String(),
				Power:           uint64(1),
				QueryType:       "SpotPrice",
				QueryId:         []byte("queryId"),
				AggregateMethod: "weighted-median",
				Value:           "1",
				MetaId:          uint64(1),
				BlockNumber:     uint64(1),
				Timestamp:       now,
			},
		},
		ValueString: "1",
		MetaId:      uint64(1),
	})
	got.ValuesWeightSum = append(got.ValuesWeightSum, &types.ValuesWeightSumStateEntry{
		MetaId:      uint64(1),
		TotalWeight: uint64(1),
	})

	// everything should be exported and imported correctly with nothing pruned
	ctx = ctx.WithBlockTime(now.Add(time.Minute * 10))
	// init with new value
	oracle.InitGenesis(ctx, k, *got)
	got2 := oracle.ExportGenesis(ctx, k)

	// sort cyclelist so order doesnt matter for comparison
	sortByteSlices(got.Cyclelist)
	sortByteSlices(got2.Cyclelist)

	fmt.Println("got.Reports", got.Reports)
	fmt.Println("got2.Reports", got2.Reports)
	require.Equal(got.Params, got2.Params)
	require.Equal(got.Cyclelist, got2.Cyclelist)
	require.Equal(got.QueryDataLimit, got2.QueryDataLimit)
	require.Equal(got.Reports[0].Timestamp.Unix(), got2.Reports[0].Timestamp.Unix())
	require.Equal(got.TipperTotal, got2.TipperTotal)
	require.Equal(got.TotalTips, got2.TotalTips)
	require.Equal(got.Nonces, got2.Nonces)
	require.Equal(got.Query, got2.Query)
	require.Equal(got.Aggregates, got2.Aggregates)
	require.Equal(got.Values[0].Value.MicroReport.Timestamp.Unix(), got2.Values[0].Value.MicroReport.Timestamp.Unix())
	require.Equal(got.AggregateValue, got2.AggregateValue)
	require.Equal(got.ValuesWeightSum, got2.ValuesWeightSum)
	require.Equal(got.ValuesWeightedMode, got2.ValuesWeightedMode)
	require.NotNil(got2)

	// Set up genesis with old data and new data to test the pruning
	got3 := types.GenesisState{
		Params:             types.DefaultParams(),
		Cyclelist:          types.InitialCycleList(),
		QueryDataLimit:     types.DefaultGenesis().QueryDataLimit,
		Reports:            []*types.MicroReport{},
		TipperTotal:        []*types.TipperTotalStateEntry{},
		TotalTips:          []*types.TotalTipsStateEntry{},
		Nonces:             []*types.NoncesStateEntry{},
		Query:              []*types.QueryMeta{},
		Aggregates:         []*types.AggregateStateEntry{},
		Values:             []*types.ValuesStateEntry{},
		AggregateValue:     []*types.AggregateValueStateEntry{},
		ValuesWeightSum:    []*types.ValuesWeightSumStateEntry{},
		ValuesWeightedMode: []*types.ValuesWeightedModeStateEntry{},
	}
	// add report that should be pruned
	got3.Reports = append(got3.Reports, &types.MicroReport{
		Reporter:        sample.AccAddressBytes().String(),
		Power:           uint64(1),
		QueryType:       "SpotPrice",
		QueryId:         []byte("queryId"),
		AggregateMethod: "weighted-median",
		Value:           "1",
		MetaId:          uint64(1),
		BlockNumber:     uint64(1),
		Timestamp:       now.Add(-time.Hour * 25 * 21), //25 days ago
	})
	// add report that should not be pruned
	got3.Reports = append(got3.Reports, &types.MicroReport{
		Reporter:        sample.AccAddressBytes().String(),
		Power:           uint64(1),
		QueryType:       "SpotPrice",
		QueryId:         []byte("queryId"),
		AggregateMethod: "weighted-median",
		Value:           "1",
		MetaId:          uint64(1),
		BlockNumber:     uint64(1),
		Timestamp:       now.Add(-time.Hour * 21), //21 hours ago
	})

	// add tipper total that should be pruned
	got3.TipperTotal = append(got3.TipperTotal, &types.TipperTotalStateEntry{
		TipperAddr:  sample.AccAddressBytes(),
		TipAmount:   math.NewInt(1),
		BlockHeight: uint64(1),
	})
	// add tipper total that should not be pruned
	got3.TipperTotal = append(got3.TipperTotal, &types.TipperTotalStateEntry{
		TipperAddr:  sample.AccAddressBytes(),
		TipAmount:   math.NewInt(1),
		BlockHeight: uint64(1134000),
	})

	// add total tips that should be pruned
	got3.TotalTips = append(got3.TotalTips, &types.TotalTipsStateEntry{
		BlockHeight: uint64(1),
		TipAmount:   math.NewInt(1),
	})
	// add total tips that should not be pruned
	got3.TotalTips = append(got3.TotalTips, &types.TotalTipsStateEntry{
		BlockHeight: uint64(1134000),
		TipAmount:   math.NewInt(1),
	})

	// add query that should be pruned
	got3.Query = append(got3.Query, &types.QueryMeta{
		Id:         uint64(100),
		Amount:     math.NewInt(0),
		Expiration: uint64(1),
		QueryData:  []byte("queryData"),
	})
	// add query that should not be pruned because it has a tip even though it is old
	got3.Query = append(got3.Query, &types.QueryMeta{
		Id:         uint64(101),
		Amount:     math.NewInt(1),
		Expiration: uint64(1),
		QueryData:  []byte("queryData"),
	})
	// add query that should not be pruned because it is not old
	got3.Query = append(got3.Query, &types.QueryMeta{
		Id:         uint64(102),
		Amount:     math.NewInt(1),
		Expiration: uint64(now.UnixMilli()),
		QueryData:  []byte("queryData"),
	})

	// add aggregate values that should be pruned
	got3.AggregateValue = append(got3.AggregateValue, &types.AggregateValueStateEntry{
		MetaId: uint64(1),
		RunningAggregate: &types.RunningAggregate{
			Value:           "1",
			CrossoverWeight: uint64(1),
		},
	})
	// add aggregate values that should not be pruned
	got3.AggregateValue = append(got3.AggregateValue, &types.AggregateValueStateEntry{
		MetaId: uint64(101),
		RunningAggregate: &types.RunningAggregate{
			Value:           "1",
			CrossoverWeight: uint64(1),
		},
	})

	// add values that should be pruned
	got3.Values = append(got3.Values, &types.ValuesStateEntry{
		MetaId: uint64(1),
		Value: &types.Value{
			CrossoverWeight: uint64(1),
			MicroReport: &types.MicroReport{
				Reporter:        sample.AccAddressBytes().String(),
				Power:           uint64(1),
				QueryType:       "SpotPrice",
				QueryId:         []byte("queryId"),
				AggregateMethod: "weighted-median",
				Value:           "1",
				MetaId:          uint64(1),
				BlockNumber:     uint64(1),
				Timestamp:       now.Add(-time.Hour * 25 * 21), //25 days ago
			},
		},
		ValueString: "1",
	})
	// add values that should not be pruned
	got3.Values = append(got3.Values, &types.ValuesStateEntry{
		MetaId: uint64(101),
		Value: &types.Value{
			CrossoverWeight: uint64(1),
			MicroReport: &types.MicroReport{
				Reporter:        sample.AccAddressBytes().String(),
				Power:           uint64(1),
				QueryType:       "SpotPrice",
				QueryId:         []byte("queryId"),
				AggregateMethod: "weighted-median",
				Value:           "1",
				MetaId:          uint64(1),
				BlockNumber:     uint64(1),
				Timestamp:       now.Add(-time.Hour * 25 * 21), //25 days ago
			},
		},
		ValueString: "1",
	})

	// add values weight sum that should be pruned
	got3.ValuesWeightSum = append(got3.ValuesWeightSum, &types.ValuesWeightSumStateEntry{
		MetaId:      uint64(1),
		TotalWeight: uint64(1),
	})
	// add values weight sum that should not be pruned
	got3.ValuesWeightSum = append(got3.ValuesWeightSum, &types.ValuesWeightSumStateEntry{
		MetaId:      uint64(101),
		TotalWeight: uint64(1),
	})

	// add aggregate that should be pruned
	got3.Aggregates = append(got.Aggregates, &types.AggregateStateEntry{
		Aggregate: &types.Aggregate{
			QueryId:           []byte("query1"),
			AggregateValue:    "1",
			AggregatePower:    uint64(1),
			Flagged:           false,
			AggregateReporter: sample.AccAddressBytes().String(),
			Index:             uint64(1),
			Height:            uint64(1),
			MicroHeight:       uint64(1),
			MetaId:            uint64(1),
		},
		Timestamp: uint64(now.Add(-time.Hour * 25 * 21).UnixMilli()), // 25 days ago
	})
	// add aggregate that should not be pruned
	got3.Aggregates = append(got3.Aggregates, &types.AggregateStateEntry{
		Aggregate: &types.Aggregate{
			QueryId:           []byte("query2"),
			AggregateValue:    "1",
			AggregatePower:    uint64(1),
			Flagged:           false,
			AggregateReporter: sample.AccAddressBytes().String(),
			Index:             uint64(1),
			Height:            uint64(1),
			MicroHeight:       uint64(1),
			MetaId:            uint64(101),
		},
		Timestamp: uint64(now.Add(-time.Hour * 21).UnixMilli()), // 21 hours ago
	})

	k, _, _, _, _, _, ctx = keepertest.OracleKeeper(t)
	ctx = ctx.WithBlockTime(now.Add(time.Minute * 10))
	ctx = ctx.WithBlockHeight(1134000 + 100)

	oracle.InitGenesis(ctx, k, got3)
	got4 := oracle.ExportGenesis(ctx, k)

	// sort cyclelist so order doesnt matter for comparison
	sortByteSlices(got3.Cyclelist)
	sortByteSlices(got4.Cyclelist)

	fmt.Println("got3.Aggregate", got3.Aggregates)
	fmt.Println("got4.Aggregate", got4.Aggregates)
	fmt.Println("Length of got3.Aggregates", len(got3.Aggregates))
	fmt.Println("Length of got4.Aggregates", len(got4.Aggregates))
	require.Equal(got3.Params, got4.Params)
	require.Equal(got3.Cyclelist, got4.Cyclelist)
	require.Equal(got3.QueryDataLimit, got4.QueryDataLimit)
	require.Equal(got3.Reports[1].Timestamp.Unix(), got4.Reports[0].Timestamp.Unix())
	require.Equal([]*types.TipperTotalStateEntry{got3.TipperTotal[1]}, got4.TipperTotal)
	require.Equal([]*types.TotalTipsStateEntry{got3.TotalTips[1]}, got4.TotalTips)
	require.Equal(got3.Nonces, got4.Nonces)
	require.Equal([]*types.QueryMeta{got3.Query[2], got3.Query[1]}, got4.Query)
	require.Equal([]*types.AggregateStateEntry{got3.Aggregates[1]}, got4.Aggregates)
	require.Equal([]*types.ValuesStateEntry{got3.Values[1]}, got4.Values)
	require.Equal([]*types.AggregateValueStateEntry{got3.AggregateValue[1]}, got4.AggregateValue)
	require.Equal([]*types.ValuesWeightSumStateEntry{got3.ValuesWeightSum[1]}, got4.ValuesWeightSum)
	require.Equal([]*types.ValuesWeightedModeStateEntry{got3.ValuesWeightedMode[1]}, got4.ValuesWeightedMode)
	require.NotNil(got4)

}
