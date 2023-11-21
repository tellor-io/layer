package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k Keeper) Refund(ctx sdk.Context, consAddr sdk.ConsAddress, power int64, refundAmount math.Int) math.Int {
	logger := k.Logger(ctx)

	if refundAmount.IsNegative() {
		panic(fmt.Errorf("attempted to slash with a negative slash factor: %v", refundAmount))
	}

	validator, found := k.stakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	if !found {
		// If not found, the validator must have been overslashed and removed - so we don't need to do anything
		// NOTE:  Correctness dependent on invariant that unbonding delegations / redelegations must also have been completely
		//        slashed in this case - which we don't explicitly check, but should be true.
		// Log the slash attempt for future reference (maybe we should tag it too)
		logger.Error(
			"WARNING: ignored attempt to slash a nonexistent validator; we recommend you investigate immediately",
			"validator", consAddr.String(),
		)
		return math.NewInt(0)
	}

	validator = k.ReturnValidatorTokens(ctx, validator, refundAmount)

	k.refundTokens(ctx, refundAmount)

	logger.Info(
		"validator refunded back slashed tokens",
		"validator", validator.GetOperator().String(),
		"refunded", refundAmount,
	)
	return refundAmount
}

// burnNotBondedTokens removes coins from the not bonded pool module account
func (k Keeper) refundTokens(ctx sdk.Context, amt math.Int) error {
	if !amt.IsPositive() {
		// skip as no coins need to be burned
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amt))

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, coins)
}

// RemoveTokens removes tokens from a validator
func AddTokens(v stakingtypes.Validator, tokens math.Int) stakingtypes.Validator {
	if tokens.IsNegative() {
		panic(fmt.Sprintf("should not happen: trying to remove negative tokens %v", tokens))
	}

	v.Tokens = v.Tokens.Add(tokens)

	return v
}

// Update the tokens of an existing validator, update the validators power index key
func (k Keeper) ReturnValidatorTokens(ctx sdk.Context,
	validator stakingtypes.Validator, tokensToAdd math.Int,
) stakingtypes.Validator {
	k.stakingKeeper.DeleteValidatorByPowerIndex(ctx, validator)
	validator = AddTokens(validator, tokensToAdd)
	k.stakingKeeper.SetValidator(ctx, validator)
	k.stakingKeeper.SetValidatorByPowerIndex(ctx, validator)

	return validator
}
