package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k msgServer) Vote(goCtx context.Context, msg *types.MsgVote) (*types.MsgVoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute, err := k.Keeper.GetDisputeById(ctx, msg.Id)
	if err != nil {
		return nil, err
	}
	if dispute.DisputeStatus != types.Voting {
		return nil, types.ErrDisputeNotInVotingState
	}

	// Get vote by disputeId
	vote, err := k.Keeper.GetVote(ctx, msg.Id)
	if err != nil {
		return nil, err
	}
	if vote.VoteStart.IsZero() {
		return nil, types.ErrVoteDoesNotExist
	}

	// Check if voter has already voted
	voter, err := k.Keeper.GetVoterVote(ctx, msg.Voter, msg.Id)
	if err != nil {
		return nil, err
	}
	if voter.Voter != "" {
		return nil, types.ErrVoterHasAlreadyVoted
	}

	// Assert again voting hasn't ended
	if vote.VoteEnd.Before(ctx.BlockTime()) {
		return nil, types.ErrVotingPeriodEnded
	}

	err = k.Keeper.SetTally(ctx, msg.Id, msg.Vote, msg.Voter)
	if err != nil {
		return nil, err
	}
	err = k.Keeper.SetVoterVote(ctx, msg)
	if err != nil {
		return nil, err
	}
	err = k.Keeper.AppendVoters(ctx, msg.Id, msg.Voter)
	if err != nil {
		return nil, err
	}

	return &types.MsgVoteResponse{}, nil
}
