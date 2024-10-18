package types

import (
	"fmt"

	"cosmossdk.io/math"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyMinCommissionRate = []byte("MinCommissionRate")
	// TODO: Determine the default value
	DefaultMinCommissionRate = math.NewUint(0)
	DefaultMinTrb            = math.NewIntWithDecimal(1, 6)
	DefaultMaxSelectors      = uint64(100)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	minCommissionRate math.Uint,
	minTrb math.Int,
) Params {
	return Params{
		MinCommissionRate: minCommissionRate,
		MinTrb:            minTrb,
		MaxSelectors:      DefaultMaxSelectors,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMinCommissionRate,
		DefaultMinTrb,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinCommissionRate, &p.MinCommissionRate, validateMinCommissionRate),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMinCommissionRate(p.MinCommissionRate); err != nil {
		return err
	}

	return nil
}

// validateMinStakeAmount validates the MinStakeAmount param
func validateMinCommissionRate(v interface{}) error {
	_, ok := v.(math.Uint)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}
