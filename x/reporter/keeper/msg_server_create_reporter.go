package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/reporter/types"
)

func (k msgServer) CreateReporter(goCtx context.Context, msg *types.MsgCreateReporter) (*types.MsgCreateReporterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	reporter := sdk.MustAccAddressFromBech32(msg.Reporter)
	reporterExists, err := k.Reporters.Has(ctx, reporter)
	if err != nil {
		return nil, err
	}
	if reporterExists {
		return nil, fmt.Errorf("Reporter already registered!")
	}
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if msg.Amount < params.MinStakeAmount {
		return nil, fmt.Errorf("Insufficient stake amount")
	}
	// check if reporter is delegated somewhere
	delegatorExists, err := k.Delegators.Has(ctx, reporter)
	if err != nil {
		return nil, err
	}
	if delegatorExists {
		return nil, fmt.Errorf("Reporter already delegated!")
	}
	if err := k.Keeper.ValidateAmount(ctx, reporter, msg.TokenOrigins, msg.Amount); err != nil {
		return nil, err
	}
	// create a new reporter
	if err := k.Reporters.Set(ctx,
		reporter,
		types.OracleReporter{
			Reporter:    msg.Reporter,
			TotalTokens: msg.Amount,
		}); err != nil {
		return nil, err
	}
	// create a new delegation
	if err := k.Delegators.Set(ctx,
		reporter,
		types.Delegation{
			Reporter: msg.Reporter,
			Amount:   msg.Amount,
		}); err != nil {
		return nil, err
	}
	return &types.MsgCreateReporterResponse{}, nil
}
