package keeper

import (
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
	reptypes "github.com/tellor-io/layer/x/reporter/types"
)

// Add invalid vote type and return all fees to the paying parties
func (k Keeper) TallyVote(ctx sdk.Context, id uint64) error {
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return err
	}
	vote, err := k.Votes.Get(ctx, id)
	if err != nil {
		return err
	}
	tally := vote.Tally

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
	tokenHolderVoteSumDec := math.LegacyNewDecFromInt(tokenHolderVoteSum)
	validatorVoteSumDec := math.LegacyNewDecFromInt(validatorVoteSum)
	userVoteSumDec := math.LegacyNewDecFromInt(userVoteSum)
	teamVoteSumDec := math.LegacyNewDecFromInt(teamVoteSum)

	// Normalize the votes for each group
	forTokenHolders := math.LegacyNewDecFromInt(tally.ForVotes.TokenHolders).Quo(tokenHolderVoteSumDec)
	forValidators := math.LegacyNewDecFromInt(tally.ForVotes.Validators).Quo(validatorVoteSumDec)
	forUsers := math.LegacyNewDecFromInt(tally.ForVotes.Users).Quo(userVoteSumDec)
	forTeam := math.LegacyNewDecFromInt(tally.ForVotes.Team).Quo(teamVoteSumDec)

	againstTokenHolders := math.LegacyNewDecFromInt(tally.AgainstVotes.TokenHolders).Quo(tokenHolderVoteSumDec)
	againstValidators := math.LegacyNewDecFromInt(tally.AgainstVotes.Validators).Quo(validatorVoteSumDec)
	againstUsers := math.LegacyNewDecFromInt(tally.AgainstVotes.Users).Quo(userVoteSumDec)
	againstTeam := math.LegacyNewDecFromInt(tally.AgainstVotes.Team).Quo(teamVoteSumDec)

	invalidTokenHolders := math.LegacyNewDecFromInt(tally.Invalid.TokenHolders).Quo(tokenHolderVoteSumDec)
	invalidValidators := math.LegacyNewDecFromInt(tally.Invalid.Validators).Quo(validatorVoteSumDec)
	invalidUsers := math.LegacyNewDecFromInt(tally.Invalid.Users).Quo(userVoteSumDec)
	invalidTeam := math.LegacyNewDecFromInt(tally.Invalid.Team).Quo(teamVoteSumDec)

	// Sum the normalized votes and divide by number of groups to scale between 0 and 1
	numGroups := math.LegacyNewDec(4)
	scaledSupport := (forTokenHolders.Add(forValidators).Add(forUsers).Add(forTeam)).Quo(numGroups)
	scaledAgainst := (againstTokenHolders.Add(againstValidators).Add(againstUsers).Add(againstTeam)).Quo(numGroups)
	scaledInvalid := (invalidTokenHolders.Add(invalidValidators).Add(invalidUsers).Add(invalidTeam)).Quo(numGroups)

	tokenHolderRatio := ratio(k.GetTotalSupply(ctx), tokenHolderVoteSum)
	totalPower, err := k.GetTotalReporterPower(ctx)
	if err != nil {
		return err
	}
	validatorRatio := ratio(totalPower, validatorVoteSum)
	totalTips, _ := k.GetTotalTips(ctx) // TODO: handle err
	userRatio := ratio(totalTips, userVoteSum)
	teamRatio := ratio(math.NewInt(1), teamVoteSum)
	totalRatio := tokenHolderRatio.Add(validatorRatio).Add(userRatio).Add(teamRatio)

	if vote.VoteResult == types.VoteResult_NO_TALLY {
		// quorum reached case
		if totalRatio.GTE(math.LegacyNewDec(51)) {
			fmt.Println("quorum reached")
			switch {
			case scaledSupport.GT(scaledAgainst) && scaledSupport.GT(scaledInvalid):
				if err := k.SetDisputeStatus(ctx, id, types.Resolved); err != nil {
					return err
				}
				if err := k.SetVoteResult(ctx, id, types.VoteResult_SUPPORT); err != nil {
					return err
				}
			case scaledAgainst.GT(scaledSupport) && scaledAgainst.GT(scaledInvalid):
				if err := k.SetDisputeStatus(ctx, id, types.Resolved); err != nil {
					return err
				}
				if err := k.SetVoteResult(ctx, id, types.VoteResult_AGAINST); err != nil {
					return err
				}
			case scaledInvalid.GT(scaledSupport) && scaledInvalid.GT(scaledAgainst):
				if err := k.SetDisputeStatus(ctx, id, types.Resolved); err != nil {
					return err
				}
				if err := k.SetVoteResult(ctx, id, types.VoteResult_INVALID); err != nil {
					return err
				}
			default:
			}
			return nil
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
				if err := k.SetDisputeStatus(ctx, id, disputeStatus); err != nil {
					return err
				}
				if err := k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT); err != nil {
					return err
				}
			case scaledAgainst.GT(scaledSupport) && scaledAgainst.GT(scaledInvalid):
				if err := k.SetDisputeStatus(ctx, id, disputeStatus); err != nil {
					return err
				}
				if err := k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST); err != nil {
					return err
				}
			case scaledInvalid.GT(scaledSupport) && scaledInvalid.GT(scaledAgainst):
				if err := k.SetDisputeStatus(ctx, id, disputeStatus); err != nil {
					return err
				}
				if err := k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_INVALID); err != nil {
					return err
				}
			default:
				iter, err := k.Voter.Indexes.VotersById.MatchExact(ctx, id)
				if err != nil {
					return err
				}
				voters, err := iter.PrimaryKeys()
				if err != nil {
					return err
				}
				if len(voters) == 0 {
					if err := k.SetDisputeStatus(ctx, id, disputeStatus); err != nil {
						return err
					}
					if err := k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_INVALID); err != nil {
						return err
					}
					return nil
				}
				return errors.New("no quorum majority")
			}
			return nil
		}
	}
	return nil
}

// Set tally numbers
func (k Keeper) SetTally(ctx sdk.Context, voterAcc sdk.AccAddress, ballot types.VoteEnum, vote types.Vote) error {
	tallies := vote.Tally
	voter := voterAcc.String()

	valP, err := k.GetReporterPower(ctx, voter)
	if err != nil {
		return err
	}
	tkHol, err := k.GetAccountBalance(ctx, voter)
	if err != nil {
		return err
	}
	usrTps, err := k.GetUserTips(ctx, voter)
	if err != nil {
		return err
	}
	team := k.IsTeamAddress(ctx, voter)

	voterPower := math.ZeroInt()
	tp, err := k.GetTotalReporterPower(ctx)
	if err != nil {
		return err
	}
	voterPower = voterPower.Add(calculateVotingPower(valP, tp))
	voterPower = voterPower.Add(calculateVotingPower(tkHol, k.GetTotalSupply(ctx)))

	totalTips, err := k.GetTotalTips(ctx)
	if err != nil {
		return err
	}
	voterPower = voterPower.Add(calculateVotingPower(usrTps, totalTips))
	voterPower = voterPower.Add(calculateVotingPower(team, math.NewInt(1)))

	switch ballot {
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
		return errors.New("invalid vote type")
	}

	if err := k.Votes.Set(ctx, vote.Id, vote); err != nil {
		return err
	}
	voterVote := types.Voter{
		Vote:       ballot,
		VoterPower: voterPower,
	}
	if err := k.Voter.Set(ctx, collections.Join(vote.Id, voterAcc), voterVote); err != nil {
		return err
	}
	return nil
}

func (k Keeper) GetReporterPower(ctx sdk.Context, voter string) (math.Int, error) {
	reporterAddr, err := sdk.AccAddressFromBech32(voter)
	if err != nil {
		return math.Int{}, err
	}
	reporter, err := k.reporterKeeper.Reporter(ctx, reporterAddr)
	if err != nil {
		if errors.Is(err, reptypes.ErrReporterDoesNotExist) {
			return math.ZeroInt(), nil
		}
		return math.Int{}, err
	}
	return reporter.TotalTokens.Quo(layertypes.PowerReduction), nil
}

func (k Keeper) GetAccountBalance(ctx sdk.Context, voter string) (math.Int, error) {
	addr, err := sdk.AccAddressFromBech32(voter)
	if err != nil {
		return math.Int{}, err
	}
	bal := k.bankKeeper.GetBalance(ctx, addr, layertypes.BondDenom)
	return bal.Amount, nil
}

func (k Keeper) GetUserTips(ctx sdk.Context, voter string) (math.Int, error) {
	voterAddr, err := sdk.AccAddressFromBech32(voter)
	if err != nil {
		return math.Int{}, err
	}
	userTips, err := k.oracleKeeper.GetUserTips(ctx, voterAddr)
	if err != nil {
		return math.Int{}, err
	}
	return userTips.Total, nil
}

func (k Keeper) IsTeamAddress(ctx sdk.Context, voter string) math.Int {
	if voter != types.TeamAddress {
		return math.ZeroInt()
	}
	return math.NewInt(1)
}

// Get total trb supply
func (k Keeper) GetTotalSupply(ctx sdk.Context) math.Int {
	return k.bankKeeper.GetSupply(ctx, layertypes.BondDenom).Amount
}

// Get total reporter power
func (k Keeper) GetTotalReporterPower(ctx sdk.Context) (math.Int, error) {
	tp, err := k.reporterKeeper.TotalReporterPower(ctx)
	if err != nil {
		return math.Int{}, err
	}
	return tp, nil
}

// Get total number of tips
func (k Keeper) GetTotalTips(ctx sdk.Context) (math.Int, error) {
	return k.oracleKeeper.GetTotalTips(ctx)
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

func (k Keeper) CalculateVoterShare(ctx sdk.Context, voters []VoterInfo, totalTokens math.Int) ([]VoterInfo, math.Int) {
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
