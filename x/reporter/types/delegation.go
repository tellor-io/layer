package types

import "fmt"

func NewDelegation(reporter string, amount uint64) Delegation {
	return Delegation{
		Reporter: reporter,
		Amount:   amount,
	}
}

// reduce delegation tokens by amount
func (d Delegation) ReduceDelegationby(amount uint64) (Delegation, error) {
	if d.Amount < amount {
		return d, fmt.Errorf("insufficient delegation amount in delegation to reduce: %v", d)
	}
	d.Amount -= amount
	return d, nil
}
