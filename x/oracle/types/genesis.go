package types

import (
	"fmt"
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
	if len(gs.Cyclelist) == 0 {
		return fmt.Errorf("cyclelist is empty")
	}
	for _, query := range gs.Cyclelist {
		if len(query) == 0 {
			return fmt.Errorf("cyclelist item is empty")
		}
	}
	return gs.Params.Validate()
}
