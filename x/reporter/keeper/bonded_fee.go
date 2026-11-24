package keeper

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// FeefromReporterStake collects exactly amt or returns an error to the message
// transaction, which discards all debits. The sum of individually floored positions
// is an admission ceiling. A split within it can still fail at a live exchange rate.
func (k Keeper) FeefromReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, amt math.Int, hashID []byte, isFirstRound bool) error {
	if amt.IsNil() || amt.IsNegative() {
		return errors.New("bonded fee must be nonnegative")
	}

	origins, collected, err := k.collectBondedFee(ctx, reporterAddr, amt, isFirstRound)
	if err != nil {
		return err
	}
	if !collected.Equal(amt) {
		return fmt.Errorf("bonded fee collection returned %s, expected %s", collected, amt)
	}
	if isFirstRound {
		key := feePaidFromStakePayerKey(hashID, reporterAddr)
		previous, err := k.FeePaidFromStake.Get(ctx, key)
		switch {
		case errors.Is(err, collections.ErrNotFound):
			previous.Total = math.ZeroInt()
		case err != nil:
			return err
		}
		if err := k.FeePaidFromStake.Set(ctx, key, types.DelegationsAmounts{
			TokenOrigins: mergeFeeOrigins(origins, previous.TokenOrigins),
			Total:        previous.Total.Add(collected),
		}); err != nil {
			return err
		}
	}
	return k.tokensToDispute(ctx, stakingtypes.BondedPoolName, collected)
}

func (k Keeper) collectBondedFee(ctx context.Context, reporterAddr sdk.AccAddress, amt math.Int, trackOrigins bool) ([]*types.TokenOriginInfo, math.Int, error) {
	collected := math.ZeroInt()
	if amt.IsZero() {
		return nil, collected, nil
	}
	var origins []*types.TokenOriginInfo
	capacities, positions, total, err := k.bondedFeePositions(ctx, reporterAddr)
	if err != nil {
		return nil, math.Int{}, err
	}
	if amt.GT(total) {
		return nil, math.Int{}, errors.New("insufficient stake to pay fee")
	}
	targets := proportionalReturnAmounts(capacities, total, amt)
	for i, target := range targets {
		delegator := sdk.AccAddress(capacities[i].DelegatorAddress)
		remaining := target
		for _, validator := range positions[i] {
			if remaining.IsZero() {
				break
			}
			taken, err := k.unbondFeePosition(ctx, delegator, validator, remaining)
			if err != nil {
				return nil, math.Int{}, err
			}
			if taken.IsZero() {
				continue
			}
			remaining = remaining.Sub(taken)
			collected = collected.Add(taken)
			if trackOrigins {
				origins = append(origins, &types.TokenOriginInfo{
					DelegatorAddress: delegator,
					ValidatorAddress: validator,
					Amount:           taken,
				})
			}
		}
		if !remaining.IsZero() {
			return nil, math.Int{}, fmt.Errorf("bonded fee target for %s is short by %s", delegator, remaining)
		}
	}
	return origins, collected, nil
}

// bondedFeePositions preserves the keeper's selector and delegation iteration
// orders. Only addresses are retained for the live debit pass.
func (k Keeper) bondedFeePositions(ctx context.Context, reporter sdk.AccAddress) ([]*types.TokenOriginInfo, [][]sdk.ValAddress, math.Int, error) {
	iter, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, reporter)
	if err != nil {
		return nil, nil, math.Int{}, err
	}
	defer iter.Close()
	var capacities []*types.TokenOriginInfo
	var positions [][]sdk.ValAddress
	total := math.ZeroInt()
	for ; iter.Valid(); iter.Next() {
		selector, err := iter.PrimaryKey()
		if err != nil {
			return nil, nil, math.Int{}, err
		}
		capacity := math.ZeroInt()
		var validators []sdk.ValAddress
		var readErr error
		err = k.stakingKeeper.IterateDelegatorDelegations(ctx, sdk.AccAddress(selector), func(del stakingtypes.Delegation) bool {
			valAddr, err := sdk.ValAddressFromBech32(del.ValidatorAddress)
			if err != nil {
				readErr = err
				return true
			}
			val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
			if err != nil {
				readErr = err
				return true
			}
			if feePositionEligible(val) {
				capacity = capacity.Add(val.TokensFromShares(del.Shares).TruncateInt())
				validators = append(validators, valAddr)
			}
			return false
		})
		if err != nil {
			return nil, nil, math.Int{}, err
		}
		if readErr != nil {
			return nil, nil, math.Int{}, readErr
		}
		if capacity.IsPositive() {
			capacities = append(capacities, &types.TokenOriginInfo{DelegatorAddress: selector, Amount: capacity})
			positions = append(positions, validators)
			total = total.Add(capacity)
		}
	}
	return capacities, positions, total, nil
}

func feePositionEligible(val stakingtypes.Validator) bool {
	return val.IsBonded() && val.Tokens.IsPositive() && val.DelegatorShares.IsPositive()
}

func (k Keeper) unbondFeePosition(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, remaining math.Int) (math.Int, error) {
	val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return math.Int{}, err
	}
	del, err := k.stakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return math.Int{}, err
	}
	if !feePositionEligible(val) {
		return math.ZeroInt(), nil
	}
	take := math.MinInt(remaining, val.TokensFromShares(del.Shares).TruncateInt())
	if !take.IsPositive() {
		return math.ZeroInt(), nil
	}

	// Divide in integer share atoms so rounding up is exact even when the
	// quotient lies less than one decimal quantum above a whole atom.
	numerator := new(big.Int).Mul(val.DelegatorShares.BigInt(), take.BigInt())
	shareAtoms, remainder := new(big.Int).QuoRem(numerator, val.Tokens.BigInt(), new(big.Int))
	if remainder.Sign() != 0 {
		shareAtoms.Add(shareAtoms, big.NewInt(1))
	}
	shares := math.LegacyMinDec(math.LegacyNewDecFromBigIntWithPrec(shareAtoms, math.LegacyPrecision), del.Shares)
	actual, err := k.stakingKeeper.Unbond(ctx, delAddr, valAddr, shares)
	if err != nil {
		return math.Int{}, err
	}
	if !actual.Equal(take) {
		return math.Int{}, fmt.Errorf("bonded fee debit for %s at %s returned %s, expected %s", delAddr, valAddr, actual, take)
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		"deducted_stake_in_dispute",
		sdk.NewAttribute("delegator", delAddr.String()),
		sdk.NewAttribute("validator", valAddr.String()),
		sdk.NewAttribute("shares", shares.String()),
		sdk.NewAttribute("amount", actual.String()),
	))
	return actual, nil
}

func mergeFeeOrigins(payment, previous []*types.TokenOriginInfo) []*types.TokenOriginInfo {
	type pair struct{ delegator, validator string }
	byPair := make(map[pair]*types.TokenOriginInfo)
	var merged []*types.TokenOriginInfo
	for _, origins := range [][]*types.TokenOriginInfo{payment, previous} {
		for _, origin := range origins {
			p := pair{string(origin.DelegatorAddress), string(origin.ValidatorAddress)}
			if existing := byPair[p]; existing != nil {
				existing.Amount = existing.Amount.Add(origin.Amount)
			} else {
				entry := *origin
				byPair[p] = &entry
				merged = append(merged, &entry)
			}
		}
	}
	return merged
}
