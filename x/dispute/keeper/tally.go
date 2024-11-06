package keeper

import (
	"context"
	"errors"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetVoters(ctx context.Context, id uint64) (
	[]collections.KeyValue[collections.Pair[uint64, []byte], types.Voter], error,
) {
	iter, err := k.Voter.Indexes.VotersById.MatchExact(ctx, id)
	if err != nil {
		return nil, err
	}
	voters, err := indexes.CollectKeyValues(ctx, k.Voter, iter)
	if err != nil {
		return nil, err
	}
	return voters, nil
}

func (k Keeper) GetAccountBalance(ctx context.Context, addr sdk.AccAddress) (math.Int, error) {
	bal := k.bankKeeper.GetBalance(ctx, addr, layertypes.BondDenom)
	return bal.Amount, nil
}

// Get total trb supply
func (k Keeper) GetTotalSupply(ctx context.Context) math.Int {
	return k.bankKeeper.GetSupply(ctx, layertypes.BondDenom).Amount
}

// The `Ratio` function calculates the percentage ratio of `part` to `total`, scaled by a factor of 4 for the total before calculation. The result is expressed as a percentage.
// Ratio gets called on each sector of voters after votes have been summed e.g Ratio(totalUserTips, userVoteSum)
func Ratio(total, part math.Int) math.Int {
	if total.IsZero() {
		return math.ZeroInt()
	}
	total = total.MulRaw(4)
	totalDec := math.LegacyNewDecFromInt(total)
	partDec := math.LegacyNewDecFromInt(part)
	powerReductionDec := math.LegacyNewDecFromInt(layertypes.PowerReduction)
	ratioDec := partDec.Mul(powerReductionDec).Quo(totalDec).Mul(math.LegacyNewDec(100))

	return ratioDec.TruncateInt()
}

// TallyVote determines whether the dispute vote has either reached quorum or the vote period has ended.
// If so, it calculates the given dispute round's outcome.
func (k Keeper) TallyVote(ctx context.Context, id uint64) error {
	numGroups := math.NewIntFromUint64(4)
	scaledSupport := math.ZeroInt()
	scaledAgainst := math.ZeroInt()
	scaledInvalid := math.ZeroInt()
	powerReductionDec := math.LegacyNewDecFromInt(layertypes.PowerReduction)

	vote, err := k.Votes.Get(ctx, id)
	if err != nil {
		return err
	}

	if vote.VoteResult != types.VoteResult_NO_TALLY {
		return errors.New("vote already tallied")
	}
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return err
	}
	info, err := k.BlockInfo.Get(ctx, dispute.HashId)
	if err != nil {
		return err
	}

	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	if err != nil {
		voteCounts = types.StakeholderVoteCounts{
			Users:        types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
			Reporters:    types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
			Tokenholders: types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
			Team:         types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
		}
	}

	totalRatio := math.ZeroInt()
	// init tallies
	tallies := types.Tally{
		ForVotes:     k.InitVoterClasses(),
		AgainstVotes: k.InitVoterClasses(),
		Invalid:      k.InitVoterClasses(),
	}

	teamAddr, err := k.GetTeamAddress(ctx)
	if err != nil {
		return err
	}
	teamDidVote, err := k.Voter.Has(ctx, collections.Join(id, teamAddr.Bytes()))
	if err != nil {
		return err
	}
	if teamDidVote {
		vote, err := k.Voter.Get(ctx, collections.Join(id, teamAddr.Bytes()))
		if err != nil {
			return err
		}
		switch vote.Vote {
		case types.VoteEnum_VOTE_SUPPORT:
			tallies.ForVotes.Team = math.OneInt()
			scaledSupport = scaledSupport.Add(layertypes.PowerReduction)
		case types.VoteEnum_VOTE_AGAINST:
			tallies.AgainstVotes.Team = math.OneInt()
			scaledAgainst = scaledAgainst.Add(layertypes.PowerReduction)
		case types.VoteEnum_VOTE_INVALID:
			tallies.Invalid.Team = math.OneInt()
			scaledInvalid = scaledInvalid.Add(layertypes.PowerReduction)
		}

		totalRatio = totalRatio.Add(math.NewInt(25).Mul(layertypes.PowerReduction))
	}

	// get user group
	tallies.ForVotes.Users = math.NewIntFromUint64(voteCounts.Users.Support)
	tallies.AgainstVotes.Users = math.NewIntFromUint64(voteCounts.Users.Against)
	tallies.Invalid.Users = math.NewIntFromUint64(voteCounts.Users.Invalid)
	userVoteSum := tallies.ForVotes.Users.Add(tallies.AgainstVotes.Users).Add(tallies.Invalid.Users)

	scaledSupportDec := math.LegacyNewDecFromInt(scaledSupport)
	scaledAgainstDec := math.LegacyNewDecFromInt(scaledAgainst)
	scaledInvalidDec := math.LegacyNewDecFromInt(scaledInvalid)
	if userVoteSum.GT(math.ZeroInt()) {
		totalRatio = totalRatio.Add(Ratio(info.TotalUserTips, userVoteSum))
		userVoteSumDec := math.LegacyNewDecFromInt(userVoteSum)

		usersForVotesDec := math.LegacyNewDecFromInt(tallies.ForVotes.Users)
		usersAgainstVotesDec := math.LegacyNewDecFromInt(tallies.AgainstVotes.Users)
		usersInvalidVotesDec := math.LegacyNewDecFromInt(tallies.Invalid.Users)

		scaledSupportDec = scaledSupportDec.Add(usersForVotesDec.Mul(powerReductionDec).Quo(userVoteSumDec))
		scaledAgainstDec = scaledAgainstDec.Add(usersAgainstVotesDec.Mul(powerReductionDec).Quo(userVoteSumDec))
		scaledInvalidDec = scaledInvalidDec.Add(usersInvalidVotesDec.Mul(powerReductionDec).Quo(userVoteSumDec))
	}

	// replace logic above with this
	tallies.ForVotes.Reporters = math.NewIntFromUint64(voteCounts.Reporters.Support)
	tallies.AgainstVotes.Reporters = math.NewIntFromUint64(voteCounts.Reporters.Against)
	tallies.Invalid.Reporters = math.NewIntFromUint64(voteCounts.Reporters.Invalid)
	reporterVoteSum := tallies.ForVotes.Reporters.Add(tallies.AgainstVotes.Reporters).Add(tallies.Invalid.Reporters)
	reporterRatio := Ratio(info.TotalReporterPower, reporterVoteSum)
	totalRatio = totalRatio.Add(reporterRatio)

	if reporterVoteSum.GT(math.ZeroInt()) {
		reporterVoteSumDec := math.LegacyNewDecFromInt(reporterVoteSum)
		reportersForVotesDec := math.LegacyNewDecFromInt(tallies.ForVotes.Reporters)
		reportersAgainstVotesDec := math.LegacyNewDecFromInt(tallies.AgainstVotes.Reporters)
		reportersInvalidVotesDec := math.LegacyNewDecFromInt(tallies.Invalid.Reporters)

		forReportersDec := reportersForVotesDec.Mul(powerReductionDec).Quo(reporterVoteSumDec)
		againstReportersDec := reportersAgainstVotesDec.Mul(powerReductionDec).Quo(reporterVoteSumDec)
		invalidReportersDec := reportersInvalidVotesDec.Mul(powerReductionDec).Quo(reporterVoteSumDec)
		scaledSupportDec = scaledSupportDec.Add(forReportersDec)
		scaledAgainstDec = scaledAgainstDec.Add(againstReportersDec)
		scaledInvalidDec = scaledInvalidDec.Add(invalidReportersDec)
	}

	if totalRatio.GTE(math.NewInt(51).Mul(layertypes.PowerReduction)) {
		numGroupsDec := math.LegacyNewDecFromInt(numGroups)
		scaledSupportDec = scaledSupportDec.Quo(numGroupsDec)
		scaledAgainstDec = scaledAgainstDec.Quo(numGroupsDec)
		scaledInvalidDec = scaledInvalidDec.Quo(numGroupsDec)

		scaledSupport = scaledSupportDec.TruncateInt()
		scaledAgainst = scaledAgainstDec.TruncateInt()
		scaledInvalid = scaledInvalidDec.TruncateInt()
		dispute.DisputeStatus = types.Resolved
		dispute.Open = false
		dispute.PendingExecution = true
		return k.UpdateDispute(ctx, id, dispute, vote, scaledSupport, scaledAgainst, scaledInvalid, true)
	}

	tokenSupply := k.GetTotalSupply(ctx)

	// replace logic above with this
	tallies.ForVotes.TokenHolders = math.NewIntFromUint64(voteCounts.Tokenholders.Support)
	tallies.AgainstVotes.TokenHolders = math.NewIntFromUint64(voteCounts.Tokenholders.Against)
	tallies.Invalid.TokenHolders = math.NewIntFromUint64(voteCounts.Tokenholders.Invalid)
	tokenHolderVoteSum := tallies.ForVotes.TokenHolders.Add(tallies.AgainstVotes.TokenHolders).Add(tallies.Invalid.TokenHolders)
	totalRatio = totalRatio.Add(Ratio(tokenSupply, tokenHolderVoteSum))

	if !tokenHolderVoteSum.IsZero() {
		tokenHolderVoteSumDec := math.LegacyNewDecFromInt(tokenHolderVoteSum)
		tokenHoldersForVotesDec := math.LegacyNewDecFromInt(tallies.ForVotes.TokenHolders)
		tokenHoldersAgainstVotesDec := math.LegacyNewDecFromInt(tallies.AgainstVotes.TokenHolders)
		tokenHoldersInvalidVotesDec := math.LegacyNewDecFromInt(tallies.Invalid.TokenHolders)

		forTokenHoldersDec := tokenHoldersForVotesDec.Mul(powerReductionDec).Quo(tokenHolderVoteSumDec)
		againstTokenHoldersDec := tokenHoldersAgainstVotesDec.Mul(powerReductionDec).Quo(tokenHolderVoteSumDec)
		invalidTokenHoldersDec := tokenHoldersInvalidVotesDec.Mul(powerReductionDec).Quo(tokenHolderVoteSumDec)
		scaledSupportDec = scaledSupportDec.Add(forTokenHoldersDec)
		scaledAgainstDec = scaledAgainstDec.Add(againstTokenHoldersDec)
		scaledInvalidDec = scaledInvalidDec.Add(invalidTokenHoldersDec)
	}
	if totalRatio.GTE(math.NewInt(51).Mul(layertypes.PowerReduction)) {
		dispute.DisputeStatus = types.Resolved
		dispute.Open = false
		dispute.PendingExecution = true
		return k.UpdateDispute(ctx, id, dispute, vote, scaledSupportDec.TruncateInt(), scaledAgainstDec.TruncateInt(), scaledInvalidDec.TruncateInt(), true)
	}
	sdkctx := sdk.UnwrapSDKContext(ctx)
	// quorum not reached case
	if vote.VoteEnd.Before(sdkctx.BlockTime()) {
		dispute.DisputeStatus = types.Unresolved
		dispute.PendingExecution = true
		// check if rounds have been exhausted or dispute has expired in order to disperse funds
		if dispute.DisputeEndTime.Before(sdkctx.BlockTime()) {
			dispute.DisputeStatus = types.Resolved
			dispute.Open = false
		}
		allvoters, err := k.GetVoters(ctx, id)
		if err != nil {
			return err
		}
		if len(allvoters) == 0 {
			if err := k.Disputes.Set(ctx, id, dispute); err != nil {
				return err
			}
			vote.VoteResult = types.VoteResult_NO_QUORUM_MAJORITY_INVALID
			vote.VoteEnd = sdkctx.BlockTime()
			return k.Votes.Set(ctx, id, vote)
		}
		return k.UpdateDispute(ctx, id, dispute, vote, scaledSupportDec.TruncateInt(), scaledAgainstDec.TruncateInt(), scaledInvalidDec.TruncateInt(), false)
	} else {
		return errors.New(types.ErrNoQuorumStillVoting.Error())
	}
}

func (k Keeper) UpdateDispute(
	ctx context.Context,
	id uint64,
	dispute types.Dispute,
	vote types.Vote,
	scaledSupport, scaledAgainst, scaledInvalid math.Int, quorum bool,
) error {
	if err := k.Disputes.Set(ctx, id, dispute); err != nil {
		return err
	}
	var result types.VoteResult
	switch {
	case scaledSupport.GT(scaledAgainst) && scaledSupport.GT(scaledInvalid):
		if quorum {
			result = types.VoteResult_SUPPORT
		} else {
			result = types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT
		}
	case scaledAgainst.GT(scaledSupport) && scaledAgainst.GT(scaledInvalid):
		if quorum {
			result = types.VoteResult_AGAINST
		} else {
			result = types.VoteResult_NO_QUORUM_MAJORITY_AGAINST
		}
	case scaledInvalid.GT(scaledSupport) && scaledInvalid.GT(scaledAgainst):
		if quorum {
			result = types.VoteResult_INVALID
		} else {
			result = types.VoteResult_NO_QUORUM_MAJORITY_INVALID
		}
	default:
		return errors.New("no majority")
	}
	vote.VoteResult = result
	vote.VoteEnd = sdk.UnwrapSDKContext(ctx).BlockTime()
	return k.Votes.Set(ctx, id, vote)
}
