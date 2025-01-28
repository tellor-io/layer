package keeper

import (
	"context"
	"errors"
	"fmt"
	"strings"

	layertypes "github.com/tellor-io/layer/types"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type selectorShares struct {
	valAddr   sdk.ValAddress
	validator stakingtypes.Validator
	shares    math.LegacyDec
}
type selectorsInfo struct {
	delAddr             sdk.AccAddress
	selectorTotalTokens math.LegacyDec
	selectorInfo        []selectorShares
}

// FeefromReporterStake enables a reporter to pay a dispute fee from their stake power.
// hashId is the dispute identifier, needed in the case where a reporter's fee is returned when a dispute is invalid.
func (k Keeper) FeefromReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, amt math.Int, hashId []byte, isFirstRound bool) error {
	reporterTotalTokens := math.LegacyZeroDec()
	fee := math.LegacyNewDecFromInt(amt)

	// Get all selectors for the reporter
	iter, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, reporterAddr)
	if err != nil {
		return err
	}
	defer iter.Close()

	selectorsList := make([]selectorsInfo, 0)
	// calculate total tokens for the reporter by summing up the total tokens of all selectors
	for ; iter.Valid(); iter.Next() {
		selectorKey, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		selectorAddr := sdk.AccAddress(selectorKey)
		// Initialize variables for the current selector
		selectorSharesList := make([]selectorShares, 0)
		selectorTotalTokens := math.LegacyZeroDec()

		// Iterate through delegations for the selector
		err = k.stakingKeeper.IterateDelegatorDelegations(ctx, selectorAddr, func(delegation stakingtypes.Delegation) bool {
			valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
			if err != nil {
				return true
			}
			validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
			if err != nil {
				return true
			}
			if validator.IsBonded() {
				delTokens := validator.TokensFromShares(delegation.Shares)
				selectorTotalTokens = selectorTotalTokens.Add(delTokens)
				selectorSharesList = append(selectorSharesList,
					selectorShares{valAddr: valAddr, validator: validator, shares: delegation.Shares})
			}
			return false
		})
		if err != nil {
			return err
		}

		// Accumulate total tokens for the reporter
		reporterTotalTokens = reporterTotalTokens.Add(selectorTotalTokens)
		selectorsList = append(selectorsList, selectorsInfo{
			delAddr:             selectorAddr,
			selectorTotalTokens: selectorTotalTokens,
			selectorInfo:        selectorSharesList,
		})
	}

	// Check if reporter has enough stake to cover the fee
	if fee.GT(reporterTotalTokens) {
		return errors.New("insufficient stake to pay fee")
	}

	feeTracker := make([]*types.TokenOriginInfo, 0)
	totalTrackedAmount := math.ZeroInt()

	// Process fee payment by unbonding shares from selectors' stake
	// undelegate a proportional amount of tokens from each selector
	for _, selectors := range selectorsList {
		feeShareAmt := selectors.selectorTotalTokens.Quo(reporterTotalTokens).Mul(fee)
		unbondAmt := feeShareAmt

		for _, info := range selectors.selectorInfo {
			// convert shares to token amount
			stakeWithValidator := info.validator.TokensFromShares(info.shares)
			// if selectors stake meets their share of the fee then unbond the amount and break
			if stakeWithValidator.GTE(unbondAmt) {
				sharesToUnbond, err := info.validator.SharesFromTokens(unbondAmt.TruncateInt())
				if err != nil {
					return err
				}
				// Unbond and move tokens out of validator
				escrowedAmt, err := k.stakingKeeper.Unbond(ctx, selectors.delAddr, info.valAddr, sharesToUnbond)
				if err != nil {
					return err
				}

				feeTracker = append(feeTracker, &types.TokenOriginInfo{
					DelegatorAddress: selectors.delAddr.Bytes(),
					ValidatorAddress: info.valAddr.Bytes(),
					Amount:           unbondAmt.TruncateInt(),
				})
				totalTrackedAmount = totalTrackedAmount.Add(escrowedAmt)
				break
			} else {
				// Unbond all shares if not enough stake with the current validator then move to the next validator
				unbondAmt = unbondAmt.Sub(stakeWithValidator)
				escrowedAmt, err := k.stakingKeeper.Unbond(ctx, selectors.delAddr, info.valAddr, info.shares)
				if err != nil {
					return err
				}
				feeTracker = append(feeTracker, &types.TokenOriginInfo{
					DelegatorAddress: selectors.delAddr.Bytes(),
					ValidatorAddress: info.valAddr.Bytes(),
					Amount:           escrowedAmt,
				})
				totalTrackedAmount = totalTrackedAmount.Add(escrowedAmt)

				if unbondAmt.IsZero() {
					break
				}

			}
		}

	}
	// check if reporter has paid some fee before for the same dispute
	hasPaid, err := k.FeePaidFromStake.Has(ctx, hashId)
	if err != nil {
		return err
	}
	prevTotal := math.ZeroInt()
	if hasPaid {
		prevFeeTracker, err := k.FeePaidFromStake.Get(ctx, hashId)
		if err != nil {
			return err
		}
		feeTracker = append(feeTracker, prevFeeTracker.TokenOrigins...)
		prevTotal = prevFeeTracker.Total
	}

	// move the tokens from the bonded pool (in staking module) to the dispute module
	if err := k.tokensToDispute(ctx, stakingtypes.BondedPoolName, totalTrackedAmount); err != nil {
		return err
	}
	// Only track the fee if this is round 1
	if isFirstRound {
		if err := k.FeePaidFromStake.Set(ctx, hashId, types.DelegationsAmounts{
			TokenOrigins: feeTracker,
			Total:        totalTrackedAmount.Add(prevTotal),
		}); err != nil {
			return err
		}
	}
	return nil
}

// EscrowReporterStake moves tokens from the reporter's stake (from staking module) to the dispute module
func (k Keeper) EscrowReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, power, height uint64, amt math.Int, queryId, hashId []byte) error {
	report, err := k.Report.Get(ctx, collections.Join(queryId, collections.Join(reporterAddr.Bytes(), height)))
	if err != nil {
		return err
	}

	totalTokens := layertypes.PowerReduction.MulRaw(int64(power))
	disputeTokens := make([]*types.TokenOriginInfo, 0)
	leftover := amt
	// loop through the selectors' tokens (validator, amount) that were part of the report and remove tokens from relevant delegations
	// amount should be proportional to the total tokens the reporter had at the time of the report
	for i, del := range report.TokenOrigins {
		delegatorShare := math.LegacyNewDecFromInt(del.Amount).Quo(math.LegacyNewDecFromInt(totalTokens)).Mul(math.LegacyNewDecFromInt(amt)).RoundInt()
		leftover = leftover.Sub(delegatorShare)

		// leftover amount is taken from the last selector in the iteration
		if i == len(report.TokenOrigins)-1 {
			delegatorShare = delegatorShare.Add(leftover)
		}

		delAddr := sdk.AccAddress(del.DelegatorAddress)
		valAddr := sdk.ValAddress(del.ValidatorAddress)

		remaining, err := k.undelegate(ctx, delAddr, valAddr, delegatorShare.ToLegacyDec(), false)
		if err != nil {
			return err
		}

		storedAmount := delegatorShare.Sub(remaining)
		if !storedAmount.IsZero() {
			disputeTokens = append(disputeTokens, &types.TokenOriginInfo{
				DelegatorAddress: del.DelegatorAddress,
				ValidatorAddress: del.ValidatorAddress,
				Amount:           storedAmount,
			})
		}

		if !remaining.IsZero() {
			dstVAl, err := k.getDstValidator(ctx, delAddr, valAddr)
			if err != nil {
				return err
			}
			_, err = k.undelegate(ctx, delAddr, dstVAl, math.LegacyNewDecFromInt(remaining), true)
			if err != nil {
				return err
			}
			disputeTokens = append(disputeTokens, &types.TokenOriginInfo{
				DelegatorAddress: del.DelegatorAddress,
				ValidatorAddress: dstVAl,
				Amount:           remaining,
			})
		}
	}

	// store the disputed amounts information to be used after dispute resolution
	return k.DisputedDelegationAmounts.Set(ctx, hashId, types.DelegationsAmounts{TokenOrigins: disputeTokens, Total: amt})
}

// get the destination validator for a redelegated delegator, used for chasing after tokens that were redelegated to a different validator
func (k Keeper) getDstValidator(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.ValAddress, error) {
	reds, err := k.stakingKeeper.GetRedelegationsFromSrcValidator(ctx, valAddr)
	if err != nil {
		return nil, err
	}
	for _, red := range reds {
		if strings.EqualFold(red.DelegatorAddress, delAddr.String()) {
			valAddr, err := sdk.ValAddressFromBech32(red.ValidatorDstAddress)
			if err != nil {
				return nil, err
			}
			return valAddr, nil
		}
	}
	return nil, errors.New("redelegation to destination validator not found")
}

// chases after unbonding delegations in order to get tokens that are part a new dispute
func (k Keeper) deductUnbondingDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, tokens math.Int) (math.Int, error) {
	ubd, err := k.stakingKeeper.GetUnbondingDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return math.Int{}, err
	}
	if len(ubd.Entries) == 0 {
		return math.Int{}, types.ErrNoUnbondingDelegationEntries
	}
	removeAmt := math.ZeroInt()
	for i, u := range ubd.Entries {
		if u.Balance.LT(tokens) {
			tokens = tokens.Sub(u.Balance)
			removeAmt = removeAmt.Add(u.Balance)
			ubd.RemoveEntry(int64(i))
		} else {
			u.Balance = u.Balance.Sub(tokens)
			u.InitialBalance = u.InitialBalance.Sub(tokens)
			ubd.Entries[i] = u
			removeAmt = removeAmt.Add(tokens)
			tokens = math.ZeroInt()
			break
		}
	}

	if len(ubd.Entries) == 0 {
		err = k.stakingKeeper.RemoveUnbondingDelegation(ctx, ubd)
	} else {
		err = k.stakingKeeper.SetUnbondingDelegation(ctx, ubd)
	}
	if err != nil {
		return math.Int{}, err
	}

	err = k.tokensToDispute(ctx, stakingtypes.NotBondedPoolName, removeAmt)
	if err != nil {
		return math.Int{}, err
	}
	return tokens, nil
}

func (k Keeper) deductFromdelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, delTokens math.LegacyDec) (math.LegacyDec, error) {
	// get delegation
	del, err := k.stakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		if errors.Is(err, stakingtypes.ErrNoDelegation) {
			return delTokens, nil
		}
		return math.LegacyDec{}, err
	}
	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return math.LegacyDec{}, err
	}

	// convert current delegation shares to tokens
	currentTokens := validator.TokensFromShares(del.Shares)
	shares := del.Shares
	if currentTokens.GTE(delTokens) {
		shares, err = validator.SharesFromTokens(delTokens.RoundInt())
		if err != nil {
			return math.LegacyDec{}, err
		}
		delTokens = math.LegacyZeroDec()
	} else {
		delTokens = delTokens.Sub(currentTokens)
	}

	if !shares.IsZero() {
		removedTokens, err := k.stakingKeeper.Unbond(ctx, delAddr, valAddr, shares)
		if err != nil {
			return math.LegacyDec{}, err
		}
		err = k.MoveTokensFromValidator(ctx, validator, removedTokens)
		if err != nil {
			return math.LegacyDec{}, err
		}
	}
	return delTokens, nil
}

func (k Keeper) MoveTokensFromValidator(ctx context.Context, validator stakingtypes.Validator, amount math.Int) error {
	var fromPool string
	switch {
	case validator.IsBonded():
		fromPool = stakingtypes.BondedPoolName
	case validator.IsUnbonding():
		fromPool = stakingtypes.NotBondedPoolName
	default:
		return fmt.Errorf("unknown validator status: %s", validator.GetStatus())
	}
	return k.tokensToDispute(ctx, fromPool, amount)
}

func (k Keeper) tokensToDispute(ctx context.Context, fromPool string, amount math.Int) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromPool, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, amount)))
}

// undelegate a selector's tokens that are part of a dispute.
// first attempt to get the tokens from known validator and if not found then chase after the tokens that were either redelegated to another validator
// or are being unbonded
func (k Keeper) undelegate(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, delTokens math.LegacyDec, isRedelegating bool) (math.Int, error) {
	remainingFromdel, err := k.deductFromdelegation(ctx, delAddr, valAddr, delTokens)
	if err != nil {
		return math.Int{}, err
	}
	// if tokens are still remaining after removing from delegation, then it could be one of two cases
	// the delegator is unbonding or the delegator has redelegated to another validator
	if remainingFromdel.IsZero() {
		return math.ZeroInt(), nil
	}

	remainingUnbonding := math.ZeroInt()
	if !isRedelegating {
		remainingUnbonding, err = k.deductUnbondingDelegation(ctx, delAddr, valAddr, remainingFromdel.TruncateInt())
		if err != nil {
			if errors.Is(err, stakingtypes.ErrNoUnbondingDelegation) {
				return remainingFromdel.TruncateInt(), nil
			}
			return math.Int{}, err
		}
		if remainingUnbonding.IsZero() {
			return math.ZeroInt(), nil
		}
	}
	return remainingUnbonding, nil
}
