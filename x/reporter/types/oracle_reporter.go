package types

import (
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func NewOracleReporter(reporter string, tokens math.Int, commission *stakingtypes.Commission) OracleReporter {
	return OracleReporter{
		Commission:  commission,
		TotalTokens: tokens,
	}
}

// alias
func NewCommissionWithTime(rate, maxRate, maxChangeRate math.LegacyDec, updatedAt time.Time) stakingtypes.Commission {
	return stakingtypes.Commission{
		CommissionRates: stakingtypes.NewCommissionRates(rate, maxRate, maxChangeRate),
		UpdateTime:      updatedAt,
	}
}

// create a new ReporterHistoricalRewards
func NewReporterHistoricalRewards(cumulativeRewardRatio sdk.DecCoins, referenceCount uint32) ReporterHistoricalRewards {
	return ReporterHistoricalRewards{
		CumulativeRewardRatio: cumulativeRewardRatio,
		ReferenceCount:        referenceCount,
	}
}

// create a new ReporterCurrentRewards
func NewReporterCurrentRewards(rewards sdk.DecCoins, period uint64) ReporterCurrentRewards {
	return ReporterCurrentRewards{
		Rewards: rewards,
		Period:  period,
	}
}

// create a new ReporterDisputeEvent
func NewReporterDisputeEvent(reporterPeriod uint64, fraction math.LegacyDec) ReporterDisputeEvent {
	return ReporterDisputeEvent{
		ReporterPeriod: reporterPeriod,
		Fraction:       fraction,
	}
}
