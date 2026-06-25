package ante

import (
	"bytes"
	"errors"
	"sort"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
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

// applyProspectiveBondedValidatorChanges accounts for validators entering or
// leaving the active set after tx handlers run, then folds those stake changes
// into the final total and per-delegator checks.
func (t TrackStakeChangesDecorator) applyProspectiveBondedValidatorChanges(ctx sdk.Context, stakeChanges *stakeChangeTracker, projection activeSetProjection) error {
	if stakeChanges == nil || !stakeChanges.activeSetDelta {
		return nil
	}
	if len(projection.changes.entering) == 0 && len(projection.changes.leaving) == 0 {
		return t.checkTotalStakeChange(ctx, stakeChanges.totalBondedDelta)
	}

	// entrants add their post-change stake; leavers remove the stake they would still have after the tx
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

// prospectiveActiveSetChanges projects staking's next active set from the
// current top validators plus validators touched by this tx. The scan is
// bounded and candidates are sorted to preserve deterministic behavior.
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

	// scan the current top set plus enough replacement candidates to cover each validator touched by this tx
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
				addr:       valAddr,
				validator:  current,
				postTokens: current.Tokens,
				postShares: current.DelegatorShares,
				wasActive:  current.IsBonded() && !current.IsJailed(),
			}
		}
		validators[validatorKey] = validator
		count++
	}

	// add validators touched by this tx that were not present in the power index scan, including MsgCreateValidator candidates
	for _, validatorKey := range sortedKeys(stakeChanges.validatorProjections) {
		if _, ok := validators[validatorKey]; ok {
			continue
		}
		validators[validatorKey] = stakeChanges.validatorProjections[validatorKey]
	}

	ordered := make([]prospectiveValidator, 0, len(validators))
	for _, validatorKey := range sortedKeys(validators) {
		validator := validators[validatorKey]
		// jailed validators cannot enter the active set until EndBlocker clears Jailed;
		// a same-tx MsgUnjail projection has already set Jailed=false, so an unjailed candidate is still considered here
		if validator.validator.Jailed {
			continue
		}
		if sdk.TokensToConsensusPower(validator.postTokens, powerReduction) == 0 {
			continue
		}
		ordered = append(ordered, validator)
	}
	// match staking's active-set ranking: consensus power first, then operator address for deterministic ties
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
		// entering/leaving is detected via IsBonded(); a jailed validator unjailed in this tx is handled by
		// checkValidatorPowerShares via wasActive (before=0), not as an entrant, so its tokens are not double-counted
		switch {
		case inNextSet && !validator.validator.IsBonded():
			changes.entering = append(changes.entering, validator)
		case !inNextSet && validator.validator.IsBonded():
			changes.leaving = append(changes.leaving, validator)
		}
	}
	return changes, nil
}

// addActiveSetValidatorDelegatorDeltas adds the per-delegator bonded stake that
// appears or disappears when a validator enters or leaves the active set.
func (t TrackStakeChangesDecorator) addActiveSetValidatorDelegatorDeltas(ctx sdk.Context, stakeChanges *stakeChangeTracker, validator prospectiveValidator, sign math.LegacyDec) error {
	delegatorShares := make(map[delegatorAddressKey]math.LegacyDec)
	validatorKey := newValidatorAddressKey(validator.addr)
	// pending validators have no stored delegations yet; their self-delegation is already in delegationShareDelta
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

// checkValidatorPowerShares enforces the validator acquisition cap: a validator
// may not gain active bonded stake while the resulting share of total bonded
// stake is strictly above max_validator_power_share. Already-bonded validators
// whose tokens do not increase (including passive denominator drift) and
// validators leaving the active set are never rejected. Entering validators are
// checked as before=0, after=postTokens.
func (t TrackStakeChangesDecorator) checkValidatorPowerShares(ctx sdk.Context, stakeChanges *stakeChangeTracker, bondedChanges activeSetChanges, effectiveBondedDelta math.Int) error {
	if stakeChanges == nil {
		return nil
	}
	if len(stakeChanges.validatorProjections) == 0 && len(bondedChanges.entering) == 0 && len(bondedChanges.leaving) == 0 && !stakeChanges.unjailOccurred {
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

	candidates := make(map[validatorAddressKey]prospectiveValidator)
	projectedBonded := make(map[validatorAddressKey]bool)

	for _, validatorKey := range sortedKeys(stakeChanges.validatorProjections) {
		validator := stakeChanges.validatorProjections[validatorKey]
		candidates[validatorKey] = validator
		projectedBonded[validatorKey] = validator.validator.IsBonded()
	}
	for _, validator := range bondedChanges.entering {
		validatorKey := newValidatorAddressKey(validator.addr)
		candidates[validatorKey] = validator
		projectedBonded[validatorKey] = true
	}
	for _, validator := range bondedChanges.leaving {
		validatorKey := newValidatorAddressKey(validator.addr)
		candidates[validatorKey] = validator
		projectedBonded[validatorKey] = false
	}

	// acquisition-only enforcement: skip the denominator fetch when no validator is gaining active bonded stake
	acquisitions := make([]validatorAddressKey, 0)
	for _, validatorKey := range sortedKeys(candidates) {
		validator := candidates[validatorKey]

		// before is the validator's pre-tx active bonded stake; use wasActive (bonded && not jailed at tx start)
		// rather than IsBonded(): a jailed validator is not in the active set even if Status is still Bonded
		before := math.ZeroInt()
		if validator.wasActive {
			before = validator.validator.Tokens
		}

		after := math.ZeroInt()
		if projectedBonded[validatorKey] {
			after = validator.postTokens
		}

		if after.GT(before) {
			acquisitions = append(acquisitions, validatorKey)
		}
	}
	if len(acquisitions) == 0 {
		return nil
	}

	currentTotalBonded, err := t.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return err
	}
	totalBondedAfter := currentTotalBonded.Add(effectiveBondedDelta)
	if !totalBondedAfter.IsPositive() {
		return nil
	}

	for _, validatorKey := range acquisitions {
		validator := candidates[validatorKey]
		if types.ExceedsPowerShare(validator.postTokens, totalBondedAfter, maxShare) {
			return errorsmod.Wrapf(types.ErrExceedsMaxValidatorPowerShare, "validator %s", validator.addr.String())
		}
	}
	return nil
}
