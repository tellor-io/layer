package keeper

import (
	layertypes "github.com/tellor-io/layer/types"

	"cosmossdk.io/math"
)

// CalculateRefundAmount calculates the amount of the fee to be refunded to the payer
// returns the amount to be refunded (amtFixed6) and the remainder (dust)
func CalculateRefundAmount(payerFee, totalFeeRd1 math.Int) (math.Int, math.Int) {
	payerFeeDec := payerFee.ToLegacyDec()
	totalFeeRd1Dec := math.LegacyNewDecFromInt(totalFeeRd1)

	// fivePercent = totalFeeRd1 / 20
	fivePercentDec := totalFeeRd1.ToLegacyDec().Quo(math.LegacyNewDec(20))
	fivePercent := fivePercentDec.TruncateInt()

	// totalFeeMinusBurn = totalFeeRd1 - fivePercent
	totalFeeMinusBurnDec := totalFeeRd1.Sub(fivePercent).ToLegacyDec()

	powerReductionDec := math.LegacyNewDecFromInt(layertypes.PowerReduction)

	// (payerFee / totalFeeRd1) * (totalFeeRd1 - burn)
	// result scaled by PowerReduction
	amtFixed12Dec := payerFeeDec.Mul(totalFeeMinusBurnDec).Mul(powerReductionDec).Quo(totalFeeRd1Dec)

	amtFixed12 := amtFixed12Dec.TruncateInt()
	remainder := amtFixed12.Mod(layertypes.PowerReduction)
	amtFixed6 := amtFixed12.Quo(layertypes.PowerReduction)

	return amtFixed6, remainder
}

// CalculateReporterBondRewardAmount calculates the portion of the reporter's bond to be rewarded to the payer
// returns the amount to be rewarded (amtFixed6) and the remainder (dust)
func CalculateReporterBondRewardAmount(payerFee, totalFeesPaid, reporterBond math.Int) (math.Int, math.Int) {
	feeDec := math.LegacyNewDecFromInt(payerFee)
	bondDec := math.LegacyNewDecFromInt(reporterBond)
	totalFeesDec := math.LegacyNewDecFromInt(totalFeesPaid)
	powerReductionDec := math.LegacyNewDecFromInt(layertypes.PowerReduction)

	// (payerFee / totalFeesPaid) * reporterBond
	// result scaled by PowerReduction
	amtFixed12Dec := feeDec.Mul(bondDec).Mul(powerReductionDec).Quo(totalFeesDec)

	amtFixed12 := amtFixed12Dec.TruncateInt()
	amtFixed6Dec := amtFixed12Dec.Quo(powerReductionDec)
	amtFixed6 := amtFixed6Dec.TruncateInt()

	remainder := amtFixed12.Mod(layertypes.PowerReduction)

	return amtFixed6, remainder
}
