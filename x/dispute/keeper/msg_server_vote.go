package keeper

import (
	"context"
	"errors"
	"strconv"

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
	// bI, err := k.BlockInfo.Get(ctx, dispute.HashId)
	// if err != nil {
	// 	return nil, err
	// }
	teampower, err := k.SetTeamVote(ctx, msg.Id, voterAcc)
	if err != nil {
		return nil, err
	}
	upower, err := k.SetVoterTips(ctx, msg.Id, voterAcc, dispute.BlockNumber)
	if err != nil {
		return nil, err
	}
	// upower = CalculateVotingPower(upower, bI.TotalUserTips)
	repP, err := k.SetVoterReporterStake(ctx, msg.Id, voterAcc, dispute.BlockNumber)
	if err != nil {
		return nil, err
	}
	// repP = CalculateVotingPower(repP, bI.TotalReporterPower)
	acctBal, err := k.GetAccountBalance(ctx, voterAcc)
	if err != nil {
		return nil, err
	}
	// totalSupply := k.GetTotalSupply(ctx)
	voterPower := teampower.Add(upower).Add(repP).Add(acctBal)
	if voterPower.IsZero() {
		return nil, errors.New("voter power is zero")
	}
	voterVote := types.Voter{
		Vote:       msg.Vote,
		VoterPower: voterPower,
	}
	if err := k.Voter.Set(ctx, collections.Join(vote.Id, voterAcc.Bytes()), voterVote); err != nil {
		return nil, err
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
