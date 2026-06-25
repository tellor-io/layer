package keeper

import (
	"context"

	"cosmossdk.io/math"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// PickUnderCapBondedValidatorExported exposes the unexported validator-cap
// scan helper to the external keeper_test package for direct unit testing.
func (k Keeper) PickUnderCapBondedValidatorExported(ctx context.Context, amount math.Int) (stakingtypes.Validator, error) {
	maxShare, err := k.maxValidatorPowerShare(ctx)
	if err != nil {
		return stakingtypes.Validator{}, err
	}
	return k.pickUnderCapBondedValidator(ctx, amount, maxShare, math.ZeroInt())
}

// BondedValidatorForDelegationExported exposes the unexported preferred-or-scan
// helper to the external keeper_test package for direct unit testing.
func (k Keeper) BondedValidatorForDelegationExported(ctx context.Context, preferred stakingtypes.Validator, amount math.Int) (stakingtypes.Validator, error) {
	maxShare, err := k.maxValidatorPowerShare(ctx)
	if err != nil {
		return stakingtypes.Validator{}, err
	}
	return k.bondedValidatorForDelegation(ctx, preferred, amount, maxShare, math.ZeroInt())
}
