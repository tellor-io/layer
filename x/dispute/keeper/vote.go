package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k Keeper) initVoterClasses() *types.VoterClasses {
	return &types.VoterClasses{
		Validators:   math.ZeroInt(),
		TokenHolders: math.ZeroInt(),
		Users:        math.ZeroInt(),
		Team:         math.ZeroInt(),
	}
}

// Set vote results
func (k Keeper) SetVoteResult(ctx sdk.Context, id uint64, result types.VoteResult) error {
	vote, err := k.Votes.Get(ctx, id)
	if err != nil {
		return err
	}
	vote.VoteResult = result
	vote.VoteEnd = ctx.BlockTime()

	return k.Votes.Set(ctx, id, vote)
}

// Set vote start info for a dispute
func (k Keeper) SetStartVote(ctx sdk.Context, id uint64) error {
	vote := types.Vote{
		Id:        id,
		VoteStart: ctx.BlockTime(),
		VoteEnd:   ctx.BlockTime().Add(TWO_DAYS),
		Tally: &types.Tally{
			ForVotes:     k.initVoterClasses(),
			AgainstVotes: k.initVoterClasses(),
			Invalid:      k.initVoterClasses(),
		},
	}
	return k.Votes.Set(ctx, id, vote)
}
