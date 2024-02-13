package types

import (
	"time"

	"cosmossdk.io/math"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func NewOracleReporter(reporter string, totalAmount uint64, commission *stakingtypes.Commission) OracleReporter {
	return OracleReporter{
		Reporter:    reporter,
		TotalTokens: totalAmount,
		Commission:  commission,
	}
}

// alias
func NewCommissionWithTime(rate, maxRate, maxChangeRate math.LegacyDec, updatedAt time.Time) stakingtypes.Commission {
	return stakingtypes.Commission{
		CommissionRates: stakingtypes.NewCommissionRates(rate, maxRate, maxChangeRate),
		UpdateTime:      updatedAt,
	}
}
