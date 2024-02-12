package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"

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
			Delegator: msg.Reporter,
			Amount:    msg.Amount,
		}); err != nil {
		return nil, err
	}
	// set the token origins
	var amount uint64
	for _, origin := range msg.TokenOrigins {
		amount += origin.Amount
	}
	if amount != msg.Amount {
		return nil, fmt.Errorf("Token origin amount does not match the stake amount chosen")
	}
	for _, origin := range msg.TokenOrigins {
		amount += origin.Amount
		valAddr, err := sdk.ValAddressFromBech32(origin.ValidatorAddress)
		if err != nil {
			return nil, err
		}
		validator := k.stakingKeeper.Validator(ctx, valAddr)
		// check if validator has delegator bond
		del, err := k.stakingKeeper.Delegation(ctx, reporter, valAddr)
		if err != nil {
			return nil, err
		}
		if del == nil {
			return nil, fmt.Errorf("Reporter has no delegation bond with validator")
		}
		// check amount is greater than or equal to the delegation amount
		if msg.Amount < validator.TokensFromShares(del.GetShares()).TruncateInt().Uint64() {
			return nil, fmt.Errorf("Reporter has insufficient tokens bonded with validator: %v", origin)
		}
		if err := k.TokenOrigin.Set(ctx, collections.Join(reporter, valAddr), *origin); err != nil {
			return nil, err
		}
	}
	return &types.MsgCreateReporterResponse{}, nil
}
