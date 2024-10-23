package keeper

import (
	"context"
	"errors"
	"fmt"

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

func (k Keeper) FeefromReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, amt math.Int, hashId []byte) error {
	reporterTotalTokens := math.LegacyZeroDec()
	fee := math.LegacyNewDecFromInt(amt)
	var iterError error
	// Calculate each delegator's share (including the reporter as a self-delegator)
	iter, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, reporterAddr)
	if err != nil {
		return err
	}

	selectorsMap := make([]selectorsInfo, 0)
	var selectorShareslist []selectorShares
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		selectorKey, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		selectorShareslist = make([]selectorShares, 0)
		selectorTotalTokens := math.LegacyZeroDec()
		err = k.stakingKeeper.IterateDelegatorDelegations(ctx, sdk.AccAddress(selectorKey), func(delegation stakingtypes.Delegation) bool {
			valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
			if err != nil {
				iterError = err
				return true
			}
			validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
			if err != nil {
				iterError = err
				return true
			}
			if validator.IsBonded() {
				delTokens := validator.TokensFromShares(delegation.Shares)
				selectorTotalTokens = selectorTotalTokens.Add(delTokens)
				selectorShareslist = append(selectorShareslist, selectorShares{valAddr: valAddr, validator: validator, shares: delegation.Shares})
			}
			return false
		})
		if err != nil {
			return err
		}
		if iterError != nil {
			return iterError
		}
		reporterTotalTokens = reporterTotalTokens.Add(selectorTotalTokens)
		selectorsMap = append(selectorsMap, selectorsInfo{delAddr: sdk.AccAddress(selectorKey), selectorTotalTokens: selectorTotalTokens, selectorInfo: selectorShareslist})

	}

	if fee.GT(reporterTotalTokens) {
		return errors.New("insufficient stake to pay fee")
	}
	feeTracker := make([]*types.TokenOriginInfo, 0)
	totalTrackedAmount := math.ZeroInt()
	for _, selectors := range selectorsMap {
		feeshareAmt := selectors.selectorTotalTokens.Quo(reporterTotalTokens).Mul(fee)
		unbondAmt := feeshareAmt
		for _, info := range selectors.selectorInfo {
			stakeWithValidator := info.validator.TokensFromShares(info.shares)
			if stakeWithValidator.GTE(unbondAmt) {
				sharesToUnbond, err := info.validator.SharesFromTokens(unbondAmt.TruncateInt())
				if err != nil {
					return err
				}
				// Unbond and move tokens from validator
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
				unbondAmt = unbondAmt.Sub(stakeWithValidator)
				escrowedAmt, err := k.stakingKeeper.Unbond(ctx, selectors.delAddr, info.valAddr, info.shares)
				if err != nil {
					return err
				}
				feeTracker = append(feeTracker, &types.TokenOriginInfo{
					DelegatorAddress: selectors.delAddr.Bytes(),
					ValidatorAddress: info.valAddr.Bytes(),
					Amount:           unbondAmt.TruncateInt(),
				})
				totalTrackedAmount = totalTrackedAmount.Add(escrowedAmt)

				if unbondAmt.IsZero() {
					break
				}

			}
		}

	}
	has, err := k.FeePaidFromStake.Has(ctx, hashId)
	if err != nil {
		return err
	}
	prevTotal := math.ZeroInt()
	if has {
		preFeeTracker, err := k.FeePaidFromStake.Get(ctx, hashId)
		if err != nil {
			return err
		}

		feeTracker = append(feeTracker, preFeeTracker.TokenOrigins...)
		prevTotal = preFeeTracker.Total
	}

	err = k.tokensToDispute(ctx, stakingtypes.BondedPoolName, totalTrackedAmount)
	if err != nil {
		return err
	}
	if err := k.FeePaidFromStake.Set(ctx, hashId, types.DelegationsAmounts{TokenOrigins: feeTracker, Total: totalTrackedAmount.Add(prevTotal)}); err != nil {
		return err
	}
	return nil
}

func (k Keeper) EscrowReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, power, height uint64, amt math.Int, hashId []byte) error {
	report, err := k.Report.Get(ctx, collections.Join(reporterAddr.Bytes(), height))
	if err != nil {
		return err
	}

	totalTokens := layertypes.PowerReduction.MulRaw(int64(power))
	disputeTokens := make([]*types.TokenOriginInfo, 0)
	leftover := math.NewUint(amt.Uint64() * 1e6)
	for i, del := range report.TokenOrigins {
		truncDelAmount := math.NewUint(del.Amount.Uint64()).QuoUint64(layertypes.PowerReduction.Uint64()).MulUint64(layertypes.PowerReduction.Uint64())
		fmt.Printf("\npower: %d, height: %d, amt: %v, totalTokens: %v, delAmount: %v\r", power, height, amt, totalTokens, truncDelAmount)
		// delegatorShare := math.LegacyNewDecFromInt(del.Amount).Quo(math.LegacyNewDecFromInt(totalTokens)).Mul(math.LegacyNewDecFromInt(amt))
		delegatorShare := truncDelAmount.MulUint64(amt.Uint64()).MulUint64(1e6).Quo(math.NewUint(totalTokens.Uint64()))
		fmt.Printf("\nLeftover: %v, delShare: %v\r", leftover, delegatorShare)
		leftover = leftover.Sub(delegatorShare)

		if i == len(report.TokenOrigins)-1 {
			delegatorShare = delegatorShare.Add(leftover)
		}

		delAddr := sdk.AccAddress(del.DelegatorAddress)
		valAddr := sdk.ValAddress(del.ValidatorAddress)

		remaining, err := k.undelegate(ctx, delAddr, valAddr, delegatorShare)
		if err != nil {
			return err
		}
		storedAmount := delegatorShare.Sub(math.NewUint(remaining.Uint64()))
		storedAmountFixed6 := storedAmount.Quo(math.NewUint(1e6))
		if !storedAmount.IsZero() {
			disputeTokens = append(disputeTokens, &types.TokenOriginInfo{
				DelegatorAddress: del.DelegatorAddress,
				ValidatorAddress: del.ValidatorAddress,
				Amount:           math.NewIntFromUint64(storedAmountFixed6.Uint64()),
			})
		}

		remainingFixed6 := remaining.Quo(math.NewUint(1e6))
		if !remaining.IsZero() {
			dstVAl, err := k.getDstValidator(ctx, delAddr, valAddr)
			if err != nil {
				return err
			}
			_, err = k.undelegate(ctx, delAddr, dstVAl, remaining)
			if err != nil {
				return err
			}
			disputeTokens = append(disputeTokens, &types.TokenOriginInfo{
				DelegatorAddress: del.DelegatorAddress,
				ValidatorAddress: dstVAl,
				Amount:           math.NewIntFromUint64(remainingFixed6.Uint64()),
			})
		}
	}

	// after escrow you should keep a new snapshot of the amounts from each that were taken instead of relying on the original snapshot
	// then you can delete it after the slashed tokens are returned
	return k.DisputedDelegationAmounts.Set(ctx, hashId, types.DelegationsAmounts{TokenOrigins: disputeTokens, Total: amt})
}

// get dst validator for a redelegated delegator
func (k Keeper) getDstValidator(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.ValAddress, error) {
	reds, err := k.stakingKeeper.GetRedelegationsFromSrcValidator(ctx, valAddr)
	if err != nil {
		return nil, err
	}
	for _, red := range reds {
		if red.DelegatorAddress == delAddr.String() {
			valAddr, err := sdk.ValAddressFromBech32(red.ValidatorDstAddress)
			if err != nil {
				return nil, err
			}
			return valAddr, nil
		}
	}
	return nil, errors.New("redelegation to destination validator not found")
}

func (k Keeper) deductUnbondingDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, tokens math.Uint) (math.Uint, error) {
	ubd, err := k.stakingKeeper.GetUnbondingDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return math.Uint{}, err
	}

	if len(ubd.Entries) == 0 {
		return math.Uint{}, types.ErrNoUnbondingDelegationEntries
	}
	removeAmt := math.ZeroUint()
	for i, u := range ubd.Entries {
		normalizedBalance := math.NewUint(u.Balance.Uint64() * 1e6)
		if normalizedBalance.LT(tokens) {
			tokens = tokens.Sub(normalizedBalance)
			removeAmt = removeAmt.Add(normalizedBalance)
			ubd.RemoveEntry(int64(i))
		} else {
			u.Balance = math.NewIntFromUint64(normalizedBalance.Sub(tokens).QuoUint64(1e6).Uint64())
			u.InitialBalance = u.InitialBalance.Sub(math.NewIntFromUint64(tokens.QuoUint64(1e6).Uint64()))
			ubd.Entries[i] = u
			removeAmt = removeAmt.Add(tokens)
			tokens = math.ZeroUint()
			break
		}
	}

	if len(ubd.Entries) == 0 {
		err = k.stakingKeeper.RemoveUnbondingDelegation(ctx, ubd)
	} else {
		err = k.stakingKeeper.SetUnbondingDelegation(ctx, ubd)
	}
	if err != nil {
		return math.Uint{}, err
	}
	err = k.tokensToDispute(ctx, stakingtypes.NotBondedPoolName, (math.NewInt(int64(removeAmt.Uint64())).QuoRaw(1e6)))
	if err != nil {
		return math.Uint{}, err
	}
	return tokens, nil
}

func (k Keeper) deductFromdelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, delTokens math.Uint) (math.Uint, error) {
	// get delegation
	del, err := k.stakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		if errors.Is(err, stakingtypes.ErrNoDelegation) {
			return delTokens, nil
		}
		return math.Uint{}, err
	}
	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return math.Uint{}, err
	}

	// convert current delegation shares to tokens
	currentTokens := validator.TokensFromShares(del.Shares)
	tokensFromShare := math.NewUint(currentTokens.BigInt().Uint64() * 1e6) // normalize to match with the normalized delTokens
	shares := del.Shares
	if tokensFromShare.GTE(delTokens) {
		shares, err = validator.SharesFromTokens(math.Int(delTokens.QuoUint64(1e6)))
		if err != nil {
			return math.Uint{}, err
		}
		delTokens = math.ZeroUint()
	} else {
		delTokens = delTokens.Sub(tokensFromShare)
	}

	if !tokensFromShare.IsZero() {
		removedTokens, err := k.stakingKeeper.Unbond(ctx, delAddr, valAddr, shares)
		if err != nil {
			return math.Uint{}, err
		}
		err = k.MoveTokensFromValidator(ctx, validator, removedTokens)
		if err != nil {
			return math.Uint{}, err
		}
	}
	// returning normalized version of delTokens
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
	fmt.Println("Tokens to dispute: ", amount)
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromPool, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, amount)))
}

func (k Keeper) undelegate(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, delTokens math.Uint) (math.Uint, error) {
	remainingFromdel, err := k.deductFromdelegation(ctx, delAddr, valAddr, delTokens)
	if err != nil {
		return math.Uint{}, err
	}

	// if tokens are still remaining after removing from delegation, then it could be one of two cases
	// the delegator is unbonding or the delegator has redelegated to another validator
	if remainingFromdel.IsZero() {
		return math.ZeroUint(), nil
	}

	remainingUnbonding, err := k.deductUnbondingDelegation(ctx, delAddr, valAddr, remainingFromdel)
	if err != nil {
		if errors.Is(err, stakingtypes.ErrNoUnbondingDelegation) {
			return remainingFromdel, nil
		}
		return math.Uint{}, err
	}
	if remainingUnbonding.IsZero() {
		return math.ZeroUint(), nil
	}
	return remainingUnbonding, nil
}
