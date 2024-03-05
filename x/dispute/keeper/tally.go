package keeper

import (
	"errors"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
	reptypes "github.com/tellor-io/layer/x/reporter/types"
)

// Add invalid vote type and return all fees to the paying parties
func (k Keeper) TallyVote(ctx sdk.Context, id uint64) error {
	dispute, err := k.GetDisputeById(ctx, id)
	if err != nil {
		return err
	}
	tally, err := k.GetTally(ctx, id)
	if err != nil {
		return err
	}
	vote, err := k.GetVote(ctx, id)
	if err != nil {
		return err
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
			case len(vote.Voters) == 0:
				if err := k.SetDisputeStatus(ctx, id, disputeStatus); err != nil {
					return err
				}
				if err := k.SetVoteResult(ctx, id, types.VoteResult_NO_QUORUM_MAJORITY_INVALID); err != nil {
					return err
				}
			default:
				return errors.New("no quorum majority") // TODO: log only?
			}
			return nil
		}
	}
	return nil
}

func (k Keeper) GetTally(ctx sdk.Context, id uint64) (*types.Tally, error) {
	store := k.voteStore(ctx)
	tallyBytes := store.Get(types.TallyKeyPrefix(id))
	var tallies types.Tally
	if err := k.cdc.Unmarshal(tallyBytes, &tallies); err != nil {
		return nil, err
	}
	// for some reason the proto files don't initialize the tallies when nullable = false
	if tallies.ForVotes == nil {
		tallies.ForVotes = k.initVoterClasses()
		tallies.AgainstVotes = k.initVoterClasses()
		tallies.Invalid = k.initVoterClasses()
	}

	return &tallies, nil
}

// Set tally numbers
func (k Keeper) SetTally(ctx sdk.Context, id uint64, voteFor types.VoteEnum, voter string) error {
	tallies, err := k.GetTally(ctx, id)
	if err != nil {
		return err
	}
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
		return errors.New("invalid vote type")
	}

	k.SetVoterTally(ctx, id, tallies)
	k.SetVoterPower(ctx, sdk.MustAccAddressFromBech32(voter), voterPower)
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
	return reporter.TotalTokens.Quo(sdk.DefaultPowerReduction), nil
}

func (k Keeper) GetAccountBalance(ctx sdk.Context, voter string) (math.Int, error) {
	addr, err := sdk.AccAddressFromBech32(voter)
	if err != nil {
		return math.Int{}, err
	}
	bal := k.bankKeeper.GetBalance(ctx, addr, layer.BondDenom)
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
	return k.bankKeeper.GetSupply(ctx, layer.BondDenom).Amount
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

func (k Keeper) CalculateVoterShare(ctx sdk.Context, voters []string, totalTokens math.Int) (map[string]math.Int, math.Int) {
	// remove duplicates from voters list
	seen := make(map[string]bool)
	var uniqueVoters []string
	for _, voter := range voters {
		if !seen[voter] {
			seen[voter] = true
			uniqueVoters = append(uniqueVoters, voter)
		}
	}
	totalPower := math.ZeroInt()
	powers := make(map[string]math.Int)
	for _, voter := range uniqueVoters {
		voterShare := k.GetVoterPower(ctx, sdk.MustAccAddressFromBech32(voter))
		totalPower = totalPower.Add(voterShare)
		powers[voter] = voterShare
	}

	// Calculate and allocate tokens based on each person's share of the total power
	tokenDistribution := make(map[string]math.Int)
	scalingFactor := math.NewInt(1_000_000) //TODO: use sdk.DefaultPowerReduction
	totalShare := math.ZeroInt()
	for voter, power := range powers {
		share := power.Mul(scalingFactor).Quo(totalPower)
		tokens := share.Mul(totalTokens).Quo(scalingFactor)
		tokenDistribution[voter] = tokens
		totalShare = totalShare.Add(tokens)
	}
	burnedRemainder := math.ZeroInt()
	if totalTokens.GT(totalShare) {
		burnedRemainder = totalTokens.Sub(totalShare)
	}

	return tokenDistribution, burnedRemainder
}
