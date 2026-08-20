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

const (
	msgVotePeelReporterTokens = uint64(100)
	msgVotePeelSelectorTokens = uint64(30)
)

func (s *KeeperTestSuite) requireNotVoteCountUnderflow(err error) {
	s.Require().NoError(err)
	s.Require().NotErrorIs(err, types.ErrVoteCountUnderflow)
}

func (s *KeeperTestSuite) setupMsgVoteDispute(disputeID, blockNum uint64, hashID []byte) {
	require := s.Require()
	ctx := s.ctx.WithBlockHeight(int64(blockNum)).WithBlockTime(time.Now().UTC())
	s.ctx = ctx

	require.NoError(s.disputeKeeper.Disputes.Set(ctx, disputeID, types.Dispute{
		HashId:        hashID,
		DisputeId:     disputeID,
		BlockNumber:   blockNum,
		Open:          true,
		DisputeStatus: types.Voting,
	}))
	require.NoError(s.disputeKeeper.Votes.Set(ctx, disputeID, types.Vote{
		Id:         disputeID,
		VoteResult: types.VoteResult_NO_TALLY,
		VoteStart:  ctx.BlockTime(),
		VoteEnd:    ctx.BlockTime().Add(48 * time.Hour),
	}))
	require.NoError(s.disputeKeeper.BlockInfo.Set(ctx, hashID, types.BlockInfo{
		TotalReporterPower: math.NewInt(1000),
		TotalUserTips:      math.NewInt(1000),
	}))
}

func (s *KeeperTestSuite) mockNoTips(voter sdk.AccAddress, blockNum uint64) {
	s.oracleKeeper.On("GetTipsAtBlockForTipper", mock.Anything, blockNum, voter).
		Return(math.ZeroInt(), collections.ErrNotFound)
}

// TestMsgVote_PeelThenReporterRevote exercises the full MsgVote path for:
//  1. reporter first-votes SUPPORT (100 tokens)
//  2. selector peels 30 tokens to AGAINST
//  3. reporter revotes INVALID
//
// Buckets must end at 0/30/70 without ever hitting ErrVoteCountUnderflow.
func (s *KeeperTestSuite) TestMsgVote_PeelThenReporterRevote() {
	require := s.Require()
	ctx := s.ctx
	disputeID := uint64(50)
	blockNum := uint64(10)
	hashID := []byte("msgvote-peel-revote")
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	s.setupMsgVoteDispute(disputeID, blockNum, hashID)

	s.reporterKeeper.On("Delegation", mock.Anything, reporter).
		Return(reportertypes.Selection{Reporter: reporter}, nil)
	s.reporterKeeper.On("GetReporterTokensAtBlock", mock.Anything, reporter.Bytes(), blockNum).
		Return(math.NewIntFromUint64(msgVotePeelReporterTokens), nil)
	s.mockNoTips(reporter, blockNum)

	_, err := s.msgServer.Vote(ctx, &types.MsgVote{
		Voter: reporter.String(),
		Id:    disputeID,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	})
	s.requireNotVoteCountUnderflow(err)

	reporterRow, err := s.disputeKeeper.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.Equal(types.VoteEnum_VOTE_SUPPORT, reporterRow.Vote)
	require.Equal(math.NewIntFromUint64(msgVotePeelReporterTokens), reporterRow.ReporterPower)

	afterReporter := s.reporterVoteCounts(disputeID)
	require.Equal(types.VoteCounts{Support: msgVotePeelReporterTokens}, afterReporter)

	s.reporterKeeper.On("Delegation", mock.Anything, selector).
		Return(reportertypes.Selection{Reporter: reporter}, nil)
	s.reporterKeeper.On("GetSelectorForStake", mock.Anything, selector).
		Return(unlockedSelection(reporter), nil)
	s.reporterKeeper.On("GetDelegatorTokensAtBlock", mock.Anything, selector.Bytes(), blockNum).
		Return(math.NewIntFromUint64(msgVotePeelSelectorTokens), nil)
	s.mockNoTips(selector, blockNum)

	_, err = s.msgServer.Vote(ctx, &types.MsgVote{
		Voter: selector.String(),
		Id:    disputeID,
		Vote:  types.VoteEnum_VOTE_AGAINST,
	})
	s.requireNotVoteCountUnderflow(err)

	afterPeel := s.reporterVoteCounts(disputeID)
	require.Equal(types.VoteCounts{Support: 70, Against: 30}, afterPeel)

	reporterRow, err = s.disputeKeeper.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.Equal(math.NewInt(70), reporterRow.ReporterPower,
		"peel via MsgVote must decrement the reporter's stored ReporterPower")

	reserve, err := s.disputeKeeper.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), disputeID))
	require.NoError(err)
	require.Equal(math.NewIntFromUint64(msgVotePeelSelectorTokens), reserve,
		"peel via MsgVote must add selector stake to the reserve")

	selectorRow, err := s.disputeKeeper.Voter.Get(ctx, collections.Join(disputeID, selector.Bytes()))
	require.NoError(err)
	require.Equal(types.VoteEnum_VOTE_AGAINST, selectorRow.Vote)
	require.Equal(math.NewIntFromUint64(msgVotePeelSelectorTokens), selectorRow.ReporterPower)

	// Revote uses stored ReporterPower; GetReporterTokensAtBlock must not be required.
	s.mockNoTips(reporter, blockNum)

	_, err = s.msgServer.Vote(ctx, &types.MsgVote{
		Voter: reporter.String(),
		Id:    disputeID,
		Vote:  types.VoteEnum_VOTE_INVALID,
	})
	s.requireNotVoteCountUnderflow(err)

	final := s.reporterVoteCounts(disputeID)
	s.requireNoUint64Wrap(final.Support, "Reporters.Support")
	require.Equal(types.VoteCounts{Support: 0, Against: 30, Invalid: 70}, final)

	reporterRow, err = s.disputeKeeper.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.Equal(types.VoteEnum_VOTE_INVALID, reporterRow.Vote)
	require.Equal(math.NewInt(70), reporterRow.ReporterPower)

	dispute, err := s.disputeKeeper.Disputes.Get(ctx, disputeID)
	require.NoError(err)
	require.Equal(types.Voting, dispute.DisputeStatus,
		"correct accounting must not manufacture quorum via a bucket wrap")
}

// TestSubtractReporterVoteCount_ErrVoteCountUnderflowOnCorruptedState asserts that
// subtracting more than a bucket holds returns ErrVoteCountUnderflow instead of wrapping.
func (s *KeeperTestSuite) TestSubtractReporterVoteCount_ErrVoteCountUnderflowOnCorruptedState() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx
	disputeID := uint64(51)

	require.NoError(k.VoteCountsByGroup.Set(ctx, disputeID, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Support: 70},
	}))

	err := k.SubtractReporterVoteCount(ctx, disputeID, 100, types.VoteEnum_VOTE_SUPPORT)
	require.ErrorIs(err, types.ErrVoteCountUnderflow)

	counts := s.reporterVoteCounts(disputeID)
	require.Equal(uint64(70), counts.Support, "underflow must leave the bucket unchanged")
}

// TestMsgVote_ReporterRevote_ErrVoteCountUnderflowOnCorruptedState seeds the pre-fix
// failure mode: buckets reflect a peel (70 SUPPORT) but the reporter row still
// records ReporterPower=100. A revote must fail closed with ErrVoteCountUnderflow.
func (s *KeeperTestSuite) TestMsgVote_ReporterRevote_ErrVoteCountUnderflowOnCorruptedState() {
	require := s.Require()
	ctx := s.ctx
	disputeID := uint64(52)
	blockNum := uint64(10)
	hashID := []byte("msgvote-corrupt-revote")
	reporter := sample.AccAddressBytes()

	s.setupMsgVoteDispute(disputeID, blockNum, hashID)

	require.NoError(s.disputeKeeper.VoteCountsByGroup.Set(ctx, disputeID, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Support: 70, Against: 30},
	}))
	require.NoError(s.disputeKeeper.Voter.Set(ctx, collections.Join(disputeID, reporter.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    math.NewIntFromUint64(msgVotePeelReporterTokens),
		ReporterPower: math.NewIntFromUint64(msgVotePeelReporterTokens),
	}))

	s.reporterKeeper.On("Delegation", mock.Anything, reporter).
		Return(reportertypes.Selection{Reporter: reporter}, nil)
	s.mockNoTips(reporter, blockNum)

	_, err := s.msgServer.Vote(ctx, &types.MsgVote{
		Voter: reporter.String(),
		Id:    disputeID,
		Vote:  types.VoteEnum_VOTE_INVALID,
	})
	require.ErrorIs(err, types.ErrVoteCountUnderflow)

	final := s.reporterVoteCounts(disputeID)
	require.Equal(types.VoteCounts{Support: 70, Against: 30, Invalid: 0}, final,
		"failed revote must not mutate buckets")

	reporterRow, err := s.disputeKeeper.Voter.Get(ctx, collections.Join(disputeID, reporter.Bytes()))
	require.NoError(err)
	require.Equal(types.VoteEnum_VOTE_SUPPORT, reporterRow.Vote,
		"failed revote must not update the stored voter row")
}
