package keeper

import (
	"fmt"
	"math/big"

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
			fmt.Println("quorum reached")
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
			return
		}
		// quorum not reached case
		if vote.VoteEnd.Before(ctx.BlockTime()) {
			disputeStatus := types.Unresolved
			// check if rounds have been exhausted or dispute has expired in order to disperse funds
			if dispute.DisputeEndTime.Before(ctx.BlockTime()) {
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
			return
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
	valP := k.GetValidatorPower(ctx, voter)
	tkHol := k.GetAccountBalance(ctx, voter)
	usrTps := k.GetUserTips(ctx, voter)
	team := k.IsTeamAddress(ctx, voter)

	voterPower := math.ZeroInt()
	voterPower = voterPower.Add(calculateVotingPower(valP, k.GetTotalValidatorPower(ctx)))
	voterPower = voterPower.Add(calculateVotingPower(tkHol, k.GetTotalSupply(ctx)))
	voterPower = voterPower.Add(calculateVotingPower(usrTps, k.GetTotalTips(ctx)))
	voterPower = voterPower.Add(calculateVotingPower(team, math.NewInt(1)))

	switch voteFor {
	case types.VoteEnum_VOTE_SUPPORT:
		tallies.ForVotes.Validators = tallies.ForVotes.Validators.Add(valP)
		tallies.ForVotes.TokenHolders = tallies.ForVotes.TokenHolders.Add(tkHol)
		tallies.ForVotes.Users = tallies.ForVotes.Users.Add(usrTps)
		tallies.ForVotes.Team = tallies.ForVotes.Team.Add(team)
	case types.VoteEnum_VOTE_AGAINST:
		tallies.AgainstVotes.Validators = tallies.AgainstVotes.Validators.Add(valP)
		tallies.AgainstVotes.TokenHolders = tallies.AgainstVotes.TokenHolders.Add(tkHol)
		tallies.AgainstVotes.Users = tallies.AgainstVotes.Users.Add(usrTps)
		tallies.AgainstVotes.Team = tallies.AgainstVotes.Team.Add(team)
	case types.VoteEnum_VOTE_INVALID:
		tallies.Invalid.Validators = tallies.Invalid.Validators.Add(valP)
		tallies.Invalid.TokenHolders = tallies.Invalid.TokenHolders.Add(tkHol)
		tallies.Invalid.Users = tallies.Invalid.Users.Add(usrTps)
		tallies.Invalid.Team = tallies.Invalid.Team.Add(team)
	default:
		panic("invalid vote type")
	}

	k.SetVoterTally(ctx, id, tallies)
	k.SetVoterPower(ctx, sdk.MustAccAddressFromBech32(voter), voterPower)
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

func (k Keeper) SetVoterTally(ctx sdk.Context, id uint64, tally *types.Tally) {
	voteStore := k.voteStore(ctx)
	voteStore.Set(types.TallyKeyPrefix(id), k.cdc.MustMarshal(tally))
}

func (k Keeper) SetVoterPower(ctx sdk.Context, voter sdk.AccAddress, vp math.Int) {
	store := k.voterPowerStore(ctx)
	store.Set(voter, vp.BigInt().Bytes())
}

func (k Keeper) GetVoterPower(ctx sdk.Context, voter sdk.AccAddress) math.Int {
	store := k.voterPowerStore(ctx)
	vpBytes := store.Get(voter)
	if vpBytes == nil {
		return math.ZeroInt()
	}
	return math.NewIntFromBigInt(new(big.Int).SetBytes(vpBytes))
}

func ratio(total, part math.Int) math.LegacyDec {
	if total.IsZero() {
		return math.LegacyZeroDec()
	}
	total = total.MulRaw(4)
	ratio := sdk.NewDecFromInt(part).Quo(sdk.NewDecFromInt(total))
	return ratio.MulInt64(100)
}

func calculateVotingPower(n, d math.Int) math.Int {
	if d.IsZero() {
		return math.ZeroInt()
	}
	scalingFactor := math.NewInt(1_000_000)
	// TODO: round?
	return n.Mul(scalingFactor).Quo(d).MulRaw(25).Quo(scalingFactor)
}

func (k Keeper) CalculateVoterShare(ctx sdk.Context, voters []string, totalTokens math.Int) map[string]math.Int {
	// remove duplicates from voters list
	seen := make(map[string]bool)
	var uniqueVoters []string
	for _, voter := range voters {
		if !seen[voter] {
			seen[voter] = true
			uniqueVoters = append(uniqueVoters, voter)
		}
	}
	var totalPower = math.ZeroInt()
	powers := make(map[string]math.Int)
	for _, voter := range uniqueVoters {
		voterShare := k.GetVoterPower(ctx, sdk.MustAccAddressFromBech32(voter))
		totalPower = totalPower.Add(voterShare)
		powers[voter] = voterShare
	}

	// Calculate and allocate tokens based on each person's share of the total power
	tokenDistribution := make(map[string]math.Int)
	scalingFactor := math.NewInt(1_000_000)
	totalShare := math.ZeroInt()
	for voter, power := range powers {
		share := power.Mul(scalingFactor).Quo(totalPower)
		tokens := share.Mul(totalTokens).Quo(scalingFactor)
		tokenDistribution[voter] = tokens
		totalShare = totalShare.Add(tokens)
	}

	if totalTokens.GT(totalShare) {
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(Denom, totalTokens.Sub(totalShare)))); err != nil {
			panic(err)
		}
	}

	return tokenDistribution
}
