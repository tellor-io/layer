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
	if n.IsZero() || d.IsZero() {
		return math.ZeroInt()
	}
	scalingFactor := math.NewInt(1_000_000)
	return n.Mul(scalingFactor).Quo(d).MulRaw(25_000_000).Quo(scalingFactor)
}

// CalculateVotingPower calculates the voting power of a given number (n) divided by another number (d).
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

	totalRatio := math.LegacyZeroDec()
	// init tallies
	tallies := types.Tally{
		ForVotes:     k.InitVoterClasses(),
		AgainstVotes: k.InitVoterClasses(),
		Invalid:      k.InitVoterClasses(),
	}

	teamVote, err := k.TeamVote(ctx, id)
	if err != nil {
		return err
	}
	if teamVote.GT(math.ZeroInt()) {
		teamAddr, err := k.GetTeamAddress(ctx)
		if err != nil {
			return err
		}
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
	userVoteSum := math.ZeroInt()
	userRng := collections.NewPrefixedPairRange[uint64, []byte](id)
	err = k.UsersGroup.Walk(ctx, userRng, func(key collections.Pair[uint64, []byte], value math.Int) (stop bool, err error) {
		vote, err := k.Voter.Get(ctx, key)
		if err != nil {
			return true, err
		}
		switch vote.Vote {
		case types.VoteEnum_VOTE_SUPPORT:
			tallies.ForVotes.Users = tallies.ForVotes.Users.Add(value)
		case types.VoteEnum_VOTE_AGAINST:
			tallies.AgainstVotes.Users = tallies.AgainstVotes.Users.Add(value)
		case types.VoteEnum_VOTE_INVALID:
			tallies.Invalid.Users = tallies.Invalid.Users.Add(value)
		}
		userVoteSum = userVoteSum.Add(value)
		return false, nil
	})
	if err != nil {
		return err
	}

	if userVoteSum.GT(math.ZeroInt()) {
		totalRatio = totalRatio.Add(Ratio(info.TotalUserTips, userVoteSum))

		userVoteSumDec := math.LegacyNewDecFromInt(userVoteSum)

		scaledSupport = scaledSupport.Add(math.LegacyNewDecFromInt(tallies.ForVotes.Users).Quo(userVoteSumDec))
		scaledAgainst = scaledAgainst.Add(math.LegacyNewDecFromInt(tallies.AgainstVotes.Users).Quo(userVoteSumDec))
		scaledInvalid = scaledInvalid.Add(math.LegacyNewDecFromInt(tallies.Invalid.Users).Quo(userVoteSumDec))
	}

	reporterRatio := math.LegacyZeroDec()
	reporterVoteSum := math.ZeroInt()
	reportersRng := collections.NewPrefixedPairRange[uint64, []byte](id)
	err = k.ReportersGroup.Walk(ctx, reportersRng, func(key collections.Pair[uint64, []byte], value math.Int) (stop bool, err error) {
		vote, err := k.Voter.Get(ctx, key)
		if err != nil {
			return true, err
		}
		switch vote.Vote {
		case types.VoteEnum_VOTE_SUPPORT:
			tallies.ForVotes.Reporters = tallies.ForVotes.Reporters.Add(value)
		case types.VoteEnum_VOTE_AGAINST:
			tallies.AgainstVotes.Reporters = tallies.AgainstVotes.Reporters.Add(value)
		case types.VoteEnum_VOTE_INVALID:
			tallies.Invalid.Reporters = tallies.Invalid.Reporters.Add(value)
		}
		reporterVoteSum = reporterVoteSum.Add(value)
		reporterRatio = reporterRatio.Add(Ratio(info.TotalReporterPower, reporterVoteSum))
		totalRatio = totalRatio.Add(reporterRatio)
		if totalRatio.GTE(math.LegacyNewDec(51)) {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return err
	}
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

	allvoters, err := k.GetVoters(ctx, id)
	if err != nil {
		return err
	}
	tokenHolderVoteSum := math.ZeroInt()
	tokenSupply := k.GetTotalSupply(ctx)
	for _, v := range allvoters {
		voterAddr := v.Key.K2()
		tkHol, err := k.GetAccountBalance(ctx, voterAddr)
		if err != nil {
			return err
		}
		switch v.Value.Vote {
		case types.VoteEnum_VOTE_SUPPORT:
			tallies.ForVotes.TokenHolders = tallies.ForVotes.TokenHolders.Add(tkHol)
		case types.VoteEnum_VOTE_AGAINST:
			tallies.AgainstVotes.TokenHolders = tallies.AgainstVotes.TokenHolders.Add(tkHol)
		case types.VoteEnum_VOTE_INVALID:
			tallies.Invalid.TokenHolders = tallies.Invalid.TokenHolders.Add(tkHol)
		}

		tokenHolderVoteSum = tokenHolderVoteSum.Add(tkHol)
		totalRatio = totalRatio.Add(Ratio(tokenSupply, tokenHolderVoteSum))

		if totalRatio.GTE(math.LegacyNewDec(51)) {
			break
		}
	}
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
		return errors.New("vote period not ended and quorum not reached")
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
