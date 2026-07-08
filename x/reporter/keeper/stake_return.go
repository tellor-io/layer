package keeper

import (
	"context"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type stakeReturnRequest struct {
	delAddr                  sdk.AccAddress
	overflowValAddr          sdk.ValAddress
	amount                   math.Int
	preferred                *stakingtypes.Validator
	maxValidatorShare        math.LegacyDec
	maxReporterShare         math.LegacyDec
	projectedBondedDelta     math.Int
	preserveUnbondedOriginal bool
	enforceUBDMaxEntries     bool
}

func (k Keeper) ReturnSlashedTokens(ctx context.Context, hashId []byte, extraReturn math.Int) (math.Int, math.Int, error) {
	bondedReturn := math.ZeroInt()
	unbondedReturn := math.ZeroInt()
	snapshot, err := k.DisputedDelegationAmounts.Get(ctx, hashId)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	if snapshot.Total.IsZero() {
		if extraReturn.IsPositive() {
			return math.ZeroInt(), math.ZeroInt(),
				fmt.Errorf("%w (hashId %x, extraReturn %s)", types.ErrReturnExtraWithoutPrincipal, hashId, extraReturn)
		}
		return math.ZeroInt(), math.ZeroInt(), k.DisputedDelegationAmounts.Remove(ctx, hashId)
	}

	if snapshot.Total.IsPositive() && len(snapshot.TokenOrigins) == 0 {
		return math.ZeroInt(), math.ZeroInt(),
			fmt.Errorf("%w (hashId %x, total %s)", types.ErrMalformedDelegationSnapshot, hashId, snapshot.Total)
	}

	maxShare, maxReporterShare, err := k.maxPowerShares(ctx)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}
	projectedBondedDelta := math.ZeroInt()

	returnTotal := snapshot.Total.Add(extraReturn)
	returnAmounts := make([]math.Int, len(snapshot.TokenOrigins))
	if extraReturn.IsPositive() {
		returnAmounts = proportionalReturnAmounts(snapshot.TokenOrigins, snapshot.Total, returnTotal)
	} else {
		for i, source := range snapshot.TokenOrigins {
			returnAmounts[i] = source.Amount
		}
	}
	for i, source := range snapshot.TokenOrigins {
		valAddr := sdk.ValAddress(source.ValidatorAddress)
		delAddr := sdk.AccAddress(source.DelegatorAddress)
		bonded, unbonded, nextProjected, err := k.returnStakeAmount(
			ctx,
			stakeReturnRequest{
				delAddr:                  delAddr,
				overflowValAddr:          valAddr,
				amount:                   returnAmounts[i],
				maxValidatorShare:        maxShare,
				maxReporterShare:         maxReporterShare,
				projectedBondedDelta:     projectedBondedDelta,
				preserveUnbondedOriginal: true,
			},
			valAddr,
		)
		if err != nil {
			return math.ZeroInt(), math.ZeroInt(), err
		}
		bondedReturn = bondedReturn.Add(bonded)
		unbondedReturn = unbondedReturn.Add(unbonded)
		projectedBondedDelta = nextProjected
	}
	return bondedReturn, unbondedReturn, k.DisputedDelegationAmounts.Remove(ctx, hashId)
}

func proportionalReturnAmounts(origins []*types.TokenOriginInfo, total, returnTotal math.Int) []math.Int {
	if len(origins) == 0 {
		return nil
	}

	type share struct {
		amount    math.Int
		remainder math.Int
	}
	shares := make([]share, len(origins))
	distributed := math.ZeroInt()
	for i, source := range origins {
		scaled := source.Amount.Mul(returnTotal)
		shares[i] = share{
			amount:    scaled.Quo(total),
			remainder: scaled.Mod(total),
		}
		distributed = distributed.Add(shares[i].amount)
	}

	for dust := returnTotal.Sub(distributed); dust.IsPositive(); dust = dust.Sub(math.OneInt()) {
		best := -1
		for i := range shares {
			if !shares[i].remainder.IsPositive() {
				continue
			}
			if best == -1 || shares[i].remainder.GT(shares[best].remainder) {
				best = i
			}
		}
		if best == -1 {
			shares[0].amount = shares[0].amount.Add(dust)
			break
		}
		shares[best].amount = shares[best].amount.Add(math.OneInt())
		shares[best].remainder = math.ZeroInt()
	}

	out := make([]math.Int, len(shares))
	for i, s := range shares {
		out[i] = s.amount
	}
	return out
}

func (k Keeper) returnViaUnbonding(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amount math.Int, enforceMaxEntries bool) error {
	if enforceMaxEntries {
		hasMaxEntries, err := k.stakingKeeper.HasMaxUnbondingDelegationEntries(ctx, delAddr, valAddr)
		if err != nil {
			return err
		}
		if hasMaxEntries {
			return stakingtypes.ErrMaxUnbondingDelegationEntries
		}
	}
	unbondingTime, err := k.stakingKeeper.UnbondingTime(ctx)
	if err != nil {
		return err
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	completionTime := sdkCtx.BlockHeader().Time.Add(unbondingTime)
	ubd, err := k.stakingKeeper.SetUnbondingDelegationEntry(ctx, delAddr, valAddr, sdkCtx.BlockHeight(), completionTime, amount)
	if err != nil {
		return err
	}
	return k.stakingKeeper.InsertUBDQueue(ctx, ubd, completionTime)
}

func (k Keeper) bondValidatorUpToHeadroom(
	ctx context.Context,
	req stakeReturnRequest,
	validator stakingtypes.Validator,
	ubdKey sdk.ValAddress,
) (bonded, unbonded, projected math.Int, err error) {
	bonded = math.ZeroInt()
	unbonded = math.ZeroInt()
	projected = req.projectedBondedDelta
	if req.amount.IsNil() || !req.amount.IsPositive() {
		return math.ZeroInt(), math.ZeroInt(), projected, nil
	}
	if len(ubdKey) == 0 {
		return math.ZeroInt(), math.ZeroInt(), math.Int{}, errors.New("no validator available for unbonding overflow key")
	}

	headroom, err := k.bondedReturnHeadroom(ctx, req.delAddr, validator, req.amount, req.maxValidatorShare, req.maxReporterShare, projected)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), math.Int{}, err
	}

	if headroom.IsPositive() {
		if _, err := k.delegateToStakeAndEmit(ctx, req.delAddr, headroom, stakingtypes.Bonded, validator); err != nil {
			return math.ZeroInt(), math.ZeroInt(), math.Int{}, err
		}
		bonded = headroom
		projected = projected.Add(headroom)
	}
	if overflow := req.amount.Sub(headroom); overflow.IsPositive() {
		if err := k.returnViaUnbonding(ctx, req.delAddr, ubdKey, overflow, req.enforceUBDMaxEntries); err != nil {
			return math.ZeroInt(), math.ZeroInt(), math.Int{}, err
		}
		unbonded = overflow
	}
	return bonded, unbonded, projected, nil
}

func (k Keeper) delegateBondedWithOverflow(ctx context.Context, req stakeReturnRequest) (bonded, unbonded, projected math.Int, err error) {
	projected = req.projectedBondedDelta
	if req.amount.IsNil() || !req.amount.IsPositive() {
		return math.ZeroInt(), math.ZeroInt(), projected, nil
	}

	if req.preferred != nil {
		headroom, err := k.bondedReturnHeadroom(ctx, req.delAddr, *req.preferred, req.amount, req.maxValidatorShare, req.maxReporterShare, projected)
		if err != nil {
			return math.ZeroInt(), math.ZeroInt(), math.Int{}, err
		}
		if headroom.IsPositive() {
			ubdKey := req.overflowValAddr
			if len(ubdKey) == 0 {
				ubdKey, err = validatorAddress(*req.preferred)
				if err != nil {
					return math.ZeroInt(), math.ZeroInt(), math.Int{}, err
				}
			}
			return k.bondValidatorUpToHeadroom(ctx, req, *req.preferred, ubdKey)
		}
	}

	scan, err := k.scanBondedHeadroom(ctx, req)
	if err != nil {
		if errors.Is(err, types.ErrExceedsMaxValidatorPowerShare) && len(req.overflowValAddr) > 0 {
			if err := k.returnViaUnbonding(ctx, req.delAddr, req.overflowValAddr, req.amount, req.enforceUBDMaxEntries); err != nil {
				return math.ZeroInt(), math.ZeroInt(), math.Int{}, err
			}
			return math.ZeroInt(), req.amount, projected, nil
		}
		return math.ZeroInt(), math.ZeroInt(), math.Int{}, err
	}

	ubdKey := req.overflowValAddr
	if len(ubdKey) == 0 {
		ubdKey = scan.ubdKey
	}
	return k.bondValidatorUpToHeadroom(ctx, req, scan.validator, ubdKey)
}

func (k Keeper) returnStakeAmount(ctx context.Context, req stakeReturnRequest, originalValAddr sdk.ValAddress) (bonded, unbonded, projected math.Int, err error) {
	projected = req.projectedBondedDelta
	if req.amount.IsNil() || !req.amount.IsPositive() {
		return math.ZeroInt(), math.ZeroInt(), projected, nil
	}
	original, getErr := k.stakingKeeper.GetValidator(ctx, originalValAddr)
	switch {
	case getErr != nil && !errors.Is(getErr, stakingtypes.ErrNoValidatorFound):
		return math.ZeroInt(), math.ZeroInt(), math.Int{}, getErr
	case getErr == nil && !original.IsBonded() && req.preserveUnbondedOriginal && !hasInvalidExchangeRate(original):
		if _, err := k.delegateToStakeAndEmit(ctx, req.delAddr, req.amount, stakingtypes.Unbonded, original); err != nil {
			return math.ZeroInt(), math.ZeroInt(), math.Int{}, err
		}
		return math.ZeroInt(), req.amount, projected, nil
	}

	if getErr == nil && original.IsBonded() && !hasInvalidExchangeRate(original) {
		req.preferred = &original
	}
	return k.delegateBondedWithOverflow(ctx, req)
}

func (k Keeper) delegateToStakeAndEmit(ctx context.Context, delegator sdk.AccAddress, amount math.Int, tokenSrc stakingtypes.BondStatus, validator stakingtypes.Validator) (math.LegacyDec, error) {
	newShares, err := k.stakingKeeper.Delegate(ctx, delegator, amount, tokenSrc, validator, false)
	if err != nil {
		return math.LegacyZeroDec(), err
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"tokens_added_to_stake",
			sdk.NewAttribute("delegator", delegator.String()),
			sdk.NewAttribute("validator", validator.OperatorAddress),
			sdk.NewAttribute("shares", newShares.String()),
			sdk.NewAttribute("amount", amount.String()),
		),
	})
	return newShares, nil
}

func (k Keeper) FeeRefund(ctx context.Context, hashId []byte, amt math.Int) (math.Int, math.Int, error) {
	bondedReturn := math.ZeroInt()
	unbondedReturn := math.ZeroInt()
	trackedFees, err := k.FeePaidFromStake.Get(ctx, hashId)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	maxShare, maxReporterShare, err := k.maxPowerShares(ctx)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}
	projectedBondedDelta := math.ZeroInt()

	for _, source := range trackedFees.TokenOrigins {
		shareAmt := math.LegacyNewDecFromInt(source.Amount).Mul(math.LegacyNewDecFromInt(amt)).Quo(math.LegacyNewDecFromInt(trackedFees.Total)).TruncateInt()
		valAddr := sdk.ValAddress(source.ValidatorAddress)
		bonded, unbonded, nextProjected, err := k.returnStakeAmount(
			ctx,
			stakeReturnRequest{
				delAddr:              sdk.AccAddress(source.DelegatorAddress),
				overflowValAddr:      valAddr,
				amount:               shareAmt,
				maxValidatorShare:    maxShare,
				maxReporterShare:     maxReporterShare,
				projectedBondedDelta: projectedBondedDelta,
				enforceUBDMaxEntries: true,
			},
			valAddr,
		)
		if err != nil {
			return math.ZeroInt(), math.ZeroInt(), err
		}
		bondedReturn = bondedReturn.Add(bonded)
		unbondedReturn = unbondedReturn.Add(unbonded)
		projectedBondedDelta = nextProjected
	}
	if err := k.FeePaidFromStake.Remove(ctx, hashId); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}
	return bondedReturn, unbondedReturn, nil
}

func (k Keeper) AddAmountToStake(ctx context.Context, acc sdk.AccAddress, amt math.Int) (math.Int, math.Int, error) {
	maxShare, maxReporterShare, err := k.maxPowerShares(ctx)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	req := stakeReturnRequest{
		delAddr:              acc,
		amount:               amt,
		maxValidatorShare:    maxShare,
		maxReporterShare:     maxReporterShare,
		projectedBondedDelta: math.ZeroInt(),
		enforceUBDMaxEntries: true,
	}

	var (
		delegated   bool
		callbackErr error
		bonded      math.Int
		unbonded    math.Int
	)

	err = k.stakingKeeper.IterateDelegatorDelegations(ctx, acc, func(delegation stakingtypes.Delegation) (stop bool) {
		if callbackErr != nil {
			return true
		}
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			callbackErr = err
			return true
		}
		val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			callbackErr = err
			return true
		}
		if !val.IsBonded() {
			return false
		}
		req.overflowValAddr = valAddr
		req.preferred = &val
		bonded, unbonded, _, callbackErr = k.delegateBondedWithOverflow(ctx, req)
		delegated = true
		return true
	})
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}
	if callbackErr != nil {
		return math.ZeroInt(), math.ZeroInt(), callbackErr
	}
	if delegated {
		return bonded, unbonded, nil
	}
	bonded, unbonded, _, err = k.delegateBondedWithOverflow(ctx, req)
	return bonded, unbonded, err
}
