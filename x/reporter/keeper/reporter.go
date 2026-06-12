package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// HasMin checks if an AccAddress has the minimum amount of tokens required with a BONDED validator
func (k Keeper) HasMin(ctx context.Context, addr sdk.AccAddress, minRequired math.Int) (bool, error) {
	tokens := math.ZeroInt()
	var iterError error
	err := k.stakingKeeper.IterateDelegatorDelegations(ctx, addr, func(delegation stakingtypes.Delegation) (stop bool) {
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
		if !val.IsBonded() {
			return false
		}
		// convert del shares to token amount
		delTokens := val.TokensFromShares(delegation.Shares).TruncateInt()
		tokens = tokens.Add(delTokens)
		// short circuit if we have enough tokens
		return tokens.GTE(minRequired)
	})
	if err != nil {
		return false, err
	}
	return tokens.GTE(minRequired), iterError
}

// ReporterStake counts the total amount of BONDED tokens for a given reporter's selectors
// at the time of reporting and returns the total amount plus stores
// the token origins for each selector which is needed during a dispute for slashing/returning tokens to appropriate parties.
// It also tracks period data for reward distribution - when delegation state changes,
// the previous period is queued for distribution.
// Ready pending reporter switches involving this reporter are finalized first (same entry path as MsgSubmitValue).
func (k Keeper) ReporterStake(ctx context.Context, repAddr sdk.AccAddress, queryId []byte) (math.Int, error) {
	if err := k.applyReadyPendingSwitchesForReporter(ctx, repAddr); err != nil {
		return math.Int{}, err
	}

	needsRecalc, err := k.needsStakeRecalc(ctx, repAddr)
	if err != nil {
		return math.Int{}, err
	}

	if !needsRecalc {
		// Stake hasn't changed, fetch cached total from last Report entry
		cached, err := k.GetDelegationsAmount(ctx, repAddr.Bytes(), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
		if err != nil {
			return math.Int{}, err
		}
		if !cached.Total.IsNil() && cached.Total.IsPositive() {
			return cached.Total, nil
		}
		// if it ain't positive, just recalculate
	}

	totalTokens, delegates, selectorShares, hash, err := k.GetReporterStake(ctx, repAddr)
	if err != nil {
		return math.Int{}, err
	}

	// Clear stake recalc flag after recalculation
	if err := k.StakeRecalcFlag.Remove(ctx, repAddr.Bytes()); err != nil {
		return math.Int{}, err
	}

	// Handle period tracking for reward distribution
	changed, err := k.handlePeriodTracking(ctx, repAddr, selectorShares, totalTokens, hash)
	if err != nil {
		return math.Int{}, err
	}
	if changed {
		// Store per-report snapshot for disputes
		err = k.ReportByBlock.Set(ctx, collections.Join3(repAddr.Bytes(), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()), queryId), types.DelegationsAmounts{TokenOrigins: delegates, Total: totalTokens})
		if err != nil {
			return math.Int{}, err
		}
	}
	return totalTokens, nil
}

// needsStakeRecalc checks if a reporter's stake needs to be recalculated
func (k Keeper) needsStakeRecalc(ctx context.Context, repAddr sdk.AccAddress) (bool, error) {
	// Check persisted recalc flag (set by hooks/msg handlers)
	flagged, err := k.StakeRecalcFlag.Has(ctx, repAddr.Bytes())
	if err != nil {
		return true, nil
	}
	if flagged {
		return true, nil
	}

	// Check if a selector's switch lock has expired since last calc
	recalcAt, err := k.RecalcAtTime.Get(ctx, repAddr.Bytes())
	if err == nil {
		blockTime := sdk.UnwrapSDKContext(ctx).BlockTime().Unix()
		if recalcAt <= blockTime {
			return true, nil
		}
	}

	// Check if validator set updated since last calculation
	lastCalcBlock, err := k.GetLastReportedAtBlock(ctx, repAddr.Bytes())
	if err != nil {
		return true, nil // means first time for reporter
	}
	if lastCalcBlock == 0 {
		return true, nil // never calculated
	}

	valSetUpdateHeight, err := k.LastValSetUpdateHeight.Get(ctx)
	if err != nil {
		return true, nil // no update height stored yet, recalc to be safe
	}

	return valSetUpdateHeight >= lastCalcBlock, nil
}

// function that iterates through a selector's delegations and checks if they meet the min requirement
// plus counts how many delegations they have
func (k Keeper) CheckSelectorsDelegations(ctx context.Context, addr sdk.AccAddress) (math.Int, int64, error) {
	tokens := math.ZeroInt()
	var count int64
	// todo: is this itererror necessary?
	var iterError error
	err := k.stakingKeeper.IterateDelegatorDelegations(ctx, addr, func(delegation stakingtypes.Delegation) (stop bool) {
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
		count++
		if val.IsBonded() {
			delTokens := val.TokensFromShares(delegation.Shares).TruncateInt()
			tokens = tokens.Add(delTokens)
		}
		return false
	})
	if err != nil {
		return math.Int{}, 0, err
	}
	if iterError != nil {
		return math.Int{}, 0, iterError
	}
	return tokens, count, nil
}

// TotalReporterPower returns the total amount of BONDED tokens in the network
func (k Keeper) TotalReporterPower(ctx context.Context) (math.Int, error) {
	valSet := k.stakingKeeper.GetValidatorSet()
	return valSet.TotalBondedTokens(ctx)
}

// PotentialStakeSelectorGas makes the power-cap selector expansion visible to gas
// accounting on top of normal store-read costs, mirroring the active-set scan
// precedent in the ante decorator.
const PotentialStakeSelectorGas = storetypes.Gas(10_000)

const potentialStakeSelectorGasMessage = "reporter power cap selector check"

// ReporterPotentialStake returns a conservative upper bound on a reporter's
// reporting power, used by the power-cap check: the bonded tokens of every
// selector currently selecting the reporter — including dispute-locked selectors
// (their stake returns when the lock expires) and regardless of reporter jail
// status — excluding selectors with a pending switch away (that stake already
// stopped counting and is committed elsewhere), plus the bonded tokens of
// selectors with a pending switch into the reporter (booked against the cap as
// soon as the switch is scheduled). Read-only: never mutates state.
func (k Keeper) ReporterPotentialStake(ctx context.Context, repAddr sdk.AccAddress) (math.Int, error) {
	gasMeter := sdk.UnwrapSDKContext(ctx).GasMeter()
	total := math.ZeroInt()
	iter, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, repAddr.Bytes())
	if err != nil {
		return math.Int{}, err
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		selectorAddr, err := iter.PrimaryKey()
		if err != nil {
			return math.Int{}, err
		}
		hasPending, err := k.hasOutgoingPendingSwitch(ctx, repAddr.Bytes(), selectorAddr)
		if err != nil {
			return math.Int{}, err
		}
		if hasPending {
			continue
		}
		gasMeter.ConsumeGas(PotentialStakeSelectorGas, potentialStakeSelectorGasMessage)
		bonded, _, err := k.CheckSelectorsDelegations(ctx, sdk.AccAddress(selectorAddr))
		if err != nil {
			return math.Int{}, err
		}
		total = total.Add(bonded)
	}

	inRange := collections.NewPrefixedPairRange[[]byte, []byte](repAddr.Bytes())
	inIter, err := k.IncomingPendingSwitchIdx.Iterate(ctx, inRange)
	if err != nil {
		return math.Int{}, err
	}
	defer inIter.Close()
	for ; inIter.Valid(); inIter.Next() {
		pk, err := inIter.Key()
		if err != nil {
			return math.Int{}, err
		}
		gasMeter.ConsumeGas(PotentialStakeSelectorGas, potentialStakeSelectorGasMessage)
		bonded, _, err := k.CheckSelectorsDelegations(ctx, sdk.AccAddress(pk.K2()))
		if err != nil {
			return math.Int{}, err
		}
		total = total.Add(bonded)
	}
	return total, nil
}

// PendingSwitchTarget returns the reporter a selector's scheduled pending switch
// is headed to, if any. Callers use it to attribute the selector's stake to the
// reporter that will actually receive it and to avoid double-booking re-sent
// switches against the power cap.
func (k Keeper) PendingSwitchTarget(ctx context.Context, selectorAddr sdk.AccAddress) (bool, []byte, error) {
	selection, err := k.Selectors.Get(ctx, selectorAddr.Bytes())
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return k.pendingSwitchToReporter(ctx, sdk.AccAddress(selection.Reporter), selectorAddr)
}

// Delegation returns a selector's reporter, delegations count, and locked time information
func (k Keeper) Delegation(ctx context.Context, delegator sdk.AccAddress) (types.Selection, error) {
	return k.Selectors.Get(ctx, delegator)
}

// Reporter returns a reporter's minimum bond requirement, commission rate, jailed status, and locked time information
func (k Keeper) Reporter(ctx context.Context, reporter sdk.AccAddress) (types.OracleReporter, error) {
	return k.Reporters.Get(ctx, reporter.Bytes())
}

// GetNumOfSelectors returns the number of selectors a reporter currently has
func (k Keeper) GetNumOfSelectors(ctx context.Context, repAddr sdk.AccAddress) (int, error) {
	iter, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, repAddr.Bytes())
	if err != nil {
		return 0, err
	}
	keys, err := iter.FullKeys()
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}

// GetNumOfSelectorsIncludingPendingIncoming returns active selectors plus selectors
// with a pending switch into this reporter that has not yet been finalized.
func (k Keeper) GetNumOfSelectorsIncludingPendingIncoming(ctx context.Context, repAddr sdk.AccAddress) (int, error) {
	count, err := k.GetNumOfSelectors(ctx, repAddr)
	if err != nil {
		return 0, err
	}
	head, err := k.reporterPendingSwitchHeadOrZero(ctx, repAddr.Bytes())
	if err != nil {
		return 0, err
	}
	return count + int(head.IncomingCount), nil
}

// CountSelectorsDelegatingToReporterExcludingSelf counts selector accounts whose
// Selection.reporter is repAddr, excluding repAddr itself (the self-reporter row).
func (k Keeper) CountSelectorsDelegatingToReporterExcludingSelf(ctx context.Context, repAddr sdk.AccAddress) (int, error) {
	iter, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, repAddr.Bytes())
	if err != nil {
		return 0, err
	}
	keys, err := iter.PrimaryKeys()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, selAddr := range keys {
		if !bytes.Equal(selAddr, repAddr.Bytes()) {
			n++
		}
	}
	return n, nil
}

// GetSelector returns the stored selection row without mutating state (safe for queries).
func (k Keeper) GetSelector(ctx context.Context, selectorAddr sdk.AccAddress) (types.Selection, error) {
	return k.Selectors.Get(ctx, selectorAddr.Bytes())
}

// GetSelectorForStake returns the selection row and clears expired dispute-jail before
// stake counting or dispute voting (mutates state when the sentence has ended).
func (k Keeper) GetSelectorForStake(ctx context.Context, selectorAddr sdk.AccAddress) (types.Selection, error) {
	sel, err := k.Selectors.Get(ctx, selectorAddr.Bytes())
	if err != nil {
		return types.Selection{}, err
	}
	if err := k.lazyClearSelectorLocksIfExpired(ctx, selectorAddr, &sel); err != nil {
		return types.Selection{}, err
	}
	return sel, nil
}

// GetReporterStake counts bonded selector stake for reporting paths. It finalizes ready
// pending switches, may lazy-unjail expired selector rows, and update RecalcAtTime when
// locks are still active.
func (k Keeper) GetReporterStake(ctx context.Context, repAddr sdk.AccAddress) (math.Int, []*types.TokenOriginInfo, []*types.SelectorShare, []byte, error) {
	return k.getReporterStake(ctx, repAddr, true)
}

// GetReporterStakeView is the read-only stake snapshot used by gRPC queries.
func (k Keeper) GetReporterStakeView(ctx context.Context, repAddr sdk.AccAddress) (math.Int, []*types.TokenOriginInfo, []*types.SelectorShare, []byte, error) {
	return k.getReporterStake(ctx, repAddr, false)
}

func (k Keeper) getReporterStake(ctx context.Context, repAddr sdk.AccAddress, mutate bool) (math.Int, []*types.TokenOriginInfo, []*types.SelectorShare, []byte, error) {
	if mutate {
		if err := k.applyReadyPendingSwitchesForReporter(ctx, repAddr); err != nil {
			return math.Int{}, nil, nil, nil, err
		}
	}

	reporter, err := k.Reporters.Get(ctx, repAddr.Bytes())
	if err != nil {
		return math.Int{}, nil, nil, nil, err
	}
	if reporter.Jailed {
		return math.Int{}, nil, nil, nil, errorsmod.Wrapf(types.ErrReporterJailed, "reporter %s is in jail", repAddr.String())
	}

	totalTokens := math.ZeroInt()
	iter, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, repAddr)
	if err != nil {
		return math.Int{}, nil, nil, nil, err
	}
	defer iter.Close()
	delegates := make([]*types.TokenOriginInfo, 0)
	selectorShares := make([]*types.SelectorShare, 0)
	// Compute hash inline as we build selector shares
	hasher := sha256.New()
	var earliestFutureLock int64 // track earliest future lock expiry (unix seconds)
	for ; iter.Valid(); iter.Next() {
		selectorAddr, err := iter.PrimaryKey()
		if err != nil {
			return math.Int{}, nil, nil, nil, err
		}
		valSet := k.stakingKeeper.GetValidatorSet()
		maxValSet, err := valSet.MaxValidators(ctx)
		if err != nil {
			return math.Int{}, nil, nil, nil, err
		}
		var selector types.Selection
		if mutate {
			selector, err = k.GetSelectorForStake(ctx, sdk.AccAddress(selectorAddr))
		} else {
			selector, err = k.GetSelector(ctx, sdk.AccAddress(selectorAddr))
		}
		if err != nil {
			return math.Int{}, nil, nil, nil, err
		}
		// Skip selectors with a pending switch away from this reporter: their
		// stake no longer counts toward the outgoing reporter until ReporterStake
		// finalizes the handoff after unlock height.
		hasPending, err := k.hasOutgoingPendingSwitch(ctx, repAddr.Bytes(), selectorAddr)
		if err != nil {
			return math.Int{}, nil, nil, nil, err
		}
		if hasPending && bytes.Equal(selector.Reporter, repAddr.Bytes()) {
			continue
		}
		// skip dispute-locked selectors (locked_until_time and/or jailed)
		now := sdk.UnwrapSDKContext(ctx).BlockTime()
		if selectorStakeLocked(selector, now) {
			lockUntil := selector.LockedUntilTime
			if selector.DisputeLockedUntil.After(lockUntil) {
				lockUntil = selector.DisputeLockedUntil
			}
			if lockUntil.After(now) {
				lockUnix := lockUntil.Unix()
				if earliestFutureLock == 0 || lockUnix < earliestFutureLock {
					earliestFutureLock = lockUnix
				}
			}
			continue
		}
		var iterError error
		selectorTotal := math.ZeroInt()
		// compare how many delegations a selector has to the max validators to detemine if you should short circuit and iterate the counts number of times
		// or iterate over all bonded validators for a selector in the case they have more delegations (with multiple validators, bonded or not) than the max bonded validators
		if selector.DelegationsCount > uint64(maxValSet) {
			// iterate over bonded validators
			err = valSet.IterateBondedValidatorsByPower(ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
				valAddrr, err := sdk.ValAddressFromBech32(validator.GetOperator())
				if err != nil {
					iterError = err
					return true
				}
				stakingdel, err := k.stakingKeeper.GetDelegation(ctx, selectorAddr, valAddrr)
				if err != nil {
					if errors.Is(err, stakingtypes.ErrNoDelegation) {
						return false
					}
					iterError = err
					return true
				}
				// get the token amount
				tokens := validator.TokensFromSharesTruncated(stakingdel.Shares).TruncateInt()
				totalTokens = totalTokens.Add(tokens)
				selectorTotal = selectorTotal.Add(tokens)
				delegates = append(delegates, &types.TokenOriginInfo{DelegatorAddress: selectorAddr, ValidatorAddress: valAddrr.Bytes(), Amount: tokens})
				return false
			})
			if err != nil {
				return math.Int{}, nil, nil, nil, err
			}
		} else {
			err = k.stakingKeeper.IterateDelegatorDelegations(ctx, selectorAddr, func(delegation stakingtypes.Delegation) (stop bool) {
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
					delTokens := val.TokensFromSharesTruncated(delegation.Shares).TruncateInt()
					totalTokens = totalTokens.Add(delTokens)
					selectorTotal = selectorTotal.Add(delTokens)
					delegates = append(delegates, &types.TokenOriginInfo{DelegatorAddress: selectorAddr, ValidatorAddress: valAddr.Bytes(), Amount: delTokens})
				}
				return false
			})
			if err != nil {
				return math.Int{}, nil, nil, nil, err
			}
		}
		if iterError != nil {
			return math.Int{}, nil, nil, nil, iterError
		}
		// Add aggregated share for this selector and update hash
		if selectorTotal.IsPositive() {
			selectorShares = append(selectorShares, &types.SelectorShare{
				SelectorAddress: selectorAddr,
				Amount:          selectorTotal,
			})
			hasher.Write(selectorAddr)
			hasher.Write(selectorTotal.BigInt().Bytes())
		}
	}
	// Update RecalcAtTime on write paths only (queries must not mutate store).
	if mutate {
		if earliestFutureLock == 0 {
			err = k.RecalcAtTime.Remove(ctx, repAddr.Bytes())
		} else {
			err = k.RecalcAtTime.Set(ctx, repAddr.Bytes(), earliestFutureLock)
		}
		if err != nil {
			return math.Int{}, nil, nil, nil, err
		}
	}

	// Finalize hash with total
	hasher.Write(totalTokens.BigInt().Bytes())
	return totalTokens, delegates, selectorShares, hasher.Sum(nil), nil
}

// Stores the token origins for each selector which is needed during a dispute for slashing/returning tokens to appropriate parties
func (k Keeper) SetReporterStakeByQueryId(ctx context.Context, repAddr sdk.AccAddress, delegates []*types.TokenOriginInfo, totalTokens math.Int, queryId []byte) error {
	return k.ReportByBlock.Set(ctx, collections.Join3(repAddr.Bytes(), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()), queryId), types.DelegationsAmounts{TokenOrigins: delegates, Total: totalTokens})
}

// handlePeriodTracking manages reward period tracking for a reporter.
// When delegation state changes (hash differs), it queues the previous period for distribution
// and stores the new period data.
func (k Keeper) handlePeriodTracking(ctx context.Context, repAddr sdk.AccAddress, selectorShares []*types.SelectorShare, totalTokens math.Int, newHash []byte) (bool, error) {
	// Get previous period data
	prevData, err := k.ReporterPeriodData.Get(ctx, repAddr)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return false, err
		}
		// First time for this reporter - just store period data
		return true, k.ReporterPeriodData.Set(ctx, repAddr, types.PeriodRewardData{
			Selectors:    selectorShares,
			Total:        totalTokens,
			RewardAmount: math.LegacyZeroDec(),
			Hash:         newHash,
		})
	}

	// Check if delegation state changed
	if bytes.Equal(prevData.Hash, newHash) {
		// No change - nothing to do, rewards will accumulate via DivvyingTips
		return false, nil
	}

	// Delegation state changed - queue previous period for distribution if it has rewards
	if prevData.RewardAmount.IsPositive() {
		if err := k.queueForDistribution(ctx, repAddr, prevData); err != nil {
			return false, err
		}
	}

	// Store new period data
	return true, k.ReporterPeriodData.Set(ctx, repAddr, types.PeriodRewardData{
		Selectors:    selectorShares,
		Total:        totalTokens,
		RewardAmount: math.LegacyZeroDec(),
		Hash:         newHash,
	})
}

// queueForDistribution adds a period's data to the distribution queue.
func (k Keeper) queueForDistribution(ctx context.Context, reporter sdk.AccAddress, data types.PeriodRewardData) error {
	// Get current queue counter
	counter, err := k.DistributionQueueCounter.Get(ctx)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		counter = types.DistributionQueueCounter{Head: 0, Tail: 0}
	}

	// Add item to queue
	item := types.DistributionQueueItem{
		Reporter:     reporter,
		Selectors:    data.Selectors,
		Total:        data.Total,
		RewardAmount: data.RewardAmount,
	}
	if err := k.DistributionQueue.Set(ctx, counter.Tail, item); err != nil {
		return err
	}

	// Increment tail
	counter.Tail++
	return k.DistributionQueueCounter.Set(ctx, counter)
}
