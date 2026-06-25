package keeper

import (
	"context"

	"cosmossdk.io/math"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// PickUnderCapBondedValidatorExported exposes the unexported validator-cap
// scan helper to the external keeper_test package for direct unit testing. It
// uses single-call semantics (no cross-origin projection).
func (k Keeper) PickUnderCapBondedValidatorExported(ctx context.Context, amount math.Int) (stakingtypes.Validator, error) {
	return k.pickUnderCapBondedValidator(ctx, amount, nil)
}

// BondedValidatorForDelegationExported exposes the unexported preferred-or-scan
// helper to the external keeper_test package for direct unit testing. It uses
// single-call semantics (no cross-origin projection).
func (k Keeper) BondedValidatorForDelegationExported(ctx context.Context, preferred stakingtypes.Validator, amount math.Int) (stakingtypes.Validator, error) {
	return k.bondedValidatorForDelegation(ctx, preferred, amount, nil)
}
