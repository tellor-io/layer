package keeper

import (
	"context"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"
	"github.com/tellor-io/layer/x/reporter/types"

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

// AfterValidatorBonded updates the signing info start height or create a new signing info
func (h Hooks) AfterValidatorBonded(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

// AfterValidatorRemoved deletes the address-pubkey relation when a validator is removed,
func (h Hooks) AfterValidatorRemoved(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

// AfterValidatorCreated adds the address-pubkey relation when a validator is created.
func (h Hooks) AfterValidatorCreated(_ context.Context, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorBeginUnbonding(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeValidatorModified(_ context.Context, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeDelegationCreated(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeDelegationSharesModified(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeDelegationRemoved(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	// TODO: check context to see how this is called in sdk!!!
	// reflect changes only when token/power decreases
	// update the reporter tokens and the delegator's tokens to reflect the new power numbers
	// also need to update the token origins to reflect the new changes when the delegator's tokens are updated
	exists, err := h.k.TokenOrigin.Has(ctx, collections.Join(delAddr, valAddr))
	if err != nil {
		return err
	}
	if exists {
		// get delegation
		delegation, err := h.k.stakingKeeper.Delegation(ctx, delAddr, valAddr)
		if err != nil {
			return err
		}
		// get validator to calculate token amount from shares
		validator, err := h.k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			return err
		}
		tokenAmount := validator.TokensFromSharesTruncated(delegation.GetShares()).TruncateInt()
		// get token origin
		source, err := h.k.TokenOrigin.Get(ctx, collections.Join(delAddr, valAddr))
		if err != nil {
			return err
		}
		// update token origin if the staked amount becomes less than what is written in the token origin struct
		if tokenAmount.LT(source.Amount) {
			// get the difference in the token change to reduce delegation and reporter tokens by.
			diff := source.Amount.Sub(tokenAmount)
			if err := h.k.UpdateOrRemoveSource(ctx, collections.Join(delAddr, valAddr), source, tokenAmount); err != nil {
				return err
			}

			// update delegator
			delegator, err := h.k.Delegators.Get(ctx, delAddr)
			if err != nil {
				return err
			}
			repAddr := sdk.MustAccAddressFromBech32(delegator.Reporter)
			// update reporter
			reporter, err := h.k.Reporters.Get(ctx, repAddr)
			if err != nil {
				return err
			}
			if err := h.k.UpdateOrRemoveDelegator(ctx, delAddr, delegator, reporter, diff); err != nil {
				return err
			}
			if err := h.k.UpdateOrRemoveReporter(ctx, repAddr, reporter, diff); err != nil {
				return err
			}
		}

	}
	return nil
}

func (h Hooks) BeforeValidatorSlashed(_ context.Context, _ sdk.ValAddress, _ sdkmath.LegacyDec) error {
	return nil
}

func (h Hooks) AfterUnbondingInitiated(_ context.Context, _ uint64) error {
	return nil
}

// AfterConsensusPubKeyUpdate triggers the functions to rotate the signing-infos also sets address pubkey relation.
func (h Hooks) AfterConsensusPubKeyUpdate(_ context.Context, oldPubKey, newPubKey cryptotypes.PubKey, _ sdk.Coin) error {
	return nil
}
