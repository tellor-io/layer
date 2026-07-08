package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k Keeper) MaxBondableAmountExported(
	ctx context.Context,
	delAddr sdk.AccAddress,
	validator stakingtypes.Validator,
	maxValidatorShare math.LegacyDec,
	maxReporterShare math.LegacyDec,
	projectedBondedDelta math.Int,
) (math.Int, error) {
	return k.maxBondableAmount(ctx, delAddr, validator, maxValidatorShare, maxReporterShare, projectedBondedDelta)
}

func (k Keeper) GetDstValidatorExported(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.ValAddress, error) {
	return k.getDstValidator(ctx, delAddr, valAddr)
}
