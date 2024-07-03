package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/reporter/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	for i := range reporters {
		reportersPtrs[i] = &reporters[i]
	}
	return &types.QueryReportersResponse{Reporters: reportersPtrs}, nil
}

// DelegatorReporter queries the reporter of a delegator
func (k Querier) DelegatorReporter(ctx context.Context, req *types.QueryDelegatorReporterRequest) (*types.QueryDelegatorReporterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	delAddr := sdk.MustAccAddressFromBech32(req.DelegatorAddress)

	delegator, err := k.Keeper.Selectors.Get(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegatorReporterResponse{Reporter: sdk.AccAddress(delegator.GetReporter()).String()}, nil
}
