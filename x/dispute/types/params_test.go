package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
)

func TestParams_NewParams(t *testing.T) {
	teamAddr := sample.AccAddressBytes()
	params := NewParams(teamAddr)
	require.NotNil(t, params)
	require.Equal(t, params.TeamAddress, teamAddr.Bytes())
}

func TestParams_DefaultParams(t *testing.T) {
	teamAddr := DefaultTeamAddress
	params := DefaultParams()
	require.NotNil(t, params)
	require.Equal(t, params.TeamAddress, teamAddr.Bytes())
}

func TestParams_ParamSetPairs(t *testing.T) {
	params := DefaultParams()
	require.NotNil(t, params)

	pairs := params.ParamSetPairs()
	require.NotNil(t, pairs)
	require.Equal(t, pairs[0].Key, KeyTeamAddress)
	require.Equal(t, pairs[0].Value, &params.TeamAddress)
}

func TestParams_Validate(t *testing.T) {
	params := DefaultParams()
	require.NotNil(t, params)
	require.NoError(t, params.Validate())
}

func TestParams_validateTeamAddress(t *testing.T) {
	params := DefaultParams()
	require.NotNil(t, params)
	require.NoError(t, validateTeamAddress(params.TeamAddress))
}
