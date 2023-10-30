package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k Keeper) Tally(ctx sdk.Context, ids []uint64) {
	for _, id := range ids {
		k.TallyVote(ctx, id)
	}
}

func (k Keeper) TallyVote(ctx sdk.Context, id uint64) {
	dispute := k.GetDisputeById(ctx, id)
	if dispute == nil {
		return
	}
	tally := k.GetTally(ctx, id)
	if tally == nil {
		return
	}
	vote := k.GetVote(ctx, id)
	if vote == nil {
		return
	}
	// Check if vote period ended
	if vote.VoteEnd.After(ctx.BlockTime()) {
		return
	}
	tokenHolderVoteSum := tally.ForVotes.TokenHolders.Add(tally.AgainstVotes.TokenHolders)
	validatorVoteSum := tally.ForVotes.Validators.Add(tally.AgainstVotes.Validators)
	userVoteSum := tally.ForVotes.Users.Add(tally.AgainstVotes.Users)
	teamVoteSum := tally.ForVotes.Team.Add(tally.AgainstVotes.Team)

	// Prevent zero-division
	if tokenHolderVoteSum.IsZero() {
		tokenHolderVoteSum = sdk.NewInt(1)
	}
	if validatorVoteSum.IsZero() {
		validatorVoteSum = sdk.NewInt(1)
	}
	if userVoteSum.IsZero() {
		userVoteSum = sdk.NewInt(1)
	}
	if teamVoteSum.IsZero() {
		teamVoteSum = sdk.NewInt(1)
	}

	// Convert to Dec for precision
	tokenHolderVoteSumDec := sdk.NewDecFromInt(tokenHolderVoteSum)
	validatorVoteSumDec := sdk.NewDecFromInt(validatorVoteSum)
	userVoteSumDec := sdk.NewDecFromInt(userVoteSum)
	teamVoteSumDec := sdk.NewDecFromInt(teamVoteSum)

	// Normalize the votes for each group
	forTokenHolders := sdk.NewDecFromInt(tally.ForVotes.TokenHolders).Quo(tokenHolderVoteSumDec)
	forValidators := sdk.NewDecFromInt(tally.ForVotes.Validators).Quo(validatorVoteSumDec)
	forUsers := sdk.NewDecFromInt(tally.ForVotes.Users).Quo(userVoteSumDec)
	forTeam := sdk.NewDecFromInt(tally.ForVotes.Team).Quo(teamVoteSumDec)

	againstTokenHolders := sdk.NewDecFromInt(tally.AgainstVotes.TokenHolders).Quo(tokenHolderVoteSumDec)
	againstValidators := sdk.NewDecFromInt(tally.AgainstVotes.Validators).Quo(validatorVoteSumDec)
	againstUsers := sdk.NewDecFromInt(tally.AgainstVotes.Users).Quo(userVoteSumDec)
	againstTeam := sdk.NewDecFromInt(tally.AgainstVotes.Team).Quo(teamVoteSumDec)

	// Sum the normalized votes and divide by number of groups to scale between 0 and 1
	numGroups := sdk.NewDec(4)
	scaledSupport := (forTokenHolders.Add(forValidators).Add(forUsers).Add(forTeam)).Quo(numGroups)
	scaledAgainst := (againstTokenHolders.Add(againstValidators).Add(againstUsers).Add(againstTeam)).Quo(numGroups)

	if scaledSupport.GT(scaledAgainst) {
		// Check if support is greater than 50%
		if scaledSupport.GT(sdk.NewDecWithPrec(5, 1)) {
			k.SetDisputeStatus(ctx, id, types.Resolved)
			k.SetVoteResult(ctx, id, types.VoteResult_PASSED)
		} else {
			k.SetDisputeStatus(ctx, id, types.Unresolved)
			k.SetVoteResult(ctx, id, types.VoteResult_UNRESOLVEDPASSED)
			// Set end time to an extra day after vote end to allow for more rounds
			k.AddTimeToDisputeEndTime(ctx, *dispute, 86400)
		}
	}

	if scaledAgainst.GT(scaledSupport) {
		// Check if against is greater than 50%
		if scaledAgainst.GT(sdk.NewDecWithPrec(5, 1)) {
			k.SetDisputeStatus(ctx, id, types.Resolved)
			k.SetVoteResult(ctx, id, types.VoteResult_FAILED)
		} else {
			k.SetDisputeStatus(ctx, id, types.Unresolved)
			k.SetVoteResult(ctx, id, types.VoteResult_UNRESOLVEDFAILED)
			// Set end time to an extra day after vote end to allow for more rounds
			k.AddTimeToDisputeEndTime(ctx, *dispute, 86400)
		}
	}
}

// Set vote results
func (k Keeper) SetVoteResult(ctx sdk.Context, id uint64, result types.VoteResult) {
	vote := k.GetVote(ctx, id)
	vote.VoteResult = result
	vote.VoteEnd = ctx.BlockTime()
	store := k.voteStore(ctx)
	store.Set(types.DisputeIdBytes(id), k.cdc.MustMarshal(vote))
}
