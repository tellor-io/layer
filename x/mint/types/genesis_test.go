package types_test

import (
	"testing"
	time "time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/mint/types"
)

func TestNewGenesisState(t *testing.T) {
	require := require.New(t)

	time := time.Now()
	genesisState := types.NewGenesisState("loya", true, &time)
	require.NotNil(genesisState)
	require.Equal(genesisState.BondDenom, "loya")
	require.True(genesisState.Initialized)
	require.NotNil(genesisState.PreviousBlockTime)
}

func TestDefaultGenesis(t *testing.T) {
	require := require.New(t)

	genesisState := types.DefaultGenesis()
	require.NotNil(genesisState)
	require.Equal(genesisState.BondDenom, "loya")
	require.False(genesisState.Initialized)
	require.Nil(genesisState.PreviousBlockTime)
}

func TestValidateGenesis(t *testing.T) {
	require := require.New(t)

	genesisState := types.DefaultGenesis()
	require.NotNil(genesisState)
	require.Equal(genesisState.BondDenom, "loya")
	err := types.ValidateGenesis(*genesisState)
	require.NoError(err)

	time := time.Now()
	emptyDenomGenesis := types.NewGenesisState("", true, &time)
	err = types.ValidateGenesis(*emptyDenomGenesis)
	require.ErrorContains(err, "bond denom cannot be empty")
}
