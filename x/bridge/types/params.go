package types

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"cosmossdk.io/math"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

// Parameter default values
var (
	// Default slash percentage: 1% (0.01)
	DefaultAttestSlashPercentage = math.LegacyNewDecWithPrec(1, 2)

	// Default rate limit window: 10 minutes in milliseconds
	DefaultAttestRateLimitWindow = uint64(10 * 60 * 1000)

	// Default valset slash percentage: 5% (0.05)
	DefaultValsetSlashPercentage = math.LegacyNewDecWithPrec(5, 2)

	// Default valset rate limit window: 10 minutes in milliseconds
	DefaultValsetRateLimitWindow = uint64(10 * 60 * 1000)

	// Default attest penalty time cutoff: 0 (no cutoff)
	DefaultAttestPenaltyTimeCutoff = uint64(0)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(attestSlashPercentage math.LegacyDec, attestRateLimitWindow uint64, valsetSlashPercentage math.LegacyDec, valsetRateLimitWindow, attestPenaltyTimeCutoff uint64) Params {
	return Params{
		AttestSlashPercentage:   attestSlashPercentage,
		AttestRateLimitWindow:   attestRateLimitWindow,
		ValsetSlashPercentage:   valsetSlashPercentage,
		ValsetRateLimitWindow:   valsetRateLimitWindow,
		AttestPenaltyTimeCutoff: attestPenaltyTimeCutoff,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultAttestSlashPercentage,
		DefaultAttestRateLimitWindow,
		DefaultValsetSlashPercentage,
		DefaultValsetRateLimitWindow,
		DefaultAttestPenaltyTimeCutoff,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair([]byte("AttestSlashPercentage"), &p.AttestSlashPercentage, validateAttestSlashPercentage),
		paramtypes.NewParamSetPair([]byte("AttestRateLimitWindow"), &p.AttestRateLimitWindow, validateAttestRateLimitWindow),
		paramtypes.NewParamSetPair([]byte("ValsetSlashPercentage"), &p.ValsetSlashPercentage, validateValsetSlashPercentage),
		paramtypes.NewParamSetPair([]byte("ValsetRateLimitWindow"), &p.ValsetRateLimitWindow, validateValsetRateLimitWindow),
		paramtypes.NewParamSetPair([]byte("AttestPenaltyTimeCutoff"), &p.AttestPenaltyTimeCutoff, validateAttestPenaltyTimeCutoff),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateAttestSlashPercentage(p.AttestSlashPercentage); err != nil {
		return err
	}

	if err := validateAttestRateLimitWindow(p.AttestRateLimitWindow); err != nil {
		return err
	}

	if err := validateValsetSlashPercentage(p.ValsetSlashPercentage); err != nil {
		return err
	}

	if err := validateValsetRateLimitWindow(p.ValsetRateLimitWindow); err != nil {
		return err
	}

	if err := validateAttestPenaltyTimeCutoff(p.AttestPenaltyTimeCutoff); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateAttestSlashPercentage validates the AttestSlashPercentage param
func validateAttestSlashPercentage(v interface{}) error {
	slashPct, ok := v.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// Slash percentage must be between 0 and 1 (0-100%)
	if slashPct.IsNegative() {
		return fmt.Errorf("attest slash percentage cannot be negative: %s", slashPct)
	}

	if slashPct.GT(math.LegacyOneDec()) {
		return fmt.Errorf("attest slash percentage too high: %s, maximum is 1.000000000000000000 (100%%)", slashPct)
	}

	return nil
}

// validateAttestRateLimitWindow validates the AttestRateLimitWindow param
func validateAttestRateLimitWindow(v interface{}) error {
	window, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// Window must be at least 1 seconds (1000 milliseconds)
	if window < 1000 {
		return fmt.Errorf("attest rate limit window too small: %d, minimum is 1000 milliseconds (1 second)", window)
	}

	// Window must be less than or equal to 21 days
	if window > 21*24*60*60*1000 {
		return fmt.Errorf("attest rate limit window too large: %d, maximum is %d milliseconds (21 days)", window, 21*24*60*60*1000)
	}

	return nil
}

func validateValsetSlashPercentage(v interface{}) error {
	slashPct, ok := v.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// Slash percentage must be between 0 and 1 (0-100%)
	if slashPct.IsNegative() {
		return fmt.Errorf("valset slash percentage cannot be negative: %s", slashPct)
	}

	if slashPct.GT(math.LegacyOneDec()) {
		return fmt.Errorf("valset slash percentage too high: %s, maximum is 1.000000000000000000 (100%%)", slashPct)
	}

	return nil
}

func validateValsetRateLimitWindow(v interface{}) error {
	window, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// Window must be at least 1 seconds (1000 milliseconds)
	if window < 1000 {
		return fmt.Errorf("valset rate limit window too small: %d, minimum is 1000 milliseconds (1 second)", window)
	}

	// Window must be less than or equal to 21 days
	if window > 21*24*60*60*1000 {
		return fmt.Errorf("valset rate limit window too large: %d, maximum is %d milliseconds (21 days)", window, 21*24*60*60*1000)
	}

	return nil
}

func validateAttestPenaltyTimeCutoff(v interface{}) error {
	cutoff, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// cutoff can be any valid uint64 timestamp (including 0 for no cutoff)
	// no specific validation needed beyond type checking
	_ = cutoff
	return nil
}
