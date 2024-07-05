package keeper_test

import (
	"errors"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
)

func (s *KeeperTestSuite) TestInitVoterClasses() {
	require := s.Require()
	k := s.disputeKeeper

	classes := k.InitVoterClasses()
	require.True(classes.Users.IsZero())
	require.True(classes.Reporters.IsZero())
	require.True(classes.Team.IsZero())
	require.True(classes.TokenHolders.IsZero())
}

func (s *KeeperTestSuite) TestSetStartVote() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx

	exptectedStartTime := ctx.BlockTime()
	expectedEndTime := exptectedStartTime.Add(time.Hour * 48)
	require.NoError(k.SetStartVote(s.ctx, 1))

	vote, err := k.Votes.Get(s.ctx, 1)
	require.NoError(err)
	require.Equal(vote.VoteStart, exptectedStartTime)
	require.Equal(vote.VoteEnd, expectedEndTime)
	require.Equal(vote.Id, uint64(1))
}

func (s *KeeperTestSuite) TestTeamVote() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx

	teamTally, err := k.TeamVote(ctx, 1)
	require.NoError(err)
	require.Equal(teamTally, math.NewInt(0))

	teamAddr, err := k.GetTeamAddress(ctx)
	require.NoError(err)
	tally, err := k.SetTeamVote(ctx, 1, teamAddr)
	require.NoError(err)
	require.Equal(tally, math.NewInt(25000000))
	tally, err = k.SetTeamVote(ctx, 1, sample.AccAddressBytes())
	require.NoError(err)
	require.Equal(tally, math.NewInt(0))

	teamTally, err = k.TeamVote(ctx, 1)
	require.NoError(err)
	require.Equal(teamTally, math.NewInt(1))
}

func (s *KeeperTestSuite) TestGetUserTotalTips() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx

	voter := sample.AccAddressBytes()
	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, int64(1), voter).Return(math.NewInt(100), nil).Once()

	userTips, err := k.GetUserTotalTips(ctx, voter, 1)
	require.NoError(err)
	require.Equal(userTips, math.NewInt(100))

	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, int64(1), voter).Return(math.NewInt(100), collections.ErrNotFound).Once()

	userTips, err = k.GetUserTotalTips(ctx, voter, 1)
	require.NoError(err)
	require.Equal(userTips, math.NewInt(0))

	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, int64(1), voter).Return(math.NewInt(100), errors.New("error")).Once()

	userTips, err = k.GetUserTotalTips(ctx, voter, 1)
	require.Error(err)
	require.Equal(userTips, math.Int{})
}

func (s *KeeperTestSuite) TestSetVoterTips() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx

	voter := sample.AccAddressBytes()
	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, int64(1), voter).Return(math.NewInt(100), nil).Once()
	tips, err := k.SetVoterTips(ctx, uint64(1), voter, 1)
	require.NoError(err)
	require.Equal(tips, math.NewInt(100))

	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, int64(1), voter).Return(math.NewInt(0), nil).Once()
	tips, err = k.SetVoterTips(ctx, uint64(1), voter, 1)
	require.NoError(err)
	require.Equal(tips, math.NewInt(0))

	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, int64(1), voter).Return(math.NewInt(100), errors.New("error")).Once()
	tips, err = k.SetVoterTips(ctx, uint64(1), voter, 1)
	require.Error(err)
	require.Equal(tips, math.Int{})
}

func (s *KeeperTestSuite) TestSetVoterReportStake() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.ctx
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	blockNum := int64(0)
	id := uint64(1)

	// delegation not found, collections not found
	rk.On("Delegation", ctx, reporter).Return(reportertypes.Delegation{}, collections.ErrNotFound).Once()
	reporterTokens, err := k.SetVoterReporterStake(ctx, id, reporter, blockNum)
	require.NoError(err)
	require.Equal(reporterTokens, math.ZeroInt())

	// delegation not found, not collections not found err
	rk.On("Delegation", ctx, reporter).Return(reportertypes.Delegation{}, errors.New("error")).Once()
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, reporter, blockNum)
	require.Error(err)
	require.Equal(reporterTokens, math.Int{})

	// delegation found, ReportersWithDelegatorsVotedBefore empty, voter is reporter, error with GetReporterTokensAtBlock
	rk.On("Delegation", ctx, reporter).Return(reportertypes.Delegation{
		Reporter: reporter,
	}, nil).Once()
	rk.On("GetReporterTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(100*1e6), errors.New("error")).Once()
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, reporter, blockNum)
	require.Error(err)
	require.Equal(reporterTokens, math.Int{})

	// delegation found, ReportersWithDelegatorsVotedBefore empty, voter is reporter, no error with GetReporterTokensAtBlock, reportersgroup does not exist
	rk.On("Delegation", ctx, reporter).Return(reportertypes.Delegation{
		Reporter: reporter,
	}, nil).Once()
	rk.On("GetReporterTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(100*1e6), nil).Once()
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, reporter, blockNum)
	require.NoError(err)
	require.Equal(reporterTokens, math.NewInt(100*1e6))
	reporterTokens, err = k.ReportersGroup.Get(ctx, collections.Join(id, reporter.Bytes()))
	require.NoError(err)
	require.Equal(reporterTokens, math.NewInt(100*1e6))

	// delegation found, ReportersWithDelegatorsVotedBefore empty, voter is reporter, no error with GetReporterTokensAtBlock, reportersgroup exists
	rk.On("Delegation", ctx, reporter).Return(reportertypes.Delegation{
		Reporter: reporter,
	}, nil).Once()
	rk.On("GetReporterTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(100*1e6), nil).Once()
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, reporter, blockNum)
	require.NoError(err)
	require.Equal(reporterTokens, math.NewInt(100*1e6))
	reporterTokens, err = k.ReportersGroup.Get(ctx, collections.Join(id, reporter.Bytes()))
	require.NoError(err)
	require.Equal(reporterTokens, math.NewInt(100*1e6))

	// delegation found, ReportersWithDelegatorsVotedBefore empty, voter is delegator, error with GetDelegatorTokensAtBlock
	rk.On("Delegation", ctx, selector).Return(reportertypes.Delegation{
		Reporter: reporter,
	}, nil).Once()
	rk.On("GetDelegatorTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(50*1e6), errors.New("error")).Once()
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, selector, blockNum)
	require.Error(err)
	require.Equal(reporterTokens, math.Int{})

	// delegation found, ReportersWithDelegatorsVotedBefore empty, voter is delegator, no error with GetDelegatorTokensAtBlock, reportersgroup set, voter not set
	rk.On("Delegation", ctx, selector).Return(reportertypes.Delegation{
		Reporter: reporter,
	}, nil).Once()
	rk.On("GetDelegatorTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(50*1e6), nil).Once()
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, selector, blockNum)
	require.Error(err)
	require.Equal(reporterTokens, math.Int{})

	// delegation found, ReportersWithDelegatorsVotedBefore empty, voter is selector, no error with GetDelegatorTokensAtBlock, reportersgroup set, voter set
	rk.On("Delegation", ctx, selector).Return(reportertypes.Delegation{
		Reporter: reporter,
	}, nil).Once()
	rk.On("GetDelegatorTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(50*1e6), nil).Once()
	require.NoError(k.Voter.Set(ctx, collections.Join(id, reporter.Bytes()), types.Voter{
		Vote:       types.VoteEnum_VOTE_SUPPORT,
		VoterPower: math.NewInt(50 * 1e6),
	}))
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, selector, blockNum)
	require.NoError(err)
	require.Equal(reporterTokens, math.NewInt(50*1e6))

	// clear ReportersGroup to get outside exists block
	require.NoError(k.ReportersGroup.Remove(ctx, collections.Join(id, reporter.Bytes())))
	// delegation found, ReportersWithDelegatorsVotedBefore empty, voter is selector, no error with GetDelegatorTokensAtBlock, reportersgroup empty
	rk.On("Delegation", ctx, selector).Return(reportertypes.Delegation{
		Reporter: reporter,
	}, nil).Once()
	rk.On("GetDelegatorTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(50*1e6), nil).Once()
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, selector, blockNum)
	require.NoError(err)
	require.Equal(reporterTokens, math.NewInt(50*1e6))
	// clear ReportersWithDelegatorsVotedBefore
	require.NoError(k.ReportersWithDelegatorsVotedBefore.Remove(ctx, collections.Join(selector.Bytes(), id)))

	// delegation found, ReportersWithDelegatorsVotedBefore set (selector has voted), voter is reporter, error with GetReporterTokensAtBlock
	rk.On("Delegation", ctx, reporter).Return(reportertypes.Delegation{
		Reporter: reporter,
	}, nil).Once()
	require.NoError(k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(selector.Bytes(), id), math.NewInt(50*1e6)))
	rk.On("GetReporterTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(100*1e6), errors.New("error")).Once()
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, reporter, blockNum)
	require.Error(err)
	require.Equal(reporterTokens, math.Int{})

	// delegation found, ReportersWithDelegatorsVotedBefore set (selector has voted), voter is reporter, error with GetReporterTokensAtBlock
	rk.On("Delegation", ctx, reporter).Return(reportertypes.Delegation{
		Reporter: reporter,
	}, nil).Once()
	require.NoError(k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(selector.Bytes(), id), math.NewInt(50*1e6)))
	rk.On("GetReporterTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(100*1e6), nil).Once()
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, reporter, blockNum)
	require.NoError(err)
	require.Equal(reporterTokens, math.NewInt(100*1e6).Sub(math.NewInt(50*1e6)))

	// delegation found, ReportersWithDelegatorsBefore set (selector has voted), voter is selector, no error with GetDelegatorTokensAtBlock, selctor has voted with all of their tokens already
	rk.On("Delegation", ctx, selector).Return(reportertypes.Delegation{
		Reporter: reporter,
	}, nil).Once()
	require.NoError(k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(selector.Bytes(), id), math.NewInt(50*1e6)))
	rk.On("GetDelegatorTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(50*1e6), nil).Once()
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, selector, blockNum)
	require.NoError(err)
	require.Equal(reporterTokens, math.ZeroInt())
	selectorTokens, err := k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(selector.Bytes(), id))
	require.NoError(err)
	require.Equal(selectorTokens, math.NewInt(50*1e6))

	// delegation found, ReportersWithDelegatorsBefore set (selector has voted), voter is selector, no error with GetDelegatorTokensAtBlock, selctor has voted already
	rk.On("Delegation", ctx, selector).Return(reportertypes.Delegation{
		Reporter: reporter,
	}, nil).Once()
	require.NoError(k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(selector.Bytes(), id), math.NewInt(50*1e6)))
	rk.On("GetDelegatorTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(60*1e6), nil).Once()
	reporterTokens, err = k.SetVoterReporterStake(ctx, id, selector, blockNum)
	require.NoError(err)
	require.Equal(reporterTokens, math.ZeroInt())
	selectorTokens, err = k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), id))
	require.NoError(err)
	require.Equal(selectorTokens, math.NewInt(160*1e6))
}

func (s *KeeperTestSuite) TestUpdateDispute() {
	k := s.disputeKeeper
	ctx := s.ctx
	require := s.Require()
	require.NotNil(k)
	require.NotNil(ctx)
}
