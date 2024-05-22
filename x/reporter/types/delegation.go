package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewDelegation(reporter string, amount math.Int) Delegation {
	repAcc := sdk.MustAccAddressFromBech32(reporter)
	return Delegation{
		Reporter: repAcc,
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
