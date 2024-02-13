package types

func NewDelegation(reporter string, amount uint64) Delegation {
	return Delegation{
		Reporter: reporter,
		Amount:   amount,
	}
}
