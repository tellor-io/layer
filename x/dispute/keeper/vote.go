package keeper

import (
	"bytes"
	"context"
	"errors"

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
	voterIsReporter := bytes.Equal(voter, reporter)
	reporterHasVoted, err := k.Voter.Has(ctx, collections.Join(id, reporter.Bytes()))
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
		// update reporter's power record for reward calculation
		reporterVote.ReporterPower = reporterVote.ReporterPower.Sub(selectorTokens)
		err = k.Voter.Set(ctx, collections.Join(id, reporter.Bytes()), reporterVote)
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

func (k Keeper) SetTokenholderVote(ctx context.Context, id uint64, voter sdk.AccAddress, blockNumber uint64, choice types.VoteEnum) (math.Int, error) {
	// get token balance
	tokenBalance, err := k.GetAccountBalance(ctx, voter)
	if err != nil {
		return math.Int{}, err
	}
	// get tokens delegated to a reporter
	selectorTokens, err := k.reporterKeeper.GetDelegatorTokensAtBlock(ctx, voter, blockNumber)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return math.Int{}, err
		}
		selectorTokens = math.ZeroInt()
	}
	tokenBalance = tokenBalance.Add(selectorTokens)

	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return math.Int{}, err
		}
		voteCounts = types.StakeholderVoteCounts{}
	}
	if choice == types.VoteEnum_VOTE_SUPPORT {
		voteCounts.Tokenholders.Support += tokenBalance.Uint64()
	} else if choice == types.VoteEnum_VOTE_AGAINST {
		voteCounts.Tokenholders.Against += tokenBalance.Uint64()
	} else {
		voteCounts.Tokenholders.Invalid += tokenBalance.Uint64()
	}
	return tokenBalance, k.VoteCountsByGroup.Set(ctx, id, voteCounts)
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
