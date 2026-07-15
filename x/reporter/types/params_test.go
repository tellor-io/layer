package types

import (
	fmt "fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

func TestParams_NewParams(t *testing.T) {
	require := require.New(t)

	params := NewParams(math.LegacyNewDec(5), math.NewInt(1), 100, 10, 10, math.LegacyNewDecWithPrec(30, 2), math.LegacyNewDecWithPrec(30, 2))
	require.NoError(params.Validate())
	require.Equal(params.MinCommissionRate, math.LegacyNewDec(5))
	require.Equal(params.MinLoya, math.NewInt(1))
	require.Equal(params.MaxSelectors, uint64(100))
	require.Equal(params.MaxNumOfDelegations, uint64(10))
	require.Equal(params.MaxPendingSwitchesPerReporter, uint64(10))
	require.Equal(params.MaxReporterPowerShare, math.LegacyNewDecWithPrec(30, 2))
	require.Equal(params.MaxValidatorPowerShare, math.LegacyNewDecWithPrec(30, 2))

	params = NewParams(math.LegacyZeroDec(), math.NewInt(0), 0, 0, 1, math.LegacyZeroDec(), math.LegacyZeroDec())
	require.NoError(params.Validate())
	require.Equal(params.MinCommissionRate, math.LegacyZeroDec())
	require.Equal(params.MinLoya, math.NewInt(0))
	require.Equal(params.MaxSelectors, uint64(0))
	require.Equal(params.MaxNumOfDelegations, uint64(0))

	params = NewParams(math.LegacyNewDec(100), math.NewInt(100), 100, 100, 10, math.LegacyOneDec(), math.LegacyOneDec())
	require.NoError(params.Validate())
	require.Equal(params.MinCommissionRate, math.LegacyNewDec(100))
	require.Equal(params.MinLoya, math.NewInt(100))
	require.Equal(params.MaxSelectors, uint64(100))
	require.Equal(params.MaxNumOfDelegations, uint64(100))

	params = NewParams(math.LegacyNewDec(100), math.NewInt(1000), 1000, 1000, 10, math.LegacyDec{}, math.LegacyDec{})
	require.NoError(params.Validate())
	require.Equal(params.MinCommissionRate, math.LegacyNewDec(100))
	require.Equal(params.MinLoya, math.NewInt(1000))
	require.Equal(params.MaxSelectors, uint64(1000))
	require.Equal(params.MaxNumOfDelegations, uint64(1000))
}

func TestParams_DefaultParams(t *testing.T) {
	require := require.New(t)

	params := DefaultParams()
	require.NoError(params.Validate())
	require.Equal(params.MinLoya, DefaultMinLoya)
	require.Equal(params.MinCommissionRate, DefaultMinCommissionRate)
	require.Equal(params.MaxSelectors, DefaultMaxSelectors)
	require.Equal(params.MaxNumOfDelegations, DefaultMaxNumOfDelegations)
	require.Equal(params.MaxPendingSwitchesPerReporter, DefaultMaxPendingSwitchesPerReporter)
	require.Equal(params.MaxReporterPowerShare, DefaultMaxReporterPowerShare)
	require.Equal(params.MaxValidatorPowerShare, DefaultMaxValidatorPowerShare)
}

func TestParams_ParamSetPairs(t *testing.T) {
	require := require.New(t)

	params := DefaultParams()
	ps := params.ParamSetPairs()

	expected := paramtypes.ParamSetPairs{
		{Key: KeyMinCommissionRate, Value: &params.MinCommissionRate, ValidatorFn: validateMinCommissionRate},
		{Key: KeyMinLoya, Value: &params.MinLoya, ValidatorFn: validateMinLoya},
		{Key: KeyMaxSelectors, Value: &params.MaxSelectors, ValidatorFn: validateMaxSelectors},
		{Key: KeyMaxNumOfDelegations, Value: &params.MaxNumOfDelegations, ValidatorFn: validateMaxNumOfDelegations},
		{Key: KeyMaxPendingSwitchesPerReporter, Value: &params.MaxPendingSwitchesPerReporter, ValidatorFn: validateMaxPendingSwitchesPerReporter},
		{Key: KeyMaxReporterPowerShare, Value: &params.MaxReporterPowerShare, ValidatorFn: validateMaxReporterPowerShare},
		{Key: KeyMaxValidatorPowerShare, Value: &params.MaxValidatorPowerShare, ValidatorFn: validateMaxValidatorPowerShare},
	}

	for i := range expected {
		require.Equal(expected[i].Key, ps[i].Key)
		require.Equal(expected[i].Value, ps[i].Value)
		require.Equal(fmt.Sprintf("%p", expected[i].ValidatorFn), fmt.Sprintf("%p", ps[i].ValidatorFn))
	}
}

func TestParams_Validate(t *testing.T) {
	require := require.New(t)

	params := DefaultParams()
	require.NoError(params.Validate())

	params.MaxReporterPowerShare = math.LegacyNewDec(-1)
	require.ErrorContains(params.Validate(), "max reporter power share")

	// nil (pre-migration) and >= 1 (disabled) are both valid
	params.MaxReporterPowerShare = math.LegacyDec{}
	require.NoError(params.Validate())
	params.MaxReporterPowerShare = math.LegacyNewDec(2)
	require.NoError(params.Validate())

	// validator power share mirrors the reporter share validation
	params = DefaultParams()
	params.MaxValidatorPowerShare = math.LegacyNewDec(-1)
	require.ErrorContains(params.Validate(), "max validator power share")
	params.MaxValidatorPowerShare = math.LegacyDec{}
	require.NoError(params.Validate())
	params.MaxValidatorPowerShare = math.LegacyZeroDec()
	require.NoError(params.Validate())
	params.MaxValidatorPowerShare = math.LegacyOneDec()
	require.NoError(params.Validate())
}

func TestValidatePowerShareParam(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantErr string
	}{
		{"nil legacy dec", math.LegacyDec{}, ""},
		{"zero", math.LegacyZeroDec(), ""},
		{"positive", math.LegacyNewDecWithPrec(30, 2), ""},
		{"one", math.LegacyOneDec(), ""},
		{"greater than one", math.LegacyNewDec(2), ""},
		{"negative", math.LegacyNewDec(-1), "test power share"},
		{"wrong type", uint64(1), "invalid parameter type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePowerShareParam("test power share", tt.value)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestPowerShareEnabled(t *testing.T) {
	tests := []struct {
		name  string
		share math.LegacyDec
		want  bool
	}{
		{"nil", math.LegacyDec{}, false},
		{"zero", math.LegacyZeroDec(), false},
		{"negative", math.LegacyNewDec(-1), false},
		{"positive below one", math.LegacyNewDecWithPrec(30, 2), true},
		{"one", math.LegacyOneDec(), false},
		{"greater than one", math.LegacyNewDec(2), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, PowerShareEnabled(tt.share))
		})
	}
}

func TestExceedsPowerShare(t *testing.T) {
	maxShare := math.LegacyNewDecWithPrec(30, 2)

	tests := []struct {
		name     string
		held     math.Int
		total    math.Int
		maxShare math.LegacyDec
		want     bool
	}{
		{"below cap", math.NewInt(29), math.NewInt(100), maxShare, false},
		{"at cap", math.NewInt(30), math.NewInt(100), maxShare, false},
		{"above cap", math.NewInt(31), math.NewInt(100), maxShare, true},
		{"nil share disabled", math.NewInt(31), math.NewInt(100), math.LegacyDec{}, false},
		{"zero share disabled", math.NewInt(31), math.NewInt(100), math.LegacyZeroDec(), false},
		{"one share disabled", math.NewInt(101), math.NewInt(100), math.LegacyOneDec(), false},
		{"non-positive total", math.NewInt(31), math.ZeroInt(), maxShare, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ExceedsPowerShare(tt.held, tt.total, tt.maxShare))
		})
	}
}

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
