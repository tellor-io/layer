package keeper

import (
	"context"
	"errors"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

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
		delTokens := val.TokensFromShares(delegation.Shares).TruncateInt()
		tokens = tokens.Add(delTokens)

		return tokens.GTE(minRequired)
	})
	if err != nil {
		return false, err
	}
	return tokens.GTE(minRequired), iterError
}

// Reporter returns the total power of a reporter that is bonded at time of the call
// Store the set of delegations for the reporter at the current block height for dispute purposes to be referenced by block height
func (k Keeper) ReporterStake(ctx context.Context, repAddr sdk.AccAddress) (math.Int, error) {
	reporter, err := k.Reporters.Get(ctx, repAddr.Bytes())
	if err != nil {
		return math.Int{}, err
	}
	if reporter.Jailed {
		return math.Int{}, errorsmod.Wrapf(types.ErrReporterJailed, "reporter %s is in jail", repAddr.String())
	}
	totalTokens := math.ZeroInt()
	iter, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, repAddr)
	if err != nil {
		return math.Int{}, err
	}
	defer iter.Close()
	delegates := make([]*types.TokenOriginInfo, 0)
	for ; iter.Valid(); iter.Next() {
		selectorAddr, err := iter.PrimaryKey()
		if err != nil {
			return math.Int{}, err
		}
		valSet := k.stakingKeeper.GetValidatorSet()
		maxValSet, err := valSet.MaxValidators(ctx)
		if err != nil {
			return math.Int{}, err
		}
		// get delegator count
		selector, err := k.Selectors.Get(ctx, selectorAddr)
		if err != nil {
			return math.Int{}, err
		}
		if selector.LockedUntilTime.After(sdk.UnwrapSDKContext(ctx).BlockTime()) {
			continue
		}
		var iterError error
		if selector.DelegationsCount > int64(maxValSet) {
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
				delegates = append(delegates, &types.TokenOriginInfo{DelegatorAddress: selectorAddr, ValidatorAddress: valAddrr.Bytes(), Amount: tokens})
				return false
			})
			if err != nil {
				return math.Int{}, err
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
					delTokens := val.TokensFromShares(delegation.Shares).TruncateInt()
					totalTokens = totalTokens.Add(delTokens)
					delegates = append(delegates, &types.TokenOriginInfo{DelegatorAddress: selectorAddr, ValidatorAddress: valAddr.Bytes(), Amount: delTokens})
				}
				return false
			})
			if err != nil {
				return math.Int{}, err
			}
		}
		if iterError != nil {
			return math.Int{}, iterError
		}
	}
	err = k.Report.Set(ctx, collections.Join(repAddr.Bytes(), sdk.UnwrapSDKContext(ctx).BlockHeight()), types.DelegationsAmounts{TokenOrigins: delegates, Total: totalTokens})
	if err != nil {
		return math.Int{}, err
	}
	return totalTokens, nil
}

// function that iterates through a selector's delegations and checks if they meet the min requirement
// plus counts how many delegations they have
func (k Keeper) CheckSelectorsDelegations(ctx context.Context, addr sdk.AccAddress) (math.Int, int64, error) {
	tokens := math.ZeroInt()
	var count int64
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

func (k Keeper) TotalReporterPower(ctx context.Context) (math.Int, error) {
	valSet := k.stakingKeeper.GetValidatorSet()
	return valSet.TotalBondedTokens(ctx)
}

// alias
func (k Keeper) Delegation(ctx context.Context, delegator sdk.AccAddress) (types.Selection, error) {
	return k.Selectors.Get(ctx, delegator)
}

func (k Keeper) Reporter(ctx context.Context, reporter sdk.AccAddress) (types.OracleReporter, error) {
	return k.Reporters.Get(ctx, reporter.Bytes())
}
