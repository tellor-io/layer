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

func hasInvalidExchangeRate(validator stakingtypes.Validator) bool {
	if validator.Tokens.IsNil() || validator.DelegatorShares.IsNil() {
		return false
	}
	return validator.InvalidExRate()
}

func validatorAddress(validator stakingtypes.Validator) (sdk.ValAddress, error) {
	return sdk.ValAddressFromBech32(validator.OperatorAddress)
}

func (k Keeper) maxPowerShares(ctx context.Context) (math.LegacyDec, math.LegacyDec, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return math.LegacyDec{}, math.LegacyDec{}, nil
		}
		return math.LegacyDec{}, math.LegacyDec{}, err
	}
	return params.MaxValidatorPowerShare, params.MaxReporterPowerShare, nil
}

func (k Keeper) maxValidatorPowerShare(ctx context.Context) (math.LegacyDec, error) {
	maxValidatorShare, _, err := k.maxPowerShares(ctx)
	return maxValidatorShare, err
}

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

	validatorAfter := validator.Tokens.Add(amount)
	check := types.ValidatorCapCheck{
		ValidatorTokensAfter: validatorAfter,
		TotalBondedAfter:     totalBondedAfter,
		MaxShare:             maxShare,
	}
	if check.Exceeds() {
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		if err != nil {
			return err
		}
		return errorsmod.Wrapf(types.ErrExceedsMaxValidatorPowerShare, "validator %s", valAddr.String())
	}

	return nil
}

func (k Keeper) CheckValidatorPowerShareDelegation(ctx context.Context, validator stakingtypes.Validator, amount math.Int) error {
	maxShare, err := k.maxValidatorPowerShare(ctx)
	if err != nil {
		return err
	}
	return k.checkValidatorPowerShareDelegation(ctx, validator, amount, maxShare, math.ZeroInt())
}

func (k Keeper) delegatorBondedTokens(ctx context.Context, delegator sdk.AccAddress) (math.LegacyDec, error) {
	tokens := math.LegacyZeroDec()
	var iterError error
	err := k.stakingKeeper.IterateDelegatorDelegations(ctx, delegator, func(delegation stakingtypes.Delegation) (stop bool) {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			iterError = err
			return true
		}
		val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			iterError = err
			return true
		}
		if val.IsBonded() {
			tokens = tokens.Add(val.TokensFromShares(delegation.Shares))
		}
		return false
	})
	if err != nil {
		return math.LegacyDec{}, err
	}
	return tokens, iterError
}

func (k Keeper) selectedReporterForCap(ctx context.Context, delegator sdk.AccAddress) (sdk.AccAddress, bool, error) {
	selection, err := k.GetSelector(ctx, delegator)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	pending, to, err := k.PendingSwitchTarget(ctx, delegator)
	if err != nil {
		return nil, false, err
	}
	if pending {
		return sdk.AccAddress(to), true, nil
	}
	return sdk.AccAddress(selection.Reporter), true, nil
}

func linearCapHeadroomInclusive(held math.LegacyDec, base math.Int, maxShare math.LegacyDec) math.Int {
	if !types.PowerShareEnabled(maxShare) || base.IsNil() || !base.IsPositive() {
		return math.NewIntFromUint64(^uint64(0) >> 1)
	}
	oneMinusShare := math.LegacyOneDec().Sub(maxShare)
	if !oneMinusShare.IsPositive() {
		return math.NewIntFromUint64(^uint64(0) >> 1)
	}
	rhs := maxShare.MulInt(base).Sub(held)
	if !rhs.IsPositive() {
		return math.ZeroInt()
	}
	return rhs.Quo(oneMinusShare).TruncateInt()
}

func linearCapHeadroomStrict(held math.LegacyDec, base math.Int, maxShare math.LegacyDec) math.Int {
	inclusive := linearCapHeadroomInclusive(held, base, maxShare)
	if inclusive.IsZero() {
		return math.ZeroInt()
	}
	total := base.Add(inclusive)
	if held.Add(inclusive.ToLegacyDec()).GTE(maxShare.MulInt(total)) {
		return inclusive.Sub(math.OneInt())
	}
	return inclusive
}

func delegatorCapShare() math.LegacyDec {
	return math.LegacyNewDecWithPrec(30, 2)
}

func (k Keeper) maxBondableAmount(
	ctx context.Context,
	delAddr sdk.AccAddress,
	validator stakingtypes.Validator,
	maxValidatorShare math.LegacyDec,
	maxReporterShare math.LegacyDec,
	projectedBondedDelta math.Int,
) (math.Int, error) {
	if projectedBondedDelta.IsNil() {
		projectedBondedDelta = math.ZeroInt()
	}
	if !types.PowerShareEnabled(maxValidatorShare) && !types.PowerShareEnabled(maxReporterShare) {
		return math.NewIntFromUint64(^uint64(0) >> 1), nil
	}

	currentTotalBonded, err := k.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return math.Int{}, err
	}
	base := currentTotalBonded.Add(projectedBondedDelta)
	if !base.IsPositive() {
		return math.NewIntFromUint64(^uint64(0) >> 1), nil
	}

	max := math.NewIntFromUint64(^uint64(0) >> 1)

	if validator.IsBonded() && types.PowerShareEnabled(maxValidatorShare) {
		headroom := linearCapHeadroomInclusive(validator.Tokens.ToLegacyDec(), base, maxValidatorShare)
		if headroom.LT(max) {
			max = headroom
		}
	}

	delegatorBonded, err := k.delegatorBondedTokens(ctx, delAddr)
	if err != nil {
		return math.Int{}, err
	}
	delegatorHeadroom := linearCapHeadroomInclusive(delegatorBonded, base, delegatorCapShare())
	if delegatorHeadroom.LT(max) {
		max = delegatorHeadroom
	}

	reporter, found, err := k.selectedReporterForCap(ctx, delAddr)
	if err != nil {
		return math.Int{}, err
	}
	if found && types.PowerShareEnabled(maxReporterShare) {
		potential, err := k.ReporterPotentialStake(ctx, reporter)
		if err != nil {
			return math.Int{}, err
		}
		reporterHeadroom := linearCapHeadroomStrict(potential.ToLegacyDec(), base, maxReporterShare)
		if reporterHeadroom.LT(max) {
			max = reporterHeadroom
		}
	}

	return max, nil
}

func (k Keeper) bondedReturnHeadroom(
	ctx context.Context,
	delAddr sdk.AccAddress,
	validator stakingtypes.Validator,
	amount math.Int,
	maxValidatorShare math.LegacyDec,
	maxReporterShare math.LegacyDec,
	projectedBondedDelta math.Int,
) (math.Int, error) {
	if amount.IsNil() || !amount.IsPositive() {
		return math.ZeroInt(), nil
	}
	max, err := k.maxBondableAmount(ctx, delAddr, validator, maxValidatorShare, maxReporterShare, projectedBondedDelta)
	if err != nil {
		return math.Int{}, err
	}
	if max.GT(amount) {
		return amount, nil
	}
	return max, nil
}

type bondedScan struct {
	validator stakingtypes.Validator
	ubdKey    sdk.ValAddress
}

func (k Keeper) scanBondedHeadroom(ctx context.Context, req stakeReturnRequest) (bondedScan, error) {
	vals, err := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return bondedScan{}, err
	}
	if len(vals) == 0 {
		return bondedScan{}, errorsmod.Wrap(types.ErrExceedsMaxValidatorPowerShare, "no bonded validator available for delegation")
	}

	var fallback bondedScan
	for i := len(vals) - 1; i >= 0; i-- {
		val := vals[i]
		if hasInvalidExchangeRate(val) {
			continue
		}
		valAddr, err := validatorAddress(val)
		if err != nil {
			return bondedScan{}, err
		}
		if len(fallback.ubdKey) == 0 {
			fallback = bondedScan{validator: val, ubdKey: valAddr}
		}
		headroom, err := k.bondedReturnHeadroom(ctx, req.delAddr, val, req.amount, req.maxValidatorShare, req.maxReporterShare, req.projectedBondedDelta)
		if err != nil {
			return bondedScan{}, err
		}
		if headroom.IsPositive() {
			return bondedScan{validator: val, ubdKey: valAddr}, nil
		}
	}
	if len(fallback.ubdKey) == 0 {
		return bondedScan{}, errorsmod.Wrap(types.ErrExceedsMaxValidatorPowerShare, "no valid bonded validator available")
	}
	return fallback, nil
}
