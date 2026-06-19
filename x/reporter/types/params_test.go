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

	params := NewParams(math.LegacyNewDec(5), math.NewInt(1), 100, 10, 10, math.LegacyNewDecWithPrec(30, 2))
	require.NoError(params.Validate())
	require.Equal(params.MinCommissionRate, math.LegacyNewDec(5))
	require.Equal(params.MinLoya, math.NewInt(1))
	require.Equal(params.MaxSelectors, uint64(100))
	require.Equal(params.MaxNumOfDelegations, uint64(10))
	require.Equal(params.MaxPendingSwitchesPerReporter, uint64(10))
	require.Equal(params.MaxReporterPowerShare, math.LegacyNewDecWithPrec(30, 2))

	params = NewParams(math.LegacyZeroDec(), math.NewInt(0), 0, 0, 1, math.LegacyZeroDec())
	require.NoError(params.Validate())
	require.Equal(params.MinCommissionRate, math.LegacyZeroDec())
	require.Equal(params.MinLoya, math.NewInt(0))
	require.Equal(params.MaxSelectors, uint64(0))
	require.Equal(params.MaxNumOfDelegations, uint64(0))

	params = NewParams(math.LegacyNewDec(100), math.NewInt(100), 100, 100, 10, math.LegacyOneDec())
	require.NoError(params.Validate())
	require.Equal(params.MinCommissionRate, math.LegacyNewDec(100))
	require.Equal(params.MinLoya, math.NewInt(100))
	require.Equal(params.MaxSelectors, uint64(100))
	require.Equal(params.MaxNumOfDelegations, uint64(100))

	params = NewParams(math.LegacyNewDec(100), math.NewInt(1000), 1000, 1000, 10, math.LegacyDec{})
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
	require.Error(params.Validate())

	// nil (pre-migration) and >= 1 (disabled) are both valid
	params.MaxReporterPowerShare = math.LegacyDec{}
	require.NoError(params.Validate())
	params.MaxReporterPowerShare = math.LegacyNewDec(2)
	require.NoError(params.Validate())
}
