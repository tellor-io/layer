package keeper

import (
	"context"
	"encoding/binary"

	layertypes "github.com/tellor-io/layer/types"
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

func (k Querier) Tally(ctx context.Context, req *types.QueryDisputesTallyRequest) (*types.QueryDisputesTallyResponse, error) {
	dispute, err := k.Keeper.Disputes.Get(ctx, req.DisputeId)
	if err != nil {
		return &types.QueryDisputesTallyResponse{}, err
	}
	voteCounts, err := k.Keeper.VoteCountsByGroup.Get(ctx, req.DisputeId)
	if err != nil {
		return &types.QueryDisputesTallyResponse{}, err
	}
	blockInfo, err := k.BlockInfo.Get(ctx, dispute.HashId)
	if err != nil {
		return &types.QueryDisputesTallyResponse{}, err
	}

	sumOfReporterVotes := voteCounts.Reporters.Against + voteCounts.Reporters.Invalid + voteCounts.Reporters.Support
	totalReporterPower := blockInfo.TotalReporterPower

	sumOfUsersVotes := voteCounts.Users.Against + voteCounts.Users.Invalid + voteCounts.Users.Support
	totalUserPower := blockInfo.TotalUserTips

	sumOfTokenHoldersVotes := voteCounts.Tokenholders.Against + voteCounts.Tokenholders.Invalid + voteCounts.Tokenholders.Support
	totalTokenHolderPower := k.Keeper.GetTotalSupply(ctx).Uint64()

	teamAddr, err := k.Keeper.GetTeamAddress(ctx)
	if err != nil {
		return &types.QueryDisputesTallyResponse{}, err
	}

	teamDidVote, err := k.Keeper.Voter.Has(ctx, collections.Join(req.DisputeId, teamAddr.Bytes()))
	if err != nil {
		return &types.QueryDisputesTallyResponse{}, err
	}

	teamVote := &types.VoteCounts{Support: 0, Against: 0, Invalid: 0}
	if teamDidVote {
		vote, err := k.Voter.Get(ctx, collections.Join(req.DisputeId, teamAddr.Bytes()))
		if err != nil {
			return &types.QueryDisputesTallyResponse{}, err
		}

		switch vote.Vote {
		case types.VoteEnum_VOTE_SUPPORT:
			teamVote.Support = math.OneInt().Mul(layertypes.PowerReduction).Uint64()
		case types.VoteEnum_VOTE_AGAINST:
			teamVote.Against = math.OneInt().Mul(layertypes.PowerReduction).Uint64()
		case types.VoteEnum_VOTE_INVALID:
			teamVote.Invalid = math.OneInt().Mul(layertypes.PowerReduction).Uint64()
		}
	}

	res := &types.QueryDisputesTallyResponse{
		Users: &types.GroupTally{
			VoteCount:       &voteCounts.Users,
			TotalPowerVoted: sumOfUsersVotes,
			TotalGroupPower: totalUserPower.Uint64(),
		},
		Reporters: &types.GroupTally{
			VoteCount:       &voteCounts.Reporters,
			TotalPowerVoted: sumOfReporterVotes,
			TotalGroupPower: totalReporterPower.Uint64(),
		},
		Tokenholders: &types.GroupTally{
			VoteCount:       &voteCounts.Tokenholders,
			TotalPowerVoted: sumOfTokenHoldersVotes,
			TotalGroupPower: totalTokenHolderPower,
		},
		Team: teamVote,
	}

	return res, nil
}
