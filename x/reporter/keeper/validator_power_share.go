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

// validatorPowerCapState carries the projected bonded-token delta accumulated
// across multiple origins in a single ReturnSlashedTokens/FeeRefund call.
//
// stakingKeeper.Delegate with tokenSrc=Bonded and a bonded destination updates
// the validator's Tokens in the staking store immediately (via
// AddValidatorTokensAndShares) and each origin re-fetches GetValidator, so the
// cap's numerator (validator tokens) already reflects earlier refunds in the
// same call. The denominator, TotalBondedTokens, is the bonded pool's bank
// balance and is NOT updated by Delegate (tokenSrc=Bonded&&IsBonded performs no
// pool transfer); the dispute caller moves those coins only after the reporter
// keeper returns. Without projecting the bonded delta into the denominator,
// later origins are checked against a stale, too-small total and a destination's
// real post-refund share is under-counted. Only the denominator is projected
// here; the numerator is left to the fresh per-origin GetValidator read.
type validatorPowerCapState struct {
	bondedDelta math.Int
}

func newValidatorPowerCapState() *validatorPowerCapState {
	return &validatorPowerCapState{bondedDelta: math.ZeroInt()}
}

// applyBondedDelegation records a bonded-pool delegation of amount so later
// origins in the same call are checked against the projected total bonded.
func (s *validatorPowerCapState) applyBondedDelegation(amount math.Int) {
	if amount.IsNil() || !amount.IsPositive() {
		return
	}
	s.bondedDelta = s.bondedDelta.Add(amount)
}

// checkValidatorPowerShareDelegationWithProjection is the cumulative-aware core
// of the validator acquisition cap. It rejects a bonded delegation when the
// validator's post-delegation share of projected total bonded stake is strictly
// above max_validator_power_share. A nil/zero amount, a non-bonded destination,
// and nil/zero/>=1 params all short-circuit. Callers that must preserve the
// validator acquisition invariant should choose a bonded destination before
// calling this helper. capState may be nil, in which case no cross-origin
// projection is applied (single-call semantics).
func (k Keeper) checkValidatorPowerShareDelegationWithProjection(
	ctx context.Context,
	validator stakingtypes.Validator,
	amount math.Int,
	capState *validatorPowerCapState,
) error {
	if amount.IsNil() || !amount.IsPositive() {
		return nil
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}

	maxShare := params.MaxValidatorPowerShare
	if maxShare.IsNil() || !maxShare.IsPositive() || maxShare.GTE(math.LegacyOneDec()) {
		return nil
	}

	// Direct keeper enforcement only checks immediate active bonded acquisition.
	// Callers that redirect not-bonded originals must do so before this point
	// by selecting a bonded fallback destination.
	if !validator.IsBonded() {
		return nil
	}

	currentTotalBonded, err := k.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return err
	}

	projectedBondedDelta := math.ZeroInt()
	if capState != nil {
		projectedBondedDelta = capState.bondedDelta
	}
	totalBondedAfter := currentTotalBonded.Add(projectedBondedDelta).Add(amount)
	if !totalBondedAfter.IsPositive() {
		return nil
	}

	// validator.Tokens is re-fetched per origin by the callers, so it already
	// reflects earlier refunds in this call; no per-validator projection here.
	validatorAfter := validator.Tokens.Add(amount)
	maxAllowed := maxShare.MulInt(totalBondedAfter)
	if validatorAfter.ToLegacyDec().GT(maxAllowed) {
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
// stake-change tracker. It is the single-call entry point with no cross-origin
// projection (used by WithdrawTip and external callers). Looping callers
// (ReturnSlashedTokens/FeeRefund) thread a capState through the unexported
// helpers so the denominator reflects all in-flight bonded refunds.
func (k Keeper) CheckValidatorPowerShareDelegation(ctx context.Context, validator stakingtypes.Validator, amount math.Int) error {
	return k.checkValidatorPowerShareDelegationWithProjection(ctx, validator, amount, nil)
}

// pickUnderCapBondedValidator scans a bounded list of bonded validators in
// staking power-store order and returns the first one that can accept the
// delegation under the validator cap. It never splits amounts and never indexes
// an empty result. If every scanned validator fails the cap, it returns
// ErrExceedsMaxValidatorPowerShare; other errors propagate. capState threads
// the projected bonded delta across origins in the same call.
func (k Keeper) pickUnderCapBondedValidator(ctx context.Context, amount math.Int, capState *validatorPowerCapState) (stakingtypes.Validator, error) {
	vals, err := k.GetBondedValidators(ctx, ValidatorPowerFallbackScanLimit)
	if err != nil {
		return stakingtypes.Validator{}, err
	}
	if len(vals) == 0 {
		return stakingtypes.Validator{}, errorsmod.Wrap(types.ErrExceedsMaxValidatorPowerShare, "no bonded validator available for delegation")
	}

	for _, val := range vals {
		err := k.checkValidatorPowerShareDelegationWithProjection(ctx, val, amount, capState)
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
// bonded validator. Non-validator-cap errors propagate. capState threads the
// projected bonded delta across origins in the same call.
func (k Keeper) bondedValidatorForDelegation(ctx context.Context, preferred stakingtypes.Validator, amount math.Int, capState *validatorPowerCapState) (stakingtypes.Validator, error) {
	if preferred.IsBonded() {
		err := k.checkValidatorPowerShareDelegationWithProjection(ctx, preferred, amount, capState)
		if err == nil {
			return preferred, nil
		}
		if !errors.Is(err, types.ErrExceedsMaxValidatorPowerShare) {
			return stakingtypes.Validator{}, err
		}
	}

	return k.pickUnderCapBondedValidator(ctx, amount, capState)
}
