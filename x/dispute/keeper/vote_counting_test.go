package keeper_test

import (
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// These tests encode the correct accounting for issues 1-4 in
// .agents/reviews/pr-1061-dispute-vote-counting-final-2026-08-19.md.
// Several of them fail on fix/dispute-vote-counting (ebcc16d7) until the
// keeper is fixed; that failure is the proof.

const (
	pr1061ReporterTokens = uint64(100)
	pr1061SelectorTokens = uint64(30)
)

// outOfRangeVote is a VoteEnum that proto3 will decode but that is not
// INVALID/SUPPORT/AGAINST. Pre-PR code parked these in Invalid via an else.
const outOfRangeVote = types.VoteEnum(3)

func unlockedSelection(reporter sdk.AccAddress) reportertypes.Selection {
	return reportertypes.Selection{
		Reporter:         reporter,
		LockedUntilTime:  time.Time{},
		DelegationsCount: 1,
	}
}

func (s *KeeperTestSuite) reporterVoteCounts(disputeID uint64) types.VoteCounts {
	counts, err := s.disputeKeeper.VoteCountsByGroup.Get(s.ctx, disputeID)
	s.Require().NoError(err)
	return counts.Reporters
}

func (s *KeeperTestSuite) requireNoUint64Wrap(bucket uint64, label string) {
	s.Require().Less(bucket, uint64(1<<63),
		"%s looks like an unsigned wrap (got %d); a bucket subtraction went past zero", label, bucket)
}

// TestSelectorVotedBeforeReporterWritesReserve is the positive counterpart to
// "peel must not write reserve": when the selector votes first, stake is
// reserved so the reporter's later first vote cannot reclaim those tokens.
func (s *KeeperTestSuite) TestSelectorVotedBeforeReporterWritesReserve() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.ctx.WithBlockHeight(10)
	s.ctx = ctx

	disputeID := uint64(7)
	blockNum := uint64(10)
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	rk.On("Delegation", ctx, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", ctx, selector).Return(unlockedSelection(reporter), nil)
	rk.On("GetDelegatorTokensAtBlock", ctx, selector.Bytes(), blockNum).
		Return(math.NewIntFromUint64(pr1061SelectorTokens), nil)

	moved, err := k.SetVoterReporterStake(ctx, disputeID, selector, blockNum, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(pr1061SelectorTokens), moved)
	require.Equal(types.VoteCounts{Support: pr1061SelectorTokens}, s.reporterVoteCounts(disputeID))

	reserve, err := k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(pr1061SelectorTokens), reserve,
		"selector-before-reporter must reserve tokens against the owner")

	rk.On("Delegation", ctx, reporter).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetReporterTokensAtBlock", ctx, reporter.Bytes(), blockNum).
		Return(math.NewIntFromUint64(pr1061ReporterTokens), nil)

	reporterPower, err := k.SetVoterReporterStake(ctx, disputeID, reporter, blockNum, types.VoteEnum_VOTE_AGAINST, nil, nil)
	require.NoError(err)
	require.Equal(math.NewInt(70), reporterPower,
		"reporter first vote must subtract the selector-before reserve (100-30)")
	require.Equal(types.VoteCounts{Support: 30, Against: 70}, s.reporterVoteCounts(disputeID))
}

// TestFinding1_ReporterRevoteAfterSelectorPeelDoesNotWrap proves finding 1.
//
// Reporter R holds 100 tokens, of which selector S holds 30.
//  1. R votes SUPPORT        → Reporters 100 / 0 / 0, ReporterPower 100, reserve 0
//  2. S votes AGAINST (peel) → Reporters  70 / 30 / 0, ReporterPower  70, reserve untouched
//  3. R revotes INVALID
//
// Correct end state is 0 / 30 / 70: the selector keeps 30 AGAINST, the reporter
// moves only the remaining 70 to INVALID via stored ReporterPower. Peel does
// not dual-write reserve.
func (s *KeeperTestSuite) TestFinding1_ReporterRevoteAfterSelectorPeelDoesNotWrap() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.ctx
	ctx = ctx.WithBlockHeight(10)
	s.ctx = ctx

	disputeID := uint64(1)
	blockNum := uint64(10)
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	rk.On("Delegation", ctx, reporter).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetReporterTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewIntFromUint64(pr1061ReporterTokens), nil)
	rk.On("Delegation", ctx, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", ctx, selector).Return(unlockedSelection(reporter), nil)
	rk.On("GetDelegatorTokensAtBlock", ctx, selector.Bytes(), blockNum).Return(math.NewIntFromUint64(pr1061SelectorTokens), nil)

	// 1. Reporter first-votes SUPPORT with the full snapshot.
	reporterPower, err := k.SetVoterReporterStake(ctx, disputeID, reporter, blockNum, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(pr1061ReporterTokens), reporterPower)
	require.NoError(k.Voter.Set(ctx, collections.Join(disputeID, reporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    reporterPower,
		ReporterPower: reporterPower,
	}))

	// 2. Selector peels 30 out of the reporter's SUPPORT into AGAINST.
	selectorPower, err := k.SetVoterReporterStake(ctx, disputeID, selector, blockNum, types.VoteEnum_VOTE_AGAINST, nil, nil)
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(pr1061SelectorTokens), selectorPower)

	afterPeel := s.reporterVoteCounts(disputeID)
	require.Equal(uint64(70), afterPeel.Support)
	require.Equal(uint64(30), afterPeel.Against)
	require.Equal(uint64(0), afterPeel.Invalid)

	_, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
	require.ErrorIs(err, collections.ErrNotFound,
		"peel must not write reserve; remaining power lives on the Voter row")

	reporterRow, err := k.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)

	// 3. Reporter revotes INVALID. Amount must be the remaining 70, not 100.
	moved, err := k.SetVoterReporterStake(ctx, disputeID, reporter, blockNum, types.VoteEnum_VOTE_INVALID, &reporterRow, nil)
	require.NoError(err)

	final := s.reporterVoteCounts(disputeID)
	s.requireNoUint64Wrap(final.Support, "Reporters.Support")
	require.Equal(types.VoteCounts{Support: 0, Against: 30, Invalid: 70}, final,
		"selector's 30 stays AGAINST; reporter's remaining 70 moves SUPPORT→INVALID")
	require.Equal(math.NewInt(70), moved,
		"reporter revote must move stored ReporterPower (70), not snapshot (100)")

	// Wrapped Support (~2^64) is promoted verbatim by TallyVote and reads as
	// full reporter participation against snapshot supply. With snapshot 1000,
	// honest 100-token turnout cannot reach 51% quorum; a wrap can.
	hashID := []byte("finding1-hash")
	require.NoError(k.Disputes.Set(ctx, disputeID, types.Dispute{
		HashId:        hashID,
		DisputeId:     disputeID,
		Open:          true,
		DisputeStatus: types.Voting,
	}))
	require.NoError(k.Votes.Set(ctx, disputeID, types.Vote{
		Id:         disputeID,
		VoteResult: types.VoteResult_NO_TALLY,
		VoteEnd:    ctx.BlockTime().Add(time.Hour),
	}))
	require.NoError(k.BlockInfo.Set(ctx, hashID, types.BlockInfo{
		TotalReporterPower: math.NewInt(1000),
		TotalUserTips:      math.NewInt(1000),
	}))
	err = k.TallyVote(ctx, disputeID)
	if err != nil {
		require.ErrorContains(err, types.ErrNoQuorumStillVoting.Error())
	}
	dispute, err := k.Disputes.Get(ctx, disputeID)
	require.NoError(err)
	require.Equal(types.Voting, dispute.DisputeStatus,
		"a wrap in Support must not manufacture quorum and resolve the dispute")
}

// TestReporterRevoteAfterFullPeelMovesZero covers selectors peeling the reporter's
// entire stake, then the reporter changing choice. Remaining ReporterPower is 0;
// revote must move 0 (not recompute snapshot) and leave selector buckets alone.
func (s *KeeperTestSuite) TestReporterRevoteAfterFullPeelMovesZero() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.ctx.WithBlockHeight(10)
	s.ctx = ctx

	disputeID := uint64(5)
	blockNum := uint64(10)
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	rk.On("Delegation", ctx, reporter).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetReporterTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewIntFromUint64(pr1061SelectorTokens), nil)
	rk.On("Delegation", ctx, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", ctx, selector).Return(unlockedSelection(reporter), nil)
	rk.On("GetDelegatorTokensAtBlock", ctx, selector.Bytes(), blockNum).Return(math.NewIntFromUint64(pr1061SelectorTokens), nil)

	reporterPower, err := k.SetVoterReporterStake(ctx, disputeID, reporter, blockNum, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)
	require.NoError(k.Voter.Set(ctx, collections.Join(disputeID, reporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    reporterPower,
		ReporterPower: reporterPower,
	}))

	_, err = k.SetVoterReporterStake(ctx, disputeID, selector, blockNum, types.VoteEnum_VOTE_AGAINST, nil, nil)
	require.NoError(err)

	reporterRow, err := k.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.True(reporterRow.ReporterPower.IsZero())

	_, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
	require.ErrorIs(err, collections.ErrNotFound,
		"full peel must not write reserve")

	moved, err := k.SetVoterReporterStake(ctx, disputeID, reporter, blockNum, types.VoteEnum_VOTE_INVALID, &reporterRow, nil)
	require.NoError(err)
	require.True(moved.IsZero(), "full peel leaves no reporter power to move on revote")

	final := s.reporterVoteCounts(disputeID)
	require.Equal(types.VoteCounts{Support: 0, Against: pr1061SelectorTokens, Invalid: 0}, final,
		"selector AGAINST must be untouched when reporter revotes with zero remaining power")
}

// TestFinding2_LockedFirstVoteThenRevoteDoesNotWrap proves finding 2.
//
// A dual-class selector (reporter stake + tip power) first-votes while
// dispute-locked. SetVoterReporterStake returns ZeroInt and writes no buckets,
// but MsgVote still stores a Voter row because tips keep voterPower non-zero.
// After the lock expires, the new early return at vote.go:193 keys off
// oldVote != nil and subtracts selectorTokens from a bucket that never
// received them.
//
// Setup: reporter already voted AGAINST with 100. Selector's stored vote is
// SUPPORT with ReporterPower 0. Unlocked revote INVALID then does
// Support -= 30 against Support == 0 → wrap.
//
// Correct: ReporterPower == 0 means first-vote side effects never ran, so
// peel (reporter has voted) or reserve — do not take the revote shortcut.
func (s *KeeperTestSuite) TestFinding2_LockedFirstVoteThenRevoteDoesNotWrap() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper

	now := time.Now().UTC().Truncate(time.Second)
	lockedCtx := s.ctx.WithBlockHeight(10).WithBlockTime(now)
	unlockedCtx := lockedCtx.WithBlockTime(now.Add(2 * time.Hour))
	lockUntil := now.Add(time.Hour)

	disputeID := uint64(2)
	blockNum := uint64(10)
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	require.NoError(k.VoteCountsByGroup.Set(lockedCtx, disputeID, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Against: pr1061ReporterTokens},
	}))
	require.NoError(k.Voter.Set(lockedCtx, collections.Join(disputeID, reporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_AGAINST,
		VoterPower:    math.NewIntFromUint64(pr1061ReporterTokens),
		ReporterPower: math.NewIntFromUint64(pr1061ReporterTokens),
	}))

	rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", mock.Anything, selector).Return(reportertypes.Selection{
		Reporter:           reporter,
		DisputeLockedUntil: lockUntil,
		DelegationsCount:   1,
	}, nil)
	rk.On("GetDelegatorTokensAtBlock", mock.Anything, selector.Bytes(), blockNum).Return(math.NewIntFromUint64(pr1061SelectorTokens), nil)

	// Locked first vote: no reporter buckets, no reserve.
	lockedPower, err := k.SetVoterReporterStake(lockedCtx, disputeID, selector, blockNum, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)
	require.True(lockedPower.IsZero())

	afterLock := s.reporterVoteCountsAt(lockedCtx, disputeID)
	require.Equal(types.VoteCounts{Against: pr1061ReporterTokens}, afterLock)

	// MsgVote still writes a Voter row when tips/team power is non-zero.
	require.NoError(k.Voter.Set(lockedCtx, collections.Join(disputeID, selector.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    math.NewInt(10), // tip power
		ReporterPower: math.ZeroInt(),
	}))
	oldVote, err := k.Voter.Get(lockedCtx, collections.Join(disputeID, selector.Bytes()))
	require.NoError(err)
	require.True(oldVote.ReporterPower.IsZero())

	// Lock expired. This must peel 30 off the reporter's AGAINST into INVALID,
	// not subtract 30 from SUPPORT (which never held the selector's tokens).
	moved, err := k.SetVoterReporterStake(unlockedCtx, disputeID, selector, blockNum, types.VoteEnum_VOTE_INVALID, &oldVote, nil)
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(pr1061SelectorTokens), moved)

	final := s.reporterVoteCountsAt(unlockedCtx, disputeID)
	s.requireNoUint64Wrap(final.Support, "Reporters.Support")
	require.Equal(types.VoteCounts{Support: 0, Against: 70, Invalid: 30}, final,
		"first counted vote after unlock must peel, because ReporterPower was 0")

	reporterRow, err := k.Voter.Get(unlockedCtx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(70), reporterRow.ReporterPower,
		"peel must drop the selector's 30 off the reporter's stored power")
}

// TestFinding2b_SwitchedSelectorUnlockPeelsFromEvidenceReporter covers the e2e
// switch+lock path: selector was on evidenceReporter for the disputed report,
// then switched to newReporter and got dispute-locked. Evidence reporter voted
// with the selector's stake; locked tip-only vote stored ReporterPower=0.
// After unlock, peel must target the evidence reporter — not ADD against the
// current (unvoted) selection, which would double-count.
func (s *KeeperTestSuite) TestFinding2b_SwitchedSelectorUnlockPeelsFromEvidenceReporter() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper

	now := time.Now().UTC().Truncate(time.Second)
	lockedCtx := s.ctx.WithBlockHeight(10).WithBlockTime(now)
	unlockedCtx := lockedCtx.WithBlockTime(now.Add(2 * time.Hour))
	lockUntil := now.Add(time.Hour)

	disputeID := uint64(22)
	blockNum := uint64(10)
	evidenceReporter := sample.AccAddressBytes()
	newReporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	require.NoError(k.VoteCountsByGroup.Set(lockedCtx, disputeID, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Against: pr1061ReporterTokens},
	}))
	require.NoError(k.Voter.Set(lockedCtx, collections.Join(disputeID, evidenceReporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_AGAINST,
		VoterPower:    math.NewIntFromUint64(pr1061ReporterTokens),
		ReporterPower: math.NewIntFromUint64(pr1061ReporterTokens),
	}))
	require.NoError(k.Voter.Set(lockedCtx, collections.Join(disputeID, selector.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    math.NewInt(10),
		ReporterPower: math.ZeroInt(),
	}))

	// Current selection is the post-switch reporter (has not voted).
	rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: newReporter}, nil)
	rk.On("GetSelectorForStake", mock.Anything, selector).Return(reportertypes.Selection{
		Reporter:           newReporter,
		DisputeLockedUntil: lockUntil,
		DelegationsCount:   1,
	}, nil)
	// Current reporter snapshot may be empty / unrelated after switch.
	rk.On("GetDelegatorTokensAtBlock", mock.Anything, selector.Bytes(), blockNum).
		Return(math.ZeroInt(), nil)
	rk.On("GetDelegatorTokensFromReporterAtBlock", mock.Anything, selector.Bytes(), evidenceReporter.Bytes(), blockNum).
		Return(math.NewIntFromUint64(pr1061SelectorTokens), nil)

	oldVote, err := k.Voter.Get(unlockedCtx, collections.Join(disputeID, selector.Bytes()))
	require.NoError(err)

	moved, err := k.SetVoterReporterStake(unlockedCtx, disputeID, selector, blockNum, types.VoteEnum_VOTE_INVALID, &oldVote, evidenceReporter)
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(pr1061SelectorTokens), moved)

	final := s.reporterVoteCountsAt(unlockedCtx, disputeID)
	require.Equal(types.VoteCounts{Support: 0, Against: 70, Invalid: 30}, final,
		"must peel from evidence reporter AGAINST into INVALID without increasing totals")

	evidenceRow, err := k.Voter.Get(unlockedCtx, collections.Join(disputeID, evidenceReporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(70), evidenceRow.ReporterPower)

	// New reporter must not have received a phantom first-vote reserve/add.
	hasNew, err := k.Voter.Has(unlockedCtx, collections.Join(disputeID, newReporter.Bytes()))
	require.NoError(err)
	require.False(hasNew)
	_, err = k.ReportersWithDelegatorsVotedBefore.Get(unlockedCtx, collections.Join(newReporter.Bytes(), disputeID))
	require.ErrorIs(err, collections.ErrNotFound)
}

// TestLiveReporterVotedEvidenceSnapshotStillHoldsTokens peels the evidence
// reporter when the live selection has already voted but the selector's
// dispute-block tokens still sit on the evidence snapshot. The old
// !peelReporterVoted gate missed this ordering.
func (s *KeeperTestSuite) TestLiveReporterVotedEvidenceSnapshotStillHoldsTokens() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.ctx.WithBlockHeight(10)
	s.ctx = ctx

	disputeID := uint64(23)
	blockNum := uint64(10)
	evidenceReporter := sample.AccAddressBytes()
	liveReporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	require.NoError(k.VoteCountsByGroup.Set(ctx, disputeID, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{
			Against: pr1061ReporterTokens, // evidence reporter
			Support: 50,                   // live reporter already voted
		},
	}))
	require.NoError(k.Voter.Set(ctx, collections.Join(disputeID, evidenceReporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_AGAINST,
		VoterPower:    math.NewIntFromUint64(pr1061ReporterTokens),
		ReporterPower: math.NewIntFromUint64(pr1061ReporterTokens),
	}))
	require.NoError(k.Voter.Set(ctx, collections.Join(disputeID, liveReporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    math.NewInt(50),
		ReporterPower: math.NewInt(50),
	}))

	rk.On("Delegation", ctx, selector).Return(reportertypes.Selection{Reporter: liveReporter}, nil)
	rk.On("GetSelectorForStake", ctx, selector).Return(unlockedSelection(liveReporter), nil)
	rk.On("GetDelegatorTokensAtBlock", ctx, selector.Bytes(), blockNum).Return(math.ZeroInt(), nil)
	rk.On("GetDelegatorTokensFromReporterAtBlock", ctx, selector.Bytes(), evidenceReporter.Bytes(), blockNum).
		Return(math.NewIntFromUint64(pr1061SelectorTokens), nil)

	moved, err := k.SetVoterReporterStake(ctx, disputeID, selector, blockNum, types.VoteEnum_VOTE_INVALID, nil, evidenceReporter)
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(pr1061SelectorTokens), moved)

	final := s.reporterVoteCounts(disputeID)
	require.Equal(types.VoteCounts{Support: 50, Against: 70, Invalid: 30}, final,
		"must peel evidence reporter, not live reporter (whose live tokens are 0)")

	evidenceRow, err := k.Voter.Get(ctx, collections.Join(disputeID, evidenceReporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(70), evidenceRow.ReporterPower)

	liveRow, err := k.Voter.Get(ctx, collections.Join(disputeID, liveReporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(50), liveRow.ReporterPower,
		"live reporter must not be peeled when tokens sit on evidence snapshot")
}

func (s *KeeperTestSuite) reporterVoteCountsAt(ctx sdk.Context, disputeID uint64) types.VoteCounts {
	counts, err := s.disputeKeeper.VoteCountsByGroup.Get(ctx, disputeID)
	s.Require().NoError(err)
	return counts.Reporters
}

// TestFinding3_SubtractOutOfRangeOptionLeavesInvalid proves finding 3.
//
// This PR added `default: return ErrInvalidVoteChoice` on the increment
// switch but left the decrement switches with no default. On main, an
// out-of-range option was counted into Invalid via else. After upgrade a
// stored Vote=3 therefore still has its power in Invalid, but
// SubtractReporterVoteCount(choice=3) matches no case and removes nothing.
//
// The subtract path must treat out-of-range as INVALID — mirroring how the
// old code added it — and must not return ErrInvalidVoteChoice, which would
// block every selector of an affected reporter from voting.
func (s *KeeperTestSuite) TestFinding3_SubtractOutOfRangeOptionLeavesInvalid() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx
	disputeID := uint64(3)

	require.NoError(k.VoteCountsByGroup.Set(ctx, disputeID, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Invalid: 100},
	}))

	err := k.SubtractReporterVoteCount(ctx, disputeID, 30, outOfRangeVote)
	require.NoError(err, "subtract must not return ErrInvalidVoteChoice; selectors of a Vote=3 reporter still have to vote")

	counts := s.reporterVoteCounts(disputeID)
	require.Equal(uint64(70), counts.Invalid,
		"stored out-of-range option was parked in Invalid; peel must take the selector's share out of Invalid")
	require.Equal(uint64(0), counts.Support)
	require.Equal(uint64(0), counts.Against)

	// Same gap on a real peel: reporter row holds Vote=3, power in Invalid.
	rk := s.reporterKeeper
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	blockNum := uint64(ctx.BlockHeight())
	peelID := uint64(31)

	require.NoError(k.VoteCountsByGroup.Set(ctx, peelID, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Invalid: 100},
	}))
	require.NoError(k.Voter.Set(ctx, collections.Join(peelID, reporter.Bytes()), types.Voter{
		Vote:          outOfRangeVote,
		VoterPower:    math.NewInt(100),
		ReporterPower: math.NewInt(100),
	}))
	rk.On("Delegation", ctx, selector).Return(reportertypes.Selection{Reporter: reporter}, nil).Once()
	rk.On("GetSelectorForStake", ctx, selector).Return(unlockedSelection(reporter), nil).Once()
	rk.On("GetDelegatorTokensAtBlock", ctx, selector.Bytes(), blockNum).Return(math.NewInt(30), nil).Once()

	_, err = k.SetVoterReporterStake(ctx, peelID, selector, blockNum, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)

	peeled := s.reporterVoteCountsAt(ctx, peelID)
	require.Equal(uint64(70), peeled.Invalid, "peel of Vote=3 must decrement Invalid")
	require.Equal(uint64(30), peeled.Support, "selector's new SUPPORT is added")
}

// TestFinding4_SelectorRevoteDoesNotDoublePeel is the coverage the PR claimed
// to add (finding 4, item 2). Reporter is already AGAINST; selector first
// votes SUPPORT (peel once), then revotes AGAINST. Reporter power must be
// subtracted exactly once; buckets move Support→Against; reserve from a
// prior selector-before-reporter stays unchanged on peel and revote.
func (s *KeeperTestSuite) TestFinding4_SelectorRevoteDoesNotDoublePeel() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.ctx.WithBlockHeight(10)
	s.ctx = ctx

	disputeID := uint64(4)
	blockNum := uint64(10)
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	require.NoError(k.VoteCountsByGroup.Set(ctx, disputeID, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Against: 150},
	}))
	require.NoError(k.Voter.Set(ctx, collections.Join(disputeID, reporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_AGAINST,
		VoterPower:    math.NewInt(150),
		ReporterPower: math.NewInt(150),
	}))
	// A leftover reserve from some other selector who voted before the reporter.
	require.NoError(k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(reporter.Bytes(), disputeID), math.NewInt(50)))

	rk.On("Delegation", ctx, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", ctx, selector).Return(unlockedSelection(reporter), nil)
	rk.On("GetDelegatorTokensAtBlock", ctx, selector.Bytes(), blockNum).Return(math.NewInt(100), nil)

	_, err := k.SetVoterReporterStake(ctx, disputeID, selector, blockNum, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)
	afterPeel := s.reporterVoteCounts(disputeID)
	require.Equal(types.VoteCounts{Support: 100, Against: 50, Invalid: 0}, afterPeel)

	reporterRow, err := k.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(50), reporterRow.ReporterPower)

	reserve, err := k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
	require.NoError(err)
	require.Equal(math.NewInt(50), reserve,
		"peel must leave the pre-existing selector-before-reporter reserve alone")

	oldVote := &types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    math.NewInt(100),
		ReporterPower: math.NewInt(100),
	}
	_, err = k.SetVoterReporterStake(ctx, disputeID, selector, blockNum, types.VoteEnum_VOTE_AGAINST, oldVote, nil)
	require.NoError(err)

	final := s.reporterVoteCounts(disputeID)
	require.Equal(types.VoteCounts{Support: 0, Against: 150, Invalid: 0}, final,
		"revote must move the selector's 100 Support→Against, not peel the reporter a second time")

	reporterRow, err = k.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(50), reporterRow.ReporterPower, "reporter power is peeled only on the first selector vote")

	reserve, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
	require.NoError(err)
	require.Equal(math.NewInt(50), reserve, "revote must not change the reserve")
}

// TestVoteThenSwitchThenRevoteConservesAndDoesNotTouchNewReporter covers:
// selector peels from evidenceReporter, switches Selection to newReporter, then
// revotes. Stored ReporterPower must move buckets only; no peel/add against the
// new reporter and no second peel of the evidence reporter.
func (s *KeeperTestSuite) TestVoteThenSwitchThenRevoteConservesAndDoesNotTouchNewReporter() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.ctx.WithBlockHeight(10)
	s.ctx = ctx

	disputeID := uint64(5)
	blockNum := uint64(10)
	evidenceReporter := sample.AccAddressBytes()
	newReporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	require.NoError(k.VoteCountsByGroup.Set(ctx, disputeID, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Against: pr1061ReporterTokens},
	}))
	require.NoError(k.Voter.Set(ctx, collections.Join(disputeID, evidenceReporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_AGAINST,
		VoterPower:    math.NewIntFromUint64(pr1061ReporterTokens),
		ReporterPower: math.NewIntFromUint64(pr1061ReporterTokens),
	}))

	// First vote while still selected to the evidence reporter: peel 30.
	rk.On("Delegation", ctx, selector).Return(reportertypes.Selection{Reporter: evidenceReporter}, nil).Once()
	rk.On("GetSelectorForStake", ctx, selector).Return(unlockedSelection(evidenceReporter), nil).Once()
	rk.On("GetDelegatorTokensAtBlock", ctx, selector.Bytes(), blockNum).
		Return(math.NewIntFromUint64(pr1061SelectorTokens), nil).Once()

	moved, err := k.SetVoterReporterStake(ctx, disputeID, selector, blockNum, types.VoteEnum_VOTE_SUPPORT, nil, evidenceReporter)
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(pr1061SelectorTokens), moved)

	afterPeel := s.reporterVoteCounts(disputeID)
	require.Equal(types.VoteCounts{Support: 30, Against: 70}, afterPeel)

	evidenceRow, err := k.Voter.Get(ctx, collections.Join(disputeID, evidenceReporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(70), evidenceRow.ReporterPower)

	_, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(evidenceReporter.Bytes(), disputeID))
	require.ErrorIs(err, collections.ErrNotFound,
		"peel must not write reserve; remaining power lives on the evidence reporter Voter row")

	// Post-switch: current Selection is newReporter (has not voted). Revote must
	// use stored ReporterPower and must not call a first-vote peel/add path.
	oldVote := &types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    math.NewIntFromUint64(pr1061SelectorTokens),
		ReporterPower: math.NewIntFromUint64(pr1061SelectorTokens),
	}
	rk.On("Delegation", ctx, selector).Return(reportertypes.Selection{Reporter: newReporter}, nil).Once()
	rk.On("GetSelectorForStake", ctx, selector).Return(unlockedSelection(newReporter), nil).Once()

	moved, err = k.SetVoterReporterStake(ctx, disputeID, selector, blockNum, types.VoteEnum_VOTE_INVALID, oldVote, evidenceReporter)
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(pr1061SelectorTokens), moved)

	final := s.reporterVoteCounts(disputeID)
	require.Equal(types.VoteCounts{Support: 0, Against: 70, Invalid: 30}, final,
		"post-switch revote must only move the selector's stored 30 Support→Invalid")

	evidenceRow, err = k.Voter.Get(ctx, collections.Join(disputeID, evidenceReporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(70), evidenceRow.ReporterPower,
		"evidence reporter must not be peeled again on post-switch revote")

	hasNew, err := k.Voter.Has(ctx, collections.Join(disputeID, newReporter.Bytes()))
	require.NoError(err)
	require.False(hasNew, "new reporter must not receive a phantom voter row")
	_, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(newReporter.Bytes(), disputeID))
	require.ErrorIs(err, collections.ErrNotFound,
		"new reporter must not receive a first-vote reserve after switch")
}

// TestCountedVoteThenLockedTipRevoteMovesBucketsWithoutWipingPower covers the
// failure mode where lock used to short-circuit before the counted-revote path,
// returning ZeroInt so MsgVote could store ReporterPower=0 while leaving power
// in the old reporter bucket. After unlock that looked like a first vote and
// double-peeled. Correct: already-counted stake still moves while locked.
func (s *KeeperTestSuite) TestCountedVoteThenLockedTipRevoteMovesBucketsWithoutWipingPower() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper

	now := time.Now().UTC().Truncate(time.Second)
	unlockedCtx := s.ctx.WithBlockHeight(10).WithBlockTime(now)
	lockUntil := now.Add(time.Hour)
	lockedCtx := unlockedCtx.WithBlockTime(now.Add(30 * time.Minute))
	s.ctx = unlockedCtx

	disputeID := uint64(6)
	blockNum := uint64(10)
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	require.NoError(k.VoteCountsByGroup.Set(unlockedCtx, disputeID, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Against: pr1061ReporterTokens},
	}))
	require.NoError(k.Voter.Set(unlockedCtx, collections.Join(disputeID, reporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_AGAINST,
		VoterPower:    math.NewIntFromUint64(pr1061ReporterTokens),
		ReporterPower: math.NewIntFromUint64(pr1061ReporterTokens),
	}))

	rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	// Unlocked for the first counted peel; locked afterward for the tip-funded revote.
	rk.On("GetSelectorForStake", unlockedCtx, selector).Return(unlockedSelection(reporter), nil).Once()
	rk.On("GetSelectorForStake", lockedCtx, selector).Return(reportertypes.Selection{
		Reporter:           reporter,
		DisputeLockedUntil: lockUntil,
		DelegationsCount:   1,
	}, nil).Once()
	rk.On("GetDelegatorTokensAtBlock", mock.Anything, selector.Bytes(), blockNum).
		Return(math.NewIntFromUint64(pr1061SelectorTokens), nil).Once()

	// Unlocked first vote peels 30 into SUPPORT.
	moved, err := k.SetVoterReporterStake(unlockedCtx, disputeID, selector, blockNum, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(pr1061SelectorTokens), moved)
	require.Equal(types.VoteCounts{Support: 30, Against: 70}, s.reporterVoteCountsAt(unlockedCtx, disputeID))

	oldVote := &types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    math.NewIntFromUint64(pr1061SelectorTokens).AddRaw(10), // stake + tip
		ReporterPower: math.NewIntFromUint64(pr1061SelectorTokens),
	}

	// Locked revote must still move the counted 30 and return that power (so
	// MsgVote does not overwrite ReporterPower with 0).
	moved, err = k.SetVoterReporterStake(lockedCtx, disputeID, selector, blockNum, types.VoteEnum_VOTE_INVALID, oldVote, nil)
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(pr1061SelectorTokens), moved,
		"locked revote of already-counted stake must return stored ReporterPower")

	final := s.reporterVoteCountsAt(lockedCtx, disputeID)
	require.Equal(types.VoteCounts{Support: 0, Against: 70, Invalid: 30}, final,
		"locked counted revote must move Support→Invalid, not leave orphaned SUPPORT")

	reporterRow, err := k.Voter.Get(lockedCtx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(70), reporterRow.ReporterPower,
		"must not double-peel the reporter during a locked revote")
}
