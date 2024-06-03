package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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

func (h Hooks) AfterValidatorBonded(ctx context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorBeginUnbonding(ctx context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorRemoved(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorCreated(_ context.Context, _ sdk.ValAddress) error { return nil }

func (h Hooks) BeforeValidatorModified(_ context.Context, _ sdk.ValAddress) error { return nil }

func (h Hooks) BeforeValidatorSlashed(_ context.Context, _ sdk.ValAddress, _ sdkmath.LegacyDec) error {
	return nil
} // todo: handle for dispute event

func (h Hooks) AfterUnbondingInitiated(_ context.Context, _ uint64) error { return nil }

func (h Hooks) AfterConsensusPubKeyUpdate(_ context.Context, _, _ cryptotypes.PubKey, _ sdk.Coin) error {
	return nil
}

func (h Hooks) BeforeDelegationSharesModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	// current shares
	del, err := h.k.stakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return err
	}

	val, err := h.k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return err
	}
	tokens := val.TokensFromShares(del.Shares).TruncateInt()
	// set temp
	if err := h.k.TempStore.Set(ctx, collections.Join(delAddr.Bytes(), valAddr.Bytes()), tokens); err != nil {
		return err
	}
	return nil
}

func (h Hooks) AfterDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	// current shares
	sDel, err := h.k.stakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return err
	}

	val, err := h.k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return err
	}
	tokens := val.TokensFromShares(sDel.Shares).TruncateInt()
	// get temp
	temp, err := h.k.TempStore.Get(ctx, collections.Join(delAddr.Bytes(), valAddr.Bytes()))
	if err != nil {
		return err
	}
	diff := tokens.Sub(temp)

	// update the delegator's total tokens
	del, err := h.k.Delegators.Get(ctx, delAddr.Bytes())
	if err != nil {
		return err
	}
	del.Amount = del.Amount.Add(diff)
	if err := h.k.Delegators.Set(ctx, delAddr.Bytes(), del); err != nil {
		return err
	}
	// update reporter's total tokens
	reporter, err := h.k.Reporters.Get(ctx, del.Reporter)
	if err != nil {
		return err
	}
	reporter.TotalTokens = reporter.TotalTokens.Add(diff)
	return h.k.Reporters.Set(ctx, del.Reporter, reporter)
}

// Check if validator is a reporter and if they have reached the maximum number of delegators of 100 for now
// if they have, then make the new delegator a reporter since they are new they shouldn't have been a reporter before
// also tracks the total tokens of the delegator
func (h Hooks) BeforeDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	if err := h.k.TempStore.Set(ctx, collections.Join(delAddr.Bytes(), valAddr.Bytes()), sdkmath.ZeroInt()); err != nil {
		return err
	}
	reporterKey := valAddr.Bytes()
	iter, err := h.k.Delegators.Indexes.Reporter.MatchExact(ctx, reporterKey)
	if err != nil {
		return err
	}
	// todo: this is an iteration and should be optimized, perhaps we can use a counter
	pks, err := iter.FullKeys()
	if err != nil {
		return err
	}

	if len(pks) == 0 {
		val, err := h.k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			return err
		}
		if err := h.k.Reporters.Set(ctx, reporterKey, types.OracleReporter{
			// inherit the commission rates from the validator
			Commission:  &val.Commission,
			TotalTokens: sdkmath.ZeroInt(),
		}); err != nil {
			return err
		}

	}

	if len(pks) >= 100 {
		reporterKey = delAddr.Bytes()
		// add reporter with default commission rates
		if err := h.k.Reporters.Set(ctx, reporterKey, types.OracleReporter{
			Commission: &stakingtypes.Commission{CommissionRates: stakingtypes.CommissionRates{
				Rate:          sdkmath.LegacyMustNewDecFromStr("0.1"),
				MaxRate:       sdkmath.LegacyMustNewDecFromStr("0.2"),
				MaxChangeRate: sdkmath.LegacyMustNewDecFromStr("0.01"),
			}},
			// Reporter: reporterKey,
			// TotalTokens: sdkmath.ZeroInt(),
		}); err != nil {
			return err
		}
	}

	return h.k.Delegators.Set(ctx, delAddr.Bytes(), types.Delegation{
		Reporter:        reporterKey,
		Amount:          sdkmath.ZeroInt(),
		DelegationCount: 1,
	})
}

// when BeforeDelegationRemoved is called, reduce the delegation count of the delegator, also
// we need to check if the delegator has any more delegations, if not remove the delegator plus check if the reporter
// has any more delegators, if not remove the reporter as well
// else just reduce the delegation count of the delegator
// Also, update the total tokens of the delegator
func (h Hooks) BeforeDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	// get temp
	temp, err := h.k.TempStore.Get(ctx, collections.Join(delAddr.Bytes(), valAddr.Bytes()))
	if err != nil {
		return err
	}
	// delAmt, err := h.k.DelegatorAmount.Get(ctx, delAddr.Bytes())
	// if err != nil {
	// 	return err
	// }
	// delAmt = delAmt.Sub(temp)
	// if delAmt.IsZero() {
	// 	if err := h.k.DelegatorAmount.Remove(ctx, delAddr.Bytes()); err != nil {
	// 		return err
	// 	}
	// } else {
	// 	if err := h.k.DelegatorAmount.Set(ctx, delAddr.Bytes(), delAmt); err != nil {
	// 		return err
	// 	}
	// }
	del, err := h.k.Delegators.Get(ctx, delAddr)
	if err != nil {
		return err
	}
	del.Amount = del.Amount.Sub(temp)
	del.DelegationCount--
	if del.DelegationCount == 0 {
		err = h.k.Delegators.Remove(ctx, delAddr)
		if err != nil {
			return err
		}
		iter, err := h.k.Delegators.Indexes.Reporter.MatchExact(ctx, del.Reporter)
		if err != nil {
			return err
		}
		pks, err := iter.FullKeys()
		if err != nil {
			return err
		}
		if len(pks) == 0 {
			return h.k.Reporters.Remove(ctx, del.Reporter)
		}
		rep, err := h.k.Reporters.Get(ctx, del.Reporter)
		if err != nil {
			return err
		}
		rep.TotalTokens = rep.TotalTokens.Sub(temp)
		return h.k.Reporters.Set(ctx, del.Reporter, rep)
	}
	rep, err := h.k.Reporters.Get(ctx, del.Reporter)
	if err != nil {
		return err
	}
	rep.TotalTokens = rep.TotalTokens.Sub(temp)
	err = h.k.Reporters.Set(ctx, del.Reporter, rep)
	if err != nil {
		return err
	}
	return h.k.Delegators.Set(ctx, delAddr, del)
}
