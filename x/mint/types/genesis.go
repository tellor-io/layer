package types

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewGenesisState creates a new GenesisState object
func NewGenesisState(bondDenom string) *GenesisState {
	return &GenesisState{
		BondDenom:   bondDenom,
		InitialMint: sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(InitialMint))),
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesis() *GenesisState {
	return NewGenesisState(DefaultBondDenom)
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data GenesisState) error {
	if data.BondDenom == "" {
		return errors.New("bond denom cannot be empty")
	}
	if !data.InitialMint.IsValid() {
		return errors.New("initial mint coins are invalid")
	}
	return nil
}
