package types

import (
	"cosmossdk.io/math"
)

func NewReporter(commission math.LegacyDec, minTokensRequired math.Int) OracleReporter {
	return OracleReporter{
		MinTokensRequired: minTokensRequired,
		CommissionRate:    commission,
	}
}

func NewSelection(reporter []byte, delegationCount int64) Selection {
	return Selection{
		Reporter:         reporter,
		DelegationsCount: delegationCount,
	}
}
