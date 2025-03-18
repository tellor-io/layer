package types

import (
	"cosmossdk.io/math"
)

func NewReporter(commission math.LegacyDec, minTokensRequired math.Int, moniker string) OracleReporter {
	return OracleReporter{
		MinTokensRequired: minTokensRequired,
		CommissionRate:    commission,
		Moniker:           moniker,
	}
}

func NewSelection(reporter []byte, delegationCount uint64) Selection {
	return Selection{
		Reporter:         reporter,
		DelegationsCount: delegationCount,
	}
}
