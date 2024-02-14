package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/reporter/types"
)

func (k msgServer) DelegateReporter(goCtx context.Context, msg *types.MsgDelegateReporter) (*types.MsgDelegateReporterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	delegator := sdk.MustAccAddressFromBech32(msg.Delegator)

	// fetch reporter
	reporter, err := k.Reporters.Get(ctx, sdk.MustAccAddressFromBech32(msg.Reporter))
	if err != nil {
		return nil, err
	}
	delegation, err := k.Delegators.Get(ctx, delegator)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		} else {
			delegation.Reporter = msg.Reporter
			delegation.Amount = msg.Amount
		}
	}
	if err == nil {
		// found delegation, update the amount
		// validate right reporter selected
		if delegation.Reporter != msg.Reporter {
			return nil, fmt.Errorf("A delegation exists but reporter, %s, does not match the selected reporter: %s", delegation.Reporter, msg.Reporter)
		}
		delegation.Amount += msg.Amount
	}
	if err := k.Keeper.ValidateAmount(ctx, delegator, msg.TokenOrigins, msg.Amount); err != nil {
		return nil, err
	}
	if err := k.Delegators.Set(ctx, delegator, delegation); err != nil {
		return nil, err
	}
	// update reporter total tokens
	reporter.TotalTokens += msg.Amount
	if err := k.Reporters.Set(ctx, sdk.MustAccAddressFromBech32(msg.Reporter), reporter); err != nil {
		return nil, err
	}

	return &types.MsgDelegateReporterResponse{}, nil
}
