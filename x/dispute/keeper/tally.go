package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k Keeper) GetVoters(ctx context.Context, id uint64) (
	[]collections.KeyValue[collections.Pair[uint64, sdk.AccAddress], types.Voter], error) {
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

// Get total reporter power
func (k Keeper) GetTotalReporterPower(ctx context.Context) (math.Int, error) {
	tp, err := k.reporterKeeper.TotalReporterPower(ctx)
	if err != nil {
		return math.Int{}, err
	}
	return tp, nil
}

func ratio(total, part math.Int) math.LegacyDec {
	if total.IsZero() {
		return math.LegacyZeroDec()
	}
	total = total.MulRaw(4)
	ratio := math.LegacyNewDecFromInt(part).Quo(math.LegacyNewDecFromInt(total))
	return ratio.MulInt64(100)
}

func calculateVotingPower(n, d math.Int) math.Int {
	if n.IsZero() || d.IsZero() {
		return math.ZeroInt()
	}
	scalingFactor := math.NewInt(1_000_000)
	return n.Mul(scalingFactor).Quo(d).MulRaw(25_000_000).Quo(scalingFactor)
}

func (k Keeper) CalculateVoterShare(ctx context.Context, voters []VoterInfo, totalTokens math.Int) ([]VoterInfo, math.Int) {
	totalPower := math.ZeroInt()
	for _, voter := range voters {
		totalPower = totalPower.Add(voter.Power)
	}

	scalingFactor := layertypes.PowerReduction
	totalShare := math.ZeroInt()
	for i, v := range voters {
		share := v.Power.Mul(scalingFactor).Quo(totalPower)
		tokens := share.Mul(totalTokens).Quo(scalingFactor)
		voters[i].Share = tokens
		totalShare = totalShare.Add(tokens)
	}
	burnedRemainder := math.ZeroInt()
	if totalTokens.GT(totalShare) {
		burnedRemainder = totalTokens.Sub(totalShare)
	}
	return voters, burnedRemainder
}

func (k Keeper) Tallyvote(ctx context.Context, id uint64) error {
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
	totalRatio := math.LegacyZeroDec()
	// init tallies
	tallies := types.Tally{
		ForVotes:     k.initVoterClasses(),
		AgainstVotes: k.initVoterClasses(),
		Invalid:      k.initVoterClasses(),
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
		vote, err := k.Voter.Get(ctx, collections.Join(id, teamAddr))
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
	userRng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](id)
	err = k.UsersGroup.Walk(ctx, userRng, func(key collections.Pair[uint64, sdk.AccAddress], value math.Int) (stop bool, err error) {
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

	totaltips, err := k.oracleKeeper.GetTotalTipsAtBlock(ctx, dispute.BlockNumber)
	if err != nil {
		return err
	}

	if userVoteSum.GT(math.ZeroInt()) {
		totalRatio = totalRatio.Add(ratio(totaltips, userVoteSum))

		userVoteSumDec := math.LegacyNewDecFromInt(userVoteSum)

		scaledSupport = scaledSupport.Add(math.LegacyNewDecFromInt(tallies.ForVotes.Users).Quo(userVoteSumDec))
		scaledAgainst = scaledAgainst.Add(math.LegacyNewDecFromInt(tallies.AgainstVotes.Users).Quo(userVoteSumDec))
		scaledInvalid = scaledInvalid.Add(math.LegacyNewDecFromInt(tallies.Invalid.Users).Quo(userVoteSumDec))
	}

	totalReporterPower, err := k.GetTotalReporterPower(ctx)
	if err != nil {
		return err
	}
	reporterRatio := math.LegacyZeroDec()
	reporterVoteSum := math.ZeroInt()
	reportersRng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](id)
	err = k.ReportersGroup.Walk(ctx, reportersRng, func(key collections.Pair[uint64, sdk.AccAddress], value math.Int) (stop bool, err error) {
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
		reporterRatio = reporterRatio.Add(ratio(totalReporterPower, reporterVoteSum))
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
		return k.UpdateDispute(ctx, id, types.Resolved, scaledSupport, scaledAgainst, scaledInvalid)
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
		totalRatio = totalRatio.Add(ratio(tokenSupply, tokenHolderVoteSum))

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

		return k.UpdateDispute(ctx, id, types.Resolved, scaledSupport, scaledAgainst, scaledInvalid)
	}
	sdkctx := sdk.UnwrapSDKContext(ctx)
	// quorum not reached case
	if vote.VoteEnd.Before(sdkctx.BlockTime()) {
		fmt.Println("quorum not reached")
		disputeStatus := types.Unresolved
		// check if rounds have been exhausted or dispute has expired in order to disperse funds
		if dispute.DisputeEndTime.Before(sdkctx.BlockTime()) {
			disputeStatus = types.Resolved
		}
		if len(allvoters) == 0 {
			if err := k.SetDisputeStatus(ctx, id, disputeStatus); err != nil {
				return err
			}
			if err := k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_INVALID); err != nil {
				return err
			}
			return nil
		}
		return k.UpdateDispute(ctx, id, disputeStatus, scaledSupport, scaledAgainst, scaledInvalid)
	} else {
		return errors.New("vote period not ended and quorum not reached")
	}

}

func (k Keeper) UpdateDispute(
	ctx context.Context,
	id uint64,
	status types.DisputeStatus,
	scaledSupport, scaledAgainst, scaledInvalid math.LegacyDec) error {
	switch {
	case scaledSupport.GT(scaledAgainst) && scaledSupport.GT(scaledInvalid):
		if err := k.SetDisputeStatus(ctx, id, status); err != nil {
			return err
		}
		if err := k.SetVoteResult(ctx, id, types.VoteResult_SUPPORT); err != nil {
			return err
		}
	case scaledAgainst.GT(scaledSupport) && scaledAgainst.GT(scaledInvalid):
		if err := k.SetDisputeStatus(ctx, id, status); err != nil {
			return err
		}
		if err := k.SetVoteResult(ctx, id, types.VoteResult_AGAINST); err != nil {
			return err
		}
	case scaledInvalid.GT(scaledSupport) && scaledInvalid.GT(scaledAgainst):
		if err := k.SetDisputeStatus(ctx, id, status); err != nil {
			return err
		}
		if err := k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_INVALID); err != nil {
			return err
		}
	default:
	}
	return nil
}
