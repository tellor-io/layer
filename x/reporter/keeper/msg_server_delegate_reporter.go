package keeper

import (
	"context"
	"errors"

	errorsmod "cosmossdk.io/errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/reporter/types"
)

func (k msgServer) DelegateReporter(goCtx context.Context, msg *types.MsgDelegateReporter) (*types.MsgDelegateReporterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	delAddr := sdk.MustAccAddressFromBech32(msg.Delegator)
	repAddr := sdk.MustAccAddressFromBech32(msg.Reporter)

	// fetch reporter
	reporter, err := k.Reporters.Get(ctx, repAddr)
	if err != nil {
		return nil, err
	}
	delegation, err := k.Delegators.Get(ctx, delAddr)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		} else {
			// create a new delegation
			// **********************  BeforeDelegationCreated  hook **************************************
			if err := k.BeforeDelegationCreated(ctx, reporter); err != nil {
				return nil, err
			}
			delegation.Reporter = msg.Reporter
			delegation.Amount = msg.Amount
		}
	}
	if err == nil {
		// found delegation, update the amount
		// validate right reporter selected
		if delegation.Reporter != msg.Reporter {
			return nil, errorsmod.Wrapf(types.ErrInvalidReporter, "Reporter mismatch for delegated address %s, expected %s, got %s", msg.Delegator, delegation.Reporter, msg.Reporter)
		}
		// **********************  BeforeDelegationModified  hook **************************************
		if err := k.BeforeDelegationModified(ctx, delAddr, delegation, reporter); err != nil {
			return nil, err
		}
		delegation.Amount = delegation.Amount.Add(msg.Amount)
	}
	if err := k.Keeper.ValidateAmount(ctx, delAddr, msg.TokenOrigins, msg.Amount); err != nil {
		return nil, err
	}
	if err := k.Delegators.Set(ctx, delAddr, delegation); err != nil {
		return nil, err
	}
	// **********************  AfterDelegationModified  hook **************************************
	if err := k.AfterDelegationModified(ctx, delAddr, sdk.ValAddress(repAddr), delegation.Amount); err != nil {
		return nil, err
	}
	// update reporter total tokens
	reporter.TotalTokens = reporter.TotalTokens.Add(msg.Amount)
	if err := k.Reporters.Set(ctx, repAddr, reporter); err != nil {
		return nil, err
	}

	return &types.MsgDelegateReporterResponse{}, nil
}
