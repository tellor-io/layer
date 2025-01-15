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
	// validate gs.Dataspec
	if len(gs.Dataspec.AbiComponents) == 0 {
		return errors.New("abi components is empty")
	}
	if gs.Dataspec.ResponseValueType == "" {
		return errors.New("response value type is empty")
	}
	if gs.Dataspec.ReportBlockWindow == 0 {
		return errors.New("report block window is 0")
	}
	if gs.Dataspec.Registrar == "" {
		return errors.New("registrar is empty")
	}
	if gs.Dataspec.AggregationMethod == "" {
		return errors.New("aggregation method is empty")
	}

	return gs.Params.Validate()
}
