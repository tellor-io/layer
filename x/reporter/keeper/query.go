package keeper

import (
	"context"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/reporter/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
		stake, _, err := k.GetReporterStake(ctx, sdk.AccAddress(repAddr), nil)
		if err != nil {
			stake = math.ZeroInt()
		}
		reportingPower := stake.Quo(layertypes.PowerReduction).Uint64()
		reporters = append(reporters, &types.Reporter{
			Address:  sdk.AccAddress(repAddr).String(),
			Metadata: &reporterMeta,
			Power:    reportingPower,
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
	return &types.QueryAllowedAmountExpirationResponse{Expiration: uint64(timeMilli)}, nil
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

func (k Querier) AvailableTips(ctx context.Context, req *types.QueryAvailableTipsRequest) (*types.QueryAvailableTipsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	selectorAcc := sdk.MustAccAddressFromBech32(req.SelectorAddress)

	rewards, err := k.Keeper.SelectorTips.Get(ctx, selectorAcc)
	if err != nil {
		return nil, err
	}
	return &types.QueryAvailableTipsResponse{AvailableTips: rewards}, nil
}

func (k Querier) SelectionsTo(ctx context.Context, req *types.QuerySelectionsToRequest) (*types.QuerySelectionsToResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// make sure reporter exists
	repAddr := sdk.MustAccAddressFromBech32(req.ReporterAddress)
	_, err := k.Keeper.Reporters.Get(ctx, repAddr.Bytes())
	if err != nil {
		return nil, err
	}

	// get all selections to this reporter
	selections := make([]*types.FormattedSelection, 0)
	iter, err := k.Keeper.Selectors.Indexes.Reporter.MatchExact(ctx, repAddr)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		selectorAddr, err := iter.PrimaryKey()
		if err != nil {
			return nil, err
		}
		// get selection
		selection, err := k.Keeper.Selectors.Get(ctx, selectorAddr)
		if err != nil {
			return nil, err
		}
		// get individual delegation(s) info with totals calculated
		individualDelegations, totalTokens, delegationCount, err := k.getIndividualDelegations(ctx, selectorAddr)
		if err != nil {
			return nil, err
		}

		formattedSelection := &types.FormattedSelection{
			Selector:              sdk.AccAddress(selectorAddr).String(),
			LockedUntilTime:       selection.GetLockedUntilTime(),
			DelegationsCount:      delegationCount,
			DelegationsTotal:      totalTokens,
			IndividualDelegations: individualDelegations,
		}
		selections = append(selections, formattedSelection)
	}
	return &types.QuerySelectionsToResponse{
		Reporter:   repAddr.String(),
		Selections: selections,
	}, nil
}

func (k Querier) getIndividualDelegations(ctx context.Context, selectorAddr sdk.AccAddress) ([]*types.IndividualDelegation, math.Int, uint64, error) {
	var individualDelegations []*types.IndividualDelegation
	totalTokens := math.ZeroInt()
	var iterError error

	err := k.stakingKeeper.IterateDelegatorDelegations(ctx, selectorAddr, func(delegation stakingtypes.Delegation) (stop bool) {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			iterError = err
			return true
		}
		val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			iterError = err
			return true
		}
		if val.IsBonded() {
			delTokens := val.TokensFromShares(delegation.Shares).TruncateInt()
			individualDelegations = append(individualDelegations, &types.IndividualDelegation{
				ValidatorAddress: delegation.ValidatorAddress,
				Amount:           delTokens,
			})
			totalTokens = totalTokens.Add(delTokens)
		}
		return false
	})
	if err != nil {
		return nil, math.ZeroInt(), 0, err
	}
	if iterError != nil {
		return nil, math.ZeroInt(), 0, iterError
	}
	return individualDelegations, totalTokens, uint64(len(individualDelegations)), nil
}
