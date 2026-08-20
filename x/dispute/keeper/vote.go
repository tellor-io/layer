package keeper

import (
	"bytes"
	"context"
	"errors"

	"github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) InitVoterClasses() *types.VoterClasses {
	return &types.VoterClasses{
		Reporters: math.ZeroInt(),
		Users:     math.ZeroInt(),
		Team:      math.ZeroInt(),
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

func (k Keeper) GetTeamAddress(ctx context.Context) (sdk.AccAddress, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	return params.TeamAddress, nil
}

func (k Keeper) SetTeamVote(ctx context.Context, id uint64, voter sdk.AccAddress, choice types.VoteEnum, oldVote *types.Voter) (math.Int, error) {
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
		switch choice {
		case types.VoteEnum_VOTE_SUPPORT:
			voteCounts.Team.Support = 1
		case types.VoteEnum_VOTE_AGAINST:
			voteCounts.Team.Against = 1
		default:
			voteCounts.Team.Invalid = 1
		}
		if oldVote != nil {
			if oldVote.Vote != choice {
				switch oldVote.Vote {
				case types.VoteEnum_VOTE_SUPPORT:
					voteCounts.Team.Support = 0
				case types.VoteEnum_VOTE_AGAINST:
					voteCounts.Team.Against = 0
				case types.VoteEnum_VOTE_INVALID:
					voteCounts.Team.Invalid = 0
				}
			}
		}
		err = k.VoteCountsByGroup.Set(ctx, id, voteCounts)
		if err != nil {
			return math.Int{}, err
		}
		// return doesnt get used in dispute calculations
		// just gets set in Voter collection as the team's voter.VoterPower which doesnt matter for tally calculations
		power := math.NewInt(100000000).Quo(math.NewInt(3))
		return power, nil
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

func (k Keeper) SetVoterTips(ctx context.Context, id uint64, voter sdk.AccAddress, blockNumber uint64, choice types.VoteEnum, oldVote *types.Voter) (math.Int, error) {
	tips, err := k.GetUserTotalTips(ctx, voter, blockNumber)
	if err != nil {
		return math.Int{}, err
	}
	if !tips.IsZero() {
		voteCounts, err := k.getVoteCounts(ctx, id)
		if err != nil {
			return math.Int{}, err
		}
		if err := voteCounts.Users.Add(choice, tips.Uint64()); err != nil {
			return math.Int{}, err
		}
		if oldVote != nil && oldVote.Vote != choice {
			if err := voteCounts.Users.Subtract(oldVote.Vote, tips.Uint64()); err != nil {
				return math.Int{}, err
			}
		}
		if err := k.VoteCountsByGroup.Set(ctx, id, voteCounts); err != nil {
			return math.Int{}, err
		}
		return tips, nil
	}
	return math.ZeroInt(), nil
}

// selectorStakeAtBlock resolves who owns the selector's dispute-block tokens.
// Prefer the evidence reporter's snapshot when it still holds the stake after a switch.
func (k Keeper) selectorStakeAtBlock(
	ctx context.Context,
	voter, liveReporter, evidenceReporter sdk.AccAddress,
	blockNumber uint64,
) (owner sdk.AccAddress, tokens math.Int, err error) {
	liveTokens, err := k.reporterKeeper.GetDelegatorTokensAtBlock(ctx, voter, blockNumber)
	if err != nil {
		return nil, math.Int{}, err
	}
	if len(evidenceReporter) == 0 || bytes.Equal(evidenceReporter, liveReporter) {
		return liveReporter, liveTokens, nil
	}
	evTokens, err := k.reporterKeeper.GetDelegatorTokensFromReporterAtBlock(
		ctx, voter.Bytes(), evidenceReporter.Bytes(), blockNumber)
	if err != nil {
		return nil, math.Int{}, err
	}
	if evTokens.IsPositive() {
		return evidenceReporter, evTokens, nil
	}
	return liveReporter, liveTokens, nil
}

func (k Keeper) reporterVotePower(
	ctx context.Context,
	id uint64,
	reporter sdk.AccAddress,
	blockNumber uint64,
	choice types.VoteEnum,
	oldVote *types.Voter,
) (math.Int, error) {
	var reporterPower math.Int
	if oldVote != nil {
		// Revote: use remaining power after peels (including zero after a full peel).
		if oldVote.ReporterPower.IsNil() {
			reporterPower = math.ZeroInt()
		} else {
			reporterPower = oldVote.ReporterPower
		}
	} else {
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
		reporterTokens, err = reporterTokens.SafeSub(tokensVotedBefore)
		if err != nil {
			return math.Int{}, err
		}
		reporterPower = reporterTokens
	}
	return reporterPower, k.AddReporterVoteCount(ctx, id, reporterPower.Uint64(), choice, oldVote)
}

func (k Keeper) peelSelectorFromOwner(
	ctx context.Context,
	id uint64,
	owner sdk.AccAddress,
	tokens math.Int,
	choice types.VoteEnum,
) (math.Int, error) {
	ownerVote, err := k.Voter.Get(ctx, collections.Join(id, owner.Bytes()))
	if err != nil {
		return math.Int{}, err
	}
	if err := k.SubtractReporterVoteCount(ctx, id, tokens.Uint64(), ownerVote.Vote); err != nil {
		return math.Int{}, err
	}
	ownerVote.ReporterPower, err = ownerVote.ReporterPower.SafeSub(tokens)
	if err != nil {
		return math.Int{}, err
	}
	ownerVote.VoterPower, err = ownerVote.VoterPower.SafeSub(tokens)
	if err != nil {
		return math.Int{}, err
	}
	if err := k.Voter.Set(ctx, collections.Join(id, owner.Bytes()), ownerVote); err != nil {
		return math.Int{}, err
	}
	return tokens, k.AddReporterVoteCount(ctx, id, tokens.Uint64(), choice, nil)
}

func (k Keeper) SetVoterReporterStake(
	ctx context.Context,
	id uint64,
	voter sdk.AccAddress,
	blockNumber uint64,
	choice types.VoteEnum,
	oldVote *types.Voter,
	evidenceReporter sdk.AccAddress,
) (math.Int, error) {
	delegation, err := k.reporterKeeper.Delegation(ctx, voter)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return math.ZeroInt(), nil
		}
		return math.Int{}, err
	}
	liveReporter := sdk.AccAddress(delegation.Reporter)

	if bytes.Equal(voter, liveReporter) {
		return k.reporterVotePower(ctx, id, liveReporter, blockNumber, choice, oldVote)
	}

	selector, err := k.reporterKeeper.GetSelectorForStake(ctx, voter)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return math.Int{}, err
		}
		return math.ZeroInt(), nil
	}
	// Revote of already-counted stake must run even while dispute-/switch-locked.
	// Lock only blocks first acquisition of reporter stake into the tally; skipping
	// a counted revote (and letting MsgVote store ReporterPower=0) orphans power in
	// the old bucket and makes a later unlock look like a first vote (double-peel).
	if oldVote != nil && !oldVote.ReporterPower.IsNil() && oldVote.ReporterPower.IsPositive() {
		return oldVote.ReporterPower, k.AddReporterVoteCount(ctx, id, oldVote.ReporterPower.Uint64(), choice, oldVote)
	}
	if reportertypes.SelectorStakeLocked(selector, sdk.UnwrapSDKContext(ctx).BlockTime()) {
		return math.ZeroInt(), nil
	}

	owner, tokens, err := k.selectorStakeAtBlock(ctx, voter, liveReporter, evidenceReporter, blockNumber)
	if err != nil {
		return math.Int{}, err
	}
	if !tokens.IsPositive() {
		return math.ZeroInt(), nil
	}

	ownerVoted, err := k.Voter.Has(ctx, collections.Join(id, owner.Bytes()))
	if err != nil {
		return math.Int{}, err
	}
	if ownerVoted {
		return k.peelSelectorFromOwner(ctx, id, owner, tokens, choice)
	}
	if err := k.addReporterDelegatorTokensVoted(ctx, owner, id, tokens); err != nil {
		return math.Int{}, err
	}
	return tokens, k.AddReporterVoteCount(ctx, id, tokens.Uint64(), choice, nil)
}

// addReporterDelegatorTokensVoted adds amount to the per-dispute reserve of selector
// stake already counted separately from the reporter's remaining vote power.
func (k Keeper) addReporterDelegatorTokensVoted(ctx context.Context, reporter sdk.AccAddress, id uint64, amount math.Int) error {
	delegatorTokensVoted, err := k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), id))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		delegatorTokensVoted = math.ZeroInt()
	}
	delegatorTokensVoted, err = delegatorTokensVoted.SafeAdd(amount)
	if err != nil {
		return err
	}
	return k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(reporter.Bytes(), id), delegatorTokensVoted)
}

func (k Keeper) getVoteCounts(ctx context.Context, id uint64) (types.StakeholderVoteCounts, error) {
	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return types.StakeholderVoteCounts{}, err
		}
		return types.StakeholderVoteCounts{}, nil
	}
	return voteCounts, nil
}

func (k Keeper) AddReporterVoteCount(ctx context.Context, id, amount uint64, choice types.VoteEnum, oldVote *types.Voter) error {
	voteCounts, err := k.getVoteCounts(ctx, id)
	if err != nil {
		return err
	}
	if err := voteCounts.Reporters.Add(choice, amount); err != nil {
		return err
	}
	if oldVote != nil && oldVote.Vote != choice {
		if err := voteCounts.Reporters.Subtract(oldVote.Vote, amount); err != nil {
			return err
		}
	}
	return k.VoteCountsByGroup.Set(ctx, id, voteCounts)
}

func (k Keeper) SubtractReporterVoteCount(ctx context.Context, id, amount uint64, choice types.VoteEnum) error {
	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := voteCounts.Reporters.Subtract(choice, amount); err != nil {
		return err
	}
	return k.VoteCountsByGroup.Set(ctx, id, voteCounts)
}
