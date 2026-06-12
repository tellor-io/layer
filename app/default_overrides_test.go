package app

import (
	"testing"

	icagenesistypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/genesis/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// Interchain accounts must ship disabled: ICA-executed messages skip the ante
// chain and would bypass the stake and reporter power limits.
func TestICADefaultGenesisDisabled(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	raw := icaModule{}.DefaultGenesis(cdc)

	var genState icagenesistypes.GenesisState
	require.NoError(t, cdc.UnmarshalJSON(raw, &genState))
	require.False(t, genState.HostGenesisState.Params.HostEnabled)
	require.Empty(t, genState.HostGenesisState.Params.AllowMessages)
	require.False(t, genState.ControllerGenesisState.Params.ControllerEnabled)
}
