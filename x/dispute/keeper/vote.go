package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) InitVoterClasses() *types.VoterClasses {
	return &types.VoterClasses{
		Reporters:    math.ZeroInt(),
		TokenHolders: math.ZeroInt(),
		Users:        math.ZeroInt(),
		Team:         math.ZeroInt(),
	}
}

// Set vote start info for a dispute
func (k Keeper) SetStartVote(ctx sdk.Context, id uint64) error {
	vote := types.Vote{
		Id:        id,
		VoteStart: ctx.BlockTime(),
		VoteEnd:   ctx.BlockTime().Add(TWO_DAYS),
	}
	return k.Votes.Set(ctx, id, vote)
}

func (k Keeper) TeamVote(ctx context.Context, id uint64) (math.Int, error) {
	teamTally := math.ZeroInt()
	voted, err := k.TeamVoter.Has(ctx, id)
	if err != nil {
		return math.Int{}, err
	}
	if voted {
		teamTally = math.OneInt()
	}

	return teamTally, nil
}

func (k Keeper) GetTeamAddress(ctx context.Context) (sdk.AccAddress, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	return params.TeamAddress, nil
}

func (k Keeper) SetTeamVote(ctx context.Context, id uint64, voter sdk.AccAddress, choice types.VoteEnum) (math.Int, error) {
	teamAddr, err := k.GetTeamAddress(ctx)
	if err != nil {
		return math.Int{}, err
	}

	if bytes.Equal(voter, teamAddr) {

		voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return math.Int{}, err
			}
			voteCounts = types.StakeholderVoteCounts{}
		}
		if choice == types.VoteEnum_VOTE_SUPPORT {
			voteCounts.Team.Support = 1
		} else if choice == types.VoteEnum_VOTE_AGAINST {
			voteCounts.Team.Against = 1
		} else {
			voteCounts.Team.Invalid = 1
		}
		err = k.VoteCountsByGroup.Set(ctx, id, voteCounts)
		if err != nil {
			return math.Int{}, err
		}
		return math.NewInt(25000000), k.TeamVoter.Set(ctx, id, true)
	}
	return math.ZeroInt(), nil
}

func (k Keeper) GetUserTotalTips(ctx context.Context, voter sdk.AccAddress, blockNumber uint64) (math.Int, error) {
	tips, err := k.oracleKeeper.GetTipsAtBlockForTipper(ctx, blockNumber, voter)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return math.Int{}, err
		}
		return math.ZeroInt(), nil
	}
	return tips, nil
}

func (k Keeper) SetVoterTips(ctx context.Context, id uint64, voter sdk.AccAddress, blockNumber uint64, choice types.VoteEnum) (math.Int, error) {

	tips, err := k.GetUserTotalTips(ctx, voter, blockNumber)
	if err != nil {
		return math.Int{}, err
	}
	if !tips.IsZero() {
		voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return math.Int{}, err
			}
			voteCounts = types.StakeholderVoteCounts{}
		}
		if choice == types.VoteEnum_VOTE_SUPPORT {
			voteCounts.Users.Support += tips.Uint64()
		} else if choice == types.VoteEnum_VOTE_AGAINST {
			voteCounts.Users.Against += tips.Uint64()
		} else {
			voteCounts.Users.Invalid += tips.Uint64()
		}
		err = k.VoteCountsByGroup.Set(ctx, id, voteCounts)
		if err != nil {
			return math.Int{}, err
		}
		return tips, k.UsersGroup.Set(ctx, collections.Join(id, voter.Bytes()), tips)
	}
	return math.ZeroInt(), nil
}

func (k Keeper) SetVoterReporterStake(ctx context.Context, id uint64, voter sdk.AccAddress, blockNumber uint64, choice types.VoteEnum) (math.Int, error) {
	// get delegation
	delegation, err := k.reporterKeeper.Delegation(ctx, voter)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return math.ZeroInt(), nil
		}
		return math.Int{}, err
	}
	reporter := sdk.AccAddress(delegation.Reporter)
	// Check if reporter has voted. If not, store voter tokens either full if reporter or delegation amount
	// this amount the amount to reduce from reporter so total amount of delegators that voted
	delegatorTokensVoted, err := k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), id))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return math.Int{}, err
		}
		if bytes.Equal(reporter, voter) {
			// if voter is reporter AND reporter votes before any selectors, we end up here
			// voter is reporter, get reporter tokens
			reporterTokens, err := k.reporterKeeper.GetReporterTokensAtBlock(ctx, reporter, blockNumber)
			if err != nil {
				return math.Int{}, err
			}
			err = k.AddReporterVoteCount(ctx, id, reporterTokens.Uint64(), choice)
			if err != nil {
				return math.Int{}, err
			}
			return reporterTokens, k.ReportersGroup.Set(ctx, collections.Join(id, voter.Bytes()), reporterTokens)
		}
		// if voter is not reporter AND voter votes before any of reporter's other selectors, we end up here
		// shouldn't this be reporterKeeper.GetDelegatorTokensAtBlock(ctx, voter, blockNumber)?
		voterDelegatedAmt, err := k.reporterKeeper.GetDelegatorTokensAtBlock(ctx, voter, blockNumber)
		if err != nil {
			return math.Int{}, err
		}
		exists, err := k.ReportersGroup.Has(ctx, collections.Join(id, reporter.Bytes()))
		if err != nil {
			return math.Int{}, err
		}
		fmt.Println("exists: ", exists)
		// if reporter has voted before, get reporter tokens and reduce the amount
		if exists {
			reporterTokens, err := k.ReportersGroup.Get(ctx, collections.Join(id, reporter.Bytes()))
			if err != nil {
				return math.Int{}, err
			}
			reporterTokens = reporterTokens.Sub(voterDelegatedAmt)
			reporterVote, err := k.Voter.Get(ctx, collections.Join(id, reporter.Bytes()))
			if err != nil {
				return math.Int{}, err
			}
			// isn't this wrong? we're subtracting a delegator's token amount from voterPower (x% * 25,000,000)
			reporterVote.VoterPower = reporterVote.VoterPower.Sub(voterDelegatedAmt)
			if err := k.Voter.Set(ctx, collections.Join(id, reporter.Bytes()), reporterVote); err != nil {
				return math.Int{}, err
			}
			if err := k.ReportersGroup.Set(ctx, collections.Join(id, reporter.Bytes()), reporterTokens); err != nil {
				return math.Int{}, err
			}

			// adjust reporter's vote count by voter's delegated token amount
			err = k.SubtractReporterVoteCount(ctx, id, voterDelegatedAmt.Uint64(), reporterVote.Vote)
			if err != nil {
				return math.Int{}, err
			}
			// add voter's delegated token amount to reporter's vote count
			err = k.AddReporterVoteCount(ctx, id, voterDelegatedAmt.Uint64(), choice)
			if err != nil {
				return math.Int{}, err
			}
			return voterDelegatedAmt, k.ReportersGroup.Set(ctx, collections.Join(id, voter.Bytes()), voterDelegatedAmt)
		}
		// if reporter has not voted before, set reporter tokens and reporter delegation amount
		if err := k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(reporter.Bytes(), id), voterDelegatedAmt); err != nil {
			return math.Int{}, err
		}
		err = k.AddReporterVoteCount(ctx, id, voterDelegatedAmt.Uint64(), choice)
		if err != nil {
			return math.Int{}, err
		}
		return voterDelegatedAmt, k.ReportersGroup.Set(ctx, collections.Join(id, voter.Bytes()), voterDelegatedAmt)
	}
	// at least one non-reporter selector has voted before
	// if voter is reporter, then get reporter tokens at block and reduce by amount that has voted already
	if bytes.Equal(reporter, voter) {
		reporterTokens, err := k.reporterKeeper.GetReporterTokensAtBlock(ctx, reporter, blockNumber)
		if err != nil {
			return math.Int{}, err
		}
		reporterVoteWeight := reporterTokens.Sub(delegatorTokensVoted)
		err = k.AddReporterVoteCount(ctx, id, reporterVoteWeight.Uint64(), choice)
		if err != nil {
			return math.Int{}, err
		}
		return reporterTokens.Sub(delegatorTokensVoted), k.ReportersGroup.Set(ctx, collections.Join(id, voter.Bytes()), reporterTokens.Sub(delegatorTokensVoted))
	} else {
		// voter is not reporter
		// shouldn't this be reporterKeeper.GetDelegatorTokensAtBlock(ctx, voter, blockNumber)?
		amt, err := k.reporterKeeper.GetDelegatorTokensAtBlock(ctx, voter, blockNumber)
		if err != nil {
			return math.Int{}, err
		}
		if err := k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(reporter.Bytes(), id), delegatorTokensVoted.Add(amt)); err != nil {
			return math.Int{}, err
		}
		// if this is the second selector to vote, and the reporter has already voted, shouldn't we subtract from the reporter's vote count here?
		// check whether reporter has voted before
		exists, err := k.Voter.Has(ctx, collections.Join(id, reporter.Bytes()))
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return math.Int{}, err
			}
		}
	}
	// why return zero in case of (selector, selector)?
	return math.ZeroInt(), nil
}

func (k Keeper) SetVoterReporterStakeREDO(ctx context.Context, id uint64, voter sdk.AccAddress, blockNumber uint64, choice types.VoteEnum) (math.Int, error) {
	// get delegation
	delegation, err := k.reporterKeeper.Delegation(ctx, voter)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return math.ZeroInt(), nil
		}
		return math.Int{}, err
	}
	reporter := sdk.AccAddress(delegation.Reporter)
	voterIsReporter := bytes.Equal(voter, reporter)
	reporterHasVoted, err := k.ReportersGroup.Has(ctx, collections.Join(id, reporter.Bytes()))
	if err != nil {
		return math.Int{}, err
	}
	// voter is reporter
	if voterIsReporter {
		reporterTokens, err := k.reporterKeeper.GetReporterTokensAtBlock(ctx, reporter, blockNumber)
		if err != nil {
			return math.Int{}, err
		}
		tokensVotedBefore, err := k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), id))
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return math.Int{}, err
			}
			tokensVotedBefore = math.ZeroInt()
		}
		reporterTokens = reporterTokens.Sub(tokensVotedBefore)
		return reporterTokens, k.AddReporterVoteCount(ctx, id, reporterTokens.Uint64(), choice)
	}
	// voter is non-reporter selector
	selectorTokens, err := k.reporterKeeper.GetDelegatorTokensAtBlock(ctx, voter, blockNumber)
	if err != nil {
		return math.Int{}, err
	}
	if reporterHasVoted {
		reporterVote, err := k.Voter.Get(ctx, collections.Join(id, reporter.Bytes()))
		if err != nil {
			return math.Int{}, err
		}
		err = k.SubtractReporterVoteCount(ctx, id, selectorTokens.Uint64(), reporterVote.Vote)
		if err != nil {
			return math.Int{}, err
		}
		return selectorTokens, k.AddReporterVoteCount(ctx, id, selectorTokens.Uint64(), choice)
	}
	delegatorTokensVoted, err := k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), id))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return math.Int{}, err
		}
		delegatorTokensVoted = math.ZeroInt()
	}
	delegatorTokensVoted = delegatorTokensVoted.Add(selectorTokens)
	err = k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(reporter.Bytes(), id), delegatorTokensVoted)
	if err != nil {
		return math.Int{}, err
	}
	return selectorTokens, k.AddReporterVoteCount(ctx, id, selectorTokens.Uint64(), choice)
}

func (k Keeper) AddReporterVoteCount(ctx context.Context, id uint64, amount uint64, choice types.VoteEnum) error {
	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		voteCounts = types.StakeholderVoteCounts{}
	}
	if choice == types.VoteEnum_VOTE_SUPPORT {
		voteCounts.Reporters.Support += amount
	} else if choice == types.VoteEnum_VOTE_AGAINST {
		voteCounts.Reporters.Against += amount
	} else {
		voteCounts.Reporters.Invalid += amount
	}
	return k.VoteCountsByGroup.Set(ctx, id, voteCounts)
}

func (k Keeper) SubtractReporterVoteCount(ctx context.Context, id uint64, amount uint64, choice types.VoteEnum) error {
	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	if err != nil {
		return err
	}
	if choice == types.VoteEnum_VOTE_SUPPORT {
		voteCounts.Reporters.Support -= amount
	} else if choice == types.VoteEnum_VOTE_AGAINST {
		voteCounts.Reporters.Against -= amount
	} else {
		voteCounts.Reporters.Invalid -= amount
	}
	return k.VoteCountsByGroup.Set(ctx, id, voteCounts)
}
