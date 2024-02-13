package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/reporter/types"
)

func (k Keeper) ValidateAmount(ctx context.Context, delegator sdk.AccAddress, originAmounts []*types.TokenOrigins, amount uint64) error {
	var _amount uint64
	for _, origin := range originAmounts {
		_amount += origin.Amount
	}
	if _amount != amount {
		return fmt.Errorf("Token origin amount does not match the stake amount chosen")
	}
	for _, origin := range originAmounts {
		valAddr, err := sdk.ValAddressFromBech32(origin.ValidatorAddress)
		if err != nil {
			return err
		}
		tokenSource, err := k.TokenOrigin.Get(ctx, collections.Join(delegator, valAddr))
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return fmt.Errorf("Unexpected unable to fetch token origin, error: %s", err)
			} else {
				tokenSource.ValidatorAddress = origin.ValidatorAddress
			}
		}
		var _sumAmount = tokenSource.Amount + origin.Amount
		validator := k.stakingKeeper.Validator(ctx, valAddr)
		if validator == nil {
			return fmt.Errorf("Validator not found: %s", valAddr)
		}
		delegation, err := k.stakingKeeper.Delegation(ctx, delegator, valAddr)
		if err != nil {
			return err
		}
		if validator.TokensFromShares(delegation.GetShares()).TruncateInt().Uint64() < _sumAmount {
			return fmt.Errorf("Insufficient tokens bonded with validator %v", valAddr)
		}
		tokenSource.Amount = _sumAmount
		if err := k.TokenOrigin.Set(ctx, collections.Join(delegator, valAddr), tokenSource); err != nil {
			return err
		}
	}
	return nil
}
