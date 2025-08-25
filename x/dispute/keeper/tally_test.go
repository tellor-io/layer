package keeper_test

import (
	"errors"
	"fmt"

	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestGetVotersExist() {
	require := s.Require()
	ctx := s.ctx
	require.NotNil(ctx)
	k := s.disputeKeeper
	require.NotNil(k)

	voter := sample.AccAddressBytes()
	require.NoError(k.Voter.Set(ctx, collections.Join(uint64(1), voter.Bytes()), types.Voter{
		Vote:          1,
		VoterPower:    math.NewInt(100),
		ReporterPower: math.NewInt(100),
		RewardClaimed: false,
	}))

	// 1 voter
	_, err := k.GetVotersExist(ctx, 1)
	require.NoError(err)
}

// func (s *KeeperTestSuite) TestGetVoters() {
// 	require := s.Require()
// 	ctx := s.ctx
// 	k := s.disputeKeeper
// 	require.NotNil(k)
// 	require.NotNil(ctx)

// 	res, err := s.disputeKeeper.GetVotersExist(ctx, 1)
// 	require.Empty(res)
// 	require.NoError(err)

// 	voter := sample.AccAddressBytes()
// 	require.NoError(k.Voter.Set(ctx, collections.Join(uint64(1), voter.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.OneInt()}))

// 	res, err = s.disputeKeeper.GetVotersExist(ctx, 1)
// 	require.NoError(err)
// 	require.Equal(res[0].Value.Vote, types.VoteEnum_VOTE_SUPPORT)
// 	require.Equal(res[0].Value.VoterPower, math.OneInt())

// 	voter2 := sample.AccAddressBytes()
// 	require.NoError(k.Voter.Set(ctx, collections.Join(uint64(1), voter2.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.OneInt()}))

// 	res, err = s.disputeKeeper.GetVotersExist(ctx, 1)
// 	require.NoError(err)
// 	require.Equal(res[0].Value.Vote, types.VoteEnum_VOTE_SUPPORT)
// 	require.Equal(res[0].Value.VoterPower, math.OneInt())
// 	require.Equal(res[1].Value.Vote, types.VoteEnum_VOTE_SUPPORT)
// 	require.Equal(res[1].Value.VoterPower, math.OneInt())
// }

func (s *KeeperTestSuite) TestGetAccountBalance() {
	require := s.Require()
	ctx := s.ctx
	k := s.disputeKeeper
	require.NotNil(k)
	require.NotNil(ctx)

	addr := sample.AccAddressBytes()
	s.bankKeeper.On("GetBalance", ctx, addr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(100)), nil)

	balance, err := k.GetAccountBalance(ctx, addr)
	require.NoError(err)
	require.Equal(balance, math.NewInt(100))
}

func (s *KeeperTestSuite) TestRatio() {
	require := s.Require()

	// 11/33 --> 11111111
	ratio := disputekeeper.Ratio(math.LegacyNewDec(33), math.LegacyNewDec(11))
	expected := math.LegacyNewDec(1).Quo(math.LegacyNewDec(9)).Mul(math.LegacyNewDec(1e8)) // 6 for power reduction, 2 for using tens place numbers
	fmt.Println(ratio)
	require.Equal(ratio.TruncateDec(), expected.TruncateDec()) // not sure of a better way to compare...
	// 25/25 --> 25/100
	ratio = disputekeeper.Ratio(math.LegacyNewDec(33), math.LegacyNewDec(33))
	expected = math.LegacyNewDec(3).Quo(math.LegacyNewDec(9)).Mul(math.LegacyNewDec(1e8))
	require.Equal(ratio.TruncateDec(), expected.TruncateDec())
	// 0/25 --> 0/100
	ratio = disputekeeper.Ratio(math.LegacyNewDec(33), math.LegacyNewDec(0))
	fmt.Println(ratio)
	require.Equal(ratio, math.LegacyNewDec(0))
	// 25/0 --> 100/0
	ratio = disputekeeper.Ratio(math.LegacyNewDec(0), math.LegacyNewDec(33))
	fmt.Println(ratio)
	require.Equal(ratio, math.LegacyNewDec(0))

	// big numbers
	// ex. total reporter power is 1_000_000 trb, all of them have voted
	ratio = disputekeeper.Ratio(math.LegacyNewDec(1_000_000), math.LegacyNewDec(1_000_000))
	expected = math.LegacyNewDec(3).Quo(math.LegacyNewDec(9)).Mul(math.LegacyNewDec(1e8))
	fmt.Println(ratio)
	require.Equal(ratio.TruncateDec(), expected.TruncateDec())

	// ex. total reporter power is 1e14 trb, 1e13 trb have voted
	ratio = disputekeeper.Ratio(math.LegacyNewDec(1e14), math.LegacyNewDec(1e13))
	expected = math.LegacyNewDec(1e13).Quo(math.LegacyNewDec(3e14)).Mul(math.LegacyNewDec(1e8))
	fmt.Println(ratio)
	require.Equal(ratio.TruncateDec(), expected.TruncateDec())
}

func (s *KeeperTestSuite) TestTallyVote() {
	k := s.disputeKeeper
	bk := s.bankKeeper
	ctx := s.ctx
	require := s.Require()
	require.NotNil(k)
	require.NotNil(ctx)

	testCases := []struct {
		name          string
		disputeId     uint64
		setup         func()
		teardown      func()
		expectedError error
		expectedVotes types.StakeholderVoteCounts
	}{
		{
			name:      "vote already tallied",
			disputeId: uint64(1),
			setup: func() {
				disputeId := uint64(1)
				require.NoError(k.Votes.Set(ctx, uint64(1), types.Vote{
					Id:         disputeId,
					VoteResult: types.VoteResult_INVALID,
				}))
			},
			teardown: func() {
				disputeId := uint64(1)
				require.NoError(k.Votes.Remove(ctx, disputeId))
			},
			expectedError: errors.New("vote already tallied"),
		},
		{
			name:      "team votes only",
			disputeId: uint64(2),
			setup: func() {
				disputeId := uint64(2)
				// get team address
				teamAddr, err := k.GetTeamAddress(ctx)
				require.NoError(err)
				// set dispute voting status
				require.NoError(k.Votes.Set(ctx, disputeId, types.Vote{
					Id:         disputeId,
					VoteResult: types.VoteResult_NO_TALLY,
				}))
				// set dispute info
				require.NoError(k.Disputes.Set(ctx, disputeId, types.Dispute{
					HashId:    []byte("hashId2"),
					DisputeId: disputeId,
				}))
				// set block info
				require.NoError(k.BlockInfo.Set(ctx, []byte("hashId2"), types.BlockInfo{
					TotalReporterPower: math.NewInt(50 * 1e6),
					TotalUserTips:      math.NewInt(50 * 1e6),
				}))
				// set vote counts by group
				require.NoError(k.VoteCountsByGroup.Set(ctx, disputeId, types.StakeholderVoteCounts{
					Team: types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
				}))
				// set team vote
				require.NoError(k.Voter.Set(ctx, collections.Join(disputeId, teamAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.NewInt(33333333)}))
				// mock for GetTotalSupply
				bk.On("GetSupply", ctx, layertypes.BondDenom).Return(sdk.Coin{Denom: layertypes.BondDenom, Amount: math.NewInt(250 * 1e6)}, nil)
			},
			teardown:      func() {},
			expectedError: errors.New("vote period not ended and quorum not reached"),
			expectedVotes: types.StakeholderVoteCounts{
				Team:      types.VoteCounts{Support: 33333333, Against: 0, Invalid: 0},
				Reporters: types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
				Users:     types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
			},
		},
		{
			name:      "team votes, some users vote, quorum not reached",
			disputeId: uint64(2),
			setup: func() {
				disputeId := uint64(2)
				require.NoError(k.VoteCountsByGroup.Set(ctx, disputeId, types.StakeholderVoteCounts{
					Team:      types.VoteCounts{Support: 33333333, Against: 0, Invalid: 0},
					Users:     types.VoteCounts{Support: 10000000, Against: 0, Invalid: 0},
					Reporters: types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
				}))
			},
			teardown:      func() {},
			expectedError: errors.New("vote period not ended and quorum not reached"),
			expectedVotes: types.StakeholderVoteCounts{
				Team:      types.VoteCounts{Support: 33333333, Against: 0, Invalid: 0},
				Reporters: types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
				Users:     types.VoteCounts{Support: 10000000, Against: 0, Invalid: 0},
			},
		},
		{
			name:      "team votes, all users vote, reporters vote, quorum reached",
			disputeId: uint64(2),
			setup: func() {
				disputeId := uint64(2)
				require.NoError(k.Disputes.Set(ctx, disputeId, types.Dispute{
					HashId:    []byte("hashId4"),
					DisputeId: disputeId,
					Open:      true,
				}))
				require.NoError(k.BlockInfo.Set(ctx, []byte("hashId4"), types.BlockInfo{
					TotalReporterPower: math.NewInt(60 * 1e6),
					TotalUserTips:      math.NewInt(60 * 1e6),
				}))
				require.NoError(k.VoteCountsByGroup.Set(ctx, disputeId, types.StakeholderVoteCounts{
					Team:      types.VoteCounts{Support: 33333333, Against: 0, Invalid: 0},
					Users:     types.VoteCounts{Support: 0, Against: 20 * 1e6, Invalid: 0},
					Reporters: types.VoteCounts{Support: 0, Against: 20 * 1e6, Invalid: 0},
				}))
			},
			teardown:      func() {},
			expectedError: nil,
			expectedVotes: types.StakeholderVoteCounts{
				Team:      types.VoteCounts{Support: 33333333, Against: 0, Invalid: 0},
				Reporters: types.VoteCounts{Support: 0, Against: 20 * 1e6, Invalid: 0},
				Users:     types.VoteCounts{Support: 0, Against: 20 * 1e6, Invalid: 0},
			},
		},
		{
			name:      "everybody votes, quorum reached and dispute is closed",
			disputeId: uint64(4),
			setup: func() {
				disputeId := uint64(4)
				// get team address
				teamAddr, err := k.GetTeamAddress(ctx)
				require.NoError(err)
				// set dispute voting status
				require.NoError(k.Votes.Set(ctx, disputeId, types.Vote{
					Id:         disputeId,
					VoteResult: types.VoteResult_NO_TALLY,
				}))
				// set dispute info
				require.NoError(k.Disputes.Set(ctx, disputeId, types.Dispute{
					HashId:    []byte("hashId4"),
					DisputeId: disputeId,
				}))
				// set block info
				require.NoError(k.BlockInfo.Set(ctx, []byte("hashId4"), types.BlockInfo{
					TotalReporterPower: math.NewInt(60 * 1e6),
					TotalUserTips:      math.NewInt(60 * 1e6),
				}))
				// set vote counts by group
				require.NoError(k.VoteCountsByGroup.Set(ctx, disputeId, types.StakeholderVoteCounts{
					Team:      types.VoteCounts{Support: 0, Against: 0, Invalid: 33333333},
					Users:     types.VoteCounts{Support: 22500000, Against: 22500000, Invalid: 15000000},
					Reporters: types.VoteCounts{Support: 27500000, Against: 22500000, Invalid: 10000000},
				}))
				// set team vote
				require.NoError(k.Voter.Set(ctx, collections.Join(disputeId, teamAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_INVALID, VoterPower: math.NewInt(25000000)}))
				// mock for GetTotalSupply
				bk.On("GetSupply", ctx, layertypes.BondDenom).Return(sdk.Coin{Denom: layertypes.BondDenom, Amount: math.NewInt(60 * 1e6)}, nil)
			},
			teardown:      func() {},
			expectedError: nil,
			expectedVotes: types.StakeholderVoteCounts{
				Team:      types.VoteCounts{Support: 0, Against: 0, Invalid: 33333333},
				Users:     types.VoteCounts{Support: 22500000, Against: 22500000, Invalid: 15000000},
				Reporters: types.VoteCounts{Support: 27500000, Against: 22500000, Invalid: 10000000},
			},
		},
		{
			name:      "51% ratio is reached but for a split vote and dispute is still in voting period",
			disputeId: uint64(6),
			setup: func() {
				disputeId := uint64(6)
				// get team address
				teamAddr, err := k.GetTeamAddress(ctx)
				require.NoError(err)
				// set dispute voting status
				require.NoError(k.Votes.Set(ctx, disputeId, types.Vote{
					Id:         disputeId,
					VoteResult: types.VoteResult_NO_TALLY,
				}))
				// set dispute info
				require.NoError(k.Disputes.Set(ctx, disputeId, types.Dispute{
					HashId:    []byte("hashId4"),
					DisputeId: disputeId,
					Open:      true,
				}))
				// set block info
				require.NoError(k.BlockInfo.Set(ctx, []byte("hashId4"), types.BlockInfo{
					TotalReporterPower: math.NewInt(60 * 1e6),
					TotalUserTips:      math.NewInt(60 * 1e6),
				}))
				// set vote counts by group
				require.NoError(k.VoteCountsByGroup.Set(ctx, disputeId, types.StakeholderVoteCounts{
					Team:      types.VoteCounts{Support: 0, Against: 0, Invalid: 33333333},
					Users:     types.VoteCounts{Support: 22500000, Against: 22500000, Invalid: 15000000},
					Reporters: types.VoteCounts{Support: 27500000, Against: 22500000, Invalid: 10000000},
				}))
				// set team vote
				require.NoError(k.Voter.Set(ctx, collections.Join(disputeId, teamAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_INVALID, VoterPower: math.NewInt(25000000)}))
				// mock for GetTotalSupply
				bk.On("GetSupply", ctx, layertypes.BondDenom).Return(sdk.Coin{Denom: layertypes.BondDenom, Amount: math.NewInt(60 * 1e6)}, nil)
			},
			teardown:      func() {},
			expectedError: errors.New(types.ErrNoQuorumStillVoting.Error()),
			expectedVotes: types.StakeholderVoteCounts{
				Team:      types.VoteCounts{Support: 0, Against: 0, Invalid: 33333333},
				Users:     types.VoteCounts{Support: 22500000, Against: 22500000, Invalid: 15000000},
				Reporters: types.VoteCounts{Support: 27500000, Against: 22500000, Invalid: 10000000},
			},
		},
	}
	for _, tc := range testCases {
		if tc.setup != nil {
			fmt.Println(tc.name)
			s.Run(tc.name, tc.setup)
		}
		err := k.TallyVote(ctx, tc.disputeId)
		if tc.expectedError != nil {
			require.Error(err)
			fmt.Println("err: ", err)
			require.ErrorContains(err, tc.expectedError.Error())
		} else {
			require.NoError(err)
			votesByGroup, err := k.VoteCountsByGroup.Get(ctx, tc.disputeId)
			fmt.Println("votesByGroup: ", votesByGroup)
			require.NoError(err)
			require.Equal(tc.expectedVotes, votesByGroup)
			dispute, err := k.Disputes.Get(ctx, tc.disputeId)
			require.NoError(err)
			fmt.Println("dispute: ", dispute)
		}
		if tc.teardown != nil {
			s.Run(tc.name, tc.teardown)
		}
	}
}

func (s *KeeperTestSuite) TestUpdateDispute() {
	k := s.disputeKeeper
	ctx := s.ctx
	require := s.Require()
	require.NotNil(k)
	require.NotNil(ctx)
	id := uint64(1)
	dispute := types.Dispute{
		HashId:          []byte("hashId"),
		DisputeId:       id,
		DisputeCategory: types.Minor,
	}

	// quorum support
	vote := types.Vote{
		Id:         id,
		VoteResult: types.VoteResult_SUPPORT,
	}
	scaledSupport := math.NewInt(100)
	scaledAgainst := math.ZeroInt()
	scaledInvalid := math.ZeroInt()

	require.NoError(k.UpdateDispute(ctx, id, dispute, vote, scaledSupport, scaledAgainst, scaledInvalid, true))
	disputeRes, err := k.Disputes.Get(ctx, id)
	require.NoError(err)
	require.Equal(disputeRes.DisputeId, dispute.DisputeId)
	require.Equal(disputeRes.HashId, dispute.HashId)
	require.Equal(disputeRes.DisputeCategory, dispute.DisputeCategory)
	voteRes, err := k.Votes.Get(ctx, id)
	require.NoError(err)
	require.Equal(voteRes.Id, vote.Id)
	require.Equal(voteRes.VoteResult, vote.VoteResult)

	// no quorum majority support
	vote = types.Vote{
		Id:         id,
		VoteResult: types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT,
	}
	scaledSupport = math.NewInt(50)
	scaledAgainst = math.ZeroInt()
	scaledInvalid = math.ZeroInt()

	require.NoError(k.UpdateDispute(ctx, id, dispute, vote, scaledSupport, scaledAgainst, scaledInvalid, false))
	disputeRes, err = k.Disputes.Get(ctx, id)
	require.NoError(err)
	require.Equal(disputeRes.DisputeId, dispute.DisputeId)
	require.Equal(disputeRes.HashId, dispute.HashId)
	require.Equal(disputeRes.DisputeCategory, dispute.DisputeCategory)
	voteRes, err = k.Votes.Get(ctx, id)
	require.NoError(err)
	require.Equal(voteRes.Id, vote.Id)
	require.Equal(voteRes.VoteResult, vote.VoteResult)

	// quorum against
	vote = types.Vote{
		Id:         id,
		VoteResult: types.VoteResult_AGAINST,
	}
	scaledSupport = math.ZeroInt()
	scaledAgainst = math.NewInt(100)
	scaledInvalid = math.ZeroInt()

	require.NoError(k.UpdateDispute(ctx, id, dispute, vote, scaledSupport, scaledAgainst, scaledInvalid, true))
	disputeRes, err = k.Disputes.Get(ctx, id)
	require.NoError(err)
	require.Equal(disputeRes.DisputeId, dispute.DisputeId)
	require.Equal(disputeRes.HashId, dispute.HashId)
	require.Equal(disputeRes.DisputeCategory, dispute.DisputeCategory)
	voteRes, err = k.Votes.Get(ctx, id)
	require.NoError(err)
	require.Equal(voteRes.Id, vote.Id)
	require.Equal(voteRes.VoteResult, vote.VoteResult)

	// no quorum majority against
	vote = types.Vote{
		Id:         id,
		VoteResult: types.VoteResult_NO_QUORUM_MAJORITY_AGAINST,
	}
	scaledSupport = math.ZeroInt()
	scaledAgainst = math.NewInt(40)
	scaledInvalid = math.ZeroInt()

	require.NoError(k.UpdateDispute(ctx, id, dispute, vote, scaledSupport, scaledAgainst, scaledInvalid, false))
	disputeRes, err = k.Disputes.Get(ctx, id)
	require.NoError(err)
	require.Equal(disputeRes.DisputeId, dispute.DisputeId)
	require.Equal(disputeRes.HashId, dispute.HashId)
	require.Equal(disputeRes.DisputeCategory, dispute.DisputeCategory)
	voteRes, err = k.Votes.Get(ctx, id)
	require.NoError(err)
	require.Equal(voteRes.Id, vote.Id)
	require.Equal(voteRes.VoteResult, vote.VoteResult)

	// quorum invalid
	vote = types.Vote{
		Id:         id,
		VoteResult: types.VoteResult_INVALID,
	}
	scaledSupport = math.ZeroInt()
	scaledAgainst = math.ZeroInt()
	scaledInvalid = math.NewInt(51)

	require.NoError(k.UpdateDispute(ctx, id, dispute, vote, scaledSupport, scaledAgainst, scaledInvalid, true))
	disputeRes, err = k.Disputes.Get(ctx, id)
	require.NoError(err)
	require.Equal(disputeRes.DisputeId, dispute.DisputeId)
	require.Equal(disputeRes.HashId, dispute.HashId)
	require.Equal(disputeRes.DisputeCategory, dispute.DisputeCategory)
	voteRes, err = k.Votes.Get(ctx, id)
	require.NoError(err)
	require.Equal(voteRes.Id, vote.Id)
	require.Equal(voteRes.VoteResult, vote.VoteResult)

	// no quorum majority invalid
	vote = types.Vote{
		Id:         id,
		VoteResult: types.VoteResult_NO_QUORUM_MAJORITY_INVALID,
	}
	scaledSupport = math.ZeroInt()
	scaledAgainst = math.ZeroInt()
	scaledInvalid = math.NewInt(49)

	require.NoError(k.UpdateDispute(ctx, id, dispute, vote, scaledSupport, scaledAgainst, scaledInvalid, false))
	disputeRes, err = k.Disputes.Get(ctx, id)
	require.NoError(err)
	require.Equal(disputeRes.DisputeId, dispute.DisputeId)
	require.Equal(disputeRes.HashId, dispute.HashId)
	require.Equal(disputeRes.DisputeCategory, dispute.DisputeCategory)
	voteRes, err = k.Votes.Get(ctx, id)
	require.NoError(err)
	require.Equal(voteRes.Id, vote.Id)
	require.Equal(voteRes.VoteResult, vote.VoteResult)
}
