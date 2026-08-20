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

// subtractUserVoteBucket removes amount from the bucket for choice. Out-of-range
// choices are treated as INVALID to match legacy increment behavior.
func subtractUserVoteBucket(counts *types.VoteCounts, choice types.VoteEnum, amount uint64) error {
	switch choice {
	case types.VoteEnum_VOTE_SUPPORT:
		if counts.Support < amount {
			return types.ErrVoteCountUnderflow
		}
		counts.Support -= amount
	case types.VoteEnum_VOTE_AGAINST:
		if counts.Against < amount {
			return types.ErrVoteCountUnderflow
		}
		counts.Against -= amount
	case types.VoteEnum_VOTE_INVALID:
		if counts.Invalid < amount {
			return types.ErrVoteCountUnderflow
		}
		counts.Invalid -= amount
	default:
		if counts.Invalid < amount {
			return types.ErrVoteCountUnderflow
		}
		counts.Invalid -= amount
	}
	return nil
}

// subtractReporterVoteBucket removes amount from the bucket for choice. Out-of-range
// choices are treated as INVALID to match legacy increment behavior.
func subtractReporterVoteBucket(counts *types.VoteCounts, choice types.VoteEnum, amount uint64) error {
	switch choice {
	case types.VoteEnum_VOTE_SUPPORT:
		if counts.Support < amount {
			return types.ErrVoteCountUnderflow
		}
		counts.Support -= amount
	case types.VoteEnum_VOTE_AGAINST:
		if counts.Against < amount {
			return types.ErrVoteCountUnderflow
		}
		counts.Against -= amount
	case types.VoteEnum_VOTE_INVALID:
		if counts.Invalid < amount {
			return types.ErrVoteCountUnderflow
		}
		counts.Invalid -= amount
	default:
		if counts.Invalid < amount {
			return types.ErrVoteCountUnderflow
		}
		counts.Invalid -= amount
	}
	return nil
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
		voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return math.Int{}, err
			}
			voteCounts = types.StakeholderVoteCounts{}
		}
		switch choice {
		case types.VoteEnum_VOTE_SUPPORT:
			voteCounts.Users.Support += tips.Uint64()
		case types.VoteEnum_VOTE_AGAINST:
			voteCounts.Users.Against += tips.Uint64()
		case types.VoteEnum_VOTE_INVALID:
			voteCounts.Users.Invalid += tips.Uint64()
		}
		if oldVote != nil {
			if oldVote.Vote != choice {
				if err := subtractUserVoteBucket(&voteCounts.Users, oldVote.Vote, tips.Uint64()); err != nil {
					return math.Int{}, err
				}
			}
		}
		err = k.VoteCountsByGroup.Set(ctx, id, voteCounts)
		if err != nil {
			return math.Int{}, err
		}
		return tips, nil
	}
	return math.ZeroInt(), nil
}

func (k Keeper) SetVoterReporterStake(ctx context.Context, id uint64, voter sdk.AccAddress, blockNumber uint64, choice types.VoteEnum, oldVote *types.Voter) (math.Int, error) {
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
	// voter is non-reporter selector
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

	// First counted reporter vote (including unlock after a locked tip-only vote).
	// Prefer peeling from the reporter who already voted with these tokens. After a
	// selector switch, current Selection.reporter may differ from the disputed
	// report's reporter — peeling against the current reporter would ADD stake and
	// double-count what the evidence reporter already voted.
	peelReporter := reporter
	peelReporterVoted := reporterHasVoted
	selectorTokens, err := k.reporterKeeper.GetDelegatorTokensAtBlock(ctx, voter, blockNumber)
	if err != nil {
		return math.Int{}, err
	}
	if !peelReporterVoted {
		dispute, digErr := k.Disputes.Get(ctx, id)
		if digErr != nil {
			if !errors.Is(digErr, collections.ErrNotFound) {
				return math.Int{}, digErr
			}
		} else {
			evidenceReporter, addrErr := sdk.AccAddressFromBech32(dispute.InitialEvidence.Reporter)
			if addrErr != nil {
				return math.Int{}, addrErr
			}
			if !bytes.Equal(evidenceReporter, peelReporter) {
				evidenceVoted, hasErr := k.Voter.Has(ctx, collections.Join(id, evidenceReporter.Bytes()))
				if hasErr != nil {
					return math.Int{}, hasErr
				}
				if evidenceVoted {
					evidenceTokens, tokErr := k.reporterKeeper.GetDelegatorTokensFromReporterAtBlock(
						ctx, voter.Bytes(), evidenceReporter.Bytes(), blockNumber)
					if tokErr != nil {
						return math.Int{}, tokErr
					}
					if evidenceTokens.IsPositive() {
						peelReporter = evidenceReporter
						peelReporterVoted = true
						selectorTokens = evidenceTokens
					}
				}
			}
		}
	}

	// First vote, and peel target already cast — peel selector power out of that vote.
	if peelReporterVoted {
		reporterVote, err := k.Voter.Get(ctx, collections.Join(id, peelReporter.Bytes()))
		if err != nil {
			return math.Int{}, err
		}
		err = k.SubtractReporterVoteCount(ctx, id, selectorTokens.Uint64(), reporterVote.Vote)
		if err != nil {
			return math.Int{}, err
		}
		// update reporter's power record for reward calculation
		reporterVote.ReporterPower, err = reporterVote.ReporterPower.SafeSub(selectorTokens)
		if err != nil {
			return math.Int{}, err
		}
		reporterVote.VoterPower, err = reporterVote.VoterPower.SafeSub(selectorTokens)
		if err != nil {
			return math.Int{}, err
		}
		err = k.Voter.Set(ctx, collections.Join(id, peelReporter.Bytes()), reporterVote)
		if err != nil {
			return math.Int{}, err
		}
		// Keep reserve in sync with peeled stake so snapshot-reserve stays consistent.
		if err := k.addReporterDelegatorTokensVoted(ctx, peelReporter, id, selectorTokens); err != nil {
			return math.Int{}, err
		}
		return selectorTokens, k.AddReporterVoteCount(ctx, id, selectorTokens.Uint64(), choice, nil)
	}
	// First vote, no reporter has voted with these tokens yet — reserve so the
	// current selection reporter can't vote them later.
	if err := k.addReporterDelegatorTokensVoted(ctx, reporter, id, selectorTokens); err != nil {
		return math.Int{}, err
	}
	return selectorTokens, k.AddReporterVoteCount(ctx, id, selectorTokens.Uint64(), choice, nil)
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

func (k Keeper) AddReporterVoteCount(ctx context.Context, id, amount uint64, choice types.VoteEnum, oldVote *types.Voter) error {
	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		voteCounts = types.StakeholderVoteCounts{}
	}
	switch choice {
	case types.VoteEnum_VOTE_SUPPORT:
		voteCounts.Reporters.Support += amount
	case types.VoteEnum_VOTE_AGAINST:
		voteCounts.Reporters.Against += amount
	case types.VoteEnum_VOTE_INVALID:
		voteCounts.Reporters.Invalid += amount
	default:
		return types.ErrInvalidVoteChoice
	}
	if oldVote != nil {
		if oldVote.Vote != choice {
			if err := subtractReporterVoteBucket(&voteCounts.Reporters, oldVote.Vote, amount); err != nil {
				return err
			}
		}
	}

	return k.VoteCountsByGroup.Set(ctx, id, voteCounts)
}

func (k Keeper) SubtractReporterVoteCount(ctx context.Context, id, amount uint64, choice types.VoteEnum) error {
	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := subtractReporterVoteBucket(&voteCounts.Reporters, choice, amount); err != nil {
		return err
	}
	return k.VoteCountsByGroup.Set(ctx, id, voteCounts)
}
