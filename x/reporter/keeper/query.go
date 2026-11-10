package keeper

import (
	"bytes"
	"context"
	"errors"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/reporter/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

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

// Reporters queries all the reporters, omitting any with a pending self-demotion.
func (k Querier) Reporters(ctx context.Context, req *types.QueryReportersRequest) (*types.QueryReportersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	reporters, pageRes, err := query.CollectionFilteredPaginate(
		ctx,
		k.Keeper.Reporters,
		req.Pagination,
		func(repAddr []byte, _ types.OracleReporter) (bool, error) {
			demoting, err := k.hasPendingSelfDemotion(ctx, repAddr)
			if err != nil {
				return false, err
			}
			return !demoting, nil
		},
		func(repAddr []byte, reporterMeta types.OracleReporter) (*types.Reporter, error) {
			return k.reporterResponse(ctx, repAddr, reporterMeta)
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if reporters == nil {
		reporters = []*types.Reporter{}
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

	// Get current settled tips for this selector
	rewards, err := k.Keeper.SelectorTips.Get(ctx, selectorAcc)
	if err != nil {
		rewards = math.LegacyZeroDec()
	}

	// Also calculate pending rewards from ReporterPeriodData
	// First, find the selector's reporter
	selector, err := k.Keeper.Selectors.Get(ctx, selectorAcc)
	if err == nil {
		// Selector exists, get their reporter's period data
		reporterAddr := sdk.AccAddress(selector.GetReporter())
		periodData, err := k.Keeper.ReporterPeriodData.Get(ctx, reporterAddr)
		if err == nil && periodData.RewardAmount.IsPositive() {
			if periodData.Total.IsZero() {
				// No selector snapshot: full net reward falls back to the reporter address on settle.
				if selectorAcc.Equals(reporterAddr) {
					rewards = rewards.Add(periodData.RewardAmount)
				}
			} else {
				// Find this selector's share in the period data
				for _, sel := range periodData.Selectors {
					if sdk.AccAddress(sel.SelectorAddress).Equals(selectorAcc) {
						// Calculate pending: (selector_amount / total) * reward_amount
						shareRatio := sel.Amount.ToLegacyDec().Quo(periodData.Total.ToLegacyDec())
						pendingReward := periodData.RewardAmount.Mul(shareRatio)
						rewards = rewards.Add(pendingReward)
						break
					}
				}
			}
		}
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
			DisputeLockedUntil:    selection.DisputeLockedUntil,
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

// JailedReporters queries all jailed reporters
func (k Querier) JailedReporters(ctx context.Context, req *types.QueryJailedReportersRequest) (*types.QueryJailedReportersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	reporters := make([]*types.Reporter, 0)
	err := k.Keeper.Reporters.Walk(ctx, nil, func(repAddr []byte, reporterMeta types.OracleReporter) (stop bool, err error) {
		if !reporterMeta.Jailed {
			return false, nil
		}
		reporter, err := k.reporterResponse(ctx, repAddr, reporterMeta)
		if err != nil {
			return true, err
		}
		reporters = append(reporters, reporter)
		return false, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryJailedReportersResponse{Reporters: reporters}, nil
}

// TipUnlocks queries pending tip unlocks for a selector (with unlock_ids for cancel).
func (k Querier) TipUnlocks(ctx context.Context, req *types.QueryTipUnlocksRequest) (*types.QueryTipUnlocksResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.SelectorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "selector address cannot be empty")
	}

	selectorAcc, err := sdk.AccAddressFromBech32(req.SelectorAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid selector address")
	}

	unlocks := make([]types.TipUnlockInfo, 0)
	rng := collections.NewPrefixedPairRange[[]byte, uint64](selectorAcc.Bytes())
	err = k.Keeper.TipUnlocks.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], entry types.TipUnlockEntry) (stop bool, err error) {
		unlocks = append(unlocks, types.TipUnlockInfo{
			UnlockId:       key.K2(),
			Amount:         entry.Amount,
			CompletionTime: entry.CompletionTime,
		})
		return false, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTipUnlocksResponse{Unlocks: unlocks}, nil
}

// Reporter queries a specific reporter by address
func (k Querier) Reporter(ctx context.Context, req *types.QueryReporterRequest) (*types.QueryReporterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	repAddr := sdk.MustAccAddressFromBech32(req.ReporterAddress)
	reporterMeta, err := k.Keeper.Reporters.Get(ctx, repAddr)
	if err != nil {
		return nil, status.Error(codes.NotFound, "reporter not found")
	}
	demoting, err := k.hasPendingSelfDemotion(ctx, repAddr)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if demoting {
		return nil, status.Error(codes.NotFound, "reporter not found")
	}

	reporter, err := k.reporterResponse(ctx, repAddr, reporterMeta)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryReporterResponse{Reporter: reporter}, nil
}

// TwoBitPendingRequests lists outstanding 2bit registration requests.
// A reporter address returns that reporter's request. A 2bit address returns
// every request naming that agent. Both empty returns every request.
func (k Querier) TwoBitPendingRequests(ctx context.Context, req *types.QueryTwoBitPendingRequestsRequest) (*types.QueryTwoBitPendingRequestsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var reporterFilter, agentFilter []byte
	if req.ReporterAddress != "" {
		reporter, err := sdk.AccAddressFromBech32(req.ReporterAddress)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid reporter address")
		}
		reporterFilter = reporter.Bytes()
	}
	if req.TwoBitAddress != "" {
		agent, err := sdk.AccAddressFromBech32(req.TwoBitAddress)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid 2bit address")
		}
		agentFilter = agent.Bytes()
	}

	// One request per reporter, so a reporter filter is a single key lookup.
	if reporterFilter != nil {
		pending, err := k.Keeper.TwoBitPendingRequests.Get(ctx, reporterFilter)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				return &types.QueryTwoBitPendingRequestsResponse{Requests: []*types.TwoBitPendingRequestInfo{}}, nil
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
		if agentFilter != nil && !bytes.Equal(pending.TwoBitReporter, agentFilter) {
			return &types.QueryTwoBitPendingRequestsResponse{Requests: []*types.TwoBitPendingRequestInfo{}}, nil
		}
		return &types.QueryTwoBitPendingRequestsResponse{
			Requests: []*types.TwoBitPendingRequestInfo{twoBitPendingRequestInfo(reporterFilter, pending)},
		}, nil
	}

	requests, pageRes, err := query.CollectionFilteredPaginate(
		ctx,
		k.Keeper.TwoBitPendingRequests,
		req.Pagination,
		func(_ []byte, pending types.TwoBitPendingRequest) (bool, error) {
			if agentFilter != nil && !bytes.Equal(pending.TwoBitReporter, agentFilter) {
				return false, nil
			}
			return true, nil
		},
		func(repAddr []byte, pending types.TwoBitPendingRequest) (*types.TwoBitPendingRequestInfo, error) {
			return twoBitPendingRequestInfo(repAddr, pending), nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if requests == nil {
		requests = []*types.TwoBitPendingRequestInfo{}
	}
	return &types.QueryTwoBitPendingRequestsResponse{Requests: requests, Pagination: pageRes}, nil
}

func (k Querier) reporterResponse(ctx context.Context, repAddr []byte, reporterMeta types.OracleReporter) (*types.Reporter, error) {
	stake, _, _, _, err := k.GetReporterStakeView(ctx, sdk.AccAddress(repAddr))
	if err != nil {
		stake = math.ZeroInt()
	}
	twoBitAddress, err := k.registeredTwoBitAddress(ctx, repAddr)
	if err != nil {
		return nil, err
	}
	return &types.Reporter{
		Address:       sdk.AccAddress(repAddr).String(),
		Metadata:      &reporterMeta,
		Power:         stake.Quo(layertypes.PowerReduction).Uint64(),
		TwoBitAddress: twoBitAddress,
	}, nil
}

func (k Querier) registeredTwoBitAddress(ctx context.Context, reporter []byte) (string, error) {
	agent, err := k.Keeper.TwoBitReporterToAgent.Get(ctx, reporter)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	if len(agent) == 0 {
		return "", nil
	}
	return sdk.AccAddress(agent).String(), nil
}

func twoBitPendingRequestInfo(reporter []byte, pending types.TwoBitPendingRequest) *types.TwoBitPendingRequestInfo {
	return &types.TwoBitPendingRequestInfo{
		Reporter:      sdk.AccAddress(reporter).String(),
		TwoBitAddress: sdk.AccAddress(pending.TwoBitReporter).String(),
		ExpiryHeight:  pending.ExpiryHeight,
	}
}
