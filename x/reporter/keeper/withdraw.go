package keeper

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	layertypes "github.com/tellor-io/layer/types"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type selectorShares struct {
	valAddr   sdk.ValAddress
	validator stakingtypes.Validator
	shares    math.LegacyDec
}
type selectorsInfo struct {
	delAddr             sdk.AccAddress
	selectorTotalTokens math.LegacyDec
	selectorInfo        []selectorShares
}

const feePaidFromStakePayerKeyPrefix = "fee-paid-from-stake-payer\x00"

func feePaidFromStakePayerKey(hashId []byte, payer sdk.AccAddress) []byte {
	key := make([]byte, 0, len(feePaidFromStakePayerKeyPrefix)+4+len(hashId)+len(payer))
	key = append(key, feePaidFromStakePayerKeyPrefix...)
	key = binary.BigEndian.AppendUint32(key, uint32(len(hashId)))
	key = append(key, hashId...)
	key = append(key, payer...)
	return key
}

func (k Keeper) FeePaidFromStakeByPayer(ctx context.Context, hashId []byte, payer sdk.AccAddress) (types.DelegationsAmounts, error) {
	return k.FeePaidFromStake.Get(ctx, feePaidFromStakePayerKey(hashId, payer))
}

func (k Keeper) HasFeePaidFromStakeByPayer(ctx context.Context, hashId []byte, payer sdk.AccAddress) (bool, error) {
	return k.FeePaidFromStake.Has(ctx, feePaidFromStakePayerKey(hashId, payer))
}

func (k Keeper) DisputedDelegationTotal(ctx context.Context, hashId []byte) (math.Int, error) {
	snapshot, err := k.DisputedDelegationAmounts.Get(ctx, hashId)
	if err != nil {
		return math.ZeroInt(), err
	}
	return snapshot.Total, nil
}

// FeefromReporterStake enables a reporter to pay a dispute fee from their stake power.
// hashId is the dispute identifier, needed in the case where a reporter's fee is returned when a dispute is invalid.
func (k Keeper) FeefromReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, amt math.Int, hashId []byte, isFirstRound bool) error {
	reporterTotalTokens := math.LegacyZeroDec()
	fee := math.LegacyNewDecFromInt(amt)

	// Get all selectors for the reporter
	iter, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, reporterAddr)
	if err != nil {
		return err
	}
	defer iter.Close()

	selectorsList := make([]selectorsInfo, 0)
	// calculate total tokens for the reporter by summing up the total tokens of all selectors
	for ; iter.Valid(); iter.Next() {
		selectorKey, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		selectorAddr := sdk.AccAddress(selectorKey)
		// Initialize variables for the current selector
		selectorSharesList := make([]selectorShares, 0)
		selectorTotalTokens := math.LegacyZeroDec()

		// Iterate through delegations for the selector
		err = k.stakingKeeper.IterateDelegatorDelegations(ctx, selectorAddr, func(delegation stakingtypes.Delegation) bool {
			valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
			if err != nil {
				return true
			}
			validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
			if err != nil {
				return true
			}
			if validator.IsBonded() {
				delTokens := validator.TokensFromShares(delegation.Shares)
				selectorTotalTokens = selectorTotalTokens.Add(delTokens)
				selectorSharesList = append(selectorSharesList,
					selectorShares{valAddr: valAddr, validator: validator, shares: delegation.Shares})
			}
			return false
		})
		if err != nil {
			return err
		}

		// Accumulate total tokens for the reporter
		reporterTotalTokens = reporterTotalTokens.Add(selectorTotalTokens)
		selectorsList = append(selectorsList, selectorsInfo{
			delAddr:             selectorAddr,
			selectorTotalTokens: selectorTotalTokens,
			selectorInfo:        selectorSharesList,
		})
	}

	// Check if reporter has enough stake to cover the fee
	if fee.GT(reporterTotalTokens) {
		return errors.New("insufficient stake to pay fee")
	}

	feeTracker := make([]*types.TokenOriginInfo, 0)
	totalTrackedAmount := math.ZeroInt()

	// Process fee payment by unbonding shares from selectors' stake
	// undelegate a proportional amount of tokens from each selector
	for _, selectors := range selectorsList {
		feeShareAmt := selectors.selectorTotalTokens.Quo(reporterTotalTokens).Mul(fee)
		unbondAmt := feeShareAmt

		for _, info := range selectors.selectorInfo {
			// convert shares to token amount
			stakeWithValidator := info.validator.TokensFromShares(info.shares)
			// if selectors stake meets their share of the fee then unbond the amount and break
			if stakeWithValidator.GTE(unbondAmt) {
				sharesToUnbond, err := info.validator.SharesFromTokens(unbondAmt.TruncateInt())
				if err != nil {
					return err
				}
				// Unbond and move tokens out of validator
				escrowedAmt, err := k.stakingKeeper.Unbond(ctx, selectors.delAddr, info.valAddr, sharesToUnbond)
				if err != nil {
					return err
				}
				sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
					sdk.NewEvent(
						"deducted_stake_in_dispute",
						sdk.NewAttribute("delegator", selectors.delAddr.String()),
						sdk.NewAttribute("validator", info.valAddr.String()),
						sdk.NewAttribute("shares", sharesToUnbond.String()),
						sdk.NewAttribute("amount", escrowedAmt.String()),
					),
				})
				feeTracker = append(feeTracker, &types.TokenOriginInfo{
					DelegatorAddress: selectors.delAddr.Bytes(),
					ValidatorAddress: info.valAddr.Bytes(),
					Amount:           escrowedAmt,
				})
				totalTrackedAmount = totalTrackedAmount.Add(escrowedAmt)
				break
			} else {
				// Unbond all shares if not enough stake with the current validator then move to the next validator
				unbondAmt = unbondAmt.Sub(stakeWithValidator)
				escrowedAmt, err := k.stakingKeeper.Unbond(ctx, selectors.delAddr, info.valAddr, info.shares)
				if err != nil {
					return err
				}
				sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
					sdk.NewEvent(
						"deducted_stake_in_dispute",
						sdk.NewAttribute("delegator", selectors.delAddr.String()),
						sdk.NewAttribute("validator", info.validator.OperatorAddress),
						sdk.NewAttribute("shares", info.shares.String()),
						sdk.NewAttribute("amount", escrowedAmt.String()),
					),
				})
				feeTracker = append(feeTracker, &types.TokenOriginInfo{
					DelegatorAddress: selectors.delAddr.Bytes(),
					ValidatorAddress: info.valAddr.Bytes(),
					Amount:           escrowedAmt,
				})
				totalTrackedAmount = totalTrackedAmount.Add(escrowedAmt)

				if unbondAmt.IsZero() {
					break
				}

			}
		}

	}
	prevTotal := math.ZeroInt()
	trackerKey := feePaidFromStakePayerKey(hashId, reporterAddr)
	if isFirstRound {
		hasPaid, err := k.FeePaidFromStake.Has(ctx, trackerKey)
		if err != nil {
			return err
		}
		if hasPaid {
			prevFeeTracker, err := k.FeePaidFromStake.Get(ctx, trackerKey)
			if err != nil {
				return err
			}
			feeTracker = append(feeTracker, prevFeeTracker.TokenOrigins...)
			prevTotal = prevFeeTracker.Total
		}
	}

	// move the tokens from the bonded pool (in staking module) to the dispute module
	if err := k.tokensToDispute(ctx, stakingtypes.BondedPoolName, totalTrackedAmount); err != nil {
		return err
	}
	// Only track the fee if this is round 1
	if isFirstRound {
		if err := k.FeePaidFromStake.Set(ctx, trackerKey, types.DelegationsAmounts{
			TokenOrigins: feeTracker,
			Total:        totalTrackedAmount.Add(prevTotal),
		}); err != nil {
			return err
		}
	}
	return nil
}

// EscrowReporterStake moves tokens from the reporter's stake (from staking module) to the dispute module
func (k Keeper) EscrowReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, power, height uint64, amt math.Int, queryId, hashId []byte) error {
	report, err := k.GetDelegationsAmount(ctx, reporterAddr, height)
	if err != nil {
		return err
	}

	allocations, err := reporterSlashAmounts(report, amt)
	if err != nil {
		return err
	}

	disputeTokens := make([]*types.TokenOriginInfo, 0)
	collected := math.ZeroInt()
	// loop through the selectors' tokens (validator, amount) that were part of the report and remove tokens from relevant delegations
	// amount should be proportional to the total tokens the reporter had at the time of the report
	for i, del := range report.TokenOrigins {
		delegatorShare := allocations[i]
		delAddr := sdk.AccAddress(del.DelegatorAddress)
		valAddr := sdk.ValAddress(del.ValidatorAddress)

		collectedFirst, err := k.undelegate(ctx, delAddr, valAddr, delegatorShare)
		if err != nil {
			return err
		}

		if collectedFirst.IsPositive() {
			disputeTokens = append(disputeTokens, &types.TokenOriginInfo{
				DelegatorAddress: del.DelegatorAddress,
				ValidatorAddress: del.ValidatorAddress,
				Amount:           collectedFirst,
			})
			collected = collected.Add(collectedFirst)
		}

		remaining := delegatorShare.Sub(collectedFirst)
		if remaining.IsPositive() {
			// collect at every redelegation hop; trails record destinations, not residual amounts
			path, err := k.getRedelegationPath(ctx, delAddr, valAddr)
			if err != nil {
				return err
			}
			for _, dstVal := range path {
				collectedHop, err := k.undelegate(ctx, delAddr, dstVal, remaining)
				if err != nil {
					return err
				}
				if collectedHop.IsPositive() {
					disputeTokens = append(disputeTokens, &types.TokenOriginInfo{
						DelegatorAddress: del.DelegatorAddress,
						ValidatorAddress: dstVal,
						Amount:           collectedHop,
					})
					collected = collected.Add(collectedHop)
				}
				remaining = remaining.Sub(collectedHop)
				if remaining.IsZero() {
					break
				}
			}
		}
	}

	// Total is coins actually escrowed, not the intended slash amount
	return k.DisputedDelegationAmounts.Set(ctx, hashId, types.DelegationsAmounts{TokenOrigins: disputeTokens, Total: collected})
}

func reporterSlashAmounts(report types.DelegationsAmounts, amt math.Int) ([]math.Int, error) {
	if amt.IsNil() || amt.IsNegative() {
		return nil, fmt.Errorf("reporter slash amount must be nonnegative")
	}
	if report.Total.IsNil() || report.Total.IsNegative() {
		return nil, fmt.Errorf("%w: reporter snapshot total must be nonnegative", types.ErrMalformedDelegationSnapshot)
	}

	originTotal := math.ZeroInt()
	for i, origin := range report.TokenOrigins {
		if origin == nil || origin.Amount.IsNil() || origin.Amount.IsNegative() {
			return nil, fmt.Errorf("%w: invalid token origin at index %d", types.ErrMalformedDelegationSnapshot, i)
		}
		originTotal = originTotal.Add(origin.Amount)
	}
	if !originTotal.Equal(report.Total) {
		return nil, fmt.Errorf("%w: reporter snapshot total %s does not match token origin total %s", types.ErrMalformedDelegationSnapshot, report.Total, originTotal)
	}
	if amt.IsPositive() && !report.Total.IsPositive() {
		return nil, fmt.Errorf("%w: cannot allocate positive slash from empty reporter snapshot", types.ErrMalformedDelegationSnapshot)
	}
	if amt.IsZero() {
		return make([]math.Int, len(report.TokenOrigins)), nil
	}

	// Use the same largest-remainder integer allocation as stake returns. It is
	// deterministic in token-origin order and assigns every unit without ever
	// producing a negative share.
	allocations := proportionalReturnAmounts(report.TokenOrigins, report.Total, amt)
	allocated := math.ZeroInt()
	for i, allocation := range allocations {
		if allocation.IsNil() || allocation.IsNegative() {
			return nil, fmt.Errorf("negative reporter slash allocation at index %d", i)
		}
		allocated = allocated.Add(allocation)
	}
	if !allocated.Equal(amt) {
		return nil, fmt.Errorf("reporter slash allocations %s do not equal requested amount %s", allocated, amt)
	}
	return allocations, nil
}

// RedelegationRecordGas charges the explicit CPU work needed to filter and
// decode each redelegation record returned by the staking keeper.
const RedelegationRecordGas = storetypes.Gas(10_000)

const redelegationRecordGasMessage = "reporter escrow redelegation record check"

// getRedelegationPath returns reachable redelegation destinations from valAddr
// in deterministic BFS order. One delegator-scoped staking-store pass builds
// source adjacency in iterator order; cycles are skipped in memory.
func (k Keeper) getRedelegationPath(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) ([]sdk.ValAddress, error) {
	adjacency := make(map[string][]sdk.ValAddress)
	gasMeter := sdk.UnwrapSDKContext(ctx).GasMeter()
	var callbackErr error
	err := k.stakingKeeper.IterateDelegatorRedelegations(ctx, delAddr, func(red stakingtypes.Redelegation) bool {
		gasMeter.ConsumeGas(RedelegationRecordGas, redelegationRecordGasMessage)
		src, err := sdk.ValAddressFromBech32(red.ValidatorSrcAddress)
		if err != nil {
			callbackErr = err
			return true
		}
		dst, err := sdk.ValAddressFromBech32(red.ValidatorDstAddress)
		if err != nil {
			callbackErr = err
			return true
		}
		adjacency[src.String()] = append(adjacency[src.String()], dst)
		return false
	})
	if err != nil {
		return nil, err
	}
	if callbackErr != nil {
		return nil, callbackErr
	}

	queue := []sdk.ValAddress{valAddr}
	seen := map[string]struct{}{valAddr.String(): {}}
	path := make([]sdk.ValAddress, 0)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacency[current.String()] {
			if _, ok := seen[next.String()]; ok {
				continue
			}
			seen[next.String()] = struct{}{}
			path = append(path, next)
			queue = append(queue, next)
		}
	}
	return path, nil
}

// chases after unbonding delegations in order to get tokens that are part a new dispute
func (k Keeper) deductUnbondingDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, owed math.Int) (math.Int, error) {
	if !owed.IsPositive() {
		return math.ZeroInt(), nil
	}
	ubd, err := k.stakingKeeper.GetUnbondingDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return math.Int{}, err
	}
	if len(ubd.Entries) == 0 {
		return math.Int{}, types.ErrNoUnbondingDelegationEntries
	}

	remaining := owed
	collected := math.ZeroInt()
	var keptEntries []stakingtypes.UnbondingDelegationEntry
	for _, entry := range ubd.Entries {
		if !remaining.IsPositive() {
			if entry.Balance.IsPositive() {
				keptEntries = append(keptEntries, entry)
			}
			continue
		}

		if entry.Balance.LTE(remaining) {
			collected = collected.Add(entry.Balance)
			remaining = remaining.Sub(entry.Balance)
			continue
		}

		entry.Balance = entry.Balance.Sub(remaining)
		entry.InitialBalance = entry.InitialBalance.Sub(remaining)
		collected = collected.Add(remaining)
		remaining = math.ZeroInt()
		if entry.Balance.IsPositive() {
			keptEntries = append(keptEntries, entry)
		}
	}
	ubd.Entries = keptEntries

	if len(ubd.Entries) == 0 {
		err = k.stakingKeeper.RemoveUnbondingDelegation(ctx, ubd)
	} else {
		err = k.stakingKeeper.SetUnbondingDelegation(ctx, ubd)
	}
	if err != nil {
		return math.Int{}, err
	}

	if collected.IsPositive() {
		err = k.tokensToDispute(ctx, stakingtypes.NotBondedPoolName, collected)
		if err != nil {
			return math.Int{}, err
		}
	}
	return collected, nil
}

func (k Keeper) deductFromdelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, owed math.Int) (math.Int, error) {
	if !owed.IsPositive() {
		return math.ZeroInt(), nil
	}

	// get delegation
	del, err := k.stakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		if errors.Is(err, stakingtypes.ErrNoDelegation) {
			return math.ZeroInt(), nil
		}
		return math.Int{}, err
	}
	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return math.Int{}, err
	}

	// convert current delegation shares to tokens
	currentTokens := validator.TokensFromShares(del.Shares)
	shares := del.Shares
	if currentTokens.GTE(math.LegacyNewDecFromInt(owed)) {
		shares, err = validator.SharesFromTokens(owed)
		if err != nil {
			return math.Int{}, err
		}
	}

	if !shares.IsZero() {
		removedTokens, err := k.stakingKeeper.Unbond(ctx, delAddr, valAddr, shares)
		if err != nil {
			return math.Int{}, err
		}
		if removedTokens.GT(owed) {
			return math.Int{}, fmt.Errorf("collected delegation amount %s exceeds owed amount %s", removedTokens, owed)
		}
		// TODO: emit event for unbonding
		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				"deducted_stake_in_dispute",
				sdk.NewAttribute("delegator", delAddr.String()),
				sdk.NewAttribute("validator", valAddr.String()),
				sdk.NewAttribute("shares", shares.String()),
				sdk.NewAttribute("amount", removedTokens.String()),
			),
		})
		if removedTokens.IsPositive() {
			err = k.MoveTokensFromValidator(ctx, validator, removedTokens)
			if err != nil {
				return math.Int{}, err
			}
		}
		return removedTokens, nil
	}
	return math.ZeroInt(), nil
}

func (k Keeper) MoveTokensFromValidator(ctx context.Context, validator stakingtypes.Validator, amount math.Int) error {
	var fromPool string
	switch {
	case validator.IsBonded():
		fromPool = stakingtypes.BondedPoolName
	case validator.IsUnbonding():
		fromPool = stakingtypes.NotBondedPoolName
	default:
		return fmt.Errorf("unknown validator status: %s", validator.GetStatus())
	}
	return k.tokensToDispute(ctx, fromPool, amount)
}

func (k Keeper) tokensToDispute(ctx context.Context, fromPool string, amount math.Int) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromPool, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, amount)))
}

// undelegate a selector's tokens that are part of a dispute.
// first attempt to get the tokens from known validator and if not found then chase after the tokens that were either redelegated to another validator
// or are being unbonded
func (k Keeper) undelegate(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, owed math.Int) (math.Int, error) {
	collected, err := k.deductFromdelegation(ctx, delAddr, valAddr, owed)
	if err != nil {
		return math.Int{}, err
	}
	// if tokens are still remaining after removing from delegation, then it could be one of two cases
	// the delegator is unbonding or the delegator has redelegated to another validator
	remaining := owed.Sub(collected)
	if !remaining.IsPositive() {
		return collected, nil
	}

	collectedUnbonding, err := k.deductUnbondingDelegation(ctx, delAddr, valAddr, remaining)
	if err != nil {
		if errors.Is(err, stakingtypes.ErrNoUnbondingDelegation) {
			return collected, nil
		}
		return math.Int{}, err
	}
	return collected.Add(collectedUnbonding), nil
}
