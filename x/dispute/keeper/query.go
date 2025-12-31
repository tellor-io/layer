package keeper

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

// convertMicroReportToStrings converts MicroReport to MicroReportStrings (string queryId)
func convertMicroReportToStrings(mr oracletypes.MicroReport) oracletypes.MicroReportStrings {
	return oracletypes.MicroReportStrings{
		Reporter:        mr.Reporter,
		Power:           mr.Power,
		QueryType:       mr.QueryType,
		QueryId:         hex.EncodeToString(mr.QueryId),
		AggregateMethod: mr.AggregateMethod,
		Value:           mr.Value,
		Timestamp:       uint64(mr.Timestamp.UnixMilli()),
		Cyclelist:       mr.Cyclelist,
		BlockNumber:     mr.BlockNumber,
		MetaId:          mr.MetaId,
	}
}

// take dispute, return hash id and all evidence as strings for display
func convertDisputeToStrings(d types.Dispute) types.DisputeStrings {
	// convert all additional evidence to string types
	additionalEvidence := make([]*oracletypes.MicroReportStrings, len(d.AdditionalEvidence))
	for i, evidence := range d.AdditionalEvidence {
		converted := convertMicroReportToStrings(*evidence)
		additionalEvidence[i] = &converted
	}

	// convert initial evidence to string type
	return types.DisputeStrings{
		HashId:             hex.EncodeToString(d.HashId),
		DisputeId:          d.DisputeId,
		DisputeCategory:    d.DisputeCategory,
		DisputeFee:         d.DisputeFee,
		DisputeStatus:      d.DisputeStatus,
		DisputeStartTime:   d.DisputeStartTime,
		DisputeEndTime:     d.DisputeEndTime,
		DisputeStartBlock:  d.DisputeStartBlock,
		DisputeRound:       d.DisputeRound,
		SlashAmount:        d.SlashAmount,
		BurnAmount:         d.BurnAmount,
		InitialEvidence:    convertMicroReportToStrings(d.InitialEvidence),
		FeeTotal:           d.FeeTotal,
		PrevDisputeIds:     d.PrevDisputeIds,
		BlockNumber:        d.BlockNumber,
		Open:               d.Open,
		AdditionalEvidence: additionalEvidence,
		VoterReward:        d.VoterReward,
		PendingExecution:   d.PendingExecution,
	}
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
		disputeStrings := convertDisputeToStrings(dispute)
		disputes = append(disputes, &types.Disputes{
			DisputeId: id,
			Metadata:  &disputeStrings,
		})
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDisputesResponse{Disputes: disputes, Pagination: pageRes}, nil
}

// Dispute queries a specific dispute by id
func (k Querier) Dispute(ctx context.Context, req *types.QueryDisputeRequest) (*types.QueryDisputeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	dispute, err := k.Keeper.Disputes.Get(ctx, req.DisputeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "dispute not found")
	}

	disputeStrings := convertDisputeToStrings(dispute)
	return &types.QueryDisputeResponse{
		Dispute: &types.Disputes{
			DisputeId: req.DisputeId,
			Metadata:  &disputeStrings,
		},
	}, nil
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

	// check if dispute has been voted on yet
	voteCounts, err := k.Keeper.VoteCountsByGroup.Get(ctx, req.DisputeId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// dispute exists but hasn't been voted on yet, return empty tally
			return &types.QueryDisputesTallyResponse{
				Users: &types.GroupTally{
					VoteCount: &types.FormattedVoteCounts{
						Support: "0.00%",
						Against: "0.00%",
						Invalid: "0.00%",
					},
					TotalPowerVoted: 0,
					TotalGroupPower: 0,
				},
				Reporters: &types.GroupTally{
					VoteCount: &types.FormattedVoteCounts{
						Support: "0.00%",
						Against: "0.00%",
						Invalid: "0.00%",
					},
					TotalPowerVoted: 0,
					TotalGroupPower: 0,
				},
				Team: &types.FormattedVoteCounts{
					Support: "0.00%",
					Against: "0.00%",
					Invalid: "0.00%",
				},
				CombinedTotal: &types.CombinedTotal{
					Support: "0.00%",
					Against: "0.00%",
					Invalid: "0.00%",
				},
			}, nil
		}
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

func (k Querier) DisputeFeePayers(ctx context.Context, req *types.QueryDisputeFeePayersRequest) (*types.QueryDisputeFeePayersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	payers := make([]*types.DisputeFeePayerInfo, 0)
	rng := collections.NewPrefixedPairRange[uint64, []byte](req.DisputeId)
	iter, err := k.Keeper.DisputeFeePayer.Iterate(ctx, rng)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		value, err := iter.Value()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		payerAddrBytes := key.K2()
		payerAddr := sdk.AccAddress(payerAddrBytes)
		payers = append(payers, &types.DisputeFeePayerInfo{
			PayerAddress: payerAddr.String(),
			PayerInfo:    value,
		})
	}

	return &types.QueryDisputeFeePayersResponse{Payers: payers}, nil
}

func (k Querier) ClaimableDisputeRewards(ctx context.Context, req *types.QueryClaimableDisputeRewardsRequest) (*types.QueryClaimableDisputeRewardsResponse, error) {
	if req == nil {
		return nil, errors.New("invalid request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	dispute, err := k.Keeper.Disputes.Get(ctx, req.DisputeId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errors.New("dispute not found")
		}
		return nil, err
	}

	rewardAmount := math.ZeroInt()
	feeRefundAmount := math.ZeroInt()
	rewardClaimed := false

	// Calculate Voter Reward
	if dispute.DisputeStatus == types.Resolved {
		// Check if they voted
		voterInfo, err := k.Keeper.Voter.Get(ctx, collections.Join(req.DisputeId, addr.Bytes()))
		if err == nil {
			// Found voter info
			rewardClaimed = voterInfo.RewardClaimed
			if !voterInfo.RewardClaimed {
				// They voted and haven't claimed yet
				// CalculateReward checks if vote.Executed and other conditions
				reward, err := k.Keeper.CalculateReward(sdkCtx, addr, req.DisputeId)
				if err == nil {
					rewardAmount = reward
				}
			}
		}
	}

	// Calculate Fee Refund
	// Check if they are a fee payer for the first round
	payerInfo, err := k.Keeper.DisputeFeePayer.Get(ctx, collections.Join(req.DisputeId, addr.Bytes()))
	if err == nil {
		// Address is a fee payer
		switch dispute.DisputeStatus {
		case types.Failed:
			// Failed dispute (underfunded) - full refund
			feeRefundAmount = payerInfo.Amount
		case types.Resolved:
			vote, err := k.Keeper.Votes.Get(ctx, req.DisputeId)
			if err == nil && vote.Executed {
				switch vote.VoteResult {
				case types.VoteResult_INVALID, types.VoteResult_NO_QUORUM_MAJORITY_INVALID:
					refund, _ := CalculateRefundAmount(payerInfo.Amount, dispute.SlashAmount, dispute.FeeTotal)
					feeRefundAmount = feeRefundAmount.Add(refund)

				case types.VoteResult_SUPPORT, types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT:
					refund, _ := CalculateRefundAmount(payerInfo.Amount, dispute.SlashAmount, dispute.FeeTotal)
					feeRefundAmount = feeRefundAmount.Add(refund)

					reward, _ := CalculateReporterBondRewardAmount(payerInfo.Amount, dispute.FeeTotal, dispute.SlashAmount)
					feeRefundAmount = feeRefundAmount.Add(reward)
				}
			}
		}
	}

	return &types.QueryClaimableDisputeRewardsResponse{
		ClaimableAmount: &types.ClaimableAmount{
			DisputeId:       req.DisputeId,
			RewardAmount:    rewardAmount,
			FeeRefundAmount: feeRefundAmount,
			RewardClaimed:   rewardClaimed,
		},
	}, nil
}
