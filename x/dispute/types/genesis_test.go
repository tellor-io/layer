package types_test

import (
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/dispute/types"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestGenesisState_GetGenesisStateFromAppState(t *testing.T) {
	require := require.New(t)

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	genState := types.GetGenesisStateFromAppState(cdc, map[string]json.RawMessage{types.ModuleName: []byte("{}")})
	require.NotNil(genState)

	genState = types.GetGenesisStateFromAppState(cdc, nil)
	require.NotNil(genState)
}
