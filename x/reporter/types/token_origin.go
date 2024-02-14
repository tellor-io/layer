package types

import (
	"fmt"
)

func (o TokenOrigin) ReduceTokenOriginAmountby(amount uint64) (TokenOrigin, error) {
	if o.Amount < amount {
		return o, fmt.Errorf("insufficient amount in token origin: %v", o)
	}
	o.Amount -= amount
	return o, nil
}
