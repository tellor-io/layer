package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

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

	tokenHolderRatio := ratio(k.GetTotalSupply(ctx), tokenHolderVoteSum)
	validatorRatio := ratio(k.GetTotalValidatorPower(ctx), validatorVoteSum)
	userRatio := ratio(k.GetTotalTips(ctx), userVoteSum)
	teamRatio := ratio(math.NewInt(1), teamVoteSum)
	totalRatio := tokenHolderRatio.Add(validatorRatio).Add(userRatio).Add(teamRatio)

	if vote.VoteResult == types.VoteResult_NO_TALLY {
		// quorum reached case
		if totalRatio.GTE(sdk.NewDec(51)) {
			switch {
			case scaledSupport.GT(scaledAgainst) && scaledSupport.GT(scaledInvalid):
				k.SetDisputeStatus(ctx, id, types.Resolved)
				k.SetVoteResult(ctx, id, types.VoteResult_SUPPORT)
			case scaledAgainst.GT(scaledSupport) && scaledAgainst.GT(scaledInvalid):
				k.SetDisputeStatus(ctx, id, types.Resolved)
				k.SetVoteResult(ctx, id, types.VoteResult_AGAINST)
			case scaledInvalid.GT(scaledSupport) && scaledInvalid.GT(scaledAgainst):
				k.SetDisputeStatus(ctx, id, types.Resolved)
				k.SetVoteResult(ctx, id, types.VoteResult_INVALID)
			default:
			}
		}
		// quorum not reached case
		if vote.VoteEnd.Before(ctx.BlockTime()) {
			disputeStatus := types.Unresolved
			// check if rounds have been exhausted or dispute has expired in order to disperse funds
			if dispute.BurnAmount.MulRaw(2).GT(dispute.SlashAmount) || dispute.DisputeEndTime.Before(ctx.BlockTime()) {
				disputeStatus = types.Resolved
			}
			switch {
			case scaledSupport.GT(scaledAgainst) && scaledSupport.GT(scaledInvalid):
				k.SetDisputeStatus(ctx, id, disputeStatus)
				k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT)
			case scaledAgainst.GT(scaledSupport) && scaledAgainst.GT(scaledInvalid):
				k.SetDisputeStatus(ctx, id, disputeStatus)
				k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST)
			case scaledInvalid.GT(scaledSupport) && scaledInvalid.GT(scaledAgainst):
				k.SetDisputeStatus(ctx, id, disputeStatus)
				k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_INVALID)
			case len(vote.Voters) == 0:
				k.SetDisputeStatus(ctx, id, disputeStatus)
				k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_INVALID)
			default:
			}
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
		tallies.ForVotes.Validators = tallies.ForVotes.Validators.Add(k.GetValidatorPower(ctx, voter))
		tallies.ForVotes.TokenHolders = tallies.ForVotes.TokenHolders.Add(k.GetAccountBalance(ctx, voter))
		tallies.ForVotes.Users = tallies.ForVotes.Users.Add(k.GetUserTips(ctx, voter))
		tallies.ForVotes.Team = tallies.ForVotes.Team.Add(k.IsTeamAddress(ctx, voter))
	case types.VoteEnum_VOTE_AGAINST:
		tallies.AgainstVotes.Validators = tallies.AgainstVotes.Validators.Add(k.GetValidatorPower(ctx, voter))
		tallies.AgainstVotes.TokenHolders = tallies.AgainstVotes.TokenHolders.Add(k.GetAccountBalance(ctx, voter))
		tallies.AgainstVotes.Users = tallies.AgainstVotes.Users.Add(k.GetUserTips(ctx, voter))
		tallies.AgainstVotes.Team = tallies.AgainstVotes.Team.Add(k.IsTeamAddress(ctx, voter))
	case types.VoteEnum_VOTE_INVALID:
		tallies.Invalid.Validators = tallies.Invalid.Validators.Add(k.GetValidatorPower(ctx, voter))
		tallies.Invalid.TokenHolders = tallies.Invalid.TokenHolders.Add(k.GetAccountBalance(ctx, voter))
		tallies.Invalid.Users = tallies.Invalid.Users.Add(k.GetUserTips(ctx, voter))
		tallies.Invalid.Team = tallies.Invalid.Team.Add(k.IsTeamAddress(ctx, voter))
	default:
		panic("invalid vote type")
	}

	store := k.voteStore(ctx)
	store.Set(types.TallyKeyPrefix(id), k.cdc.MustMarshal(tallies))
	return nil
}

func (k Keeper) GetValidatorPower(ctx sdk.Context, voter string) math.Int {
	addr := sdk.MustAccAddressFromBech32(voter)
	validator, found := k.stakingKeeper.GetValidator(ctx, sdk.ValAddress(addr))
	if !found {
		return sdk.ZeroInt()
	}
	power := validator.GetConsensusPower(sdk.DefaultPowerReduction)
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
	userTips := k.oracleKeeper.GetUserTips(ctx, sdk.MustAccAddressFromBech32(voter))
	return userTips.Total.Amount
}

func (k Keeper) IsTeamAddress(ctx sdk.Context, voter string) math.Int {
	if voter != types.TeamAddress {
		return math.ZeroInt()
	}
	return math.NewInt(1)
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
	return k.oracleKeeper.GetTotalTips(ctx).Amount
}

func ratio(total, part math.Int) math.LegacyDec {
	if total.IsZero() {
		return math.LegacyZeroDec()
	}
	total = total.MulRaw(4)
	ratio := sdk.NewDecFromInt(part).Quo(sdk.NewDecFromInt(total))
	return ratio.MulInt64(100)
}
