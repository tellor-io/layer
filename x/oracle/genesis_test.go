package oracle_test

import (
	"bytes"
	"sort"
	"testing"

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
		CyclelistSequence:  uint64(0),
		TipperTotal:        []*types.TipperTotalStateEntry{},
		TotalTips:          []*types.TotalTipsStateEntry{},
		Nonces:             []*types.NoncesStateEntry{},
		Query:              []*types.QueryMeta{},
		QuerySequencer:     uint64(0),
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
	require.Equal(genesisState.CyclelistSequence, got.CyclelistSequence)
	require.Equal(genesisState.TipperTotal, got.TipperTotal)
	require.Equal(genesisState.TotalTips, got.TotalTips)
	require.Equal(genesisState.Nonces, got.Nonces)
	require.Equal(genesisState.Query, got.Query)
	require.Equal(genesisState.QuerySequencer, got.QuerySequencer)
	require.Equal(genesisState.Aggregates, got.Aggregates)
	require.Equal(genesisState.Values, got.Values)
	require.Equal(genesisState.AggregateValue, got.AggregateValue)
	require.Equal(genesisState.ValuesWeightSum, got.ValuesWeightSum)
	require.Equal(genesisState.ValuesWeightedMode, got.ValuesWeightedMode)
	require.NotNil(got)

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
	})
	got.Cyclelist = append(got.Cyclelist, []byte("BonusCyclelistItem"))
	got.CyclelistSequence = uint64(3)
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
	got.QuerySequencer = uint64(1)
	got.Reports = append(got.Reports, &types.MicroReport{
		Reporter:        sample.AccAddressBytes().String(),
		Power:           uint64(1),
		QueryType:       "SpotPrice",
		QueryId:         []byte("queryId"),
		AggregateMethod: "weighted-median",
		Value:           "1",
		MetaId:          uint64(1),
		BlockNumber:     uint64(1),
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
			},
		},
		ValueString: "1",
		MetaId:      uint64(1),
	})
	got.ValuesWeightSum = append(got.ValuesWeightSum, &types.ValuesWeightSumStateEntry{
		MetaId:      uint64(1),
		TotalWeight: uint64(1),
	})

	// init with new value
	oracle.InitGenesis(ctx, k, *got)
	got2 := oracle.ExportGenesis(ctx, k)

	// sort cyclelist so order doesnt matter for comparison
	sortByteSlices(got.Cyclelist)
	sortByteSlices(got2.Cyclelist)

	require.Equal(got.Params, got2.Params)
	require.Equal(got.Cyclelist, got2.Cyclelist)
	require.Equal(got.QueryDataLimit, got2.QueryDataLimit)
	require.Equal(got.Reports, got2.Reports)
	require.Equal(got.CyclelistSequence, got2.CyclelistSequence)
	require.Equal(got.TipperTotal, got2.TipperTotal)
	require.Equal(got.TotalTips, got2.TotalTips)
	require.Equal(got.Nonces, got2.Nonces)
	require.Equal(got.Query, got2.Query)
	require.Equal(got.QuerySequencer, got2.QuerySequencer)
	require.Equal(got.Aggregates, got2.Aggregates)
	require.Equal(got.Values, got2.Values)
	require.Equal(got.AggregateValue, got2.AggregateValue)
	require.Equal(got.ValuesWeightSum, got2.ValuesWeightSum)
	require.Equal(got.ValuesWeightedMode, got2.ValuesWeightedMode)
	require.NotNil(got2)

}
