package keeper_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestInitVoterClasses() {
	require := s.Require()
	k := s.disputeKeeper

	classes := k.InitVoterClasses()
	require.True(classes.Users.IsZero())
	require.True(classes.Reporters.IsZero())
	require.True(classes.Team.IsZero())
}

func (s *KeeperTestSuite) TestSetStartVote() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx
	ctx = ctx.WithBlockTime(time.Now())

	// start vote with disputeId 1, expected end is 2 days after start
	expectedStartTime := ctx.BlockTime()
	expectedEndTime := expectedStartTime.Add(time.Hour * 48)
	require.NoError(k.SetStartVote(ctx, 1))
	// check on vote
	vote, err := k.Votes.Get(ctx, 1)
	require.NoError(err)
	require.Equal(vote.VoteStart, expectedStartTime)
	require.Equal(vote.VoteEnd, expectedEndTime)
	require.Equal(vote.Id, uint64(1))

	// start vote with max disputeId, expected end is 2 days after start
	ctx = ctx.WithBlockTime(time.Now().Add(1 * time.Hour))
	expectedStartTime = ctx.BlockTime()
	expectedEndTime = ctx.BlockTime().Add(time.Hour * 48)
	maxUint64 := ^uint64(0)
	require.NoError(k.SetStartVote(ctx, maxUint64))
	// check on vote
	vote, err = k.Votes.Get(ctx, maxUint64)
	require.NoError(err)
	require.Equal(vote.VoteStart, expectedStartTime)
	require.Equal(vote.VoteEnd, expectedEndTime)
	require.Equal(vote.Id, maxUint64)
}

func (s *KeeperTestSuite) TestTeamVote_SetTeamVote() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx

	disputeId := uint64(1)

	// team votes SUPPORT
	teamAddr, err := k.GetTeamAddress(ctx)
	require.NoError(err)
	teamVote, err := k.SetTeamVote(ctx, disputeId, teamAddr, types.VoteEnum_VOTE_SUPPORT)
	require.NoError(err)
	require.Equal(teamVote, math.NewInt(100000000).Quo(math.NewInt(3)))
	// check on vote
	votesByGroup, err := k.VoteCountsByGroup.Get(ctx, disputeId)
	require.Equal(votesByGroup.Team.Against, uint64(0))
	require.Equal(votesByGroup.Team.Support, uint64(1))
	require.Equal(votesByGroup.Team.Invalid, uint64(0))
	require.NoError(err)

	// vote from bad account, expect return 0
	badTeamVote, err := k.SetTeamVote(ctx, disputeId, sample.AccAddressBytes(), types.VoteEnum_VOTE_SUPPORT)
	require.NoError(err)
	require.Equal(badTeamVote, math.NewInt(0))

	// note: voters can only vote once
}

func (s *KeeperTestSuite) TestGetUserTotalTips() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx

	// voter who has tipped 100 trb, expect return 100
	voter := sample.AccAddressBytes()
	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, uint64(1), voter).Return(math.NewInt(100), nil).Once()
	userTips, err := k.GetUserTotalTips(ctx, voter, 1)
	require.NoError(err)
	require.Equal(userTips, math.NewInt(100))

	// get voter who has not tipped before, expect return 0
	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, uint64(1), voter).Return(math.ZeroInt(), collections.ErrNotFound).Once()
	userTips, err = k.GetUserTotalTips(ctx, voter, 1)
	require.NoError(err)
	require.Equal(userTips, math.ZeroInt())

	// get non collections not found err, expect return math.Int{}
	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, uint64(1), voter).Return(math.Int{}, errors.New("error")).Once()
	userTips, err = k.GetUserTotalTips(ctx, voter, 1)
	require.Error(err)
	require.Equal(userTips, math.Int{})
}

func (s *KeeperTestSuite) TestSetVoterTips() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx
	ctx = ctx.WithBlockHeight(10)

	disputeId := uint64(1)
	blockNum := uint64(ctx.BlockHeight())
	// user1 has tipped 100 trb, votes support
	user1 := sample.AccAddressBytes()
	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, blockNum, user1).Return(math.NewInt(100), nil).Once()
	tips, err := k.SetVoterTips(ctx, disputeId, user1, uint64(10), types.VoteEnum_VOTE_SUPPORT)
	require.NoError(err)
	require.Equal(tips, math.NewInt(100))
	// check on vote
	votesByGroup, err := k.VoteCountsByGroup.Get(ctx, disputeId)
	require.Equal(votesByGroup.Users.Against, uint64(0))
	require.Equal(votesByGroup.Users.Support, uint64(100))
	require.Equal(votesByGroup.Users.Invalid, uint64(0))
	require.NoError(err)

	// user2 has tipped 200 trb, votes against
	user2 := sample.AccAddressBytes()
	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, blockNum, user2).Return(math.NewInt(200), nil).Once()
	tips, err = k.SetVoterTips(ctx, disputeId, user2, uint64(10), types.VoteEnum_VOTE_AGAINST)
	require.NoError(err)
	require.Equal(tips, math.NewInt(200))
	// check on vote
	votesByGroup, err = k.VoteCountsByGroup.Get(ctx, disputeId)
	require.Equal(votesByGroup.Users.Against, uint64(200))
	require.Equal(votesByGroup.Users.Support, uint64(100))
	require.Equal(votesByGroup.Users.Invalid, uint64(0))
	require.NoError(err)

	// nonUser, expect return 0
	nonUser := sample.AccAddressBytes()
	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, blockNum, nonUser).Return(math.NewInt(0), nil).Once()
	tips, err = k.SetVoterTips(ctx, disputeId, nonUser, blockNum, types.VoteEnum_VOTE_SUPPORT)
	require.NoError(err)
	require.Equal(tips, math.NewInt(0))

	// non collections not found error on GetUserTotalTips, expect return math.Int{}, err
	s.oracleKeeper.On("GetTipsAtBlockForTipper", ctx, blockNum, nonUser).Return(math.NewInt(0), errors.New("error")).Once()
	tips, err = k.SetVoterTips(ctx, disputeId, nonUser, blockNum, types.VoteEnum_VOTE_SUPPORT)
	require.Error(err)
	require.Equal(tips, math.Int{})
}

func (s *KeeperTestSuite) TestSetVoterReportStake() {
	require := s.Require()
	k := s.disputeKeeper
	rk := s.reporterKeeper
	ctx := s.ctx
	ctx = ctx.WithBlockHeight(10)
	ctx = ctx.WithBlockTime(time.Now())

	blockNum := uint64(10)
	disputeId := uint64(1)
	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()

	testCases := []struct {
		name           string
		voter          sdk.AccAddress
		setup          func()
		expectedError  bool
		expectedTokens math.Int
		expectedVote   types.StakeholderVoteCounts
		teardown       func()
	}{
		{
			name:  "delegation not found, collections not found",
			voter: reporter,
			setup: func() {
				rk.On("Delegation", ctx, reporter).Return(reportertypes.Selection{}, collections.ErrNotFound).Once()
			},
			expectedError:  false,
			expectedTokens: math.ZeroInt(),
			expectedVote:   types.StakeholderVoteCounts{},
		},
		{
			name:  "delegation not found, not collections not found",
			voter: reporter,
			setup: func() {
				rk.On("Delegation", ctx, reporter).Return(reportertypes.Selection{}, errors.New("error!")).Once()
			},
			expectedError:  true,
			expectedTokens: math.Int{},
			expectedVote:   types.StakeholderVoteCounts{},
		},
		{
			name:  "voter is reporter, error on GetReporterTokensAtBlock",
			voter: reporter,
			setup: func() {
				rk.On("Delegation", ctx, reporter).Return(reportertypes.Selection{
					Reporter: reporter,
				}, nil).Once()
				rk.On("GetReporterTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.Int{}, errors.New("error!")).Once()
			},
			expectedError:  true,
			expectedTokens: math.Int{},
			expectedVote:   types.StakeholderVoteCounts{},
		},
		{
			name:  "voter is reporter, hasnt voted with any reporter tokens",
			voter: reporter,
			setup: func() {
				rk.On("Delegation", ctx, reporter).Return(reportertypes.Selection{
					Reporter: reporter,
				}, nil).Once()
				// reporter has 100 tokens, hasnt voted with any of them
				rk.On("GetReporterTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(100), nil).Once()
			},
			expectedError:  false,
			expectedTokens: math.NewInt(100),
			expectedVote: types.StakeholderVoteCounts{
				Reporters: types.VoteCounts{
					Support: 100,
					Against: 0,
					Invalid: 0,
				},
			},
			teardown: func() {
				require.NoError(k.VoteCountsByGroup.Remove(ctx, disputeId))
			},
		},
		{
			name:  "voter is reporter, has voted with some reporter tokens",
			voter: reporter,
			setup: func() {
				rk.On("Delegation", ctx, reporter).Return(reportertypes.Selection{
					Reporter: reporter,
				}, nil).Once()
				// reporter has 100 tokens, has voted with 50 of them
				rk.On("GetReporterTokensAtBlock", ctx, reporter.Bytes(), blockNum).Return(math.NewInt(100), nil).Once()
				require.NoError(k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(reporter.Bytes(), disputeId), math.NewInt(50)))
			},
			expectedError:  false,
			expectedTokens: math.NewInt(50),
			expectedVote: types.StakeholderVoteCounts{
				Reporters: types.VoteCounts{
					Support: 50,
					Against: 0,
					Invalid: 0,
				},
			},
			teardown: func() {
				require.NoError(k.ReportersWithDelegatorsVotedBefore.Remove(ctx, collections.Join(reporter.Bytes(), disputeId)))
				require.NoError(k.VoteCountsByGroup.Remove(ctx, disputeId))
			},
		},
		{
			name:  "voter is selector, reporter has not voted",
			voter: selector,
			setup: func() {
				rk.On("Delegation", ctx, selector).Return(reportertypes.Selection{
					Reporter: reporter,
				}, nil).Once()
				// selector has 100 selected to reporter
				rk.On("GetDelegatorTokensAtBlock", ctx, selector.Bytes(), blockNum).Return(math.NewInt(100), nil).Once()
				rk.On("GetSelector", ctx, selector).Return(reportertypes.Selection{
					Reporter:         reporter,
					LockedUntilTime:  ctx.BlockTime().Add(time.Hour * -24),
					DelegationsCount: 10,
				}, nil).Once()
			},
			expectedError:  false,
			expectedTokens: math.NewInt(100),
			expectedVote: types.StakeholderVoteCounts{
				Reporters: types.VoteCounts{
					Support: 100,
					Against: 0,
					Invalid: 0,
				},
			},
			teardown: func() {
				require.NoError(k.VoteCountsByGroup.Remove(ctx, disputeId))
				require.NoError(k.ReportersWithDelegatorsVotedBefore.Remove(ctx, collections.Join(reporter.Bytes(), disputeId)))
			},
		},
		{
			name:  "voter is selector, reporter has voted",
			voter: selector,
			setup: func() {
				rk.On("Delegation", ctx, selector).Return(reportertypes.Selection{
					Reporter: reporter,
				}, nil).Once()
				// selector has 100 selected to reporter
				rk.On("GetDelegatorTokensAtBlock", ctx, selector.Bytes(), blockNum).Return(math.NewInt(100), nil).Once()
				// reporter has voted against with 150
				require.NoError(k.Voter.Set(ctx, collections.Join(disputeId, reporter.Bytes()), types.Voter{
					Vote:       types.VoteEnum_VOTE_AGAINST,
					VoterPower: math.NewInt(150),
				}))
				require.NoError(k.VoteCountsByGroup.Set(ctx, disputeId, types.StakeholderVoteCounts{
					Reporters: types.VoteCounts{
						Against: 150,
					},
				}))
				require.NoError(k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(reporter.Bytes(), disputeId), math.NewInt(50)))
				rk.On("GetSelector", ctx, selector).Return(reportertypes.Selection{
					Reporter:         reporter,
					LockedUntilTime:  ctx.BlockTime().Add(time.Hour * -24),
					DelegationsCount: 10,
				}, nil).Once()
			},
			expectedError:  false,
			expectedTokens: math.NewInt(100),
			expectedVote: types.StakeholderVoteCounts{
				Reporters: types.VoteCounts{
					Support: 100,
					Against: 50,
					Invalid: 0,
				},
			},
			teardown: func() {
				require.NoError(k.ReportersWithDelegatorsVotedBefore.Remove(ctx, collections.Join(reporter.Bytes(), disputeId)))
				require.NoError(k.VoteCountsByGroup.Remove(ctx, disputeId))
			},
		},
		{
			name:  "voter is selector, selector has recently switched reporters",
			voter: selector,
			setup: func() {
				rk.On("Delegation", ctx, selector).Return(reportertypes.Selection{
					Reporter: reporter,
				}, nil).Once()
				// selector has 100 selected to reporter
				rk.On("GetDelegatorTokensAtBlock", ctx, selector.Bytes(), blockNum).Return(math.NewInt(100), nil).Once()
				rk.On("GetSelector", ctx, selector).Return(reportertypes.Selection{
					Reporter:         reporter,
					LockedUntilTime:  ctx.BlockTime().Add(time.Hour * 24),
					DelegationsCount: 10,
				}, nil).Once()
			},
			expectedError:  false,
			expectedTokens: math.ZeroInt(),
			expectedVote:   types.StakeholderVoteCounts{},
			teardown: func() {
				require.NoError(k.VoteCountsByGroup.Remove(ctx, disputeId))
				require.NoError(k.ReportersWithDelegatorsVotedBefore.Remove(ctx, collections.Join(reporter.Bytes(), disputeId)))
			},
		},
	}
	for _, tc := range testCases {
		fmt.Println(tc.name)
		if tc.setup != nil {
			s.Run(tc.name, tc.setup)
		}
		tokensVoted, err := k.SetVoterReporterStake(ctx, disputeId, tc.voter, blockNum, types.VoteEnum_VOTE_SUPPORT)
		if tc.expectedError {
			require.Error(err)
		} else {
			require.NoError(err)
		}
		fmt.Println("tokensVoted: ", tokensVoted)
		fmt.Println("expectedTokens: ", tc.expectedTokens)
		require.Equal(tc.expectedTokens, tokensVoted)
		if tc.expectedVote != (types.StakeholderVoteCounts{}) {
			votesByGroup, err := k.VoteCountsByGroup.Get(ctx, disputeId)
			fmt.Println("votesByGroup", votesByGroup)
			require.Equal(tc.expectedVote, votesByGroup)
			require.NoError(err)
		}
		if tc.teardown != nil {
			s.Run(tc.name, tc.teardown)
		}
	}
}

// func (s *KeeperTestSuite) TestSetTokenholderVote() {
// 	require := s.Require()
// 	k := s.disputeKeeper
// 	bk := s.bankKeeper
// 	rk := s.reporterKeeper
// 	ctx := s.ctx
// 	ctx = ctx.WithBlockHeight(10)

// 	disputeId := uint64(1)
// 	blockNum := uint64(10)
// 	tokenHolder := sample.AccAddressBytes()
// 	// reporter := sample.AccAddressBytes()

// 	testCases := []struct {
// 		name           string
// 		voter          sdk.AccAddress
// 		setup          func()
// 		expectedError  bool
// 		expectedTokens math.Int
// 		expectedVote   types.StakeholderVoteCounts
// 		teardown       func()
// 	}{
// 		{
// 			name:  "err from GetDelegatorTokensAtBlock ",
// 			voter: tokenHolder,
// 			setup: func() {
// 				// 100 free floating
// 				bk.On("GetBalance", ctx, tokenHolder, layertypes.BondDenom).Return(sdk.Coin{
// 					Denom:  layertypes.BondDenom,
// 					Amount: math.NewInt(100),
// 				}, nil).Once()
// 				rk.On("GetDelegatorTokensAtBlock", ctx, tokenHolder.Bytes(), blockNum).Return(math.Int{}, errors.New("error!")).Once()
// 			},
// 			expectedError:  true,
// 			expectedTokens: math.Int{},
// 			expectedVote:   types.StakeholderVoteCounts{},
// 			teardown:       nil,
// 		},
// 		{
// 			name:  "err from VoteCountsByGroup",
// 			voter: tokenHolder,
// 			setup: func() {
// 				// 100 free floating
// 				bk.On("GetBalance", ctx, tokenHolder, layertypes.BondDenom).Return(sdk.Coin{
// 					Denom:  layertypes.BondDenom,
// 					Amount: math.NewInt(100),
// 				}, nil).Once()
// 				rk.On("GetDelegatorTokensAtBlock", ctx, tokenHolder.Bytes(), blockNum).Return(math.ZeroInt(), errors.New("error!")).Once()
// 			},
// 			expectedError:  true,
// 			expectedTokens: math.Int{},
// 			expectedVote:   types.StakeholderVoteCounts{},
// 			teardown:       nil,
// 		},
// 		{
// 			name:  "no delegated token, vote success",
// 			voter: tokenHolder,
// 			setup: func() {
// 				// 200 free floating, 0 delegated, 0 selected
// 				bk.On("GetBalance", ctx, tokenHolder, layertypes.BondDenom).Return(sdk.Coin{
// 					Denom:  layertypes.BondDenom,
// 					Amount: math.NewInt(200),
// 				}, nil).Once()
// 				rk.On("GetDelegatorTokensAtBlock", ctx, tokenHolder.Bytes(), blockNum).Return(math.ZeroInt(), nil).Once()
// 			},
// 			expectedError:  false,
// 			expectedTokens: math.NewInt(200),
// 			expectedVote: types.StakeholderVoteCounts{
// 				Tokenholders: types.VoteCounts{
// 					Support: 200,
// 				},
// 			},
// 			teardown: func() {
// 				require.NoError(k.VoteCountsByGroup.Remove(ctx, disputeId))
// 			},
// 		},
// 		{
// 			name:  "delegated token, vote success",
// 			voter: tokenHolder,
// 			setup: func() {
// 				// 200 free floating, 100 delegated, 10 selected
// 				bk.On("GetBalance", ctx, tokenHolder, layertypes.BondDenom).Return(sdk.Coin{
// 					Denom:  layertypes.BondDenom,
// 					Amount: math.NewInt(200),
// 				}, nil).Once()
// 				rk.On("GetDelegatorTokensAtBlock", ctx, tokenHolder.Bytes(), blockNum).Return(math.NewInt(100), nil).Once()
// 			},
// 			expectedError:  false,
// 			expectedTokens: math.NewInt(300),
// 			expectedVote: types.StakeholderVoteCounts{
// 				Tokenholders: types.VoteCounts{
// 					Support: 300,
// 				},
// 			},
// 			teardown: func() {
// 				require.NoError(k.VoteCountsByGroup.Remove(ctx, disputeId))
// 			},
// 		},
// 	}
// 	for _, tc := range testCases {
// 		if tc.setup != nil {
// 			tc.setup()
// 		}
// 		tokensVoted, err := k.SetTokenholderVote(ctx, disputeId, tc.voter, blockNum, types.VoteEnum_VOTE_SUPPORT)
// 		if tc.expectedError {
// 			require.Error(err)
// 		} else {
// 			require.NoError(err)
// 		}
// 		require.Equal(tokensVoted, tc.expectedTokens)
// 		if tc.expectedVote != (types.StakeholderVoteCounts{}) {
// 			votesByGroup, err := k.VoteCountsByGroup.Get(ctx, disputeId)
// 			require.Equal(votesByGroup, tc.expectedVote)
// 			require.NoError(err)
// 		}
// 		if tc.teardown != nil {
// 			tc.teardown()
// 		}
// 	}
// }

func (s *KeeperTestSuite) TestAddAndSubtractReporterVoteCount() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx

	ctx = ctx.WithBlockHeight(10)
	ctx = ctx.WithBlockTime(time.Now())
	disputeId := uint64(1)

	// add 100 support
	err := k.AddReporterVoteCount(ctx, disputeId, 100, types.VoteEnum_VOTE_SUPPORT)
	require.NoError(err)
	votesByGroup, err := k.VoteCountsByGroup.Get(ctx, disputeId)
	require.NoError(err)
	require.Equal(votesByGroup.Reporters.Support, uint64(100))

	err = k.SubtractReporterVoteCount(ctx, disputeId, 100, types.VoteEnum_VOTE_SUPPORT)
	require.NoError(err)
	votesByGroup, err = k.VoteCountsByGroup.Get(ctx, disputeId)
	require.NoError(err)
	require.Equal(votesByGroup.Reporters.Support, uint64(0))

	err = k.AddReporterVoteCount(ctx, disputeId, 100, types.VoteEnum_VOTE_AGAINST)
	require.NoError(err)
	votesByGroup, err = k.VoteCountsByGroup.Get(ctx, disputeId)
	require.NoError(err)
	require.Equal(votesByGroup.Reporters.Against, uint64(100))

	err = k.SubtractReporterVoteCount(ctx, disputeId, 100, types.VoteEnum_VOTE_AGAINST)
	require.NoError(err)
	votesByGroup, err = k.VoteCountsByGroup.Get(ctx, disputeId)
	require.NoError(err)
	require.Equal(votesByGroup.Reporters.Against, uint64(0))

	err = k.AddReporterVoteCount(ctx, disputeId, 100, types.VoteEnum_VOTE_INVALID)
	require.NoError(err)
	votesByGroup, err = k.VoteCountsByGroup.Get(ctx, disputeId)
	require.NoError(err)
	require.Equal(votesByGroup.Reporters.Invalid, uint64(100))

	err = k.SubtractReporterVoteCount(ctx, disputeId, 100, types.VoteEnum_VOTE_INVALID)
	require.NoError(err)
	votesByGroup, err = k.VoteCountsByGroup.Get(ctx, disputeId)
	require.NoError(err)
	require.Equal(votesByGroup.Reporters.Invalid, uint64(0))
}
