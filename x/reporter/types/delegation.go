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

// create a new DelegatorStartingInfo
func NewDelegatorStartingInfo(previousPeriod uint64, stake math.Int, height uint64) DelegatorStartingInfo {
	return DelegatorStartingInfo{
		PreviousPeriod: previousPeriod,
		Stake:          stake,
		Height:         height,
	}
}
