package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/reporter/types"
)

type Querier struct {
	Keeper
}

var _ types.QueryServer = Querier{}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

// Reporters queries all the reporters
func (k Querier) Reporters(ctx context.Context, req *types.QueryReportersRequest) (*types.QueryReportersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	iter, err := k.Keeper.Reporters.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}

	reporters, err := iter.Values()
	if err != nil {
		return nil, err
	}
	reportersPtrs := make([]*types.OracleReporter, len(reporters))
	for i, reporter := range reporters {
		reportersPtrs[i] = &reporter
	}
	return &types.QueryReportersResponse{Reporters: reportersPtrs}, nil

}

// DelegatorReporter queries the reporter of a delegator
func (k Querier) DelegatorReporter(ctx context.Context, req *types.QueryDelegatorReporterRequest) (*types.QueryDelegatorReporterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	delAddr := sdk.MustAccAddressFromBech32(req.DelegatorAddress)

	delegator, err := k.Keeper.Delegators.Get(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegatorReporterResponse{Reporter: delegator.GetReporter()}, nil
}

// ReporterStake queries the total tokens of a reporter
func (k Querier) ReporterStake(ctx context.Context, req *types.QueryReporterStakeRequest) (*types.QueryReporterStakeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporterAddr := sdk.MustAccAddressFromBech32(req.ReporterAddress)

	reporter, err := k.Keeper.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, types.ErrReporterDoesNotExist
		}
		return nil, err
	}

	return &types.QueryReporterStakeResponse{Stake: reporter.TotalTokens}, nil
}

// DelegationRewards the total rewards accrued by a delegation
func (k Querier) DelegationRewards(ctx context.Context, req *types.QueryDelegationRewardsRequest) (*types.QueryDelegationRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporterAddr := sdk.MustAccAddressFromBech32(req.ReporterAddress)
	delAddr := sdk.MustAccAddressFromBech32(req.DelegatorAddress)

	reporter, err := k.Keeper.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}

	delegation, err := k.Keeper.Delegators.Get(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	if reporter.GetReporter() != delegation.GetReporter() {
		return nil, types.ErrReporterMismatch
	}

	endingPeriod, err := k.Keeper.IncrementReporterPeriod(ctx, reporter)
	if err != nil {
		return nil, err
	}

	rewards, err := k.Keeper.CalculateDelegationRewards(ctx, reporterAddr.Bytes(), delAddr, delegation, endingPeriod)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegationRewardsResponse{Rewards: rewards}, nil
}

// ReporterOutstandingRewards queries rewards of a reporter address
func (k Querier) ReporterOutstandingRewards(ctx context.Context, req *types.QueryReporterOutstandingRewardsRequest) (*types.QueryReporterOutstandingRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporterAddr := sdk.MustAccAddressFromBech32(req.ReporterAddress)

	exists, err := k.Keeper.Reporters.Has(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errorsmod.Wrapf(types.ErrReporterDoesNotExist, req.ReporterAddress)
	}

	rewards, err := k.Keeper.ReporterOutstandingRewards.Get(ctx, reporterAddr.Bytes())
	if err != nil {
		return nil, err
	}

	return &types.QueryReporterOutstandingRewardsResponse{Rewards: rewards}, nil
}

// RporterCommission queries accumulated commission for a reporter
func (k Querier) ReporterCommission(ctx context.Context, req *types.QueryReporterCommissionRequest) (*types.QueryReporterCommissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporterAddr := sdk.MustAccAddressFromBech32(req.ReporterAddress)

	exists, err := k.Keeper.Reporters.Has(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errorsmod.Wrapf(types.ErrReporterDoesNotExist, req.ReporterAddress)
	}
	commission, err := k.Keeper.ReportersAccumulatedCommission.Get(ctx, reporterAddr.Bytes())
	if err != nil {
		return nil, err
	}

	return &types.QueryReporterCommissionResponse{Commission: commission}, nil
}
