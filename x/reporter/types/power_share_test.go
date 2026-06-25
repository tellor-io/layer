package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
)

func TestExceedsDelegatorStakeShare(t *testing.T) {
	total := math.NewInt(1000)
	require.False(t, ExceedsDelegatorStakeShare(math.LegacyNewDec(300), total))
	require.True(t, ExceedsDelegatorStakeShare(math.LegacyNewDec(301), total))
}

func TestExceedsReporterPowerShare(t *testing.T) {
	maxShare := math.LegacyNewDecWithPrec(30, 2)
	total := math.NewInt(1000)
	maxAllowed := maxShare.MulInt(total)

	require.False(t, ExceedsReporterPowerShare(maxAllowed.Sub(math.LegacyNewDec(1)), math.LegacyZeroDec(), total, maxShare))
	require.True(t, ExceedsReporterPowerShare(maxAllowed.Sub(math.LegacyNewDec(1)), math.LegacyNewDec(1), total, maxShare))
	require.True(t, ExceedsReporterPowerShare(maxAllowed, math.LegacyZeroDec(), total, maxShare))
}

func TestValidatorCapCheckEquivalence(t *testing.T) {
	maxShare := math.LegacyNewDecWithPrec(30, 2)
	total := math.NewInt(1000)

	// ante-style acquisition at exactly 30% is allowed (strict >)
	atCap := math.NewInt(300)
	check := ValidatorCapCheck{
		ValidatorTokensAfter: atCap,
		TotalBondedAfter:     total,
		MaxShare:             maxShare,
	}
	require.False(t, check.Exceeds())
	require.False(t, ExceedsPowerShare(atCap, total, maxShare))

	// keeper-style cumulative check matches for the same end state
	overCap := math.NewInt(301)
	checkOver := ValidatorCapCheck{
		ValidatorTokensAfter: overCap,
		TotalBondedAfter:     total,
		MaxShare:             maxShare,
	}
	require.True(t, checkOver.Exceeds())
	require.True(t, ExceedsPowerShare(overCap, total, maxShare))
}
