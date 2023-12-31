package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k msgServer) Vote(goCtx context.Context, msg *types.MsgVote) (*types.MsgVoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute := k.Keeper.GetDisputeById(ctx, msg.Id)
	if dispute.DisputeStatus != types.Voting {
		return nil, types.ErrDisputeNotInVotingState
	}

	// Get vote by disputeId
	vote := k.Keeper.GetVote(ctx, msg.Id)
	if vote.VoteStart.IsZero() {
		return nil, types.ErrVoteDoesNotExist
	}

	// Check if voter has already voted
	voter := k.Keeper.GetVoterVote(ctx, msg.Voter, msg.Id)
	if voter.Voter != "" {
		return nil, types.ErrVoterHasAlreadyVoted
	}

	// Assert again voting hasn't ended
	if vote.VoteEnd.Before(ctx.BlockTime()) {
		return nil, types.ErrVotingPeriodEnded
	}

	k.Keeper.SetTally(ctx, msg.Id, msg.Vote, msg.Voter)
	k.Keeper.SetVoterVote(ctx, msg)
	k.Keeper.AppendVoters(ctx, msg.Id, msg.Voter)

	return &types.MsgVoteResponse{}, nil
}
