package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

// Get vote by dispute id
func (k Keeper) GetVote(ctx sdk.Context, id uint64) *types.Vote {
	store := k.voteStore(ctx)
	voteBytes := store.Get(types.DisputeIdBytes(id))
	var vote types.Vote
	if err := k.cdc.Unmarshal(voteBytes, &vote); err != nil {
		return nil
	}
	return &vote
}

// Get a voter's vote by voter & dispute id
func (k Keeper) GetVoterVote(ctx sdk.Context, voter string, id uint64) *types.MsgVote {
	store := k.voteStore(ctx)
	voteBytes := store.Get(types.VoterKeyPrefix(voter, id))
	var vote types.MsgVote
	if err := k.cdc.Unmarshal(voteBytes, &vote); err != nil {
		return nil
	}
	return &vote

}

// Set vote by voter
func (k Keeper) SetVoterVote(ctx sdk.Context, msg *types.MsgVote) {
	store := k.voteStore(ctx)
	voterKey := types.VoterKeyPrefix(msg.Voter, msg.Id)
	store.Set(voterKey, k.cdc.MustMarshal(msg))
}

// Append voters to vote struct
func (k Keeper) AppendVoters(ctx sdk.Context, id uint64, voter string) {
	vote := k.GetVote(ctx, id)
	vote.Voters = append(vote.Voters, voter)
	k.SetVote(ctx, id, vote)
}

func (k Keeper) initVoterClasses() *types.VoterClasses {
	return &types.VoterClasses{
		Validators:   math.ZeroInt(),
		TokenHolders: math.ZeroInt(),
		Users:        math.ZeroInt(),
		Team:         math.ZeroInt(),
	}
}

func (k Keeper) SetVote(ctx sdk.Context, id uint64, vote *types.Vote) {
	store := k.voteStore(ctx)
	store.Set(types.DisputeIdBytes(id), k.cdc.MustMarshal(vote))
}

// Set vote results
func (k Keeper) SetVoteResult(ctx sdk.Context, id uint64, result types.VoteResult) {
	vote := k.GetVote(ctx, id)
	vote.VoteResult = result
	vote.VoteEnd = ctx.BlockTime()
	k.SetVote(ctx, id, vote)
}

// Set vote start info for a dispute
func (k Keeper) SetStartVote(ctx sdk.Context, id uint64) {
	store := k.voteStore(ctx)
	vote := types.Vote{
		Id:        id,
		VoteStart: ctx.BlockTime(),
		VoteEnd:   ctx.BlockTime().Add(TWO_DAYS),
	}
	store.Set(types.DisputeIdBytes(id), k.cdc.MustMarshal(&vote))
}
