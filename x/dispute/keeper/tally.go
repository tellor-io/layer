package keeper

import (
	"bytes"
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

type AddressInfo struct {
	Address  string
	Reporter string
	Stake    math.Int
}

func (k Keeper) fetchData(ctx context.Context, voters []collections.KeyValue[collections.Pair[uint64, sdk.AccAddress], types.Voter]) ([]AddressInfo, error) {
	info := make([]AddressInfo, 0)
	for _, v := range voters {
		voterAddr := v.Key.K2().String()
		delegation, err := k.reporterKeeper.Delegation(ctx, v.Key.K2())
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return nil, err
			}
			info = append(info, AddressInfo{Address: voterAddr, Reporter: "", Stake: math.ZeroInt()})
			continue
		}
		reporterAcc := sdk.MustAccAddressFromBech32(delegation.Reporter)
		reporter, err := k.reporterKeeper.Reporter(ctx, reporterAcc)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(reporterAcc, v.Key.K2()) {
			info = append(info, AddressInfo{Address: voterAddr, Reporter: voterAddr, Stake: reporter.TotalTokens})
		} else {
			info = append(info, AddressInfo{Address: voterAddr, Reporter: delegation.Reporter, Stake: delegation.Amount})
		}
	}
	return info, nil
}
func (k Keeper) calculateReportingEffectivePowers(voters []AddressInfo) map[string]math.Int {
	effectivePowers := make(map[string]math.Int)
	for _, v := range voters {
		effectivePowers[v.Address] = v.Stake
	}
	for _, v := range voters {
		if v.Address != v.Reporter && v.Reporter != "" {
			effectivePowers[v.Reporter] = effectivePowers[v.Reporter].Sub(v.Stake)
		}
	}
	return effectivePowers

}

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

func (k Keeper) getReportingPower(ctx context.Context, voters []collections.KeyValue[collections.Pair[uint64, sdk.AccAddress], types.Voter]) (map[string]math.Int, error) {
	info, err := k.fetchData(ctx, voters)
	if err != nil {
		return nil, err
	}
	effectivePowers := k.calculateReportingEffectivePowers(info)
	return effectivePowers, nil
}
func (k Keeper) Tallyvote(ctx context.Context, id uint64) error {
	voters, err := k.GetVoters(ctx, id)
	if err != nil {
		return err
	}
	totalReporterPower, err := k.GetTotalReporterPower(ctx)
	if err != nil {
		return err
	}
	totalTips, err := k.GetTotalTips(ctx)
	if err != nil {
		return err
	}
	tokenSupply := k.GetTotalSupply(ctx)

	tallies := types.Tally{
		ForVotes:     k.initVoterClasses(),
		AgainstVotes: k.initVoterClasses(),
		Invalid:      k.initVoterClasses(),
	}
	repP, err := k.getReportingPower(ctx, voters)
	if err != nil {
		return err
	}
	for _, v := range voters {
		voterAddr := v.Key.K2()
		tkHol, err := k.GetAccountBalance(ctx, voterAddr)
		if err != nil {
			return err
		}
		usrTps, err := k.GetUserTips(ctx, voterAddr)
		if err != nil {
			return err
		}
		team := k.IsTeamAddress(ctx, voterAddr)

		voterPower := math.ZeroInt()
		voterPower = voterPower.Add(calculateVotingPower(repP[voterAddr.String()], totalReporterPower))
		voterPower = voterPower.Add(calculateVotingPower(tkHol, tokenSupply))
		voterPower = voterPower.Add(calculateVotingPower(usrTps, totalTips))
		voterPower = voterPower.Add(calculateVotingPower(team, math.OneInt()))

		voter, err := k.Voter.Get(ctx, v.Key)
		if err != nil {
			return err
		}
		voter.VoterPower = voterPower
		err = k.Voter.Set(ctx, v.Key, voter)
		if err != nil {
			return err
		}

		switch v.Value.Vote {
		case types.VoteEnum_VOTE_SUPPORT:
			tallies.ForVotes.Reporters = tallies.ForVotes.Reporters.Add(repP[voterAddr.String()])
			tallies.ForVotes.TokenHolders = tallies.ForVotes.TokenHolders.Add(tkHol)
			tallies.ForVotes.Users = tallies.ForVotes.Users.Add(usrTps)
			tallies.ForVotes.Team = tallies.ForVotes.Team.Add(team)
		case types.VoteEnum_VOTE_AGAINST:
			tallies.AgainstVotes.Reporters = tallies.AgainstVotes.Reporters.Add(repP[voterAddr.String()])
			tallies.AgainstVotes.TokenHolders = tallies.AgainstVotes.TokenHolders.Add(tkHol)
			tallies.AgainstVotes.Users = tallies.AgainstVotes.Users.Add(usrTps)
			tallies.AgainstVotes.Team = tallies.AgainstVotes.Team.Add(team)
		case types.VoteEnum_VOTE_INVALID:
			tallies.Invalid.Reporters = tallies.Invalid.Reporters.Add(repP[voterAddr.String()])
			tallies.Invalid.TokenHolders = tallies.Invalid.TokenHolders.Add(tkHol)
			tallies.Invalid.Users = tallies.Invalid.Users.Add(usrTps)
			tallies.Invalid.Team = tallies.Invalid.Team.Add(team)
		}
	}
	tokenHolderVoteSum := tallies.ForVotes.TokenHolders.Add(tallies.AgainstVotes.TokenHolders).Add(tallies.Invalid.TokenHolders)
	reporterVoteSum := tallies.ForVotes.Reporters.Add(tallies.AgainstVotes.Reporters).Add(tallies.Invalid.Reporters)
	userVoteSum := tallies.ForVotes.Users.Add(tallies.AgainstVotes.Users).Add(tallies.Invalid.Users)
	teamVoteSum := tallies.ForVotes.Team.Add(tallies.AgainstVotes.Team).Add(tallies.Invalid.Team)
	// Prevent zero-division
	if tokenHolderVoteSum.IsZero() {
		tokenHolderVoteSum = math.OneInt()
	}
	if reporterVoteSum.IsZero() {
		reporterVoteSum = math.OneInt()
	}
	if userVoteSum.IsZero() {
		userVoteSum = math.OneInt()
	}
	if teamVoteSum.IsZero() {
		teamVoteSum = math.OneInt()
	}

	// Convert to Dec for precision
	tokenHolderVoteSumDec := math.LegacyNewDecFromInt(tokenHolderVoteSum)
	reporterVoteSumDec := math.LegacyNewDecFromInt(reporterVoteSum)
	userVoteSumDec := math.LegacyNewDecFromInt(userVoteSum)
	teamVoteSumDec := math.LegacyNewDecFromInt(teamVoteSum)

	// Normalize the votes for each group
	forTokenHolders := math.LegacyNewDecFromInt(tallies.ForVotes.TokenHolders).Quo(tokenHolderVoteSumDec)
	forReporters := math.LegacyNewDecFromInt(tallies.ForVotes.Reporters).Quo(reporterVoteSumDec)
	forUsers := math.LegacyNewDecFromInt(tallies.ForVotes.Users).Quo(userVoteSumDec)
	forTeam := math.LegacyNewDecFromInt(tallies.ForVotes.Team).Quo(teamVoteSumDec)

	againstTokenHolders := math.LegacyNewDecFromInt(tallies.AgainstVotes.TokenHolders).Quo(tokenHolderVoteSumDec)
	againstValidators := math.LegacyNewDecFromInt(tallies.AgainstVotes.Reporters).Quo(reporterVoteSumDec)
	againstUsers := math.LegacyNewDecFromInt(tallies.AgainstVotes.Users).Quo(userVoteSumDec)
	againstTeam := math.LegacyNewDecFromInt(tallies.AgainstVotes.Team).Quo(teamVoteSumDec)

	invalidTokenHolders := math.LegacyNewDecFromInt(tallies.Invalid.TokenHolders).Quo(tokenHolderVoteSumDec)
	invalidValidators := math.LegacyNewDecFromInt(tallies.Invalid.Reporters).Quo(reporterVoteSumDec)
	invalidUsers := math.LegacyNewDecFromInt(tallies.Invalid.Users).Quo(userVoteSumDec)
	invalidTeam := math.LegacyNewDecFromInt(tallies.Invalid.Team).Quo(teamVoteSumDec)

	// Sum the normalized votes and divide by number of groups to scale between 0 and 1
	numGroups := math.LegacyNewDec(4)
	scaledSupport := (forTokenHolders.Add(forReporters).Add(forUsers).Add(forTeam)).Quo(numGroups)
	scaledAgainst := (againstTokenHolders.Add(againstValidators).Add(againstUsers).Add(againstTeam)).Quo(numGroups)
	scaledInvalid := (invalidTokenHolders.Add(invalidValidators).Add(invalidUsers).Add(invalidTeam)).Quo(numGroups)
	tokenHolderRatio := ratio(tokenSupply, tokenHolderVoteSum)
	reporterRatio := ratio(totalReporterPower, reporterVoteSum)
	userRatio := ratio(totalTips, userVoteSum)
	teamRatio := ratio(math.OneInt(), teamVoteSum)
	totalRatio := tokenHolderRatio.Add(reporterRatio).Add(userRatio).Add(teamRatio)

	vote, err := k.Votes.Get(ctx, id)
	if err != nil {
		return err
	}
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
		sdkctx := sdk.UnwrapSDKContext(ctx)
		// quorum not reached case
		if vote.VoteEnd.Before(sdkctx.BlockTime()) {
			dispute, err := k.Disputes.Get(ctx, id)
			if err != nil {
				return err
			}
			disputeStatus := types.Unresolved
			// check if rounds have been exhausted or dispute has expired in order to disperse funds
			if dispute.DisputeEndTime.Before(sdkctx.BlockTime()) {
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

func (k Keeper) GetAccountBalance(ctx context.Context, addr sdk.AccAddress) (math.Int, error) {
	bal := k.bankKeeper.GetBalance(ctx, addr, layertypes.BondDenom)
	return bal.Amount, nil
}

func (k Keeper) GetUserTips(ctx context.Context, voterAddr sdk.AccAddress) (math.Int, error) {
	userTips, err := k.oracleKeeper.GetUserTips(ctx, voterAddr)
	if err != nil {
		return math.Int{}, err
	}
	return userTips.Total, nil
}

func (k Keeper) IsTeamAddress(ctx context.Context, voter sdk.AccAddress) math.Int {
	params, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}

	if !bytes.Equal(voter, sdk.MustAccAddressFromBech32(params.TeamAddress)) {
		return math.ZeroInt()
	}
	return math.NewInt(1)
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

// Get total number of tips
func (k Keeper) GetTotalTips(ctx context.Context) (math.Int, error) {
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
