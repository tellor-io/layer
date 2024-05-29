package keeper

import (
	"context"
	"errors"

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

// func (k Keeper) UpdateAddToDelegation(
// 	ctx context.Context, delAcc sdk.AccAddress, valAddr sdk.ValAddress, tokens sdkmath.Int, commission stakingtypes.Commission) error {
// 	// get the delegation
// 	delegation, err := k.Delegators.Get(ctx, delAcc)
// 	if err != nil {
// 		if !errors.Is(err, collections.ErrNotFound) {
// 			return err
// 		} else {
// 			delegation.Reporter = valAddr.Bytes()
// 			delegation.Amount = sdkmath.ZeroInt()
// 		}
// 	}
// 	delegation.Amount = delegation.Amount.Add(tokens)
// 	err = k.DelegatorCheckpoint.Set(ctx, collections.Join(delAcc.Bytes(), sdk.UnwrapSDKContext(ctx).BlockHeight()), delegation.Amount)
// 	if err != nil {
// 		return err
// 	}

// 	reporter, err := k.Reporters.Get(ctx, delegation.Reporter)
// 	if err != nil {
// 		if !errors.Is(err, collections.ErrNotFound) {
// 			return err
// 		} else {
// 			reporter.TotalTokens = sdkmath.ZeroInt()
// 			reporter.Commission = &commission
// 		}
// 	}
// 	reporter.TotalTokens = reporter.TotalTokens.Add(tokens)
// 	if err := k.Reporters.Set(ctx, delegation.Reporter, reporter); err != nil {
// 		return err
// 	}

// 	return k.Delegators.Set(ctx, delAcc, delegation)
// }

// AfterValidatorBonded updates the signing info start height or create a new signing info
// func (h Hooks) AfterValidatorBonded(ctx context.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) error {
// 	val, err := h.k.stakingKeeper.GetValidator(ctx, valAddr)
// 	if err != nil {
// 		return err
// 	}
// 	delegations, err := h.k.stakingKeeper.GetValidatorDelegations(ctx, valAddr)
// 	if err != nil {
// 		return err
// 	}
// 	// update the delegator's tokens to reflect the new power numbers
// 	for _, delegation := range delegations {
// 		delAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
// 		if err != nil {
// 			return err
// 		}
// 		tokens := val.TokensFromSharesTruncated(delegation.Shares).TruncateInt()
// 		if err := h.k.UpdateAddToDelegation(ctx, delAddr, valAddr, tokens, val.Commission); err != nil {
// 			return err
// 		}

//		}
//		return nil
//	}
//
//	func (k Keeper) UpdateSubDelegation(ctx context.Context, delAcc sdk.AccAddress, tokens sdkmath.Int) error {
//		// get the delegation
//		delegation, err := k.Delegators.Get(ctx, delAcc)
//		if err != nil {
//			return err
//		}
//		delegation.Amount = delegation.Amount.Sub(tokens)
//		err = k.DelegatorCheckpoint.Set(ctx, collections.Join(delAcc.Bytes(), sdk.UnwrapSDKContext(ctx).BlockHeight()), delegation.Amount)
//		if err != nil {
//			return err
//		}
//		if delegation.Amount.IsZero() {
//			return k.Delegators.Remove(ctx, delAcc)
//		}
//		return k.Delegators.Set(ctx, delAcc, delegation)
//	}
//
//	func (h Hooks) AfterValidatorBeginUnbonding(ctx context.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) error {
//		val, err := h.k.stakingKeeper.GetValidator(ctx, valAddr)
//		if err != nil {
//			return err
//		}
//		delegations, err := h.k.stakingKeeper.GetValidatorDelegations(ctx, valAddr)
//		if err != nil {
//			return err
//		}
//		for _, delegation := range delegations {
//			delAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
//			if err != nil {
//				return err
//			}
//			tokens := val.TokensFromSharesTruncated(delegation.Shares).TruncateInt()
//			if err := h.k.UpdateSubDelegation(ctx, delAddr, tokens); err != nil {
//				return err
//			}
//		}
//		return nil
//	}
func (h Hooks) AfterValidatorBonded(ctx context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}
func (h Hooks) AfterValidatorBeginUnbonding(ctx context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}
func (h Hooks) AfterValidatorRemoved(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}
func (h Hooks) AfterValidatorCreated(_ context.Context, _ sdk.ValAddress) error   { return nil }
func (h Hooks) BeforeValidatorModified(_ context.Context, _ sdk.ValAddress) error { return nil }
func (h Hooks) BeforeDelegationSharesModified(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}
func (h Hooks) AfterDelegationModified(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}
func (h Hooks) BeforeValidatorSlashed(_ context.Context, _ sdk.ValAddress, _ sdkmath.LegacyDec) error {
	return nil
}                                                                         //todo: handle for dispute event
func (h Hooks) AfterUnbondingInitiated(_ context.Context, _ uint64) error { return nil }
func (h Hooks) AfterConsensusPubKeyUpdate(_ context.Context, _, _ cryptotypes.PubKey, _ sdk.Coin) error {
	return nil
}

/*
switched a few things thats different from our talk, but the idea is the same to track the delegations and be one with total Power between reporter and staking
when a delegation is created assign the delegator to a val-reporter if the reporter has less than 100 delegators else create reporter where the reporter is the sk-delegator,
when a delegation is created increment the delegation count of the delegator, this is useful during the loop to check token amounts of a delegator that is a part of a total power of a reporter.
the count helps limit the number of iterations to calculate tokens that  are staked with a bonded validator, the count is compared with the max valset count, if the count is greater than the max valset count
then only iterate valset count times else iterate over the delegations of the delegator.

things to do still are have the ability for delegators to switch reporters...
plus...

just rough idea of the flow
*/
func (h Hooks) BeforeDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	reporterKey := valAddr.Bytes()
	iter, err := h.k.Delegators.Indexes.Reporter.MatchExact(ctx, reporterKey)
	if err != nil {
		return err
	}

	pks, err := iter.PrimaryKeys()
	if err != nil {
		return err
	}

	if len(pks) == 0 {
		val, err := h.k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			return err
		}
		reporter := types.OracleReporter{
			TotalTokens: sdkmath.ZeroInt(),
			Commission:  &val.Commission,
			Reporter:    reporterKey,
		}
		if err := h.k.Reporters.Set(ctx, valAddr.Bytes(), reporter); err != nil {
			return err
		}

	}

	if len(pks) > 100 {
		reporterKey = delAddr.Bytes()
		reporterExists, err := h.k.Reporters.Has(ctx, reporterKey)
		if err != nil {
			return err
		}
		if !reporterExists {
			val, err := h.k.stakingKeeper.GetValidator(ctx, valAddr)
			if err != nil {
				return err
			}
			reporter := types.OracleReporter{
				TotalTokens: sdkmath.ZeroInt(),
				Commission:  &val.Commission,
				Reporter:    reporterKey,
			}
			if err := h.k.Reporters.Set(ctx, reporterKey, reporter); err != nil {
				return err
			}
		}
	}

	del, err := h.k.Delegators.Get(ctx, delAddr)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		} else {
			del.DelegationCount = 0
			del.Reporter = reporterKey
		}
	}
	del.DelegationCount++

	return h.k.Delegators.Set(ctx, delAddr, del)
}
func (h Hooks) BeforeDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, _ sdk.ValAddress) error {

	del, err := h.k.Delegators.Get(ctx, delAddr)
	if err != nil {
		return err
	}
	del.DelegationCount--
	if del.DelegationCount == 0 {
		return h.k.Delegators.Remove(ctx, delAddr)
	}
	return h.k.Delegators.Set(ctx, delAddr, del)
}

func (k Keeper) Reporte(ctx context.Context, repAddr sdk.AccAddress) (*types.OracleReporter, error) {
	totalPower := sdkmath.ZeroInt()
	delegators, err := k.Delegators.Indexes.Reporter.MatchExact(ctx, repAddr)
	if err != nil {
		return nil, err
	}
	for ; delegators.Valid(); delegators.Next() {
		key, err := delegators.PrimaryKey()
		if err != nil {
			return nil, err
		}
		valSet := k.stakingKeeper.GetValidatorSet()
		maxVal, err := valSet.MaxValidators(ctx)
		if err != nil {
			return nil, err
		}
		// get delegator count
		delCount, err := k.Delegators.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		if delCount.DelegationCount > uint64(maxVal) {
			// iterate over bonded validators
			err = valSet.IterateBondedValidatorsByPower(ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
				valAddrr, err := sdk.ValAddressFromBech32(validator.GetOperator())
				if err != nil {
					return true
				}
				del, err := k.stakingKeeper.GetDelegation(ctx, key, valAddrr)
				if err != nil {
					return true
				}
				// get the token amount
				tokens := validator.TokensFromSharesTruncated(del.Shares).TruncateInt()
				totalPower = totalPower.Add(tokens)
				return false
			})
			if err != nil {
				return nil, err
			}
		} else {
			err := k.stakingKeeper.IterateDelegatorDelegations(ctx, key, func(delegation stakingtypes.Delegation) bool {
				validatorAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
				if err != nil {
					panic(err) // shouldn't happen
				}
				validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddr)
				if err == nil && validator.IsBonded() {
					shares := delegation.Shares
					tokens := validator.TokensFromSharesTruncated(shares).TruncateInt()
					totalPower = totalPower.Add(tokens)
				}
				return false
			})
			if err != nil {
				return nil, err
			}
		}

	}
	// still todo: populate reporter with other fields
	return &types.OracleReporter{TotalTokens: totalPower}, nil
}
func (k Keeper) GetTokenSourcesForReporter(ctx context.Context, repAddr sdk.AccAddress) (types.DelegationsPreUpdate, error) {
	delegators, err := k.Delegators.Indexes.Reporter.MatchExact(ctx, repAddr)
	if err != nil {
		return types.DelegationsPreUpdate{}, err
	}

	var tokenSources []*types.TokenOriginInfo
	for ; delegators.Valid(); delegators.Next() {
		key, err := delegators.PrimaryKey()
		if err != nil {
			return types.DelegationsPreUpdate{}, err
		}
		rng := collections.NewPrefixedPairRange[[]byte, []byte](key)
		err = k.TokenOrigin.Walk(ctx, rng, func(key collections.Pair[[]byte, []byte], value sdkmath.Int) (bool, error) {
			tokenSources = append(tokenSources, &types.TokenOriginInfo{
				DelegatorAddress: key.K1(),
				ValidatorAddress: key.K2(),
				Amount:           value,
			})
			return false, nil
		})
		if err != nil {
			return types.DelegationsPreUpdate{}, err
		}
	}
	return types.DelegationsPreUpdate{
		TokenOrigins: tokenSources,
	}, nil
}
