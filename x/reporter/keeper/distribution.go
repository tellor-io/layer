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

// divvy up the tips that a reporter has earned from reporting in the oracle module amongst the reporters' selectors
// purpose of height argument is to only pay out selectors that were part of the reporter at height of the report
// DivvyingTips distributes the reward among the reporter and its selectors based on their shares at the given height.
//
// The function performs the following steps:
// 1. Retrieves the reporter's information using the reporter's address.
// 2. Converts the reward and commission rate to legacy decimals for calculations.
// 3. Calculates the commission for the reporter and the net reward after commission.
// 4. Retrieves the selectors' addresses and their respective shares.
// 5. Distributes the net reward among the selectors based on their shares.
// 6. Adds the commission to the reporter's share if the reporter is also a selector.
// 7. Updates the selectors' tips with the new calculated shares.
func (k Keeper) DivvyingTips(ctx context.Context, reporterAddr sdk.AccAddress, reward math.LegacyDec, queryId []byte, height uint64) error {
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return err
	}

	// selector's commission = reporter's commission rate * reward
	commission := reward.Mul(reporter.CommissionRate)
	// Calculate net reward
	netReward := reward.Sub(commission)

	delAddrs, err := k.Report.Get(ctx, collections.Join(queryId, collections.Join(reporterAddr.Bytes(), height)))
	if err != nil {
		return err
	}

	for _, del := range delAddrs.TokenOrigins {
		// delegator share = netReward * selector's share / total shares
		delAmountDec := del.Amount.ToLegacyDec()
		delTotalDec := delAddrs.Total.ToLegacyDec()
		delegatorShare := netReward.Mul(delAmountDec).Quo(delTotalDec)

		if bytes.Equal(del.DelegatorAddress, reporterAddr.Bytes()) {
			delegatorShare = delegatorShare.Add(commission)
		}
		// get selector's previous tips
		oldTips, err := k.SelectorTips.Get(ctx, del.DelegatorAddress)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				oldTips = math.LegacyZeroDec()
			} else {
				return err
			}
		}
		// add the new tip to the old tips
		newTips := oldTips.Add(delegatorShare)
		// set new tip total
		err = k.SelectorTips.Set(ctx, del.DelegatorAddress, newTips)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReturnSlashedTokens returns the slashed tokens to the delegators,
// called in dispute module after dispute is resolved with result invalid or reporter wins
func (k Keeper) ReturnSlashedTokens(ctx context.Context, amt math.Int, hashId []byte) (string, error) {
	var pool string
	// get the snapshot of the metadata of the tokens that were slashed ie selectors' shares amounts and validator they were delegated to
	snapshot, err := k.DisputedDelegationAmounts.Get(ctx, hashId)
	if err != nil {
		return "", err
	}

	// winningpurse represents the amount of tokens that a disputed reporter possibly receives for winning a dispute
	winningpurse := amt.Sub(snapshot.Total)
	// for each selector-validator pair, bond the tokens back to the validator, if the validator still exists
	// if not, then find a bonded validator to bond the tokens to
	for _, source := range snapshot.TokenOrigins {
		valAddr := sdk.ValAddress(source.ValidatorAddress)

		var val stakingtypes.Validator
		val, err = k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			if !errors.Is(err, stakingtypes.ErrNoValidatorFound) {
				return "", err
			}
			vals, err := k.GetBondedValidators(ctx, 1)
			if err != nil {
				return "", err
			}
			// this should never happen since there should always be a bonded validator
			if len(vals) == 0 {
				return "", errors.New("no validators found in staking module to return tokens to")
			}
			val = vals[0]
		}
		delAddr := sdk.AccAddress(source.DelegatorAddress)

		// the refund amount is either the amount of tokens that were slashed
		// or the proportion of the slashed tokens plus the winning purse
		shareAmt := math.LegacyNewDecFromInt(source.Amount)
		if winningpurse.IsPositive() {
			// convert args needed for calculations to legacy decimals
			shareAmt = shareAmt.Quo(math.LegacyNewDecFromInt(snapshot.Total)).Mul(math.LegacyNewDecFromInt(amt))
		}
		// set token source to bonded if validator is bonded
		// if not, set to unbonded
		// this causes the delegate method (in staking module) to not transfer tokens since tokens
		// are transferred via dispute module where ReturnSlashedTokens is called
		var tokenSrc stakingtypes.BondStatus
		if val.IsBonded() {
			tokenSrc = stakingtypes.Bonded
			pool = stakingtypes.BondedPoolName
		} else {
			tokenSrc = stakingtypes.Unbonded
			pool = stakingtypes.NotBondedPoolName
		}
		_, err = k.stakingKeeper.Delegate(ctx, delAddr, shareAmt.TruncateInt(), tokenSrc, val, false) // false means to not subtract tokens from an account
		if err != nil {
			return "", err
		}
	}
	return pool, k.DisputedDelegationAmounts.Remove(ctx, hashId)
}

// called in dispute module after dispute is resolved
// returns the fee to the selectors that passively paid minus the burn amount
// refunds the fee paid (minus the burned amount) from the stake to the selectors.
// It retrieves the tracked fees using the provided hashId and calculates the share
// of the amount to be refunded to each selector based on their contribution.
// If the validator associated with the selector is not found or not bonded, it
// selects a bonded validator to delegate the refund amount to.
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
		} else if !val.IsBonded() {
			vals, err := k.GetBondedValidators(ctx, 1)
			if err != nil {
				return err
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

// GetBondedValidators returns a list of BONDED validators up to a given maximum number.
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

// TODO: this should be in dispute module, no reason for it to be in reporter module
// Stakes a given amount of tokens to a BONDED validator from a given address
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
