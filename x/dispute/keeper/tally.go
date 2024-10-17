package keeper

import (
	"context"
	"errors"
	"fmt"

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
func Ratio(total, part math.Int) math.LegacyDec {
	if total.IsZero() {
		return math.LegacyZeroDec()
	}
	total = total.MulRaw(4)
	ratio := math.LegacyNewDecFromInt(part).Quo(math.LegacyNewDecFromInt(total))
	return ratio.MulInt64(100)
}

// CalculateVotingPower calculates the voting power of a given number (n) divided by another number (d).
func CalculateVotingPower(n, d math.Int) math.Int {
	fmt.Println("CalculateVotingPower")
	fmt.Println("n", n)
	fmt.Println("d", d)
	if n.IsZero() || d.IsZero() {
		return math.ZeroInt()
	}
	scalingFactor := math.NewInt(1_000_000)
	result := n.Mul(scalingFactor).Quo(d).MulRaw(25_000_000).Quo(scalingFactor)
	fmt.Println("result", result)
	// shouldn't this just be n.MulRaw(25_000_000).Quo(d) ?
	return n.Mul(scalingFactor).Quo(d).MulRaw(25_000_000).Quo(scalingFactor)
}

// TallyVote determines whether the dispute vote has either reached quorum or the vote period has ended.
// If so, it calculates the given dispute round's outcome.
func (k Keeper) TallyVote(ctx context.Context, id uint64) error {
	numGroups := math.LegacyNewDec(4)
	scaledSupport := math.LegacyZeroDec()
	scaledAgainst := math.LegacyZeroDec()
	scaledInvalid := math.LegacyZeroDec()
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
		return err
	}

	totalRatio := math.LegacyZeroDec()
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
			scaledSupport = scaledSupport.Add(math.LegacyOneDec())
		case types.VoteEnum_VOTE_AGAINST:
			tallies.AgainstVotes.Team = math.OneInt()
			scaledAgainst = scaledAgainst.Add(math.LegacyOneDec())
		case types.VoteEnum_VOTE_INVALID:
			tallies.Invalid.Team = math.OneInt()
			scaledInvalid = scaledInvalid.Add(math.LegacyOneDec())
		}

		totalRatio = totalRatio.Add(math.LegacyNewDec(25))
	}

	// get user group
	// userVoteSum := math.ZeroInt()
	tallies.ForVotes.Users = math.NewIntFromUint64(voteCounts.Users.Support)
	tallies.AgainstVotes.Users = math.NewIntFromUint64(voteCounts.Users.Against)
	tallies.Invalid.Users = math.NewIntFromUint64(voteCounts.Users.Invalid)
	userVoteSum := tallies.ForVotes.Users.Add(tallies.AgainstVotes.Users).Add(tallies.Invalid.Users)

	if userVoteSum.GT(math.ZeroInt()) {
		totalRatio = totalRatio.Add(Ratio(info.TotalUserTips, userVoteSum))

		userVoteSumDec := math.LegacyNewDecFromInt(userVoteSum)

		scaledSupport = scaledSupport.Add(math.LegacyNewDecFromInt(tallies.ForVotes.Users).Quo(userVoteSumDec))
		scaledAgainst = scaledAgainst.Add(math.LegacyNewDecFromInt(tallies.AgainstVotes.Users).Quo(userVoteSumDec))
		scaledInvalid = scaledInvalid.Add(math.LegacyNewDecFromInt(tallies.Invalid.Users).Quo(userVoteSumDec))
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
		forReporters := math.LegacyNewDecFromInt(tallies.ForVotes.Reporters).Quo(reporterVoteSumDec)
		againstReporters := math.LegacyNewDecFromInt(tallies.AgainstVotes.Reporters).Quo(reporterVoteSumDec)
		invalidReporters := math.LegacyNewDecFromInt(tallies.Invalid.Reporters).Quo(reporterVoteSumDec)
		scaledSupport = scaledSupport.Add(forReporters)
		scaledAgainst = scaledAgainst.Add(againstReporters)
		scaledInvalid = scaledInvalid.Add(invalidReporters)
	}

	if totalRatio.GTE(math.LegacyNewDec(51)) {
		scaledSupport = scaledSupport.Quo(numGroups)
		scaledAgainst = scaledAgainst.Quo(numGroups)
		scaledInvalid = scaledInvalid.Quo(numGroups)
		fmt.Println("quorum reached")
		dispute.DisputeStatus = types.Resolved
		dispute.Open = false
		return k.UpdateDispute(ctx, id, dispute, vote, scaledSupport, scaledAgainst, scaledInvalid, true)
	}

	tokenSupply := k.GetTotalSupply(ctx)

	// replace logic above with this
	tallies.ForVotes.TokenHolders = math.NewIntFromUint64(voteCounts.Tokenholders.Support)
	tallies.AgainstVotes.TokenHolders = math.NewIntFromUint64(voteCounts.Tokenholders.Against)
	tallies.Invalid.TokenHolders = math.NewIntFromUint64(voteCounts.Tokenholders.Invalid)
	tokenHolderVoteSum := tallies.ForVotes.TokenHolders.Add(tallies.AgainstVotes.TokenHolders).Add(tallies.Invalid.TokenHolders)
	totalRatio = totalRatio.Add(Ratio(tokenSupply, tokenHolderVoteSum))

	tokenHolderVoteSumDec := math.LegacyNewDecFromInt(tokenHolderVoteSum)

	if !tokenHolderVoteSum.IsZero() {
		forTokenHolders := math.LegacyNewDecFromInt(tallies.ForVotes.TokenHolders).Quo(tokenHolderVoteSumDec)
		againstTokenHolders := math.LegacyNewDecFromInt(tallies.AgainstVotes.TokenHolders).Quo(tokenHolderVoteSumDec)
		invalidTokenHolders := math.LegacyNewDecFromInt(tallies.Invalid.TokenHolders).Quo(tokenHolderVoteSumDec)
		scaledSupport = scaledSupport.Add(forTokenHolders).Quo(numGroups)
		scaledAgainst = scaledAgainst.Add(againstTokenHolders).Quo(numGroups)
		scaledInvalid = scaledInvalid.Add(invalidTokenHolders).Quo(numGroups)
	}
	if totalRatio.GTE(math.LegacyNewDec(51)) {
		dispute.DisputeStatus = types.Resolved
		dispute.Open = false
		return k.UpdateDispute(ctx, id, dispute, vote, scaledSupport, scaledAgainst, scaledInvalid, true)
	}
	sdkctx := sdk.UnwrapSDKContext(ctx)
	// quorum not reached case
	if vote.VoteEnd.Before(sdkctx.BlockTime()) {
		fmt.Println("quorum not reached")
		dispute.DisputeStatus = types.Unresolved
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
		return k.UpdateDispute(ctx, id, dispute, vote, scaledSupport, scaledAgainst, scaledInvalid, false)
	} else {
		return errors.New(types.ErrNoQuorumStillVoting.Error())
	}
}

func (k Keeper) UpdateDispute(
	ctx context.Context,
	id uint64,
	dispute types.Dispute,
	vote types.Vote,
	scaledSupport, scaledAgainst, scaledInvalid math.LegacyDec, quorum bool,
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
