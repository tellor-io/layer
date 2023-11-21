package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k Keeper) Slash(ctx sdk.Context, opAddr sdk.ValAddress, power int64, slashFactor math.LegacyDec) math.Int {
	logger := k.Logger(ctx)

	if slashFactor.IsNegative() {
		panic(fmt.Errorf("attempted to slash with a negative slash factor: %v", slashFactor))
	}

	// Amount of slashing = slash slashFactor * power at time of infraction
	amount := k.stakingKeeper.TokensFromConsensusPower(ctx, power)
	slashAmountDec := sdk.NewDecFromInt(amount).Mul(slashFactor)
	slashAmount := slashAmountDec.TruncateInt()
	validator, found := k.stakingKeeper.GetValidator(ctx, opAddr)
	if !found {
		// If not found, the validator must have been overslashed and removed - so we don't need to do anything
		// NOTE:  Correctness dependent on invariant that unbonding delegations / redelegations must also have been completely
		//        slashed in this case - which we don't explicitly check, but should be true.
		// Log the slash attempt for future reference (maybe we should tag it too)
		logger.Error(
			"WARNING: ignored attempt to slash a nonexistent validator; we recommend you investigate immediately",
			"validator", opAddr.String(),
		)
		return math.ZeroInt()
	}

	// should not be slashing an unbonded validator
	if validator.IsUnbonded() {
		panic(fmt.Sprintf("should not be slashing unbonded validator: %s", validator.GetOperator()))
	}

	// cannot decrease balance below zero
	tokensToBurn := sdk.MinInt(slashAmount, validator.Tokens)
	tokensToBurn = sdk.MaxInt(tokensToBurn, math.ZeroInt()) // defensive.

	// we need to calculate the *effective* slash fraction for distribution
	if validator.Tokens.IsPositive() {
		effectiveFraction := sdk.NewDecFromInt(tokensToBurn).QuoRoundUp(sdk.NewDecFromInt(validator.Tokens))
		// possible if power has changed
		if effectiveFraction.GT(math.LegacyOneDec()) {
			effectiveFraction = math.LegacyOneDec()
		}
	}

	// Deduct from validator's bonded tokens and update the validator.
	// Burn the slashed tokens from the pool account and decrease the total supply.
	validator = k.stakingKeeper.RemoveValidatorTokens(ctx, validator, tokensToBurn)

	switch validator.GetStatus() {
	case stakingtypes.Bonded:
		if err := k.escrowBondedTokens(ctx, tokensToBurn); err != nil {
			panic(err)
		}
	case stakingtypes.Unbonding, stakingtypes.Unbonded:
		if err := k.escrowNotBondedTokens(ctx, tokensToBurn); err != nil {
			panic(err)
		}
	default:
		panic("invalid validator status")
	}

	logger.Info(
		"validator slashed by slash factor",
		"validator", validator.GetOperator().String(),
		"slash_factor", slashFactor.String(),
		"burned", tokensToBurn,
	)
	return tokensToBurn
}

// burnBondedTokens removes coins from the bonded pool module account
func (k Keeper) escrowBondedTokens(ctx sdk.Context, amt math.Int) error {
	if !amt.IsPositive() {
		// skip as no coins need to be burned
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amt))

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, stakingtypes.BondedPoolName, types.ModuleName, coins)
}

// burnNotBondedTokens removes coins from the not bonded pool module account
func (k Keeper) escrowNotBondedTokens(ctx sdk.Context, amt math.Int) error {
	if !amt.IsPositive() {
		// skip as no coins need to be burned
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amt))

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, stakingtypes.NotBondedPoolName, types.ModuleName, coins)
}
