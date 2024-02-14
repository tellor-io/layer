package keeper

import (
	"context"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/reporter/types"
)

func (k msgServer) UndelegateReporter(goCtx context.Context, msg *types.MsgUndelegateReporter) (*types.MsgUndelegateReporterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// fetch delegation
	delAddr := sdk.MustAccAddressFromBech32(msg.Delegator)
	delegation, err := k.Delegators.Get(ctx, delAddr)
	if err != nil {
		return nil, err
	}
	// fetch reporter
	repAddr := sdk.MustAccAddressFromBech32(delegation.Reporter)
	reporter, err := k.Reporters.Get(ctx, repAddr)
	if err != nil {
		return nil, err
	}
	var _amount uint64
	for _, source := range msg.TokenOrigins {
		valAddr, err := sdk.ValAddressFromBech32(source.ValidatorAddress)
		if err != nil {
			return nil, err
		}
		_tokenSource, err := k.TokenOrigin.Get(ctx, collections.Join(delAddr, valAddr))
		if err != nil {
			return nil, err
		}
		_tokenSource, err = _tokenSource.ReduceTokenOriginAmountby(source.Amount)
		if err != nil {
			return nil, err
		}
		if _tokenSource.Amount <= 0 {
			if err := k.TokenOrigin.Remove(ctx, collections.Join(delAddr, valAddr)); err != nil {
				return nil, err
			}
		} else {
			if err := k.TokenOrigin.Set(ctx, collections.Join(delAddr, valAddr), _tokenSource); err != nil {
				return nil, err
			}
		}
		_amount += source.Amount
	}
	delegation, err = delegation.ReduceDelegationby(_amount)
	if err != nil {
		return nil, err
	}
	reporter, err = reporter.ReduceReporterTokensby(_amount)
	if err != nil {
		return nil, err
	}
	if reporter.TotalTokens <= 0 {
		if err := k.Reporters.Remove(ctx, repAddr); err != nil {
			return nil, err
		}
	} else {
		if err := k.Reporters.Set(ctx, repAddr, reporter); err != nil {
			return nil, err
		}
	}
	if delegation.Amount <= 0 {
		if err := k.Delegators.Remove(ctx, delAddr); err != nil {
			return nil, err
		}
	} else {
		if err := k.Delegators.Set(ctx, delAddr, delegation); err != nil {
			return nil, err
		}
	}

	return &types.MsgUndelegateReporterResponse{}, nil
}
