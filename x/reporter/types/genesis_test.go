package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/reporter/types"
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
			},
			valid: true,
		},
		{
			desc: "valid two bit registration",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TwoBitReporters: []*types.TwoBitReporterStateEntry{{
					AgentAddress:    []byte("agent"),
					ReporterAddress: []byte("reporter"),
				}},
			},
			valid: true,
		},
		{
			desc: "duplicate two bit agent",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TwoBitReporters: []*types.TwoBitReporterStateEntry{
					{AgentAddress: []byte("agent"), ReporterAddress: []byte("reporter-a")},
					{AgentAddress: []byte("agent"), ReporterAddress: []byte("reporter-b")},
				},
			},
			valid: false,
		},
		{
			desc: "duplicate two bit reporter",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TwoBitReporters: []*types.TwoBitReporterStateEntry{
					{AgentAddress: []byte("agent-a"), ReporterAddress: []byte("reporter")},
					{AgentAddress: []byte("agent-b"), ReporterAddress: []byte("reporter")},
				},
			},
			valid: false,
		},
		{
			desc: "two bit agent equals reporter",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TwoBitReporters: []*types.TwoBitReporterStateEntry{{
					AgentAddress:    []byte("same"),
					ReporterAddress: []byte("same"),
				}},
			},
			valid: false,
		},
		{
			desc: "empty two bit address",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TwoBitReporters: []*types.TwoBitReporterStateEntry{{
					AgentAddress:    []byte{},
					ReporterAddress: []byte("reporter"),
				}},
			},
			valid: false,
		},
		{
			desc: "nil two bit entry",
			genState: &types.GenesisState{
				Params:          types.DefaultParams(),
				TwoBitReporters: []*types.TwoBitReporterStateEntry{nil},
			},
			valid: false,
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
