package types

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

func TestParams_NewParams(t *testing.T) {
	require := require.New(t)

	params := NewParams(math.NewInt(10*1e6), 10*time.Second)
	require.NoError(params.Validate())
	require.Equal(params.MinStakeAmount, math.NewInt(10*1e6))
	require.Equal(params.Offset, 10*time.Second)
}

func TestParams_DefaultParams(t *testing.T) {
	require := require.New(t)

	params := DefaultParams()
	require.NoError(params.Validate())
	require.Equal(params.MinStakeAmount, math.NewInt(1*1e6))
	require.Equal(params.Offset, 6*time.Second)
}

func TestParams_ParamsSetPairs(t *testing.T) {
	require := require.New(t)

	params := DefaultParams()
	ps := params.ParamSetPairs()

	expected := paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinStakeAmount, &params.MinStakeAmount, validateMinStakeAmount),
		paramtypes.NewParamSetPair(KeyOffset, &params.Offset, validateOffset),
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

	params = NewParams(math.NewInt(0), 3*time.Second)
	require.NoError(validateMinStakeAmount(math.ZeroInt()))
	require.NoError(validateOffset(params.Offset))

	params = NewParams(math.NewInt(100*1e6), 10*time.Second)
	require.NoError(validateMinStakeAmount(math.NewInt(100 * 1e6)))
	require.NoError(validateOffset(params.Offset))
}
