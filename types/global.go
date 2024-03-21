package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	BondDenom = "loya"

	OneTrb = math.NewInt(1_000_000)
	// ten percent of 1TRB
	TenPercent = math.NewInt(10_000)

	PowerReduction = sdk.DefaultPowerReduction
)
