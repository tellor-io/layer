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
	// DefaultMaxReporterPowerShare: 30%; values >= 1 disable the check.
	DefaultMaxReporterPowerShare = math.LegacyNewDecWithPrec(30, 2)
	KeyMaxValidatorPowerShare    = []byte("MaxValidatorPowerShare")
	// DefaultMaxValidatorPowerShare: 30%; values >= 1 disable the check.
	DefaultMaxValidatorPowerShare = math.LegacyNewDecWithPrec(30, 2)
)

const (
	delegatorStakeShareNumerator   int64 = 3
	delegatorStakeShareDenominator int64 = 10
)

// DelegatorStakeShare is the hardcoded 30% bonded-stake cap for a single delegator.
func DelegatorStakeShare() math.LegacyDec {
	return math.LegacyNewDecWithPrec(30, 2)
}

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
	maxValidatorPowerShare math.LegacyDec,
) Params {
	return Params{
		MinCommissionRate:             minCommissionRate,
		MinLoya:                       minLoya,
		MaxSelectors:                  maxSelectors,
		MaxNumOfDelegations:           maxNumOfDelegations,
		MaxPendingSwitchesPerReporter: maxPendingSwitchesPerReporter,
		MaxReporterPowerShare:         maxReporterPowerShare,
		MaxValidatorPowerShare:        maxValidatorPowerShare,
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
		DefaultMaxValidatorPowerShare,
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
		paramtypes.NewParamSetPair(KeyMaxValidatorPowerShare, &p.MaxValidatorPowerShare, validateMaxValidatorPowerShare),
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
	if err := validateMaxValidatorPowerShare(p.MaxValidatorPowerShare); err != nil {
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

// validateMaxReporterPowerShare allows nil (disabled) and any non-negative share;
// shares >= 1 disable the check at enforcement.
func validateMaxReporterPowerShare(v interface{}) error {
	return ValidatePowerShareParam("max reporter power share", v)
}

func validateMaxValidatorPowerShare(v interface{}) error {
	return ValidatePowerShareParam("max validator power share", v)
}

func ValidatePowerShareParam(name string, v interface{}) error {
	share, ok := v.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}
	if share.IsNil() {
		return nil
	}
	if share.IsNegative() {
		return fmt.Errorf("%s cannot be negative", name)
	}
	return nil
}

func PowerShareEnabled(share math.LegacyDec) bool {
	return !share.IsNil() && share.IsPositive() && !share.GTE(math.LegacyOneDec())
}

func ExceedsPowerShare(held, total math.Int, maxShare math.LegacyDec) bool {
	if !PowerShareEnabled(maxShare) || held.IsNil() || total.IsNil() || !total.IsPositive() {
		return false
	}
	return held.ToLegacyDec().GT(maxShare.MulInt(total))
}

func ExceedsDelegatorStakeShare(held math.LegacyDec, total math.Int) bool {
	if held.IsNil() || total.IsNil() || !total.IsPositive() {
		return false
	}
	return held.MulInt64(delegatorStakeShareDenominator).GT(total.ToLegacyDec().MulInt64(delegatorStakeShareNumerator))
}

func ExceedsReporterPowerShare(potential, addition math.LegacyDec, total math.Int, maxShare math.LegacyDec) bool {
	if !PowerShareEnabled(maxShare) || total.IsNil() || !total.IsPositive() {
		return false
	}
	maxAllowed := maxShare.MulInt(total)
	return potential.Add(addition).GTE(maxAllowed)
}

// ValidatorCapCheck is shared by ante and keeper validator-cap enforcement.
type ValidatorCapCheck struct {
	ValidatorTokensAfter math.Int
	TotalBondedAfter     math.Int
	MaxShare             math.LegacyDec
}

func (c ValidatorCapCheck) Exceeds() bool {
	return ExceedsPowerShare(c.ValidatorTokensAfter, c.TotalBondedAfter, c.MaxShare)
}
