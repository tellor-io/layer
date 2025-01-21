package registry_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/nullify"
	registry "github.com/tellor-io/layer/x/registry/module"
	"github.com/tellor-io/layer/x/registry/types"
)

const (
	genQueryTypeSpotPrice     = "spotprice"
	genQueryTypeBridgeDeposit = "trbbridge"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:   types.DefaultParams(),
		Dataspec: types.GenesisDataSpec(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, _, _, ctx := keepertest.RegistryKeeper(t)
	registry.InitGenesis(ctx, k, genesisState)
	got := registry.ExportGenesis(ctx, k)
	require.NotNil(t, got)
	nullify.Fill(&genesisState)
	nullify.Fill(got)
	bridgeDS, err := k.HasSpec(ctx, genQueryTypeBridgeDeposit)
	require.NoError(t, err)
	priceDS, _ := k.HasSpec(ctx, genQueryTypeSpotPrice)
	require.NoError(t, err)
	require.Equal(t, bridgeDS, true)
	require.Equal(t, priceDS, true)

	dataspec, err := k.SpecRegistry.Get(ctx, genQueryTypeBridgeDeposit)
	require.NoError(t, err)
	require.Equal(t, dataspec.QueryType, genQueryTypeBridgeDeposit)

	dataspec, err = k.SpecRegistry.Get(ctx, genQueryTypeSpotPrice)
	require.NoError(t, err)
	require.Equal(t, dataspec.QueryType, genQueryTypeSpotPrice)

	iter, err := k.SpecRegistry.Iterate(ctx, nil)
	require.NoError(t, err)
	var i int
	for ; iter.Valid(); iter.Next() {
		dataspec, err := iter.Value()
		require.NoError(t, err)
		fmt.Println("dataspec.QueryType: ", dataspec.QueryType)
		i++
	}
	require.Equal(t, i, 2)

	k.SpecRegistry.Set(ctx, "question", types.DataSpec{
		DocumentHash:      "",
		ResponseValueType: "uint256",
		AbiComponents: []*types.ABIComponent{
			{
				Name:            "question",
				FieldType:       "string",
				NestedComponent: []*types.ABIComponent{},
			},
			{
				Name:            "answer",
				FieldType:       "string",
				NestedComponent: []*types.ABIComponent{},
			},
		},
		AggregationMethod: "weighted-mode",
		Registrar:         "genesis",
		ReportBlockWindow: 200,
		QueryType:         "question",
	})

	got = registry.ExportGenesis(ctx, k)
	require.NotNil(t, got)
	registry.InitGenesis(ctx, k, *got)

	dataspec, err = k.SpecRegistry.Get(ctx, "question")
	require.NoError(t, err)
	require.Equal(t, dataspec.QueryType, "question")

	// this line is used by starport scaffolding # genesis/test/assert
}
