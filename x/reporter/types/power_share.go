package types

import (
	"fmt"

	"cosmossdk.io/math"
)

const (
	delegatorStakeShareNumerator   int64 = 3
	delegatorStakeShareDenominator int64 = 10
)

func ValidatePowerShareParam(name string, v interface{}) error {
	share, ok := v.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}
	if share.IsNil() {
		return nil
	}
	if share.IsNegative() {
		return fmt.Errorf("%s cannot be negative", name)
	}
	return nil
}

func PowerShareEnabled(share math.LegacyDec) bool {
	return !share.IsNil() && share.IsPositive() && !share.GTE(math.LegacyOneDec())
}

func ExceedsPowerShare(held math.Int, total math.Int, maxShare math.LegacyDec) bool {
	if !PowerShareEnabled(maxShare) || held.IsNil() || total.IsNil() || !total.IsPositive() {
		return false
	}
	return held.ToLegacyDec().GT(maxShare.MulInt(total))
}

// ExceedsDelegatorStakeShare reports whether held strictly exceeds 30% of total
// bonded stake (hardcoded delegator cap boundary).
func ExceedsDelegatorStakeShare(held math.LegacyDec, total math.Int) bool {
	if held.IsNil() || total.IsNil() || !total.IsPositive() {
		return false
	}
	return held.MulInt64(delegatorStakeShareDenominator).GT(total.ToLegacyDec().MulInt64(delegatorStakeShareNumerator))
}

// ExceedsReporterPowerShare reports whether potential plus addition reaches or
// exceeds maxShare of total bonded stake (reporter cap uses >=).
func ExceedsReporterPowerShare(potential math.LegacyDec, addition math.LegacyDec, total math.Int, maxShare math.LegacyDec) bool {
	if !PowerShareEnabled(maxShare) || total.IsNil() || !total.IsPositive() {
		return false
	}
	maxAllowed := maxShare.MulInt(total)
	return potential.Add(addition).GTE(maxAllowed)
}

// ValidatorCapCheck is the shared validator acquisition cap comparison used by
// ante and keeper enforcement paths.
type ValidatorCapCheck struct {
	ValidatorTokensAfter math.Int
	TotalBondedAfter     math.Int
	MaxShare             math.LegacyDec
}

func (c ValidatorCapCheck) Exceeds() bool {
	return ExceedsPowerShare(c.ValidatorTokensAfter, c.TotalBondedAfter, c.MaxShare)
}
