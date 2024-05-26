package types

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	BondDenom = "loya"

	OneTrb = math.NewInt(1_000_000)
	// one percent of 1TRB
	// min stake is 1TRB so min dispute fee would be 1% of 1TRB
	OnePercent = math.NewInt(1_000)

	PowerReduction = sdk.DefaultPowerReduction
)
