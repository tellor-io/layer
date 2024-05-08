package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k msgServer) Vote(goCtx context.Context, msg *types.MsgVote) (*types.MsgVoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	voterAcc, err := sdk.AccAddressFromBech32(msg.Voter)
	if err != nil {
		return nil, err
	}
	dispute, err := k.Keeper.Disputes.Get(ctx, msg.Id)
	if err != nil {
		return nil, err
	}
	if dispute.DisputeStatus != types.Voting {
		return nil, types.ErrDisputeNotInVotingState
	}

	vote, err := k.Keeper.Votes.Get(ctx, msg.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, types.ErrVoteDoesNotExist
		}
		return nil, err
	}

	voted, err := k.Voter.Has(ctx, collections.Join(msg.Id, voterAcc))
	if err != nil {
		return nil, err
	}
	// Check if voter has already voted
	if voted {
		return nil, types.ErrVoterHasAlreadyVoted
	}

	// Assert again voting hasn't ended
	if vote.VoteEnd.Before(ctx.BlockTime()) {
		return nil, types.ErrVotingPeriodEnded
	}

	err = k.Keeper.SetTally(ctx, voterAcc, msg.Vote, vote)
	if err != nil {
		return nil, err
	}

	return &types.MsgVoteResponse{}, nil
}
