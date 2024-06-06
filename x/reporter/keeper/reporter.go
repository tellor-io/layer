package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

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
	totalPower := math.ZeroInt()
	delegators, err := k.Delegators.Indexes.Reporter.MatchExact(ctx, repAddr)
	if err != nil {
		return math.Int{}, err
	}
	defer delegators.Close()
	delegates := make([]*types.TokenOriginInfo, 0)
	for ; delegators.Valid(); delegators.Next() {
		key, err := delegators.PrimaryKey()
		if err != nil {
			return math.Int{}, err
		}
		valSet := k.stakingKeeper.GetValidatorSet()
		maxVal, err := valSet.MaxValidators(ctx)
		if err != nil {
			return math.Int{}, err
		}
		// get delegator count
		delCount, err := k.Delegators.Get(ctx, key)
		if err != nil {
			return math.Int{}, err
		}
		// if the delegator has more than the max validators, iterate over all bonded validators
		// else iterate over the delegations, so that can we iterate over the shorter list
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
				delegates = append(delegates, &types.TokenOriginInfo{DelegatorAddress: key, ValidatorAddress: valAddrr.Bytes(), Amount: tokens})
				return false
			})
			if err != nil {
				return math.Int{}, err
			}
		} else {
			err := k.stakingKeeper.IterateDelegatorDelegations(ctx, key, func(delegation stakingtypes.Delegation) bool {
				validatorAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
				if err != nil {
					panic(err)
				}
				validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddr)
				if err == nil && validator.IsBonded() {
					shares := delegation.Shares
					tokens := validator.TokensFromSharesTruncated(shares).TruncateInt()
					totalPower = totalPower.Add(tokens)
					delegates = append(delegates, &types.TokenOriginInfo{DelegatorAddress: key, ValidatorAddress: validatorAddr.Bytes(), Amount: tokens})
				}
				return false
			})
			if err != nil {
				return math.Int{}, err
			}
		}

	}
	err = k.Report.Set(ctx, collections.Join(repAddr.Bytes(), sdk.UnwrapSDKContext(ctx).BlockHeight()), types.DelegationsAmounts{TokenOrigins: delegates, Total: totalPower})
	if err != nil {
		return math.Int{}, err
	}
	return totalPower, nil
}

func (k Keeper) TotalReporterPower(ctx context.Context) (math.Int, error) {
	valSet := k.stakingKeeper.GetValidatorSet()
	return valSet.TotalBondedTokens(ctx)
}

// alias
func (k Keeper) Delegation(ctx context.Context, delegator sdk.AccAddress) (types.Delegation, error) {
	return k.Delegators.Get(ctx, delegator)
}

func (k Keeper) Reporter(ctx context.Context, reporter sdk.AccAddress) (types.OracleReporter, error) {
	return k.Reporters.Get(ctx, reporter.Bytes())
}
