package keeper

import (
	layertypes "github.com/tellor-io/layer/types"

	"cosmossdk.io/math"
)

// CalculateRefundAmount splits the opening quote less the burn on the retained
// round-one fee. Later-round fees are fully consumed and never enter this split.
func CalculateRefundAmount(payerFee, quote, retainedFee math.Int) (math.Int, math.Int) {
	return calculateProRataAmount(payerFee, quote, quote.Sub(retainedFee.QuoRaw(20)))
}

func CalculateAgainstRefundAmount(payerFee, quote, retainedFee math.Int) (math.Int, math.Int) {
	return calculateProRataAmount(payerFee, quote, quote.Sub(retainedFee))
}

func CalculateReporterBondRewardAmount(payerFee, quote, reporterBond math.Int) (math.Int, math.Int) {
	return calculateProRataAmount(payerFee, quote, reporterBond)
}

// Return whole loya and the fractional remainder at the module's dust precision.
func calculateProRataAmount(payerFee, quote, pool math.Int) (math.Int, math.Int) {
	if quote.IsZero() {
		return math.ZeroInt(), math.ZeroInt()
	}
	scaled := payerFee.Mul(pool).Mul(layertypes.PowerReduction).Quo(quote)
	return scaled.Quo(layertypes.PowerReduction), scaled.Mod(layertypes.PowerReduction)
}

func CalculateActualStakeFeeRefundAmount(actualFee, intendedFee, intendedRefund math.Int) math.Int {
	burnShare := intendedFee.Sub(intendedRefund)
	if !burnShare.IsPositive() {
		return actualFee
	}
	if actualFee.LTE(burnShare) {
		return math.ZeroInt()
	}
	return actualFee.Sub(burnShare)
}
