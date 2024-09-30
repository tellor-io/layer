package types

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v2"

	"cosmossdk.io/math"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyMinStakeAmount = []byte("MinStakeAmount")
	// TODO: Determine the default value
	DefaultMinStakeAmount = math.NewInt(1_000_000) // one TRB

	KeyOffset     = []byte("Offset")
	DefaultOffset = 6 * time.Second
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(minStakeAmount math.Int, offset time.Duration) Params {
	return Params{
		MinStakeAmount: minStakeAmount,
		Offset:         offset,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultMinStakeAmount, DefaultOffset)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinStakeAmount, &p.MinStakeAmount, validateMinStakeAmount),
		paramtypes.NewParamSetPair(KeyOffset, &p.Offset, validateOffset),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateMinStakeAmount validates the MinStakeAmount param
func validateMinStakeAmount(v interface{}) error {
	_, ok := v.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}

func validateOffset(v interface{}) error {
	_, ok := v.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}
