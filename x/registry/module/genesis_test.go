package registry_test

import (
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
	// this line is used by starport scaffolding # genesis/test/assert
}
