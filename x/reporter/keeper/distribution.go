package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// distributes tips paid in oracle module to delegators that were part of reporting the tip's report
func (k Keeper) DivvyingTips(ctx context.Context, reporterAddr sdk.AccAddress, reward types.BigUint, height uint64) error {
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return err
	}

	// Convert arguments needed for calculations to legacy decimals
	rewardDec := k.LegacyDecFromMathUint(reward.Value)
	commissionRateDec := k.LegacyDecFromMathUint(reporter.CommissionRate)

	// Calculate commission: commission = reward * commissionRate
	commissionDec := rewardDec.Mul(commissionRateDec).Quo(math.LegacyNewDec(1000000))

	commission := k.TruncateUint(commissionDec)

	// Calculate net reward
	netReward := reward.Value.Sub(commission)

	delAddrs, err := k.Report.Get(ctx, collections.Join(reporterAddr.Bytes(), height))
	if err != nil {
		return err
	}

	for _, del := range delAddrs.TokenOrigins {
		// convert args needed for calculations to legacy decimals
		netRewardDec := k.LegacyDecFromMathUint(netReward)
		delAmountDec := math.LegacyNewDecFromInt(del.Amount)
		delTotalDec := math.LegacyNewDecFromInt(delAddrs.Total)
		delegatorShareDec := netRewardDec.Mul(delAmountDec).Quo(delTotalDec)
		delegatorShare := k.TruncateUint(delegatorShareDec)
		if bytes.Equal(del.DelegatorAddress, reporterAddr.Bytes()) {
			delegatorShare = delegatorShare.Add(commission)
		}
		// get delegator's tips and add the new tip
		oldTips, err := k.SelectorTips.Get(ctx, del.DelegatorAddress)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				oldTips = types.BigUint{Value: math.ZeroUint()}
			} else {
				return err
			}
		}
		newTips := types.BigUint{Value: oldTips.Value.Add(delegatorShare)}
		err = k.SelectorTips.Set(ctx, del.DelegatorAddress, newTips)
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
		shareAmt := math.NewUint(source.Amount.Uint64())
		if extra.IsPositive() {
			// add extra tokens based on the share of the delegator
			// convert args needed for calculations to legacy decimals
			sourceAmountDec := math.LegacyNewDecFromInt(source.Amount)
			amountDec := math.LegacyNewDecFromInt(amt)
			snapshotTotalDec := math.LegacyNewDecFromInt(snapshot.Total)
			shareAmtDec := sourceAmountDec.Mul(amountDec).Quo(snapshotTotalDec)
			shareAmt = k.TruncateUint(shareAmtDec)
		}
		_, err = k.stakingKeeper.Delegate(ctx, delAddr, math.NewInt(int64(shareAmt.Uint64())), tokenSrc, val, false)
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
		// convert args needed for calculations to legacy decimals
		sourceAmountDec := math.LegacyNewDecFromInt(source.Amount)
		trackedFeesTotalDec := math.LegacyNewDecFromInt(trackedFees.Total)
		amtDec := math.LegacyNewDecFromInt(amt)
		shareAmtDec := sourceAmountDec.Mul(amtDec).Quo(trackedFeesTotalDec)
		shareAmt := shareAmtDec.TruncateInt()
		_, err = k.stakingKeeper.Delegate(ctx, sdk.AccAddress(source.DelegatorAddress), shareAmt, stakingtypes.Bonded, val, false)
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

// Converts a math.Uint to a legacy decimal
func (k Keeper) LegacyDecFromMathUint(value math.Uint) math.LegacyDec {
	return math.LegacyNewDecFromInt(math.NewIntFromUint64(value.Uint64()))
}

// Truncates a legacy decimal to a math.Uint
func (k Keeper) TruncateUint(value math.LegacyDec) math.Uint {
	return math.NewUint(value.TruncateInt().Uint64())
}
