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
	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)
	delegation, err := k.Delegators.Get(ctx, delAddr)
	if err != nil {
		return nil, err
	}
	// fetch reporter
	repAddr := sdk.AccAddress(delegation.Reporter)
	reporter, err := k.Reporters.Get(ctx, repAddr)
	if err != nil {
		return nil, err
	}
	var reducedbyAmount = math.ZeroInt()
	for _, source := range msg.TokenOrigins {
		valAddr := sdk.ValAddress(source.ValidatorAddress)
		currentSourceAmt, err := k.TokenOrigin.Get(ctx, collections.Join(delAddr.Bytes(), valAddr.Bytes()))
		if err != nil {
			return nil, err
		}
		err = k.UndelegateSource(ctx, collections.Join(delAddr.Bytes(), valAddr.Bytes()), currentSourceAmt, source.Amount)
		if err != nil {
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
