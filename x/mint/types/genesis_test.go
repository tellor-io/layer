package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/mint/types"
)

func TestNewGenesisState(t *testing.T) {
	require := require.New(t)

	genesisState := types.NewGenesisState("loya")
	require.NotNil(genesisState)
	require.Equal(genesisState.BondDenom, "loya")
}

func TestDefaultGenesis(t *testing.T) {
	require := require.New(t)

	genesisState := types.DefaultGenesis()
	require.NotNil(genesisState)
	require.Equal(genesisState.BondDenom, "loya")
}

func TestValidateGenesis(t *testing.T) {
	require := require.New(t)

	genesisState := types.DefaultGenesis()
	require.NotNil(genesisState)
	require.Equal(genesisState.BondDenom, "loya")
	err := types.ValidateGenesis(*genesisState)
	require.NoError(err)

	emptyDenomGenesis := types.NewGenesisState("")
	err = types.ValidateGenesis(*emptyDenomGenesis)
	require.ErrorContains(err, "bond denom cannot be empty")
}
