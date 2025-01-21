package types

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"cosmossdk.io/math"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyMinStakeAmount = []byte("MinStakeAmount")
	KeyMinTipAmount   = []byte("MinTipAmount")
	// TODO: Determine the default value
	DefaultMinStakeAmount = math.NewInt(1_000_000) // one TRB

	DefaultMinTipAmount = math.NewInt(10_000)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(minStakeAmount, minTipAmount math.Int) Params {
	return Params{
		MinStakeAmount: minStakeAmount,
		MinTipAmount:   minTipAmount,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultMinStakeAmount, DefaultMinTipAmount)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinStakeAmount, &p.MinStakeAmount, validateMinStakeAmount),
		paramtypes.NewParamSetPair(KeyMinTipAmount, &p.MinTipAmount, validateMinTipAmount),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if p.MinStakeAmount.IsNil() {
		return fmt.Errorf("min stake amount is nil")
	}

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

func validateMinTipAmount(v interface{}) error {
	_, ok := v.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}
