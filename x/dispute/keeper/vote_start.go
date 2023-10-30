package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

// Set vote start time by dispute id
func (k Keeper) SetStartVote(ctx sdk.Context, id uint64) {
	store := k.voteStore(ctx)
	var vote types.Vote
	vote.Id = id
	vote.VoteStart = ctx.BlockTime()
	vote.VoteEnd = ctx.BlockTime().Add(86400 * 2)
	store.Set(types.DisputeIdBytes(id), k.cdc.MustMarshal(&vote))
}

// Check if disputes are ready to start voting
func (k Keeper) StartVoting(ctx sdk.Context, ids []uint64) {
	for _, disputeId := range ids {
		// get dispute by id
		dispute := k.GetDisputeById(ctx, disputeId)
		if dispute.SlashAmount.GTE(dispute.FeeTotal) && dispute.DisputeStatus == types.Prevote {
			// set dispute status to voting
			k.SetDisputeStatus(ctx, disputeId, types.Voting)
			k.AddTimeToDisputeEndTime(ctx, *dispute, 86400*2)
			k.SetStartVote(ctx, disputeId)
		}
	}
}
