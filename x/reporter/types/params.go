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
	DefaultMinCommissionRate   = math.LegacyZeroDec()
	KeyMinLoya                 = []byte("MinLoya")
	DefaultMinLoya             = math.NewIntWithDecimal(1, 6)
	KeyMaxSelectors            = []byte("MaxSelectors")
	DefaultMaxSelectors        = uint64(100)
	KeyMaxNumOfDelegations     = []byte("MaxNumOfDelegations")
	DefaultMaxNumOfDelegations = uint64(10)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	minCommissionRate math.LegacyDec,
	minLoya math.Int,
	maxSelectors uint64,
	maxNumOfDelegations uint64,
) Params {
	return Params{
		MinCommissionRate:   minCommissionRate,
		MinLoya:             minLoya,
		MaxSelectors:        maxSelectors,
		MaxNumOfDelegations: maxNumOfDelegations,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMinCommissionRate,
		DefaultMinLoya,
		DefaultMaxSelectors,
		DefaultMaxNumOfDelegations,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinCommissionRate, &p.MinCommissionRate, validateMinCommissionRate),
		paramtypes.NewParamSetPair(KeyMinLoya, &p.MinLoya, validateMinLoya),
		paramtypes.NewParamSetPair(KeyMaxSelectors, &p.MaxSelectors, validateMaxSelectors),
		paramtypes.NewParamSetPair(KeyMaxNumOfDelegations, &p.MaxNumOfDelegations, validateMaxNumOfDelegations),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMinCommissionRate(p.MinCommissionRate); err != nil {
		return err
	}
	if err := validateMinLoya(p.MinLoya); err != nil {
		return err
	}
	if err := validateMaxSelectors(p.MaxSelectors); err != nil {
		return err
	}
	if err := validateMaxNumOfDelegations(p.MaxNumOfDelegations); err != nil {
		return err
	}

	return nil
}

// validateMinStakeAmount validates the MinStakeAmount param
func validateMinCommissionRate(v interface{}) error {
	_, ok := v.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}

func validateMinLoya(v interface{}) error {
	_, ok := v.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}

func validateMaxSelectors(v interface{}) error {
	_, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}

func validateMaxNumOfDelegations(v interface{}) error {
	_, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}
