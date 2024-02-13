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
	minStakeAmount, err := k.MinStakeAmount(ctx)
	if err != nil {
		return nil, err
	}
	if msg.Amount < minStakeAmount {
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
	minCommRate, err := k.MinCommissionRate(ctx)
	if err != nil {
		return nil, err
	}
	if msg.Commission.Rate.LT(minCommRate) {
		return nil, fmt.Errorf("cannot set reporter commission to less than minimum rate of %s", minCommRate)
	}

	commission := types.NewCommissionWithTime(msg.Commission.Rate, msg.Commission.MaxRate,
		msg.Commission.MaxChangeRate, ctx.HeaderInfo().Time)

	if err := commission.Validate(); err != nil {
		return nil, err
	}
	// create a new reporter
	newOracleReporter := types.NewOracleReporter(msg.Reporter, msg.Amount, &commission)
	if err := k.Reporters.Set(ctx, reporter, newOracleReporter); err != nil {
		return nil, err
	}
	// create a new delegation
	newDelegation := types.NewDelegation(msg.Reporter, msg.Amount)
	if err := k.Delegators.Set(ctx, reporter, newDelegation); err != nil {
		return nil, err
	}
	return &types.MsgCreateReporterResponse{}, nil
}
