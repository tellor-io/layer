package ante

import (
	"bytes"
	"sort"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type activeSetChanges struct {
	entering []prospectiveValidator
	leaving  []prospectiveValidator
}

type activeSetProjection struct {
	changes     activeSetChanges
	bondedDelta math.Int
}

type stakeCapContext struct {
	currentTotalBonded math.Int
	activeSet          activeSetProjection
	delegatorDelta     math.Int
	validatorDelta     math.Int
}

func newActiveSetProjection(changes activeSetChanges) activeSetProjection {
	projection := activeSetProjection{
		changes:     changes,
		bondedDelta: math.ZeroInt(),
	}
	for _, validator := range changes.entering {
		projection.bondedDelta = projection.bondedDelta.Add(validator.postTokens)
	}
	for _, validator := range changes.leaving {
		projection.bondedDelta = projection.bondedDelta.Sub(validator.postTokens)
	}
	return projection
}

func (c stakeCapContext) totalBondedAfterDelegator() math.Int {
	return c.currentTotalBonded.Add(c.delegatorDelta)
}

func (c stakeCapContext) totalBondedAfterValidator() math.Int {
	return c.currentTotalBonded.Add(c.validatorDelta)
}

func (t TrackStakeChangesDecorator) buildStakeCapContext(ctx sdk.Context, stakeChanges *stakeChangeTracker) (stakeCapContext, error) {
	jailCleared := stakeChanges.jailCleared()
	needsActiveSetProjection := stakeChanges.activeSetDelta || jailCleared

	projection := newActiveSetProjection(activeSetChanges{})
	if needsActiveSetProjection {
		changes, err := t.prospectiveActiveSetChanges(ctx, stakeChanges)
		if err != nil {
			return stakeCapContext{}, err
		}
		projection = newActiveSetProjection(changes)
	}

	delegatorDelta := stakeChanges.totalBondedDelta
	validatorDelta := stakeChanges.totalBondedDelta

	if stakeChanges.activeSetDelta {
		if err := t.applyProspectiveBondedValidatorChanges(ctx, stakeChanges, projection); err != nil {
			return stakeCapContext{}, err
		}
		delegatorDelta = stakeChanges.totalBondedDelta
		validatorDelta = stakeChanges.totalBondedDelta
	} else {
		if err := t.checkTotalStakeChange(ctx, stakeChanges.totalBondedDelta); err != nil {
			return stakeCapContext{}, err
		}
		if jailCleared {
			// Unjail: adjust validator-cap denominator only.
			validatorDelta = validatorDelta.Add(projection.bondedDelta)
		}
	}

	var currentTotalBonded math.Int
	if len(stakeChanges.delegatorBondedDelta) > 0 ||
		len(stakeChanges.selectionChanges) > 0 ||
		needsActiveSetProjection ||
		len(stakeChanges.validatorProjections) > 0 {
		var err error
		currentTotalBonded, err = t.stakingKeeper.TotalBondedTokens(ctx)
		if err != nil {
			return stakeCapContext{}, err
		}
	}

	return stakeCapContext{
		currentTotalBonded: currentTotalBonded,
		activeSet:          projection,
		delegatorDelta:     delegatorDelta,
		validatorDelta:     validatorDelta,
	}, nil
}

// applyProspectiveBondedValidatorChanges folds active-set enter/leave stake into final checks.
func (t TrackStakeChangesDecorator) applyProspectiveBondedValidatorChanges(ctx sdk.Context, stakeChanges *stakeChangeTracker, projection activeSetProjection) error {
	if stakeChanges == nil || !stakeChanges.activeSetDelta {
		return nil
	}
	if len(projection.changes.entering) == 0 && len(projection.changes.leaving) == 0 {
		return t.checkTotalStakeChange(ctx, stakeChanges.totalBondedDelta)
	}

	stakeChanges.addTotalDelta(projection.bondedDelta)
	if err := t.checkTotalStakeChange(ctx, stakeChanges.totalBondedDelta); err != nil {
		return err
	}
	for _, validator := range projection.changes.entering {
		if err := t.addActiveSetValidatorDelegatorDeltas(ctx, stakeChanges, validator, math.LegacyOneDec()); err != nil {
			return err
		}
	}
	for _, validator := range projection.changes.leaving {
		if err := t.addActiveSetValidatorDelegatorDeltas(ctx, stakeChanges, validator, math.LegacyNewDec(-1)); err != nil {
			return err
		}
	}
	return nil
}

// prospectiveActiveSetChanges projects staking's next active set from the current
// top validators plus validators touched by this tx.
func (t TrackStakeChangesDecorator) prospectiveActiveSetChanges(ctx sdk.Context, stakeChanges *stakeChangeTracker) (activeSetChanges, error) {
	maxValidators, err := t.stakingKeeper.MaxValidators(ctx)
	if err != nil {
		return activeSetChanges{}, err
	}
	if maxValidators == 0 {
		return activeSetChanges{}, nil
	}
	powerReduction := t.stakingKeeper.PowerReduction(ctx)
	validators := make(map[validatorAddressKey]prospectiveValidator)
	iterator, err := t.stakingKeeper.ValidatorsPowerStoreIterator(ctx)
	if err != nil {
		return activeSetChanges{}, err
	}
	defer iterator.Close()

	scanLimit := int(maxValidators) + len(stakeChanges.validatorProjections)
	for count := 0; iterator.Valid() && count < scanLimit; iterator.Next() {
		valAddr := sdk.ValAddress(iterator.Value())
		validatorKey := newValidatorAddressKey(valAddr)
		if _, ok := validators[validatorKey]; ok {
			continue
		}
		validator, ok := stakeChanges.validatorProjections[validatorKey]
		if !ok {
			current, err := t.stakingKeeper.GetValidator(ctx, valAddr)
			if err != nil {
				return activeSetChanges{}, err
			}
			validator = prospectiveValidator{
				addr:        valAddr,
				validator:   current,
				postTokens:  current.Tokens,
				postShares:  current.DelegatorShares,
				preTxActive: current.IsBonded() && !current.IsJailed(),
			}
		}
		validators[validatorKey] = validator
		count++
	}

	for _, validatorKey := range sortedKeys(stakeChanges.validatorProjections) {
		if _, ok := validators[validatorKey]; ok {
			continue
		}
		validators[validatorKey] = stakeChanges.validatorProjections[validatorKey]
	}

	ordered := make([]prospectiveValidator, 0, len(validators))
	for _, validatorKey := range sortedKeys(validators) {
		validator := validators[validatorKey]
		// jailed validators stay out until EndBlocker; same-tx MsgUnjail already clears Jailed here
		if validator.validator.Jailed {
			continue
		}
		if sdk.TokensToConsensusPower(validator.postTokens, powerReduction) == 0 {
			continue
		}
		ordered = append(ordered, validator)
	}
	// consensus power desc, then operator address for ties
	sort.Slice(ordered, func(i, j int) bool {
		iPower := sdk.TokensToConsensusPower(ordered[i].postTokens, powerReduction)
		jPower := sdk.TokensToConsensusPower(ordered[j].postTokens, powerReduction)
		if iPower == jPower {
			return bytes.Compare(ordered[i].addr, ordered[j].addr) < 0
		}
		return iPower > jPower
	})

	limit := int(maxValidators)
	if len(ordered) < limit {
		limit = len(ordered)
	}
	nextSet := make(map[validatorAddressKey]struct{}, limit)
	for _, validator := range ordered[:limit] {
		nextSet[newValidatorAddressKey(validator.addr)] = struct{}{}
	}

	changes := activeSetChanges{}
	for _, validatorKey := range sortedKeys(validators) {
		validator := validators[validatorKey]
		_, inNextSet := nextSet[validatorKey]
		switch {
		case inNextSet && !validator.validator.IsBonded():
			changes.entering = append(changes.entering, validator)
		case !inNextSet && validator.preTxActive:
			changes.leaving = append(changes.leaving, validator)
		}
	}
	return changes, nil
}

func (t TrackStakeChangesDecorator) addActiveSetValidatorDelegatorDeltas(ctx sdk.Context, stakeChanges *stakeChangeTracker, validator prospectiveValidator, sign math.LegacyDec) error {
	delegatorShares := make(map[delegatorAddressKey]math.LegacyDec)
	validatorKey := newValidatorAddressKey(validator.addr)
	// pending validators have no stored delegations; self-delegation is already in delegationShareDelta
	if _, pending := stakeChanges.pendingValidators[validatorKey]; !pending {
		delegations, err := t.stakingKeeper.GetValidatorDelegations(ctx, validator.addr)
		if err != nil {
			return err
		}
		for _, delegation := range delegations {
			ctx.GasMeter().ConsumeGas(ActiveSetDelegationCheckGas, activeSetDelegationGasMessage)
			delegator, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
			if err != nil {
				return err
			}
			addDec(delegatorShares, newDelegatorAddressKey(delegator), delegation.Shares)
		}
	}
	for _, delegatorKey := range sortedKeys(stakeChanges.delegationShareDelta[validatorKey]) {
		delta := stakeChanges.delegationShareDelta[validatorKey][delegatorKey]
		addDec(delegatorShares, delegatorKey, delta)
	}
	postValidator := validator.postState()
	for _, delegatorKey := range sortedKeys(delegatorShares) {
		shares := delegatorShares[delegatorKey]
		if shares.IsPositive() {
			delegator, err := delegatorKey.address()
			if err != nil {
				return err
			}
			amount := postValidator.TokensFromShares(shares)
			stakeChanges.addDelegatorDelta(delegator, amount.Mul(sign))
		}
	}
	return nil
}
