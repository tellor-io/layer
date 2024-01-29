package types

import (
	"fmt"
	"time"

	cosmosmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DailyMintRate     = 146940000 // loya per day
	DefaultBondDenom  = "loya"
	MillisecondsInDay = 24 * 60 * 60 * 1000
	InitialMint       = 300 * 1e6 // 300k loya
)

// NewMinter returns a new Minter object.
func NewMinter(bondDenom string) Minter {
	return Minter{
		BondDenom: bondDenom,
	}
}

// DefaultMinter returns a Minter object with default values.
func DefaultMinter() Minter {
	return NewMinter(DefaultBondDenom)
}

// Validate returns an error if the minter is invalid.
func (m Minter) Validate() error {
	if m.PreviousBlockTime == nil {
		return fmt.Errorf("previous block time cannot be nil")
	}
	if m.BondDenom == "" {
		return fmt.Errorf("bond denom should not be empty string")
	}
	return nil
}

// CalculateBlockProvision returns the total number of coins that should be
// minted due to time elapsed for the current block.
func (m Minter) CalculateBlockProvision(current time.Time, previous time.Time) (sdk.Coin, error) {
	if current.Before(previous) {
		return sdk.Coin{}, fmt.Errorf("current time %v cannot be before previous time %v", current, previous)
	}
	timeElapsed := current.Sub(previous).Milliseconds()
	mintAmount := DailyMintRate * timeElapsed / MillisecondsInDay
	return sdk.NewCoin(DefaultBondDenom, cosmosmath.NewInt(mintAmount)), nil
}
