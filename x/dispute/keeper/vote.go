package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

// Get vote by dispute id
func (k Keeper) GetVote(ctx sdk.Context, id uint64) (*types.Vote, error) {
	store := k.voteStore(ctx)
	voteBytes := store.Get(types.DisputeIdBytes(id))
	var vote types.Vote
	if err := k.cdc.Unmarshal(voteBytes, &vote); err != nil {
		return nil, err
	}
	return &vote, nil
}

// Get a voter's vote by voter & dispute id
func (k Keeper) GetVoterVote(ctx sdk.Context, voter string, id uint64) (*types.MsgVote, error) {
	store := k.voteStore(ctx)
	voteBytes := store.Get(types.VoterKeyPrefix(voter, id))
	var vote types.MsgVote
	if err := k.cdc.Unmarshal(voteBytes, &vote); err != nil {
		return nil, err
	}
	return &vote, nil
}

// Set vote by voter
func (k Keeper) SetVoterVote(ctx sdk.Context, msg *types.MsgVote) error {
	store := k.voteStore(ctx)
	voterKey := types.VoterKeyPrefix(msg.Voter, msg.Id)
	bz, err := k.cdc.Marshal(msg)
	if err != nil {
		return err
	}
	store.Set(voterKey, bz)
	return nil
}

// Append voters to vote struct
func (k Keeper) AppendVoters(ctx sdk.Context, id uint64, voter string) error {
	vote, err := k.GetVote(ctx, id)
	if err != nil {
		return err
	}
	vote.Voters = append(vote.Voters, voter)

	return k.SetVote(ctx, id, vote)
}

func (k Keeper) initVoterClasses() *types.VoterClasses {
	return &types.VoterClasses{
		Validators:   math.ZeroInt(),
		TokenHolders: math.ZeroInt(),
		Users:        math.ZeroInt(),
		Team:         math.ZeroInt(),
	}
}

func (k Keeper) SetVote(ctx sdk.Context, id uint64, vote *types.Vote) error {
	store := k.voteStore(ctx)
	bz, err := k.cdc.Marshal(vote)
	if err != nil {
		return err
	}
	store.Set(types.DisputeIdBytes(id), bz)
	return nil
}

// Set vote results
func (k Keeper) SetVoteResult(ctx sdk.Context, id uint64, result types.VoteResult) error {
	vote, err := k.GetVote(ctx, id)
	if err != nil {
		return err
	}
	vote.VoteResult = result
	vote.VoteEnd = ctx.BlockTime()

	return k.SetVote(ctx, id, vote)
}

// Set vote start info for a dispute
func (k Keeper) SetStartVote(ctx sdk.Context, id uint64) error {
	store := k.voteStore(ctx)
	vote := types.Vote{
		Id:        id,
		VoteStart: ctx.BlockTime(),
		VoteEnd:   ctx.BlockTime().Add(TWO_DAYS),
	}
	bz, err := k.cdc.Marshal(&vote)
	if err != nil {
		return err
	}
	store.Set(types.DisputeIdBytes(id), bz)
	return nil
}
