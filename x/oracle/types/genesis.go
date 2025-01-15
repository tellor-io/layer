package types

import (
	"fmt"

	regtypes "github.com/tellor-io/layer/x/registry/types"
)

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params:    DefaultParams(),
		Cyclelist: InitialCycleList(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate
	// check if the cyclelist is empty
	if len(gs.Cyclelist) == 0 {
		return fmt.Errorf("cyclelist is empty")
	}

	for _, query := range gs.Cyclelist {
		// check if any cyclelist items are empty
		if len(query) == 0 {
			return fmt.Errorf("cyclelist item is empty")
		}
		// check if the queryType of the given queryData has a decodable query type
		_, _, err := regtypes.DecodeQueryType(query)
		if err != nil {
			return fmt.Errorf("failed to decode query type: %v", err)
		}
	}

	return gs.Params.Validate()
}
