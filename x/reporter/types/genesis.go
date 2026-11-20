package types

import (
	"bytes"
	"fmt"
)

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate

	if err := gs.Params.Validate(); err != nil {
		return err
	}
	agents := make(map[string]struct{}, len(gs.TwoBitReporters))
	reporters := make(map[string]struct{}, len(gs.TwoBitReporters))
	for _, entry := range gs.TwoBitReporters {
		if entry == nil || len(entry.AgentAddress) == 0 || len(entry.ReporterAddress) == 0 {
			return fmt.Errorf("two bit reporter addresses cannot be empty")
		}
		if bytes.Equal(entry.AgentAddress, entry.ReporterAddress) {
			return fmt.Errorf("two bit agent address cannot equal the reporter")
		}
		agent := string(entry.AgentAddress)
		reporter := string(entry.ReporterAddress)
		if _, ok := agents[agent]; ok {
			return fmt.Errorf("duplicate two bit agent address")
		}
		if _, ok := reporters[reporter]; ok {
			return fmt.Errorf("duplicate two bit reporter address")
		}
		agents[agent] = struct{}{}
		reporters[reporter] = struct{}{}
	}
	return nil
}
