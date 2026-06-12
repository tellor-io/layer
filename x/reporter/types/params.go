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
	DefaultMinCommissionRate             = math.LegacyZeroDec()
	KeyMinLoya                           = []byte("MinLoya")
	DefaultMinLoya                       = math.NewIntWithDecimal(1, 6)
	KeyMaxSelectors                      = []byte("MaxSelectors")
	DefaultMaxSelectors                  = uint64(100)
	KeyMaxNumOfDelegations               = []byte("MaxNumOfDelegations")
	DefaultMaxNumOfDelegations           = uint64(10)
	KeyMaxPendingSwitchesPerReporter     = []byte("MaxPendingSwitchesPerReporter")
	DefaultMaxPendingSwitchesPerReporter = uint64(10)
	KeyMaxReporterPowerShare             = []byte("MaxReporterPowerShare")
	// DefaultMaxReporterPowerShare caps a single reporter's potential stake below
	// 30% of total bonded tokens; values >= 1 disable the check (small networks).
	DefaultMaxReporterPowerShare = math.LegacyNewDecWithPrec(30, 2)
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
	maxPendingSwitchesPerReporter uint64,
	maxReporterPowerShare math.LegacyDec,
) Params {
	return Params{
		MinCommissionRate:             minCommissionRate,
		MinLoya:                       minLoya,
		MaxSelectors:                  maxSelectors,
		MaxNumOfDelegations:           maxNumOfDelegations,
		MaxPendingSwitchesPerReporter: maxPendingSwitchesPerReporter,
		MaxReporterPowerShare:         maxReporterPowerShare,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMinCommissionRate,
		DefaultMinLoya,
		DefaultMaxSelectors,
		DefaultMaxNumOfDelegations,
		DefaultMaxPendingSwitchesPerReporter,
		DefaultMaxReporterPowerShare,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinCommissionRate, &p.MinCommissionRate, validateMinCommissionRate),
		paramtypes.NewParamSetPair(KeyMinLoya, &p.MinLoya, validateMinLoya),
		paramtypes.NewParamSetPair(KeyMaxSelectors, &p.MaxSelectors, validateMaxSelectors),
		paramtypes.NewParamSetPair(KeyMaxNumOfDelegations, &p.MaxNumOfDelegations, validateMaxNumOfDelegations),
		paramtypes.NewParamSetPair(KeyMaxPendingSwitchesPerReporter, &p.MaxPendingSwitchesPerReporter, validateMaxPendingSwitchesPerReporter),
		paramtypes.NewParamSetPair(KeyMaxReporterPowerShare, &p.MaxReporterPowerShare, validateMaxReporterPowerShare),
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
	if err := validateMaxPendingSwitchesPerReporter(p.MaxPendingSwitchesPerReporter); err != nil {
		return err
	}
	if err := validateMaxReporterPowerShare(p.MaxReporterPowerShare); err != nil {
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

func validateMaxPendingSwitchesPerReporter(v interface{}) error {
	n, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}
	if n == 0 {
		return fmt.Errorf("max pending switches per reporter must be positive")
	}
	return nil
}

// validateMaxReporterPowerShare allows nil (pre-migration state, check disabled)
// and any positive share; shares >= 1 disable the check.
func validateMaxReporterPowerShare(v interface{}) error {
	share, ok := v.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}
	if share.IsNil() {
		return nil
	}
	if share.IsNegative() {
		return fmt.Errorf("max reporter power share cannot be negative")
	}
	return nil
}
