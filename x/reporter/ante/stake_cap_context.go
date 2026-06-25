package ante

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// stakeCapContext holds projected total-bonded denominators for concentration-cap
// checks after a tx's stake-change side effects are applied. Delegator and
// reporter caps share delegatorDelta; the validator cap uses validatorDelta,
// which includes unjail-only active-set re-entry per ADR M2 without affecting
// the other caps.
type stakeCapContext struct {
	currentTotalBonded math.Int
	activeSet          activeSetProjection
	delegatorDelta     math.Int
	validatorDelta     math.Int
}

func (c stakeCapContext) totalBondedAfterDelegator() math.Int {
	return c.currentTotalBonded.Add(c.delegatorDelta)
}

func (c stakeCapContext) totalBondedAfterValidator() math.Int {
	return c.currentTotalBonded.Add(c.validatorDelta)
}

func (t TrackStakeChangesDecorator) buildStakeCapContext(ctx sdk.Context, stakeChanges *stakeChangeTracker) (stakeCapContext, error) {
	projection := newActiveSetProjection(activeSetChanges{})
	if stakeChanges.activeSetDelta || stakeChanges.unjailOccurred {
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
		if stakeChanges.unjailOccurred {
			validatorDelta = validatorDelta.Add(projection.bondedDelta)
		}
	}

	var currentTotalBonded math.Int
	if len(stakeChanges.delegatorBondedDelta) > 0 ||
		len(stakeChanges.selectionChanges) > 0 ||
		stakeChanges.unjailOccurred ||
		len(stakeChanges.validatorProjections) > 0 ||
		len(projection.changes.entering) > 0 ||
		len(projection.changes.leaving) > 0 {
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
