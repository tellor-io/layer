package keeper_test

import (
	"bytes"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	disputeOpenHeight = uint64(100)
	groupSnapshot     = int64(100)
	selectorSnapshot  = int64(30)
)

// expectDisputeHeight stubs who addr was selected to at the dispute open height.
// liveAccurate is false when that reporter is a resolved history row.
func (s *KeeperTestSuite) expectDisputeHeight(addr, selected sdk.AccAddress, liveAccurate, unfinished bool) {
	s.reporterKeeper.On("ReporterAtHeight", mock.Anything, addr, disputeOpenHeight).
		Return(selected, liveAccurate, nil)
	s.reporterKeeper.On("SelectorSwitchUnfinishedAt", mock.Anything, addr, disputeOpenHeight).
		Return(unfinished, nil)
}

func (s *KeeperTestSuite) voteCtx() sdk.Context {
	// Current block is after the dispute opened. Vote power must use disputeOpenHeight.
	return s.ctx.WithBlockHeight(int64(disputeOpenHeight) + 400)
}

func (s *KeeperTestSuite) seedCountedReporterVote(ctx sdk.Context, id uint64, voter sdk.AccAddress, power int64, choice types.VoteEnum) {
	s.Require().NoError(s.disputeKeeper.VoteCountsByGroup.Set(ctx, id, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Support: uint64(power)},
	}))
	s.Require().NoError(s.disputeKeeper.Voter.Set(ctx, collections.Join(id, voter.Bytes()), types.Voter{
		Vote:          choice,
		VoterPower:    math.NewInt(power),
		ReporterPower: math.NewInt(power),
	}))
}

// TestDisputeHeight_SelfDemotionAfterOpenStillVotesGroup covers a reporter who
// was selected to themselves at the open height. A self-demotion that finalizes
// later leaves that group power in place, whether they vote before or after it
// finalizes. A selector still on them peels, and that selector vote succeeds.
func (s *KeeperTestSuite) TestDisputeHeight_SelfDemotionAfterOpenStillVotesGroup() {
	for _, tc := range []struct {
		name       string
		liveTarget func(reporter, other sdk.AccAddress) sdk.AccAddress
	}{
		{name: "vote before demotion finalizes", liveTarget: func(reporter, _ sdk.AccAddress) sdk.AccAddress { return reporter }},
		{name: "vote after demotion finalizes", liveTarget: func(_, other sdk.AccAddress) sdk.AccAddress { return other }},
	} {
		s.Run(tc.name, func() {
			s.SetupTest()
			require := s.Require()
			k := s.disputeKeeper
			rk := s.reporterKeeper
			ctx := s.voteCtx()
			disputeID := uint64(1)
			reporter := sample.AccAddressBytes()
			other := sample.AccAddressBytes()
			selector := sample.AccAddressBytes()
			live := tc.liveTarget(reporter, other)

			rk.On("Delegation", mock.Anything, reporter).Return(reportertypes.Selection{Reporter: live}, nil)
			s.expectDisputeHeight(reporter, reporter, bytes.Equal(live, reporter), false)
			rk.On("GetReporterTokensAtBlock", mock.Anything, reporter.Bytes(), disputeOpenHeight).
				Return(math.NewInt(groupSnapshot), nil)

			power, err := k.SetVoterReporterStake(ctx, disputeID, reporter, disputeOpenHeight, types.VoteEnum_VOTE_SUPPORT, nil, nil)
			require.NoError(err)
			require.Equal(math.NewInt(groupSnapshot), power)
			require.NoError(k.Voter.Set(ctx, collections.Join(disputeID, reporter.Bytes()), types.Voter{
				Vote:          types.VoteEnum_VOTE_SUPPORT,
				VoterPower:    power,
				ReporterPower: power,
			}))

			rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
			rk.On("GetSelectorForStake", mock.Anything, selector).Return(unlockedSelection(reporter), nil)
			s.expectDisputeHeight(selector, reporter, true, false)
			rk.On("GetDelegatorTokensAtBlock", mock.Anything, selector.Bytes(), disputeOpenHeight).
				Return(math.NewInt(selectorSnapshot), nil)

			selectorPower, err := k.SetVoterReporterStake(ctx, disputeID, selector, disputeOpenHeight, types.VoteEnum_VOTE_AGAINST, nil, nil)
			require.NoError(err)
			require.Equal(math.NewInt(selectorSnapshot), selectorPower, "selector vote succeeds and peels")
			require.Equal(types.VoteCounts{Support: uint64(groupSnapshot - selectorSnapshot), Against: uint64(selectorSnapshot)}, s.reporterVoteCountsAt(ctx, disputeID))

			owner, err := k.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
			require.NoError(err)
			require.Equal(math.NewInt(groupSnapshot-selectorSnapshot), owner.ReporterPower)
			_, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
			require.ErrorIs(err, collections.ErrNotFound)
		})
	}
}

// TestDisputeHeight_PendingSelfDemotionHasNoGroupPower is an open self-demotion
// at the dispute height. ReporterAtHeight still returns the reporter, and their
// power is 0. Their selector does not peel; the selector vote succeeds and counts
// that selector's own snapshot on the selector's choice.
func (s *KeeperTestSuite) TestDisputeHeight_PendingSelfDemotionHasNoGroupPower() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.voteCtx()
	disputeID := uint64(2)
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	s.seedCountedReporterVote(ctx, disputeID, reporter, 80, types.VoteEnum_VOTE_SUPPORT)

	rk.On("Delegation", mock.Anything, reporter).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", mock.Anything, reporter).Return(unlockedSelection(reporter), nil)
	// Open self-demotion: selection has not moved, and the switch is unfinished.
	s.expectDisputeHeight(reporter, reporter, true, true)

	power, err := k.SetVoterReporterStake(ctx, disputeID, reporter, disputeOpenHeight, types.VoteEnum_VOTE_AGAINST, nil, nil)
	require.NoError(err)
	require.True(power.IsZero())
	require.Equal(types.VoteCounts{Support: 80}, s.reporterVoteCountsAt(ctx, disputeID))

	rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", mock.Anything, selector).Return(unlockedSelection(reporter), nil)
	s.expectDisputeHeight(selector, reporter, true, false)
	rk.On("GetDelegatorTokensAtBlock", mock.Anything, selector.Bytes(), disputeOpenHeight).
		Return(math.NewInt(selectorSnapshot), nil)

	selectorPower, err := k.SetVoterReporterStake(ctx, disputeID, selector, disputeOpenHeight, types.VoteEnum_VOTE_AGAINST, nil, nil)
	require.NoError(err)
	require.Equal(math.NewInt(selectorSnapshot), selectorPower)
	require.Equal(types.VoteCounts{Support: 80, Against: uint64(selectorSnapshot)}, s.reporterVoteCountsAt(ctx, disputeID))

	owner, err := k.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(80), owner.ReporterPower)
	require.Equal(types.VoteEnum_VOTE_SUPPORT, owner.Vote)
	_, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
	require.ErrorIs(err, collections.ErrNotFound)
}

// TestDisputeHeight_SelectorSwitchUnfinishedHasNoPower is a normal selector whose
// switch was unfinished at the dispute height, including one that completes later.
func (s *KeeperTestSuite) TestDisputeHeight_SelectorSwitchUnfinishedHasNoPower() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.voteCtx()
	disputeID := uint64(3)
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", mock.Anything, selector).Return(unlockedSelection(reporter), nil)
	s.expectDisputeHeight(selector, reporter, true, true)

	power, err := k.SetVoterReporterStake(ctx, disputeID, selector, disputeOpenHeight, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)
	require.True(power.IsZero())
	_, err = k.VoteCountsByGroup.Get(ctx, disputeID)
	require.ErrorIs(err, collections.ErrNotFound)
	_, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
	require.ErrorIs(err, collections.ErrNotFound)
}

// TestDisputeHeight_SwitchAfterOpenUsesReporterAtOpen is a selector whose switch
// begins and completes after the dispute opened. Stake stays with the reporter
// they were on at the open height, and peel follows that reporter's status then.
func (s *KeeperTestSuite) TestDisputeHeight_SwitchAfterOpenUsesReporterAtOpen() {
	s.Run("reporter was voting the group", func() {
		s.SetupTest()
		require := s.Require()
		k := s.disputeKeeper
		rk := s.reporterKeeper
		ctx := s.voteCtx()
		disputeID := uint64(4)
		oldReporter := sample.AccAddressBytes()
		newReporter := sample.AccAddressBytes()
		selector := sample.AccAddressBytes()

		s.seedCountedReporterVote(ctx, disputeID, oldReporter, groupSnapshot, types.VoteEnum_VOTE_SUPPORT)
		s.expectDisputeHeight(oldReporter, oldReporter, true, false)

		rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: newReporter}, nil)
		rk.On("GetSelectorForStake", mock.Anything, selector).Return(unlockedSelection(newReporter), nil)
		s.expectDisputeHeight(selector, oldReporter, false, false)
		rk.On("GetDelegatorTokensFromReporterAtBlock", mock.Anything, selector.Bytes(), oldReporter.Bytes(), disputeOpenHeight).
			Return(math.NewInt(selectorSnapshot), nil)

		power, err := k.SetVoterReporterStake(ctx, disputeID, selector, disputeOpenHeight, types.VoteEnum_VOTE_AGAINST, nil, newReporter)
		require.NoError(err)
		require.Equal(math.NewInt(selectorSnapshot), power)
		require.Equal(types.VoteCounts{
			Support: uint64(groupSnapshot - selectorSnapshot),
			Against: uint64(selectorSnapshot),
		}, s.reporterVoteCountsAt(ctx, disputeID))

		_, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(newReporter.Bytes(), disputeID))
		require.ErrorIs(err, collections.ErrNotFound)
		hasNew, err := k.Voter.Has(ctx, collections.Join(disputeID, newReporter.Bytes()))
		require.NoError(err)
		require.False(hasNew)
	})

	s.Run("reporter was not voting the group", func() {
		s.SetupTest()
		require := s.Require()
		k := s.disputeKeeper
		rk := s.reporterKeeper
		ctx := s.voteCtx()
		disputeID := uint64(5)
		oldReporter := sample.AccAddressBytes()
		elsewhere := sample.AccAddressBytes()
		newReporter := sample.AccAddressBytes()
		selector := sample.AccAddressBytes()

		s.seedCountedReporterVote(ctx, disputeID, oldReporter, 40, types.VoteEnum_VOTE_SUPPORT)
		s.expectDisputeHeight(oldReporter, elsewhere, false, false)

		rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: newReporter}, nil)
		rk.On("GetSelectorForStake", mock.Anything, selector).Return(unlockedSelection(newReporter), nil)
		s.expectDisputeHeight(selector, oldReporter, false, false)
		rk.On("GetDelegatorTokensFromReporterAtBlock", mock.Anything, selector.Bytes(), oldReporter.Bytes(), disputeOpenHeight).
			Return(math.NewInt(selectorSnapshot), nil)

		power, err := k.SetVoterReporterStake(ctx, disputeID, selector, disputeOpenHeight, types.VoteEnum_VOTE_AGAINST, nil, newReporter)
		require.NoError(err)
		require.Equal(math.NewInt(selectorSnapshot), power)
		require.Equal(types.VoteCounts{Support: 40, Against: uint64(selectorSnapshot)}, s.reporterVoteCountsAt(ctx, disputeID))

		owner, err := k.Voter.Get(ctx, collections.Join(disputeID, oldReporter.Bytes()))
		require.NoError(err)
		require.Equal(math.NewInt(40), owner.ReporterPower)
		_, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(oldReporter.Bytes(), disputeID))
		require.ErrorIs(err, collections.ErrNotFound)
	})
}

// TestDisputeHeight_ReselectDoesNotGrantGroupPower is a reporter who was not an
// idle self-selection at the open height and later selects themselves again.
func (s *KeeperTestSuite) TestDisputeHeight_ReselectDoesNotGrantGroupPower() {
	s.Run("selected to someone else at open", func() {
		s.SetupTest()
		require := s.Require()
		k := s.disputeKeeper
		rk := s.reporterKeeper
		ctx := s.voteCtx()
		disputeID := uint64(6)
		reporter := sample.AccAddressBytes()
		other := sample.AccAddressBytes()

		rk.On("Delegation", mock.Anything, reporter).Return(reportertypes.Selection{Reporter: reporter}, nil)
		rk.On("GetSelectorForStake", mock.Anything, reporter).Return(unlockedSelection(reporter), nil)
		s.expectDisputeHeight(reporter, other, false, false)
		s.expectDisputeHeight(other, other, true, false)
		rk.On("GetDelegatorTokensFromReporterAtBlock", mock.Anything, reporter.Bytes(), other.Bytes(), disputeOpenHeight).
			Return(math.NewInt(25), nil)

		power, err := k.SetVoterReporterStake(ctx, disputeID, reporter, disputeOpenHeight, types.VoteEnum_VOTE_SUPPORT, nil, nil)
		require.NoError(err)
		require.Equal(math.NewInt(25), power)
		require.Equal(types.VoteCounts{Support: 25}, s.reporterVoteCountsAt(ctx, disputeID))

		reserved, err := k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(other.Bytes(), disputeID))
		require.NoError(err)
		require.Equal(math.NewInt(25), reserved)
		_, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
		require.ErrorIs(err, collections.ErrNotFound)
	})

	s.Run("switch unfinished at open", func() {
		s.SetupTest()
		require := s.Require()
		k := s.disputeKeeper
		rk := s.reporterKeeper
		ctx := s.voteCtx()
		reporter := sample.AccAddressBytes()

		rk.On("Delegation", mock.Anything, reporter).Return(reportertypes.Selection{Reporter: reporter}, nil)
		rk.On("GetSelectorForStake", mock.Anything, reporter).Return(unlockedSelection(reporter), nil)
		s.expectDisputeHeight(reporter, reporter, true, true)

		power, err := k.SetVoterReporterStake(ctx, 7, reporter, disputeOpenHeight, types.VoteEnum_VOTE_SUPPORT, nil, nil)
		require.NoError(err)
		require.True(power.IsZero())
	})
}

// TestDisputeHeight_ReselectDoesNotRemoveGroupPower is a reporter who was an idle
// self-selection at the open height and later selects themselves again.
// The open-height answer is still idle self, so the vote is the group snapshot.
func (s *KeeperTestSuite) TestDisputeHeight_ReselectDoesNotRemoveGroupPower() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.voteCtx()
	disputeID := uint64(8)
	reporter := sample.AccAddressBytes()

	rk.On("Delegation", mock.Anything, reporter).Return(reportertypes.Selection{Reporter: reporter}, nil)
	s.expectDisputeHeight(reporter, reporter, true, false)
	rk.On("GetReporterTokensAtBlock", mock.Anything, reporter.Bytes(), disputeOpenHeight).
		Return(math.NewInt(groupSnapshot), nil)

	power, err := k.SetVoterReporterStake(ctx, disputeID, reporter, disputeOpenHeight, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)
	require.Equal(math.NewInt(groupSnapshot), power)
	require.Equal(types.VoteCounts{Support: uint64(groupSnapshot)}, s.reporterVoteCountsAt(ctx, disputeID))
}

func (s *KeeperTestSuite) TestDisputeHeight_SelectorFirstReservesGroupVote() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.voteCtx()
	disputeID := uint64(9)
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", mock.Anything, selector).Return(unlockedSelection(reporter), nil)
	s.expectDisputeHeight(selector, reporter, true, false)
	s.expectDisputeHeight(reporter, reporter, true, false)
	rk.On("GetDelegatorTokensAtBlock", mock.Anything, selector.Bytes(), disputeOpenHeight).
		Return(math.NewInt(selectorSnapshot), nil)

	selectorPower, err := k.SetVoterReporterStake(ctx, disputeID, selector, disputeOpenHeight, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)
	require.Equal(math.NewInt(selectorSnapshot), selectorPower)
	reserved, err := k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
	require.NoError(err)
	require.Equal(math.NewInt(selectorSnapshot), reserved)

	rk.On("Delegation", mock.Anything, reporter).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetReporterTokensAtBlock", mock.Anything, reporter.Bytes(), disputeOpenHeight).
		Return(math.NewInt(groupSnapshot), nil)

	reporterPower, err := k.SetVoterReporterStake(ctx, disputeID, reporter, disputeOpenHeight, types.VoteEnum_VOTE_AGAINST, nil, nil)
	require.NoError(err)
	require.Equal(math.NewInt(groupSnapshot-selectorSnapshot), reporterPower)
	require.Equal(types.VoteCounts{
		Support: uint64(selectorSnapshot),
		Against: uint64(groupSnapshot - selectorSnapshot),
	}, s.reporterVoteCountsAt(ctx, disputeID))
}

func (s *KeeperTestSuite) TestDisputeHeight_RevoteAfterPeelMovesStoredPower() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.voteCtx()
	disputeID := uint64(10)
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	rk.On("Delegation", mock.Anything, reporter).Return(reportertypes.Selection{Reporter: reporter}, nil)
	s.expectDisputeHeight(reporter, reporter, true, false)
	rk.On("GetReporterTokensAtBlock", mock.Anything, reporter.Bytes(), disputeOpenHeight).
		Return(math.NewInt(groupSnapshot), nil).Once()

	power, err := k.SetVoterReporterStake(ctx, disputeID, reporter, disputeOpenHeight, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)
	require.NoError(k.Voter.Set(ctx, collections.Join(disputeID, reporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    power,
		ReporterPower: power,
	}))

	rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", mock.Anything, selector).Return(unlockedSelection(reporter), nil)
	s.expectDisputeHeight(selector, reporter, true, false)
	rk.On("GetDelegatorTokensAtBlock", mock.Anything, selector.Bytes(), disputeOpenHeight).
		Return(math.NewInt(selectorSnapshot), nil)

	_, err = k.SetVoterReporterStake(ctx, disputeID, selector, disputeOpenHeight, types.VoteEnum_VOTE_AGAINST, nil, nil)
	require.NoError(err)

	row, err := k.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(groupSnapshot-selectorSnapshot), row.ReporterPower)

	moved, err := k.SetVoterReporterStake(ctx, disputeID, reporter, disputeOpenHeight, types.VoteEnum_VOTE_INVALID, &row, nil)
	require.NoError(err)
	require.Equal(math.NewInt(groupSnapshot-selectorSnapshot), moved)
	require.Equal(types.VoteCounts{
		Support: 0,
		Against: uint64(selectorSnapshot),
		Invalid: uint64(groupSnapshot - selectorSnapshot),
	}, s.reporterVoteCountsAt(ctx, disputeID))
}

// TestDisputeHeight_SelectedToOtherVotesOwnStake is a reporter already selected
// to someone else at the open height, with no switch of their own unfinished.
// They vote only their snapshot in that reporter. Selectors whose open-height
// reporter is this address do not peel.
func (s *KeeperTestSuite) TestDisputeHeight_SelectedToOtherVotesOwnStake() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.voteCtx()
	disputeID := uint64(11)
	reporter := sample.AccAddressBytes()
	other := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	ownStake := int64(25)

	rk.On("Delegation", mock.Anything, reporter).Return(reportertypes.Selection{Reporter: other}, nil)
	rk.On("GetSelectorForStake", mock.Anything, reporter).Return(unlockedSelection(other), nil)
	s.expectDisputeHeight(reporter, other, false, false)
	s.expectDisputeHeight(other, other, true, false)
	rk.On("GetDelegatorTokensFromReporterAtBlock", mock.Anything, reporter.Bytes(), other.Bytes(), disputeOpenHeight).
		Return(math.NewInt(ownStake), nil)

	power, err := k.SetVoterReporterStake(ctx, disputeID, reporter, disputeOpenHeight, types.VoteEnum_VOTE_SUPPORT, nil, nil)
	require.NoError(err)
	require.Equal(math.NewInt(ownStake), power)
	require.NoError(k.Voter.Set(ctx, collections.Join(disputeID, reporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    power,
		ReporterPower: power,
	}))

	rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", mock.Anything, selector).Return(unlockedSelection(reporter), nil)
	s.expectDisputeHeight(selector, reporter, true, false)
	rk.On("GetDelegatorTokensAtBlock", mock.Anything, selector.Bytes(), disputeOpenHeight).
		Return(math.NewInt(selectorSnapshot), nil)

	selectorPower, err := k.SetVoterReporterStake(ctx, disputeID, selector, disputeOpenHeight, types.VoteEnum_VOTE_AGAINST, nil, nil)
	require.NoError(err)
	require.Equal(math.NewInt(selectorSnapshot), selectorPower)
	require.Equal(types.VoteCounts{Support: uint64(ownStake), Against: uint64(selectorSnapshot)}, s.reporterVoteCountsAt(ctx, disputeID))

	owner, err := k.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(ownStake), owner.ReporterPower)
	_, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
	require.ErrorIs(err, collections.ErrNotFound)
}

// TestDisputeHeight_HistoricalZeroDoesNotFallThrough keeps a zero snapshot on the
// resolved dispute-height reporter. Live and evidence snapshots are not consulted.
func (s *KeeperTestSuite) TestDisputeHeight_HistoricalZeroDoesNotFallThrough() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.voteCtx()
	disputeID := uint64(12)
	selector := sample.AccAddressBytes()
	oldReporter := sample.AccAddressBytes()
	liveReporter := sample.AccAddressBytes()
	evidence := sample.AccAddressBytes()

	rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: liveReporter}, nil)
	rk.On("GetSelectorForStake", mock.Anything, selector).Return(unlockedSelection(liveReporter), nil)
	s.expectDisputeHeight(selector, oldReporter, false, false)
	rk.On("GetDelegatorTokensFromReporterAtBlock", mock.Anything, selector.Bytes(), oldReporter.Bytes(), disputeOpenHeight).
		Return(math.ZeroInt(), nil)

	power, err := k.SetVoterReporterStake(ctx, disputeID, selector, disputeOpenHeight, types.VoteEnum_VOTE_SUPPORT, nil, evidence)
	require.NoError(err)
	require.True(power.IsZero())
	_, err = k.VoteCountsByGroup.Get(ctx, disputeID)
	require.ErrorIs(err, collections.ErrNotFound)
}

// TestDisputeHeight_CountedRevoteWhileUnfinishedMovesStoredPower keeps a lock and
// an unfinished switch from erasing reporter power that was already counted.
func (s *KeeperTestSuite) TestDisputeHeight_CountedRevoteWhileUnfinishedMovesStoredPower() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.voteCtx()
	disputeID := uint64(13)
	selector := sample.AccAddressBytes()
	reporter := sample.AccAddressBytes()

	require.NoError(k.VoteCountsByGroup.Set(ctx, disputeID, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Support: uint64(selectorSnapshot)},
	}))
	old := &types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    math.NewInt(selectorSnapshot),
		ReporterPower: math.NewInt(selectorSnapshot),
	}
	rk.On("Delegation", mock.Anything, selector).Return(reportertypes.Selection{Reporter: reporter}, nil)
	rk.On("GetSelectorForStake", mock.Anything, selector).Return(reportertypes.Selection{
		Reporter:         reporter,
		LockedUntilTime:  ctx.BlockTime().Add(time.Hour),
		DelegationsCount: 1,
	}, nil)
	s.expectDisputeHeight(selector, reporter, true, true)

	moved, err := k.SetVoterReporterStake(ctx, disputeID, selector, disputeOpenHeight, types.VoteEnum_VOTE_INVALID, old, nil)
	require.NoError(err)
	require.Equal(math.NewInt(selectorSnapshot), moved)
	require.Equal(types.VoteCounts{Invalid: uint64(selectorSnapshot)}, s.reporterVoteCountsAt(ctx, disputeID))
}
