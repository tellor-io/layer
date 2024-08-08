package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/reporter/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
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
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	repStore := prefix.NewStore(store, types.ReportersKey)
	reporters := make([]*types.Reporter, 0)
	pageRes, err := query.Paginate(repStore, req.Pagination, func(repAddr, value []byte) error {
		var reporterMeta types.OracleReporter
		err := k.cdc.Unmarshal(value, &reporterMeta)
		if err != nil {
			return err
		}

		reporters = append(reporters, &types.Reporter{
			Address:  sdk.AccAddress(repAddr).String(),
			Metadata: &reporterMeta,
		})
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryReportersResponse{Reporters: reporters, Pagination: pageRes}, nil
}

// SelectorReporter queries the reporter of a selector
func (k Querier) SelectorReporter(ctx context.Context, req *types.QuerySelectorReporterRequest) (*types.QuerySelectorReporterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	delAddr := sdk.MustAccAddressFromBech32(req.SelectorAddress)

	selector, err := k.Keeper.Selectors.Get(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	return &types.QuerySelectorReporterResponse{Reporter: sdk.AccAddress(selector.GetReporter()).String()}, nil
}

// get the current staking/unstaking amount allowed w/out triggering 5% change
func (k Querier) AllowedAmount(ctx context.Context, req *types.QueryAllowedAmountRequest) (*types.QueryAllowedAmountResponse, error) {
	amt, err := k.Keeper.Tracker.Get(ctx)
	if err != nil {
		return nil, err
	}
	currentAmount, err := k.Keeper.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return nil, err
	}
	fivePercentIncrease := amt.Amount.Add(amt.Amount.QuoRaw(20))
	fivePercentDecrease := amt.Amount.Sub(amt.Amount.QuoRaw(20))

	stakingAmountAllowed := math.ZeroInt()
	unstakingAmountAllowed := math.ZeroInt()
	if currentAmount.LT(fivePercentIncrease) {
		stakingAmountAllowed = fivePercentIncrease.Sub(currentAmount)
	}
	if currentAmount.GT(fivePercentDecrease) {
		unstakingAmountAllowed = fivePercentDecrease.Sub(currentAmount)
	}
	return &types.QueryAllowedAmountResponse{
		StakingAmount:   stakingAmountAllowed,
		UnstakingAmount: unstakingAmountAllowed,
	}, nil
}

func (k Querier) AllowedAmountExpiration(ctx context.Context, req *types.QueryAllowedAmountExpirationRequest) (*types.QueryAllowedAmountExpirationResponse, error) {
	tracker, err := k.Keeper.Tracker.Get(ctx)
	if err != nil {
		return nil, err
	}
	timeMilli := tracker.Expiration.UnixMilli()
	return &types.QueryAllowedAmountExpirationResponse{Expiration: timeMilli}, nil
}

// query for num of selectors in reporter
func (k Querier) NumOfSelectorsByReporter(ctx context.Context, req *types.QueryNumOfSelectorsByReporterRequest) (*types.QueryNumOfSelectorsByReporterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	repAddr := sdk.MustAccAddressFromBech32(req.ReporterAddress)
	count, err := k.Keeper.GetNumOfSelectors(ctx, repAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryNumOfSelectorsByReporterResponse{NumOfSelectors: int32(count)}, nil
}

// query for num of space available in reporter
func (k Querier) SpaceAvailableByReporter(ctx context.Context, req *types.QuerySpaceAvailableByReporterRequest) (*types.QuerySpaceAvailableByReporterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	repAddr := sdk.MustAccAddressFromBech32(req.ReporterAddress)
	count, err := k.Keeper.GetNumOfSelectors(ctx, repAddr)
	if err != nil {
		return nil, err
	}
	params, err := k.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	remaining := int(params.MaxSelectors) - count
	return &types.QuerySpaceAvailableByReporterResponse{SpaceAvailable: int32(remaining)}, nil
}
