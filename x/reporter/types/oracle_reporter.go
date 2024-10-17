package types

import (
	"cosmossdk.io/math"
)

func NewReporter(commission math.Uint, minTokensRequired math.Int) OracleReporter {
	return OracleReporter{
		MinTokensRequired: minTokensRequired,
		CommissionRate:    commission,
	}
}

func NewSelection(reporter []byte, delegationCount uint64) Selection {
	return Selection{
		Reporter:         reporter,
		DelegationsCount: delegationCount,
	}
}
