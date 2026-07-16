package keeper

import (
	"context"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type stakeReturnPolicy struct {
	preserveUnbondedOriginal bool
	enforceUBDMaxEntries     bool
}

var (
	slashReturnPolicy = stakeReturnPolicy{preserveUnbondedOriginal: true}
	feeReturnPolicy   = stakeReturnPolicy{enforceUBDMaxEntries: true}
)

type stakeReturnRequest struct {
	delAddr              sdk.AccAddress
	overflowValAddr      sdk.ValAddress
	amount               math.Int
	preferred            *stakingtypes.Validator
	maxValidatorShare    math.LegacyDec
	maxReporterShare     math.LegacyDec
	projectedBondedDelta math.Int
	policy               stakeReturnPolicy
}

func (k Keeper) ReturnSlashedTokens(ctx context.Context, hashId []byte, extraReturn math.Int) (math.Int, math.Int, error) {
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

	bondedReturn, unbondedReturn, err := k.returnOrigins(ctx, snapshot.TokenOrigins, snapshot.Total, snapshot.Total.Add(extraReturn), slashReturnPolicy)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}
	return bondedReturn, unbondedReturn, k.DisputedDelegationAmounts.Remove(ctx, hashId)
}

func (k Keeper) returnOrigins(ctx context.Context, origins []*types.TokenOriginInfo, total, returnAmt math.Int, policy stakeReturnPolicy) (bonded, unbonded math.Int, err error) {
	bonded = math.ZeroInt()
	unbonded = math.ZeroInt()
	maxShare, maxReporterShare, err := k.maxPowerShares(ctx)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	returnAmounts := make([]math.Int, len(origins))
	if returnAmt.Equal(total) {
		for i, source := range origins {
			returnAmounts[i] = source.Amount
		}
	} else {
		returnAmounts = proportionalReturnAmounts(origins, total, returnAmt)
	}

	projectedBondedDelta := math.ZeroInt()
	for i, source := range origins {
		valAddr := sdk.ValAddress(source.ValidatorAddress)
		bondedPart, unbondedPart, nextProjected, err := k.returnStakeAmount(
			ctx,
			stakeReturnRequest{
				delAddr:              sdk.AccAddress(source.DelegatorAddress),
				overflowValAddr:      valAddr,
				amount:               returnAmounts[i],
				maxValidatorShare:    maxShare,
				maxReporterShare:     maxReporterShare,
				projectedBondedDelta: projectedBondedDelta,
				policy:               policy,
			},
			valAddr,
		)
		if err != nil {
			return math.ZeroInt(), math.ZeroInt(), err
		}
		bonded = bonded.Add(bondedPart)
		unbonded = unbonded.Add(unbondedPart)
		projectedBondedDelta = nextProjected
	}
	return bonded, unbonded, nil
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
		if err := k.returnViaUnbonding(ctx, req.delAddr, ubdKey, overflow, req.policy.enforceUBDMaxEntries); err != nil {
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
			if err := k.returnViaUnbonding(ctx, req.delAddr, req.overflowValAddr, req.amount, req.policy.enforceUBDMaxEntries); err != nil {
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
	case getErr == nil && !original.IsBonded() && req.policy.preserveUnbondedOriginal && !hasInvalidExchangeRate(original):
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

func (k Keeper) feePaidFromStakeRefundSnapshot(ctx context.Context, hashId []byte, payer sdk.AccAddress) ([]byte, types.DelegationsAmounts, error) {
	payerKey := feePaidFromStakePayerKey(hashId, payer)
	trackedFees, err := k.FeePaidFromStake.Get(ctx, payerKey)
	if err == nil {
		return payerKey, trackedFees, nil
	}
	if !errors.Is(err, collections.ErrNotFound) {
		return nil, types.DelegationsAmounts{}, err
	}

	trackedFees, err = k.FeePaidFromStake.Get(ctx, hashId)
	if err != nil {
		return nil, types.DelegationsAmounts{}, err
	}
	return hashId, trackedFees, nil
}

func (k Keeper) FeePaidFromStakeTotalByPayer(ctx context.Context, hashId []byte, payer sdk.AccAddress) (math.Int, error) {
	_, trackedFees, err := k.feePaidFromStakeRefundSnapshot(ctx, hashId, payer)
	if err != nil {
		return math.ZeroInt(), err
	}
	return trackedFees.Total, nil
}

func (k Keeper) FeeRefund(ctx context.Context, hashId []byte, payer sdk.AccAddress, amt math.Int) (math.Int, math.Int, error) {
	trackerKey, trackedFees, err := k.feePaidFromStakeRefundSnapshot(ctx, hashId, payer)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	originTotal := math.ZeroInt()
	for _, origin := range trackedFees.TokenOrigins {
		originTotal = originTotal.Add(origin.Amount)
	}
	if !originTotal.Equal(trackedFees.Total) {
		return math.ZeroInt(), math.ZeroInt(),
			fmt.Errorf("%w (hashId %x, payer %s, total %s, originTotal %s)", types.ErrMalformedDelegationSnapshot, hashId, payer.String(), trackedFees.Total, originTotal)
	}

	refundAmt := amt
	if trackedFees.Total.LT(refundAmt) {
		refundAmt = trackedFees.Total
	}
	if !trackedFees.Total.IsPositive() || !refundAmt.IsPositive() {
		if err := k.FeePaidFromStake.Remove(ctx, trackerKey); err != nil {
			return math.ZeroInt(), math.ZeroInt(), err
		}
		return math.ZeroInt(), math.ZeroInt(), nil
	}

	bondedReturn, unbondedReturn, err := k.returnOrigins(ctx, trackedFees.TokenOrigins, trackedFees.Total, refundAmt, feeReturnPolicy)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}
	if err := k.FeePaidFromStake.Remove(ctx, trackerKey); err != nil {
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
		policy:               feeReturnPolicy,
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
