package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

func TestParams_NewParams(t *testing.T) {
	require := require.New(t)

	params := NewParams(math.NewInt(10*1e6), math.NewInt(10*1e3), math.NewInt(25*1e6))
	require.NoError(params.Validate())
	require.Equal(params.MinStakeAmount, math.NewInt(10*1e6))
	require.Equal(params.MinTipAmount, math.NewInt(10_000))
	require.Equal(params.MaxTipAmount, math.NewInt(25_000_000))
}

func TestParams_DefaultParams(t *testing.T) {
	require := require.New(t)

	params := DefaultParams()
	require.NoError(params.Validate())
	require.Equal(params.MinStakeAmount, math.NewInt(1*1e6))
	require.Equal(params.MinTipAmount, math.NewInt(10_000))
	require.Equal(params.MaxTipAmount, math.NewInt(25_000_000))
}

func TestParams_ParamsSetPairs(t *testing.T) {
	require := require.New(t)

	params := DefaultParams()
	ps := params.ParamSetPairs()

	expected := paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinStakeAmount, &params.MinStakeAmount, validateMinStakeAmount),
		paramtypes.NewParamSetPair(KeyMinTipAmount, &params.MinTipAmount, validateMinTipAmount),
		paramtypes.NewParamSetPair(KeyMaxTipAmount, &params.MaxTipAmount, validateMaxTipAmount),
	}

	require.Equal(len(expected), len(ps))
	for i := range expected {
		require.Equal(expected[i].Key, ps[i].Key)
		require.Equal(expected[i].Value, ps[i].Value)
		require.Equal(fmt.Sprintf("%p", expected[i].ValidatorFn), fmt.Sprintf("%p", ps[i].ValidatorFn))
	}
}

func TestParams_Validate(t *testing.T) {
	require := require.New(t)

	params := DefaultParams()
	require.NoError(validateMinStakeAmount(params.MinStakeAmount))
	require.NoError(validateMinTipAmount(params.MinTipAmount))
	require.NoError(validateMaxTipAmount(params.MaxTipAmount))
	params = NewParams(math.NewInt(0), math.NewInt(0), math.NewInt(0))
	require.NoError(validateMinStakeAmount(math.ZeroInt()))
	require.NoError(validateMinTipAmount(math.ZeroInt()))
	require.NoError(validateMaxTipAmount(math.ZeroInt()))

	params = NewParams(math.NewInt(100*1e6), math.NewInt(10*1e3), math.NewInt(20*1e6))
	require.NoError(validateMinStakeAmount(math.NewInt(100 * 1e6)))
	require.NoError(validateMinTipAmount(math.NewInt(10 * 1e3)))
	require.NoError(validateMaxTipAmount(math.NewInt(20 * 1e6)))
}
