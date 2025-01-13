package keeper

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	voted, err := k.Voter.Has(ctx, collections.Join(msg.Id, voterAcc.Bytes()))
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
	teampower, err := k.SetTeamVote(ctx, msg.Id, voterAcc, msg.Vote)
	if err != nil {
		return nil, err
	}
	upower, err := k.SetVoterTips(ctx, msg.Id, voterAcc, dispute.BlockNumber, msg.Vote)
	if err != nil {
		return nil, err
	}
	repP, err := k.SetVoterReporterStake(ctx, msg.Id, voterAcc, dispute.BlockNumber, msg.Vote)
	if err != nil {
		return nil, err
	}
	// totalSupply := k.GetTotalSupply(ctx)
	voterPower := teampower.Add(upower).Add(repP)
	if voterPower.IsZero() {
		return nil, errors.New("voter power is zero")
	}
	voterVote := types.Voter{
		Vote:             msg.Vote,
		VoterPower:       voterPower,
		ReporterPower:    repP,
	}
	if err := k.Voter.Set(ctx, collections.Join(vote.Id, voterAcc.Bytes()), voterVote); err != nil {
		return nil, err
	}

	// try to tally the vote
	err = k.Keeper.TallyVote(ctx, msg.Id)
	if err != nil {
		if !strings.EqualFold(err.Error(), types.ErrNoQuorumStillVoting.Error()) {
			return nil, err
		}
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"voted_on_dispute",
			sdk.NewAttribute("voter", msg.Voter),
			sdk.NewAttribute("voter_power", voterPower.String()),
			sdk.NewAttribute("dispute_id", strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute("choice", msg.Vote.String()),
		),
	})
	return &types.MsgVoteResponse{}, nil
}
