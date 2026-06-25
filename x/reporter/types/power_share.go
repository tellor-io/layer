package types

import (
	"fmt"

	"cosmossdk.io/math"
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
