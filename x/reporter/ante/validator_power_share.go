package ante

import (
	"errors"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type validatorCapCandidate struct {
	validator    prospectiveValidator
	activeAfter  bool
	beforeActive math.Int
	afterActive  math.Int
}

func (c validatorCapCandidate) isAcquisition() bool {
	return c.afterActive.GT(c.beforeActive)
}

func validatorCapBeforeActive(validator prospectiveValidator) math.Int {
	if validator.preTxActive {
		return validator.validator.Tokens
	}
	return math.ZeroInt()
}

func validatorCapActiveAfter(validator prospectiveValidator) bool {
	return validator.validator.IsBonded() && !validator.validator.IsJailed()
}

func validatorCapAfterActive(validator prospectiveValidator, activeAfter bool) math.Int {
	if activeAfter {
		return validator.postTokens
	}
	return math.ZeroInt()
}

func (t TrackStakeChangesDecorator) upsertValidatorCapCandidate(
	candidates map[validatorAddressKey]validatorCapCandidate,
	validatorKey validatorAddressKey,
	validator prospectiveValidator,
	activeAfter bool,
) {
	entry, ok := candidates[validatorKey]
	if !ok {
		entry = validatorCapCandidate{validator: validator, activeAfter: activeAfter}
	} else {
		entry.validator = validator
		entry.activeAfter = activeAfter
	}
	entry.beforeActive = validatorCapBeforeActive(entry.validator)
	entry.afterActive = validatorCapAfterActive(entry.validator, entry.activeAfter)
	candidates[validatorKey] = entry
}

// checkValidatorPowerShares rejects validator stake acquisitions above max_validator_power_share.
func (t TrackStakeChangesDecorator) checkValidatorPowerShares(ctx sdk.Context, stakeChanges *stakeChangeTracker, capCtx stakeCapContext) error {
	bondedChanges := capCtx.activeSet.changes
	if stakeChanges == nil {
		return nil
	}
	if len(stakeChanges.validatorProjections) == 0 && len(bondedChanges.entering) == 0 && len(bondedChanges.leaving) == 0 {
		return nil
	}

	params, err := t.reporterKeeper.Params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}
	maxShare := params.MaxValidatorPowerShare
	if !types.PowerShareEnabled(maxShare) {
		return nil
	}

	candidates := make(map[validatorAddressKey]validatorCapCandidate)

	for _, validatorKey := range sortedKeys(stakeChanges.validatorProjections) {
		validator := stakeChanges.validatorProjections[validatorKey]
		t.upsertValidatorCapCandidate(candidates, validatorKey, validator, validatorCapActiveAfter(validator))
	}
	for _, validator := range bondedChanges.entering {
		validatorKey := newValidatorAddressKey(validator.addr)
		t.upsertValidatorCapCandidate(candidates, validatorKey, validator, true)
	}
	for _, validator := range bondedChanges.leaving {
		validatorKey := newValidatorAddressKey(validator.addr)
		t.upsertValidatorCapCandidate(candidates, validatorKey, validator, false)
	}

	totalBondedAfter := capCtx.totalBondedAfterValidator()

	for _, validatorKey := range sortedKeys(candidates) {
		candidate := candidates[validatorKey]
		if !candidate.isAcquisition() {
			continue
		}
		if !totalBondedAfter.IsPositive() {
			return nil
		}
		check := types.ValidatorCapCheck{
			ValidatorTokensAfter: candidate.validator.postTokens,
			TotalBondedAfter:     totalBondedAfter,
			MaxShare:             maxShare,
		}
		if check.Exceeds() {
			return errorsmod.Wrapf(types.ErrExceedsMaxValidatorPowerShare, "validator %s", candidate.validator.addr.String())
		}
	}
	return nil
}
