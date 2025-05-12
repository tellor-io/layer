package oracle_test

import (
	"bytes"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
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
		Params:         types.DefaultParams(),
		Cyclelist:      types.InitialCycleList(),
		QueryDataLimit: types.DefaultGenesis().QueryDataLimit,
		QuerySequencer: types.DefaultGenesis().QuerySequencer,
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
	require.Equal(genesisState.QuerySequencer, got.QuerySequencer)
	require.NotNil(got)

	now := time.Now()

	// everything should be exported and imported correctly with nothing pruned
	ctx = ctx.WithBlockTime(now.Add(time.Minute * 10))
	// init with new value
	oracle.InitGenesis(ctx, k, *got)
	got2 := oracle.ExportGenesis(ctx, k)

	// sort cyclelist so order doesnt matter for comparison
	sortByteSlices(got.Cyclelist)
	sortByteSlices(got2.Cyclelist)

	require.Equal(got.Params, got2.Params)
	require.Equal(got.Cyclelist, got2.Cyclelist)
	require.Equal(got.QueryDataLimit, got2.QueryDataLimit)
	require.Equal(got.QuerySequencer, got2.QuerySequencer)
	require.NotNil(got2)

	// Set up genesis with old data and new data to test the pruning
	got3 := types.GenesisState{
		Params:         types.DefaultParams(),
		Cyclelist:      types.InitialCycleList(),
		QueryDataLimit: types.DefaultGenesis().QueryDataLimit,
		QuerySequencer: types.DefaultGenesis().QuerySequencer,
	}

	k, _, _, _, _, _, ctx = keepertest.OracleKeeper(t)
	ctx = ctx.WithBlockTime(now.Add(time.Minute * 10))
	ctx = ctx.WithBlockHeight(1134000 + 100)

	oracle.InitGenesis(ctx, k, got3)
	got4 := oracle.ExportGenesis(ctx, k)

	// sort cyclelist so order doesnt matter for comparison
	sortByteSlices(got3.Cyclelist)
	sortByteSlices(got4.Cyclelist)

	require.Equal(got3.Params, got4.Params)
	require.Equal(got3.Cyclelist, got4.Cyclelist)
	require.Equal(got3.QueryDataLimit, got4.QueryDataLimit)
	require.Equal(got3.QuerySequencer, got4.QuerySequencer)
	require.NotNil(got4)

	err := os.Remove("oracle_module_state.json")
	require.NoError(err)

}
