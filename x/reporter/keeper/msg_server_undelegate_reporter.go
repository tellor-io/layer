package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
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
	var reducedbyAmount = math.ZeroInt()
	for _, source := range msg.TokenOrigins {
		valAddr, err := sdk.ValAddressFromBech32(source.ValidatorAddress)
		if err != nil {
			return nil, err
		}
		_source, err := k.TokenOrigin.Get(ctx, collections.Join(delAddr, valAddr))
		if err != nil {
			return nil, err
		}
		if err := k.UpdateOrRemoveSource(ctx, collections.Join(delAddr, valAddr), _source, source.Amount); err != nil {
			return nil, err
		}
		reducedbyAmount = reducedbyAmount.Add(source.Amount)
	}

	if err := k.UpdateOrRemoveDelegator(ctx, delAddr, delegation, reporter, reducedbyAmount); err != nil {
		return nil, err
	}
	if err := k.UpdateOrRemoveReporter(ctx, repAddr, reporter, reducedbyAmount); err != nil {
		return nil, err
	}

	return &types.MsgUndelegateReporterResponse{}, nil
}
