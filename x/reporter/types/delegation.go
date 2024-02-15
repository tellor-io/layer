package types

import (
	"cosmossdk.io/math"
)

func NewDelegation(reporter string, amount math.Int) Delegation {
	return Delegation{
		Reporter: reporter,
		Amount:   amount,
	}
}
