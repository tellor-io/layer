package types

import "errors"

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params:   DefaultParams(),
		Dataspec: GenesisDataSpec(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate
	// validate each gs.Dataspec
	for _, dataspec := range gs.Dataspec {
		if dataspec.QueryType == "" {
			return errors.New("query type is empty")
		}
		if len(dataspec.AbiComponents) == 0 {
			return errors.New("abi components is empty")
		}
		if dataspec.ResponseValueType == "" {
			return errors.New("response value type is empty")
		}
		if dataspec.ReportBlockWindow == 0 {
			return errors.New("report block window is 0")
		}
		if dataspec.Registrar == "" {
			return errors.New("registrar is empty")
		}
		if dataspec.AggregationMethod == "" {
			return errors.New("aggregation method is empty")
		}
	}
	return gs.Params.Validate()
}
