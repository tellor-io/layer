package keeper

import (
	"context"
	"errors"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.StakingHooks = Hooks{}

// Hooks wrapper struct for reporter keeper
type Hooks struct {
	k Keeper
}

// Return the reporter hooks
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

func (h Hooks) AfterValidatorBonded(ctx context.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) error {
	repAddr := sdk.AccAddress(valAddr)
	iter, err := h.k.Selectors.Indexes.Reporter.MatchExact(ctx, repAddr)
	if err != nil {
		return err
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		selectorAddr, err := iter.PrimaryKey()
		if err != nil {
			return err
		}

		selector, err := h.k.Selectors.Get(ctx, selectorAddr)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				continue
			} else {
				return err
			}
		}
		selector.DelegationsCount++
		if err := h.k.Selectors.Set(ctx, repAddr, selector); err != nil {
			return nil
		}
	}
	return nil
}

func (h Hooks) AfterValidatorBeginUnbonding(ctx context.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) error {
	repAddr := sdk.AccAddress(valAddr)
	iter, err := h.k.Selectors.Indexes.Reporter.MatchExact(ctx, repAddr)
	if err != nil {
		return err
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		selectorAddr, err := iter.PrimaryKey()
		if err != nil {
			return err
		}

		selector, err := h.k.Selectors.Get(ctx, selectorAddr)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				continue
			} else {
				return err
			}
		}
		selector.DelegationsCount--
		if err := h.k.Selectors.Set(ctx, repAddr, selector); err != nil {
			return nil
		}
	}
	return nil
}

func (h Hooks) AfterValidatorRemoved(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorCreated(_ context.Context, _ sdk.ValAddress) error { return nil }

func (h Hooks) BeforeValidatorModified(_ context.Context, _ sdk.ValAddress) error { return nil }

func (h Hooks) BeforeValidatorSlashed(_ context.Context, _ sdk.ValAddress, _ sdkmath.LegacyDec) error {
	return nil
}

func (h Hooks) AfterUnbondingInitiated(_ context.Context, _ uint64) error { return nil }

func (h Hooks) AfterConsensusPubKeyUpdate(_ context.Context, _, _ cryptotypes.PubKey, _ sdk.Coin) error {
	return nil
}

func (h Hooks) BeforeDelegationSharesModified(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterDelegationModified(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}

// this hook is called in the staking module when a delegation is being created and its implemented here to update a selector's delegations count
// in the event that a selector delegates to a new validator (new relationship)
// purpose of keeping count of delegations for a selector is to short circuit the stake calcuation process that is necessary
// to calculate a reporters stake total; otherwise every bonded validator is checked for delegations from a selector
func (h Hooks) BeforeDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, _ sdk.ValAddress) error {
	selector, err := h.k.Selectors.Get(ctx, delAddr.Bytes())
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		} else {
			return err
		}
	}
	selector.DelegationsCount++
	return h.k.Selectors.Set(ctx, delAddr.Bytes(), selector)
}

// decrement a selector's delegations count when a delegation is removed
func (h Hooks) BeforeDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	selector, err := h.k.Selectors.Get(ctx, delAddr.Bytes())
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		} else {
			return err
		}
	}
	selector.DelegationsCount--
	return h.k.Selectors.Set(ctx, delAddr.Bytes(), selector)
}
