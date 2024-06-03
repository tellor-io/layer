package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateReporter(goCtx context.Context, msg *types.MsgCreateReporter) (*types.MsgCreateReporterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	reporter := sdk.MustAccAddressFromBech32(msg.Reporter)
	reporterExists, err := k.Reporters.Has(ctx, reporter)
	if err != nil {
		return nil, err
	}
	if reporterExists {
		return nil, errors.Wrapf(types.ErrReporterExists, "cannot create reporter with address %s, it already exists", msg.Reporter)
	}
	// check if reporter is delegated somewhere
	delegatorExists, err := k.Delegators.Has(ctx, reporter)
	if err != nil {
		return nil, err
	}
	if delegatorExists {
		return nil, errors.Wrapf(types.ErrAddressDelegated, "cannot use address %s as reporter as it is already delegated", msg.Reporter)
	}
	if err := k.Keeper.ValidateAndSetAmount(ctx, reporter, msg.TokenOrigins, msg.Amount); err != nil {
		return nil, err
	}
	minCommRate, err := k.MinCommissionRate(ctx)
	if err != nil {
		return nil, err
	}
	if msg.Commission.Rate.LT(minCommRate) {
		return nil, errors.Wrapf(types.ErrCommissionLTMinRate, "cannot set validator commission to less than minimum rate of %s", minCommRate)
	}

	commission := types.NewCommissionWithTime(msg.Commission.Rate, msg.Commission.MaxRate,
		msg.Commission.MaxChangeRate, ctx.HeaderInfo().Time)

	if err := commission.Validate(); err != nil {
		return nil, err
	}
	// create a new reporter
	newOracleReporter := types.NewOracleReporter(msg.Reporter, msg.Amount, &commission)
	if err := k.ReporterCheckpoint.Set(ctx, collections.Join(reporter.Bytes(), sdk.UnwrapSDKContext(ctx).BlockHeight()), newOracleReporter.TotalTokens); err != nil {
		return nil, err
	}
	if err := k.UpdateTotalPower(ctx, newOracleReporter.TotalTokens, false); err != nil {
		return nil, err
	}
	if err := k.Reporters.Set(ctx, reporter, newOracleReporter); err != nil {
		return nil, err
	}
	// **********************  AfterReporterCreated  hook **************************************
	if err := k.Keeper.AfterReporterCreated(ctx, newOracleReporter); err != nil {
		return nil, err
	}
	// ************************************************************************************************
	// create a new delegation
	// **********************  BeforeDelegationCreated  hook **************************************
	if err := k.Keeper.BeforeDelegationCreated(ctx, newOracleReporter); err != nil {
		return nil, err
	}
	// ************************************************************************************************
	newDelegation := types.NewDelegation(msg.Reporter, msg.Amount)
	if err := k.DelegatorCheckpoint.Set(ctx, collections.Join(reporter.Bytes(), sdk.UnwrapSDKContext(ctx).BlockHeight()), newDelegation.Amount); err != nil {
		return nil, err
	}
	if err := k.Delegators.Set(ctx, reporter, newDelegation); err != nil {
		return nil, err
	}
	// **********************  AfterDelegationModified  hook **************************************
	if err := k.Keeper.AfterDelegationModified(ctx, reporter, sdk.ValAddress(reporter), newDelegation.Amount); err != nil {
		return nil, err
	}
	if err := k.AfterReporterModified(ctx, reporter); err != nil {
		return nil, err
	}
	// ************************************************************************************************
	return &types.MsgCreateReporterResponse{}, nil
}
