package keeper

import (
	"context"
	"errors"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetVotersExist(ctx context.Context, id uint64) (bool, error) {
	iter, err := k.Voter.Indexes.VotersById.MatchExact(ctx, id)
	if err != nil {
		return false, err
	}

	valid := iter.Valid()
	if !valid {
		return false, nil
	}

	return true, nil
}

func (k Keeper) GetAccountBalance(ctx context.Context, addr sdk.AccAddress) (math.Int, error) {
	bal := k.bankKeeper.GetBalance(ctx, addr, layertypes.BondDenom)
	return bal.Amount, nil
}

// The `Ratio` function calculates the percentage ratio of `part` to `total`, scaled by a factor of 4 for the total before calculation. The result is expressed as a percentage.
// Ratio gets called on each sector of voters after votes have been summed e.g Ratio(totalUserTips, userVoteSum)
func Ratio(total, part math.Int) math.Int {
	if total.IsZero() {
		return math.ZeroInt()
	}
	total = total.MulRaw(3)
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
	k.Logger(ctx).Info("Vote", "vote", vote)

	if vote.VoteResult != types.VoteResult_NO_TALLY {
		return errors.New("vote already tallied")
	}
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return err
	}
	k.Logger(ctx).Info("Dispute", "dispute", dispute)
	info, err := k.BlockInfo.Get(ctx, dispute.HashId)
	if err != nil {
		return err
	}
	k.Logger(ctx).Info("BlockInfo", "BlockInfo", info)

	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	k.Logger(ctx).Info("VoteCountsByGroup no error", "VoteCountsByGroup", voteCounts)
	if err != nil {
		k.Logger(ctx).Info("VoteCountsByGroup error", "VoteCountsByGroup", voteCounts)
		voteCounts = types.StakeholderVoteCounts{
			Users:     types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
			Reporters: types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
			Team:      types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
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
	var teamDidVote bool
	// get team vote
	teamVote, err := k.Voter.Get(ctx, collections.Join(id, teamAddr.Bytes()))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
	} else {
		teamDidVote = true
	}
	if teamDidVote {
		vote := teamVote.Vote

		switch vote {
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

		totalRatio = totalRatio.Add(math.NewInt(33).Mul(layertypes.PowerReduction))
	}
	k.Logger(ctx).Info("TotalRatio after team vote", "TotalRatio", totalRatio)
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
	k.Logger(ctx).Info("TotalRatio after user vote", "TotalRatio", totalRatio)

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
	k.Logger(ctx).Info("TotalRatio after reporter vote", "TotalRatio", totalRatio)
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

	sdkctx := sdk.UnwrapSDKContext(ctx)
	// quorum not reached case
	if vote.VoteEnd.Before(sdkctx.BlockTime()) {
		dispute.DisputeStatus = types.Unresolved
		dispute.PendingExecution = true
		// check if rounds have been exhausted or dispute has expired in order to disperse funds
		// todo:  check if rounds have been exhausted
		// if dispute.DisputeEndTime.Before(sdkctx.BlockTime()) || dispute.DisputeRound == 5 {
		if dispute.DisputeEndTime.Before(sdkctx.BlockTime()) {
			dispute.DisputeStatus = types.Resolved
			dispute.Open = false
		}
		voterExists, err := k.GetVotersExist(ctx, id)
		if err != nil {
			return err
		}
		if !voterExists {
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
		k.Logger(ctx).Error("Vote tally", "result", "no majority")
		return nil
	}
	vote.VoteResult = result
	vote.VoteEnd = sdk.UnwrapSDKContext(ctx).BlockTime()
	return k.Votes.Set(ctx, id, vote)
}
