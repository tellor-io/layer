package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// distributes tips paid in oracle module to delegators that were part of reporting the tip's report
func (k Keeper) DivvyingTips(ctx context.Context, reporterAddr sdk.AccAddress, reward math.LegacyDec, height int64) error {
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return err
	}
	// Calculate commission
	commission := reward.Mul(reporter.CommissionRate)

	// Calculate net reward
	netReward := reward.Sub(commission)

	delAddrs, err := k.Report.Get(ctx, collections.Join(reporterAddr.Bytes(), height))
	if err != nil {
		return err
	}

	for _, del := range delAddrs.TokenOrigins {
		delegatorShare := netReward.Mul(math.LegacyNewDecFromInt(del.Amount)).Quo(math.LegacyNewDecFromInt(delAddrs.Total))
		if bytes.Equal(del.DelegatorAddress, reporterAddr.Bytes()) {
			delegatorShare = delegatorShare.Add(commission)
		}
		// get delegator's tips and add the new tip
		oldTips, err := k.SelectorTips.Get(ctx, del.DelegatorAddress)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				oldTips = math.LegacyZeroDec()
			} else {
				return err
			}
		}
		err = k.SelectorTips.Set(ctx, del.DelegatorAddress, oldTips.Add(delegatorShare))
		if err != nil {
			return err
		}
	}

	return nil
}

// ReturnSlashedTokens returns the slashed tokens to the delegators,
// called in dispute module after dispute is resolved with result invalid or reporter wins
func (k Keeper) ReturnSlashedTokens(ctx context.Context, amt math.Int, hashId []byte) error {
	snapshot, err := k.DisputedDelegationAmounts.Get(ctx, hashId)
	if err != nil {
		return err
	}
	// possible additional tokens to give reporter in case of a dispute for the reporter
	extra := amt.Sub(snapshot.Total)
	for _, source := range snapshot.TokenOrigins {
		valAddr := sdk.ValAddress(source.ValidatorAddress)

		var val stakingtypes.Validator
		val, err = k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			if !errors.Is(err, stakingtypes.ErrNoValidatorFound) {
				return err
			}
			vals, err := k.GetBondedValidators(ctx, 1)
			if err != nil {
				return err
			}
			if len(vals) == 0 {
				return errors.New("no validators found in staking module to return tokens to")
			}
			val = vals[0]
		}
		delAddr := sdk.AccAddress(source.DelegatorAddress)

		// set token source to bonded if validator is bonded
		// if not, set to unbonded
		// this causes the delegate method to not transfer tokens since tokens
		// are transferred via dispute module where ReturnSlashedTokens is called
		var tokenSrc stakingtypes.BondStatus
		if val.IsBonded() {
			tokenSrc = stakingtypes.Bonded
		} else {
			tokenSrc = stakingtypes.Unbonded
		}
		shareAmt := math.LegacyNewDecFromInt(source.Amount)
		if extra.IsPositive() {
			// add extra tokens based on the share of the delegator
			shareAmt = math.LegacyNewDecFromInt(source.Amount).Quo(math.LegacyNewDecFromInt(snapshot.Total)).Mul(math.LegacyNewDecFromInt(amt))
		}
		_, err = k.stakingKeeper.Delegate(ctx, delAddr, shareAmt.TruncateInt(), tokenSrc, val, false)
		if err != nil {
			return err
		}

	}

	return k.DisputedDelegationAmounts.Remove(ctx, hashId)
}

// called in dispute module after dispute is resolved
// returns the fee to the delegators that paid minus burn amount
func (k Keeper) FeeRefund(ctx context.Context, hashId []byte, amt math.Int) error {
	trackedFees, err := k.FeePaidFromStake.Get(ctx, hashId)
	if err != nil {
		return err
	}

	for _, source := range trackedFees.TokenOrigins {
		val, err := k.stakingKeeper.GetValidator(ctx, sdk.ValAddress(source.ValidatorAddress))
		if err != nil {
			if !errors.Is(err, stakingtypes.ErrNoValidatorFound) {
				return err
			}
			vals, err := k.GetBondedValidators(ctx, 1)
			if err != nil {
				return err
			}
			if len(vals) == 0 {
				return errors.New("no validators found in staking module to return tokens to")
			}
			val = vals[0]
		}
		// since fee paid is returned minus the voter/burned amount, calculate by accordingly
		shareAmt := math.LegacyNewDecFromInt(source.Amount).Quo(math.LegacyNewDecFromInt(trackedFees.Total)).Mul(math.LegacyNewDecFromInt(amt))
		_, err = k.stakingKeeper.Delegate(ctx, sdk.AccAddress(source.DelegatorAddress), shareAmt.TruncateInt(), stakingtypes.Bonded, val, false)
		if err != nil {
			return err
		}
	}
	return k.FeePaidFromStake.Remove(ctx, hashId)
}

func (k Keeper) GetBondedValidators(ctx context.Context, max uint32) ([]stakingtypes.Validator, error) {
	validators := make([]stakingtypes.Validator, max)

	iterator, err := k.stakingKeeper.ValidatorsPowerStoreIterator(ctx)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	i := 0
	for ; iterator.Valid() && i < int(max); iterator.Next() {
		address := iterator.Value()
		validator, err := k.stakingKeeper.GetValidator(ctx, address)
		if err != nil {
			return nil, fmt.Errorf("validator record not found for address: %X", address)
		}

		if validator.IsBonded() {
			validators[i] = validator
			i++
		}
	}

	return validators[:i], nil // trim
}

func (k Keeper) AddAmountToStake(ctx context.Context, acc sdk.AccAddress, amt math.Int) error {
	vals, err := k.GetBondedValidators(ctx, 1)
	if err != nil {
		return err
	}
	validator := vals[0]

	_, err = k.stakingKeeper.Delegate(ctx, acc, amt, stakingtypes.Bonded, validator, false)
	if err != nil {
		return err
	}
	return nil
}
