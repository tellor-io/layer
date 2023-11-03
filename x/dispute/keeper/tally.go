package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k Keeper) Tally(ctx sdk.Context, ids []uint64) {
	for _, id := range ids {
		k.TallyVote(ctx, id)
	}
}

// Add invalid vote type and return all fees to the paying parties
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

	tokenHolderVoteSum := tally.ForVotes.TokenHolders.Add(tally.AgainstVotes.TokenHolders).Add(tally.Invalid.TokenHolders)
	validatorVoteSum := tally.ForVotes.Validators.Add(tally.AgainstVotes.Validators).Add(tally.Invalid.Validators)
	userVoteSum := tally.ForVotes.Users.Add(tally.AgainstVotes.Users).Add(tally.Invalid.Users)
	teamVoteSum := tally.ForVotes.Team.Add(tally.AgainstVotes.Team).Add(tally.Invalid.Team)

	// Prevent zero-division
	if tokenHolderVoteSum.IsZero() {
		tokenHolderVoteSum = math.NewInt(1)
	}
	if validatorVoteSum.IsZero() {
		validatorVoteSum = math.NewInt(1)
	}
	if userVoteSum.IsZero() {
		userVoteSum = math.NewInt(1)
	}
	if teamVoteSum.IsZero() {
		teamVoteSum = math.NewInt(1)
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

	invalidTokenHolders := sdk.NewDecFromInt(tally.Invalid.TokenHolders).Quo(tokenHolderVoteSumDec)
	invalidValidators := sdk.NewDecFromInt(tally.Invalid.Validators).Quo(validatorVoteSumDec)
	invalidUsers := sdk.NewDecFromInt(tally.Invalid.Users).Quo(userVoteSumDec)
	invalidTeam := sdk.NewDecFromInt(tally.Invalid.Team).Quo(teamVoteSumDec)

	// Sum the normalized votes and divide by number of groups to scale between 0 and 1
	numGroups := sdk.NewDec(4)
	scaledSupport := (forTokenHolders.Add(forValidators).Add(forUsers).Add(forTeam)).Quo(numGroups)
	scaledAgainst := (againstTokenHolders.Add(againstValidators).Add(againstUsers).Add(againstTeam)).Quo(numGroups)
	scaledInvalid := (invalidTokenHolders.Add(invalidValidators).Add(invalidUsers).Add(invalidTeam)).Quo(numGroups)

	totalQuorum := sdk.NewDecWithPrec(51, 2) // 51% quorum

	totalVotesDec := tokenHolderVoteSumDec.Add(validatorVoteSumDec).Add(userVoteSumDec).Add(teamVoteSumDec)
	// The maximum potential votes include total supply of tokens, total validator power, total tips, and the team's vote.
	// The team's vote is represented as 1 since it's controlled by a single multisig address.
	maximumVotesDec := sdk.NewDecFromInt(k.GetTotalSupply(ctx)).Add(sdk.NewDecFromInt(k.GetTotalValidatorPower(ctx))).Add(sdk.NewDecFromInt(k.GetTotalTips(ctx))).Add(sdk.NewDecFromInt(math.NewInt(1)))

	participationRate := totalVotesDec.Quo(maximumVotesDec)

	if participationRate.GTE(totalQuorum) {
		combinedSupport := scaledSupport.Mul(participationRate)
		combinedAgainst := scaledAgainst.Mul(participationRate)
		combinedInvalid := scaledInvalid.Mul(participationRate)

		switch {
		case combinedSupport.GT(combinedAgainst) && combinedSupport.GT(combinedInvalid):
			k.SetDisputeStatus(ctx, id, types.Resolved)
			k.SetVoteResult(ctx, id, types.VoteResult_SUPPORT)
		case combinedAgainst.GT(combinedSupport) && combinedAgainst.GT(combinedInvalid):
			k.SetDisputeStatus(ctx, id, types.Resolved)
			k.SetVoteResult(ctx, id, types.VoteResult_AGAINST)
		case combinedInvalid.GT(combinedSupport) && combinedInvalid.GT(combinedAgainst):
			k.SetDisputeStatus(ctx, id, types.Resolved)
			k.SetVoteResult(ctx, id, types.VoteResult_INVALID)
		default:
		}
	} else {
		// Check if vote period ended
		if vote.VoteEnd.After(ctx.BlockTime()) {
			return
		}
		switch {
		case scaledSupport.GT(scaledAgainst) && scaledSupport.GT(scaledInvalid):
			k.SetDisputeStatus(ctx, id, types.Unresolved)
			k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT)
		case scaledAgainst.GT(scaledSupport) && scaledAgainst.GT(scaledInvalid):
			k.SetDisputeStatus(ctx, id, types.Unresolved)
			k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST)
		case scaledInvalid.GT(scaledSupport) && scaledInvalid.GT(scaledAgainst):
			k.SetDisputeStatus(ctx, id, types.Unresolved)
			k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_INVALID)
		default:
		}
	}

}

func (k Keeper) GetTally(ctx sdk.Context, id uint64) *types.Tally {
	store := k.voteStore(ctx)
	tallyBytes := store.Get(types.TallyKeyPrefix(id))
	var tallies types.Tally
	if err := k.cdc.Unmarshal(tallyBytes, &tallies); err != nil {
		return nil
	}
	// for some reason the proto files don't initialize the tallies when nullable = false
	if tallies.ForVotes == nil {
		tallies.ForVotes = k.initVoterClasses()
		tallies.AgainstVotes = k.initVoterClasses()
		tallies.Invalid = k.initVoterClasses()
	}

	return &tallies
}

// Set tally numbers
func (k Keeper) SetTally(ctx sdk.Context, id uint64, voteFor types.VoteEnum, voter string) error {
	var tallies *types.Tally
	tallies = k.GetTally(ctx, id)
	switch voteFor {
	case types.VoteEnum_VOTE_SUPPORT:
		tallies.ForVotes.Validators.Add(k.GetValidatorPower(ctx, voter))
		tallies.ForVotes.TokenHolders.Add(k.GetAccountBalance(ctx, voter))
		tallies.ForVotes.Users.Add(k.GetUserTips(ctx, voter))
		tallies.ForVotes.Team.Add(k.IsTeamAddress(ctx, voter))
	case types.VoteEnum_VOTE_AGAINST:
		tallies.AgainstVotes.Validators.Add(k.GetValidatorPower(ctx, voter))
		tallies.AgainstVotes.TokenHolders.Add(k.GetAccountBalance(ctx, voter))
		tallies.AgainstVotes.Users.Add(k.GetUserTips(ctx, voter))
		tallies.AgainstVotes.Team.Add(k.IsTeamAddress(ctx, voter))
	case types.VoteEnum_VOTE_INVALID:
		tallies.Invalid.Validators.Add(k.GetValidatorPower(ctx, voter))
		tallies.Invalid.TokenHolders.Add(k.GetAccountBalance(ctx, voter))
		tallies.Invalid.Users.Add(k.GetUserTips(ctx, voter))
		tallies.Invalid.Team.Add(k.IsTeamAddress(ctx, voter))
	default:
		panic("invalid vote type")
	}

	store := k.voteStore(ctx)
	store.Set(types.TallyKeyPrefix(id), k.cdc.MustMarshal(tallies))
	return nil
}

func (k Keeper) GetValidatorPower(ctx sdk.Context, voter string) math.Int {
	addr, err := sdk.AccAddressFromBech32(voter)
	if err != nil {
		panic(err)
	}
	validator, found := k.stakingKeeper.GetValidator(ctx, sdk.ValAddress(addr))
	if !found {
		return sdk.ZeroInt()
	}
	power := validator.GetConsensusPower(validator.GetBondedTokens())
	return math.NewInt(power)
}

func (k Keeper) GetAccountBalance(ctx sdk.Context, voter string) math.Int {
	addr, err := sdk.AccAddressFromBech32(voter)
	if err != nil {
		panic(err)
	}
	return k.bankKeeper.GetBalance(ctx, addr, sdk.DefaultBondDenom).Amount
}

func (k Keeper) GetUserTips(ctx sdk.Context, voter string) math.Int {
	return sdk.ZeroInt()
}

func (k Keeper) IsTeamAddress(ctx sdk.Context, voter string) math.Int {
	return math.NewIntFromUint64(1)
}

// Get total trb supply
func (k Keeper) GetTotalSupply(ctx sdk.Context) math.Int {
	return k.bankKeeper.GetSupply(ctx, sdk.DefaultBondDenom).Amount
}

// Get total validator power
// TODO: this changes with every block, so we need to store it somewhere?
func (k Keeper) GetTotalValidatorPower(ctx sdk.Context) math.Int {
	return k.stakingKeeper.GetLastTotalPower(ctx)
}

// Get total number of tips
func (k Keeper) GetTotalTips(ctx sdk.Context) math.Int {
	return math.NewIntFromUint64(1)
}
