package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// HasMin checks if an AccAddress has the minimum amount of tokens required with a BONDED validator
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
		// convert del shares to token amount
		delTokens := val.TokensFromShares(delegation.Shares).TruncateInt()
		tokens = tokens.Add(delTokens)
		// short circuit if we have enough tokens
		return tokens.GTE(minRequired)
	})
	if err != nil {
		return false, err
	}
	return tokens.GTE(minRequired), iterError
}

// ReporterStake counts the total amount of BONDED tokens for a given reporter's selectors
// at the time of reporting and returns the total amount plus stores
// the token origins for each selector which is needed during a dispute for slashing/returning tokens to appropriate parties
func (k Keeper) ReporterStake(ctx context.Context, repAddr sdk.AccAddress, queryId []byte) (math.Int, error) {
	totalTokens, delegates, err := k.GetReporterStake(ctx, repAddr, queryId)
	if err != nil {
		return math.Int{}, err
	}
	err = k.Report.Set(ctx, collections.Join(queryId, collections.Join(repAddr.Bytes(), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))), types.DelegationsAmounts{TokenOrigins: delegates, Total: totalTokens})
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
	// todo: is this itererror necessary?
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

// TotalReporterPower returns the total amount of BONDED tokens in the network
func (k Keeper) TotalReporterPower(ctx context.Context) (math.Int, error) {
	valSet := k.stakingKeeper.GetValidatorSet()
	return valSet.TotalBondedTokens(ctx)
}

// Delegation returns a selector's reporter, delegations count, and locked time information
func (k Keeper) Delegation(ctx context.Context, delegator sdk.AccAddress) (types.Selection, error) {
	return k.Selectors.Get(ctx, delegator)
}

// Reporter returns a reporter's minimum bond requirement, commission rate, jailed status, and locked time information
func (k Keeper) Reporter(ctx context.Context, reporter sdk.AccAddress) (types.OracleReporter, error) {
	return k.Reporters.Get(ctx, reporter.Bytes())
}

// GetNumOfSelectors returns the number of selectors a reporter currently has
func (k Keeper) GetNumOfSelectors(ctx context.Context, repAddr sdk.AccAddress) (int, error) {
	iter, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, repAddr.Bytes())
	if err != nil {
		return 0, err
	}
	keys, err := iter.FullKeys()
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}

func (k Keeper) GetSelector(ctx context.Context, selectorAddr sdk.AccAddress) (types.Selection, error) {
	return k.Selectors.Get(ctx, selectorAddr)
}

// GetReporterStake counts the total amount of BONDED tokens for a given reporter's selectors
// at the time of reporting and returns the total amount plus stores
// the token origins for each selector which is needed during a dispute for slashing/returning tokens to appropriate parties
func (k Keeper) GetReporterStake(ctx context.Context, repAddr sdk.AccAddress, queryId []byte) (math.Int, []*types.TokenOriginInfo, error) {
	reporter, err := k.Reporters.Get(ctx, repAddr.Bytes())
	if err != nil {
		return math.Int{}, nil, err
	}
	if reporter.Jailed {
		return math.Int{}, nil, errorsmod.Wrapf(types.ErrReporterJailed, "reporter %s is in jail", repAddr.String())
	}
	selection, err := k.Selectors.Get(ctx, repAddr.Bytes())
	if err != nil {
		return math.Int{}, nil, err
	}
	k.logger.Info(fmt.Sprintf("Rep selection: %v", selection))
	totalTokens := math.ZeroInt()
	iter, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, repAddr)
	if err != nil {
		return math.Int{}, nil, err
	}
	k.logger.Info(fmt.Sprintf("Is Iter Valid: %t", iter.Valid()))
	defer iter.Close()
	delegates := make([]*types.TokenOriginInfo, 0)
	for ; iter.Valid(); iter.Next() {
		selectorAddr, err := iter.PrimaryKey()
		k.logger.Info("Selector Addr: ", hex.EncodeToString(selectorAddr))
		if err != nil {
			return math.Int{}, nil, err
		}
		valSet := k.stakingKeeper.GetValidatorSet()
		maxValSet, err := valSet.MaxValidators(ctx)
		if err != nil {
			return math.Int{}, nil, err
		}
		// get delegator count
		selector, err := k.Selectors.Get(ctx, selectorAddr)
		if err != nil {
			return math.Int{}, nil, err
		}
		// skip selectors that are locked out for switching reporters
		if selector.LockedUntilTime.After(sdk.UnwrapSDKContext(ctx).BlockTime()) {
			continue
		}
		var iterError error
		// compare how many delegations a selector has to the max validators to detemine if you should short circuit and iterate the counts number of times
		// or iterate over all bonded validators for a selector in the case they have more delegations (with multiple validators, bonded or not) than the max bonded validators
		if selector.DelegationsCount > uint64(maxValSet) {
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
				return math.Int{}, nil, err
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
					delTokens := val.TokensFromSharesTruncated(delegation.Shares).TruncateInt()
					totalTokens = totalTokens.Add(delTokens)
					delegates = append(delegates, &types.TokenOriginInfo{DelegatorAddress: selectorAddr, ValidatorAddress: valAddr.Bytes(), Amount: delTokens})
				}
				return false
			})
			if err != nil {
				return math.Int{}, nil, err
			}
		}
		if iterError != nil {
			return math.Int{}, nil, iterError
		}
	}
	return totalTokens, delegates, nil
}
