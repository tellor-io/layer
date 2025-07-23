package keeper

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/x/dispute/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

func (k Querier) Disputes(ctx context.Context, req *types.QueryDisputesRequest) (*types.QueryDisputesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	disputeStore := prefix.NewStore(store, types.DisputesPrefix)
	disputes := make([]*types.Disputes, 0)
	pageRes, err := query.Paginate(disputeStore, req.Pagination, func(disputeID, value []byte) error {
		var dispute types.Dispute
		err := k.cdc.Unmarshal(value, &dispute)
		if err != nil {
			return err
		}
		id := binary.BigEndian.Uint64(disputeID)
		disputes = append(disputes, &types.Disputes{
			DisputeId: id,
			Metadata:  &dispute,
		})
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDisputesResponse{Disputes: disputes, Pagination: pageRes}, nil
}

func (k Querier) OpenDisputes(ctx context.Context, req *types.QueryOpenDisputesRequest) (*types.QueryOpenDisputesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	openDisputes, err := k.Keeper.GetOpenDisputes(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var openDisputesArray types.OpenDisputes
	openDisputesArray.Ids = openDisputes
	return &types.QueryOpenDisputesResponse{OpenDisputes: &openDisputesArray}, nil
}

func (k Querier) TeamVote(ctx context.Context, req *types.QueryTeamVoteRequest) (*types.QueryTeamVoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	teamAddr, err := k.Keeper.GetTeamAddress(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	vote, err := k.Keeper.Voter.Get(ctx, collections.Join(req.DisputeId, teamAddr.Bytes()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryTeamVoteResponse{TeamVote: vote}, nil
}

func (k Querier) TeamAddress(ctx context.Context, req *types.QueryTeamAddressRequest) (*types.QueryTeamAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	teamAddr, err := k.Keeper.GetTeamAddress(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryTeamAddressResponse{TeamAddress: teamAddr.String()}, nil
}

func (k Querier) Tally(ctx context.Context, req *types.QueryDisputesTallyRequest) (*types.QueryDisputesTallyResponse, error) {
	// get dispute so we can get the vote counts and hashID
	dispute, err := k.Keeper.Disputes.Get(ctx, req.DisputeId)
	if err != nil {
		return &types.QueryDisputesTallyResponse{}, err
	}
	voteCounts, err := k.Keeper.VoteCountsByGroup.Get(ctx, req.DisputeId)
	if err != nil {
		return &types.QueryDisputesTallyResponse{}, err
	}
	// use hashID to get blockInfo which has tips and reporter power stored in it
	blockInfo, err := k.Keeper.BlockInfo.Get(ctx, dispute.HashId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			blockInfo.TotalReporterPower = math.ZeroInt()
			blockInfo.TotalUserTips = math.ZeroInt()
		} else {
			return &types.QueryDisputesTallyResponse{}, err
		}
	}

	// add up reporter votes and convert reporter choice totals to percentages
	sumOfReporterVotes := voteCounts.Reporters.Against + voteCounts.Reporters.Invalid + voteCounts.Reporters.Support
	totalReporterPower := blockInfo.TotalReporterPower
	var reporterForPerc, reporterAgainstPerc, reporterInvalidPerc float64
	if totalReporterPower.Uint64() > 0 {
		reporterForPerc = float64(voteCounts.Reporters.Support) / float64(totalReporterPower.Uint64()) * 100 / 3
		reporterAgainstPerc = float64(voteCounts.Reporters.Against) / float64(totalReporterPower.Uint64()) * 100 / 3
		reporterInvalidPerc = float64(voteCounts.Reporters.Invalid) / float64(totalReporterPower.Uint64()) * 100 / 3
	} else {
		reporterForPerc = 0
		reporterAgainstPerc = 0
		reporterInvalidPerc = 0
	}

	// add up user votes and convert user choice totals to percentages
	sumOfUsersVotes := voteCounts.Users.Against + voteCounts.Users.Invalid + voteCounts.Users.Support
	totalUserPower := blockInfo.TotalUserTips
	var userForPerc, userAgainstPerc, userInvalidPerc float64
	if totalUserPower.Uint64() > 0 {
		userForPerc = (float64(voteCounts.Users.Support) / float64(totalUserPower.Uint64())) * 100 / 3
		userAgainstPerc = (float64(voteCounts.Users.Against) / float64(totalUserPower.Uint64())) * 100 / 3
		userInvalidPerc = (float64(voteCounts.Users.Invalid) / float64(totalUserPower.Uint64())) * 100 / 3
	} else {
		userForPerc = 0
		userAgainstPerc = 0
		userInvalidPerc = 0
	}

	supportFloat := userForPerc + reporterForPerc
	againstFloat := userAgainstPerc + reporterAgainstPerc
	invalidFloat := userInvalidPerc + reporterInvalidPerc

	// get team address and check if they voted
	teamVote := &types.FormattedVoteCounts{Support: "0.00%", Against: "0.00%", Invalid: "0.00%"}
	teamAddr, err := k.Keeper.GetTeamAddress(ctx)
	if err != nil {
		return &types.QueryDisputesTallyResponse{}, err
	}
	teamDidVote, err := k.Keeper.Voter.Has(ctx, collections.Join(req.DisputeId, teamAddr.Bytes()))
	if err != nil {
		return &types.QueryDisputesTallyResponse{}, err
	}
	// if team voted, add their vote to the choice total and set the team vote percentage
	if teamDidVote {
		vote, err := k.Voter.Get(ctx, collections.Join(req.DisputeId, teamAddr.Bytes()))
		if err != nil {
			return &types.QueryDisputesTallyResponse{}, err
		}
		teamVoteWeight := float64(100.0 / 3.0)
		switch vote.Vote {
		case types.VoteEnum_VOTE_SUPPORT:
			teamVote.Support = fmt.Sprintf("%.2f%%", teamVoteWeight)
			supportFloat += teamVoteWeight
		case types.VoteEnum_VOTE_AGAINST:
			teamVote.Against = fmt.Sprintf("%.2f%%", teamVoteWeight)
			againstFloat += teamVoteWeight
		case types.VoteEnum_VOTE_INVALID:
			teamVote.Invalid = fmt.Sprintf("%.2f%%", teamVoteWeight)
			invalidFloat += teamVoteWeight
		}
	}

	// sum up each vote choice total
	combinedTotal := &types.CombinedTotal{
		Support: fmt.Sprintf("%.2f%%", supportFloat),
		Against: fmt.Sprintf("%.2f%%", againstFloat),
		Invalid: fmt.Sprintf("%.2f%%", invalidFloat),
	}

	// return % for each category by group, and total % for each category
	res := &types.QueryDisputesTallyResponse{
		Users: &types.GroupTally{
			VoteCount: &types.FormattedVoteCounts{
				Support: fmt.Sprintf("%.2f%%", userForPerc),
				Against: fmt.Sprintf("%.2f%%", userAgainstPerc),
				Invalid: fmt.Sprintf("%.2f%%", userInvalidPerc),
			},
			TotalPowerVoted: sumOfUsersVotes,
			TotalGroupPower: totalUserPower.Uint64(),
		},
		Reporters: &types.GroupTally{
			VoteCount: &types.FormattedVoteCounts{
				Support: fmt.Sprintf("%.2f%%", reporterForPerc),
				Against: fmt.Sprintf("%.2f%%", reporterAgainstPerc),
				Invalid: fmt.Sprintf("%.2f%%", reporterInvalidPerc),
			},
			TotalPowerVoted: sumOfReporterVotes,
			TotalGroupPower: totalReporterPower.Uint64(),
		},
		Team:          teamVote,
		CombinedTotal: combinedTotal,
	}

	return res, nil
}

func (k Querier) VoteResult(ctx context.Context, req *types.QueryDisputeVoteResultRequest) (*types.QueryDisputeVoteResultResponse, error) {
	vote, err := k.Keeper.Votes.Get(ctx, req.DisputeId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryDisputeVoteResultResponse{VoteResult: vote.VoteResult}, nil
}
