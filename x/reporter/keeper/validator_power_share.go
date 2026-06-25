package keeper

import (
	"context"
	"errors"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// ValidatorPowerFallbackScanLimit bounds the deterministic scan used when a
// preferred bonded destination is over the validator cap. GetBondedValidators
// returns bonded validators in highest-power-first order, so the first under-
// cap validator found is chosen. Amounts are never split.
const ValidatorPowerFallbackScanLimit uint32 = 32

// maxValidatorPowerShare reads the governance param once per high-level keeper
// operation. Returns nil (cap disabled) if params are unset.
func (k Keeper) maxValidatorPowerShare(ctx context.Context) (math.LegacyDec, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return math.LegacyDec{}, nil
		}
		return math.LegacyDec{}, err
	}
	return params.MaxValidatorPowerShare, nil
}

// checkValidatorPowerShareDelegation is the core of the validator acquisition
// cap. It rejects a bonded delegation when the validator's post-delegation share
// of projected total bonded stake is strictly above max_validator_power_share.
// A nil/zero amount, a non-bonded destination, and nil/zero/>=1 maxShare all
// short-circuit. projectedBondedDelta carries the cumulative bonded delta across
// origins in the same call so the denominator reflects in-flight refunds.
func (k Keeper) checkValidatorPowerShareDelegation(
	ctx context.Context,
	validator stakingtypes.Validator,
	amount math.Int,
	maxShare math.LegacyDec,
	projectedBondedDelta math.Int,
) error {
	if amount.IsNil() || !amount.IsPositive() {
		return nil
	}
	if !types.PowerShareEnabled(maxShare) {
		return nil
	}

	// direct keeper enforcement only checks immediate active bonded acquisition
	if !validator.IsBonded() {
		return nil
	}

	currentTotalBonded, err := k.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return err
	}

	totalBondedAfter := currentTotalBonded.Add(projectedBondedDelta).Add(amount)
	if !totalBondedAfter.IsPositive() {
		return nil
	}

	// validator.Tokens is re-fetched per origin by the callers, so it already
	// reflects earlier refunds in this call; no per-validator projection here.
	validatorAfter := validator.Tokens.Add(amount)
	if types.ExceedsPowerShare(validatorAfter, totalBondedAfter, maxShare) {
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		if err != nil {
			return err
		}
		return errorsmod.Wrapf(types.ErrExceedsMaxValidatorPowerShare, "validator %s", valAddr.String())
	}

	return nil
}

// CheckValidatorPowerShareDelegation enforces the validator acquisition cap for
// keeper paths that call stakingKeeper.Delegate directly, bypassing the ante
// stake-change tracker. It is the single-call entry point (no cross-origin
// projection) used by WithdrawTip and external callers.
func (k Keeper) CheckValidatorPowerShareDelegation(ctx context.Context, validator stakingtypes.Validator, amount math.Int) error {
	maxShare, err := k.maxValidatorPowerShare(ctx)
	if err != nil {
		return err
	}
	return k.checkValidatorPowerShareDelegation(ctx, validator, amount, maxShare, math.ZeroInt())
}

// pickUnderCapBondedValidator scans a bounded list of bonded validators in
// staking power-store order and returns the first one that can accept the
// delegation under the validator cap. It never splits amounts and never indexes
// an empty result. If every scanned validator fails the cap, it returns
// ErrExceedsMaxValidatorPowerShare; other errors propagate.
func (k Keeper) pickUnderCapBondedValidator(ctx context.Context, amount math.Int, maxShare math.LegacyDec, projectedBondedDelta math.Int) (stakingtypes.Validator, error) {
	vals, err := k.GetBondedValidators(ctx, ValidatorPowerFallbackScanLimit)
	if err != nil {
		return stakingtypes.Validator{}, err
	}
	if len(vals) == 0 {
		return stakingtypes.Validator{}, errorsmod.Wrap(types.ErrExceedsMaxValidatorPowerShare, "no bonded validator available for delegation")
	}

	for _, val := range vals {
		err := k.checkValidatorPowerShareDelegation(ctx, val, amount, maxShare, projectedBondedDelta)
		if err == nil {
			return val, nil
		}
		if !errors.Is(err, types.ErrExceedsMaxValidatorPowerShare) {
			return stakingtypes.Validator{}, err
		}
	}

	return stakingtypes.Validator{}, errorsmod.Wrapf(
		types.ErrExceedsMaxValidatorPowerShare,
		"no bonded validator among first %d can accept delegation",
		ValidatorPowerFallbackScanLimit,
	)
}

// bondedValidatorForDelegation preserves a preferred bonded destination when it
// is under the cap, and otherwise falls back to a bounded scan for an under-cap
// bonded validator. Non-validator-cap errors propagate.
func (k Keeper) bondedValidatorForDelegation(ctx context.Context, preferred stakingtypes.Validator, amount math.Int, maxShare math.LegacyDec, projectedBondedDelta math.Int) (stakingtypes.Validator, error) {
	if preferred.IsBonded() {
		err := k.checkValidatorPowerShareDelegation(ctx, preferred, amount, maxShare, projectedBondedDelta)
		if err == nil {
			return preferred, nil
		}
		if !errors.Is(err, types.ErrExceedsMaxValidatorPowerShare) {
			return stakingtypes.Validator{}, err
		}
	}

	return k.pickUnderCapBondedValidator(ctx, amount, maxShare, projectedBondedDelta)
}
