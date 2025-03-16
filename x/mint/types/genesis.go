package types

import (
	"errors"
	time "time"
)

// NewGenesisState creates a new GenesisState object
func NewGenesisState(bondDenom string, initialized bool, previousBlockTime *time.Time) *GenesisState {
	return &GenesisState{
		BondDenom:         bondDenom,
		Initialized:       initialized,
		PreviousBlockTime: previousBlockTime,
	}
}

// DefaultGenesis creates a default GenesisState object
func DefaultGenesis() *GenesisState {
	return NewGenesisState(DefaultBondDenom, false, nil)
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data GenesisState) error {
	if data.BondDenom == "" {
		return errors.New("bond denom cannot be empty")
	}
	return nil
}
