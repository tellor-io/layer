package keeper_test

import (
	"fmt"
	"time"

	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestDisputesQuery() {
	require := s.Require()
	k := s.disputeKeeper
	q := keeper.NewQuerier(k)
	require.NotNil(q)
	ctx := s.ctx

	testCases := []struct {
		name           string
		req            *types.QueryDisputesRequest
		setup          func()
		expectedLength int
		err            bool
	}{
		{
			name: "nil request",
			req:  nil,
			err:  true,
		},
		{
			name:           "empty request, no disputes",
			req:            &types.QueryDisputesRequest{},
			expectedLength: 0,
			err:            false,
		},
		{
			name: "one dispute",
			setup: func() {
				require.NoError(k.Disputes.Set(ctx, 1, types.Dispute{
					HashId:           []byte{1},
					DisputeId:        1,
					DisputeCategory:  types.Warning,
					DisputeFee:       math.NewInt(1000000),
					DisputeStatus:    types.Voting,
					DisputeStartTime: time.Now(),
					DisputeEndTime:   time.Now().Add(time.Hour * 24),
					Open:             true,
					DisputeRound:     1,
					SlashAmount:      math.NewInt(1000000),
					BurnAmount:       math.NewInt(100),
					InitialEvidence: oracletypes.MicroReport{
						Reporter:  "cosmos1v9j474hfk7clqc4g50z0y3ftm43hj32c9mapfk",
						Timestamp: time.Now(),
					},
				}))
			},
			req:            &types.QueryDisputesRequest{},
			expectedLength: 1,
			err:            false,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if tc.setup != nil {
				tc.setup()
			}
			resp, err := q.Disputes(ctx, tc.req)
			if tc.err {
				require.Error(err)
				return
			} else {
				require.NoError(err)
				require.NotNil(resp)
				require.Equal(tc.expectedLength, len(resp.Disputes))
			}
		})
	}
}

func (s *KeeperTestSuite) TestDisputeQuery() {
	require := s.Require()
	k := s.disputeKeeper
	q := keeper.NewQuerier(k)
	require.NotNil(q)
	ctx := s.ctx

	// Create a test dispute
	testDispute := types.Dispute{
		HashId:           []byte{1},
		DisputeId:        1,
		DisputeCategory:  types.Warning,
		DisputeFee:       math.NewInt(1000000),
		DisputeStatus:    types.Voting,
		DisputeStartTime: time.Now(),
		DisputeEndTime:   time.Now().Add(time.Hour * 24),
		Open:             true,
		DisputeRound:     1,
		SlashAmount:      math.NewInt(1000000),
		BurnAmount:       math.NewInt(100),
		InitialEvidence: oracletypes.MicroReport{
			Reporter:  "cosmos1v9j474hfk7clqc4g50z0y3ftm43hj32c9mapfk",
			Timestamp: time.Now(),
		},
	}
	require.NoError(k.Disputes.Set(ctx, 1, testDispute))

	// Query the specific dispute
	resp, err := q.Dispute(ctx, &types.QueryDisputeRequest{DisputeId: 1})
	require.NoError(err)
	require.NotNil(resp)
	require.NotNil(resp.Dispute)
	require.Equal(uint64(1), resp.Dispute.DisputeId)
	require.Equal(testDispute.DisputeCategory, resp.Dispute.Metadata.DisputeCategory)
	require.Equal(testDispute.DisputeFee, resp.Dispute.Metadata.DisputeFee)
	require.Equal(testDispute.DisputeStatus, resp.Dispute.Metadata.DisputeStatus)
	require.True(resp.Dispute.Metadata.Open)

	// Query non-existent dispute
	_, err = q.Dispute(ctx, &types.QueryDisputeRequest{DisputeId: 999})
	require.Error(err)

	// nil request
	_, err = q.Dispute(ctx, nil)
	require.Error(err)
}

func (s *KeeperTestSuite) TestOpenDisputesQuery() {
	require := s.Require()
	k := s.disputeKeeper
	q := keeper.NewQuerier(k)
	require.NotNil(q)

	// nil
	ctx := s.ctx
	resp, err := q.OpenDisputes(ctx, nil)
	require.Error(err)
	require.Nil(resp)

	// empty request
	resp, err = q.OpenDisputes(ctx, &types.QueryOpenDisputesRequest{})
	require.NoError(err)
	require.NotNil(resp)

	// one open dispute
	require.NoError(k.Disputes.Set(ctx, 1, types.Dispute{
		HashId:           []byte{1},
		DisputeId:        1,
		DisputeCategory:  types.Warning,
		DisputeFee:       math.NewInt(1000000),
		DisputeStatus:    types.Voting,
		DisputeStartTime: time.Now(),
		DisputeEndTime:   time.Now().Add(time.Hour * 24),
		Open:             true,
		DisputeRound:     1,
		SlashAmount:      math.NewInt(1000000),
		BurnAmount:       math.NewInt(100),
		InitialEvidence: oracletypes.MicroReport{
			Reporter:  "cosmos1v9j474hfk7clqc4g50z0y3ftm43hj32c9mapfk",
			Timestamp: time.Now(),
		},
	}))
	resp, err = q.OpenDisputes(ctx, &types.QueryOpenDisputesRequest{})
	require.NoError(err)
	require.NotNil(resp)
	require.Equal(1, len(resp.OpenDisputes.Ids))

	// two open disputes
	require.NoError(k.Disputes.Set(ctx, 2, types.Dispute{
		HashId:           []byte{1},
		DisputeId:        2,
		DisputeCategory:  types.Warning,
		DisputeFee:       math.NewInt(1000000),
		DisputeStatus:    types.Voting,
		DisputeStartTime: time.Now(),
		DisputeEndTime:   time.Now().Add(time.Hour * 24),
		Open:             true,
		DisputeRound:     1,
		SlashAmount:      math.NewInt(1000000),
		BurnAmount:       math.NewInt(100),
		InitialEvidence: oracletypes.MicroReport{
			Reporter:  "cosmos1v9j474hfk7clqc4g50z0y3ftm43hj32c9mapfk",
			Timestamp: time.Now(),
		},
	}))
	resp, err = q.OpenDisputes(ctx, &types.QueryOpenDisputesRequest{})
	require.NoError(err)
	require.NotNil(resp)
	require.Equal(2, len(resp.OpenDisputes.Ids))

	// two open and one closed dispute
	require.NoError(k.Disputes.Set(ctx, 3, types.Dispute{
		HashId:           []byte{1},
		DisputeId:        3,
		DisputeCategory:  types.Warning,
		DisputeFee:       math.NewInt(1000000),
		DisputeStatus:    types.Resolved,
		DisputeStartTime: time.Now().Add(-time.Hour * 24),
		DisputeEndTime:   time.Now().Add(-time.Hour),
		Open:             false,
		DisputeRound:     1,
		SlashAmount:      math.NewInt(1000000),
		BurnAmount:       math.NewInt(100),
		InitialEvidence: oracletypes.MicroReport{
			Reporter:  "cosmos1v9j474hfk7clqc4g50z0y3ftm43hj32c9mapfk",
			Timestamp: time.Now().Add(-time.Hour * 24),
		},
	}))
	resp, err = q.OpenDisputes(ctx, &types.QueryOpenDisputesRequest{})
	require.NoError(err)
	require.NotNil(resp)
	require.Equal(2, len(resp.OpenDisputes.Ids))
}

func (s *KeeperTestSuite) TestTallyQuery() {
	require := s.Require()
	k := s.disputeKeeper
	q := keeper.NewQuerier(k)
	require.NotNil(q)
	ctx := s.ctx

	require.NoError(k.Disputes.Set(ctx, 1, types.Dispute{
		HashId:           []byte{1},
		DisputeId:        1,
		DisputeCategory:  types.Warning,
		DisputeFee:       math.NewInt(1000000),
		DisputeStatus:    types.Voting,
		DisputeStartTime: time.Now(),
		DisputeEndTime:   time.Now().Add(time.Hour * 24),
		Open:             true,
		DisputeRound:     1,
		SlashAmount:      math.NewInt(1000000),
		BurnAmount:       math.NewInt(100),
		InitialEvidence: oracletypes.MicroReport{
			Reporter:  "cosmos1v9j474hfk7clqc4g50z0y3ftm43hj32c9mapfk",
			Timestamp: time.Now(),
		},
	}))

	require.NoError(k.VoteCountsByGroup.Set(ctx, 1, types.StakeholderVoteCounts{
		Users:     types.VoteCounts{Support: 1000, Against: 100, Invalid: 500},
		Reporters: types.VoteCounts{Support: 10000, Against: 100, Invalid: 560},
		Team:      types.VoteCounts{Support: 1000, Against: 0, Invalid: 0},
	}))

	require.NoError(q.BlockInfo.Set(ctx, []byte{1}, types.BlockInfo{TotalReporterPower: math.NewInt(25000), TotalUserTips: math.NewInt(5000)}))

	s.bankKeeper.On("GetSupply", ctx, "loya").Return(sdk.NewCoin("loya", math.NewInt(100000)))

	res, err := q.Tally(ctx, &types.QueryDisputesTallyRequest{DisputeId: 1})
	require.NoError(err)
	fmt.Println(res)

	require.Equal(res.Users.TotalPowerVoted, uint64(1600))
	require.Equal(res.Reporters.TotalPowerVoted, uint64(10660))

	require.Equal(res.Users.TotalGroupPower, uint64(5000))
	require.Equal(res.Reporters.TotalGroupPower, uint64(25000))

	// Test dispute with no votes yet
	require.NoError(k.Disputes.Set(ctx, 2, types.Dispute{
		HashId:           []byte{2},
		DisputeId:        2,
		DisputeCategory:  types.Warning,
		DisputeFee:       math.NewInt(1000000),
		DisputeStatus:    types.Voting,
		DisputeStartTime: time.Now(),
		DisputeEndTime:   time.Now().Add(time.Hour * 24),
		Open:             true,
		DisputeRound:     1,
		SlashAmount:      math.NewInt(1000000),
		BurnAmount:       math.NewInt(100),
		InitialEvidence: oracletypes.MicroReport{
			Reporter:  "cosmos1v9j474hfk7clqc4g50z0y3ftm43hj32c9mapfk",
			Timestamp: time.Now(),
		},
	}))

	// Query tally for dispute with no votes - should return empty tally
	res, err = q.Tally(ctx, &types.QueryDisputesTallyRequest{DisputeId: 2})
	require.NoError(err)
	require.NotNil(res)
	require.Equal("0.00%", res.Users.VoteCount.Support)
	require.Equal("0.00%", res.Users.VoteCount.Against)
	require.Equal("0.00%", res.Users.VoteCount.Invalid)
	require.Equal(uint64(0), res.Users.TotalPowerVoted)
	require.Equal(uint64(0), res.Users.TotalGroupPower)
	require.Equal("0.00%", res.Reporters.VoteCount.Support)
	require.Equal("0.00%", res.Reporters.VoteCount.Against)
	require.Equal("0.00%", res.Reporters.VoteCount.Invalid)
	require.Equal(uint64(0), res.Reporters.TotalPowerVoted)
	require.Equal(uint64(0), res.Reporters.TotalGroupPower)
	require.Equal("0.00%", res.Team.Support)
	require.Equal("0.00%", res.Team.Against)
	require.Equal("0.00%", res.Team.Invalid)
	require.Equal("0.00%", res.CombinedTotal.Support)
	require.Equal("0.00%", res.CombinedTotal.Against)
	require.Equal("0.00%", res.CombinedTotal.Invalid)
}

func (s *KeeperTestSuite) TestVoteResultQuery() {
	require := s.Require()
	k := s.disputeKeeper
	q := keeper.NewQuerier(k)
	require.NotNil(q)
	ctx := s.ctx

	require.NoError(k.Votes.Set(ctx, 1, types.Vote{
		Id:         1,
		VoteStart:  time.Now().Add(time.Hour * -24),
		VoteEnd:    time.Now(),
		VoteResult: types.VoteResult_SUPPORT,
	}))

	res, err := q.VoteResult(ctx, &types.QueryDisputeVoteResultRequest{DisputeId: 1})
	require.NoError(err)
	require.Equal(res.VoteResult, types.VoteResult_SUPPORT)
}

func (s *KeeperTestSuite) TestTeamVoteQuery() {
	require := s.Require()
	k := s.disputeKeeper
	q := keeper.NewQuerier(k)
	require.NotNil(q)
	ctx := s.ctx

	teamAddr, err := k.GetTeamAddress(ctx)
	require.NoError(err)

	require.NoError(k.Voter.Set(ctx, collections.Join(uint64(1), teamAddr.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_SUPPORT,
		VoterPower:    math.NewInt(100000000).Quo(math.NewInt(3)),
		ReporterPower: math.NewInt(0),
		RewardClaimed: false,
	}))

	res, err := q.TeamVote(ctx, &types.QueryTeamVoteRequest{DisputeId: 1})
	require.NoError(err)
	require.Equal(res.TeamVote.Vote, types.VoteEnum_VOTE_SUPPORT)
	require.Equal(res.TeamVote.VoterPower, math.NewInt(100000000).Quo(math.NewInt(3)))
	require.Equal(res.TeamVote.ReporterPower, math.NewInt(0))
	require.Equal(res.TeamVote.RewardClaimed, false)

	// on second dispute team votes against
	require.NoError(k.Voter.Set(ctx, collections.Join(uint64(2), teamAddr.Bytes()), types.Voter{
		Vote:          types.VoteEnum_VOTE_AGAINST,
		VoterPower:    math.NewInt(100000000).Quo(math.NewInt(3)),
		ReporterPower: math.NewInt(0),
		RewardClaimed: false,
	}))

	res, err = q.TeamVote(ctx, &types.QueryTeamVoteRequest{DisputeId: 2})
	require.NoError(err)
	require.Equal(res.TeamVote.Vote, types.VoteEnum_VOTE_AGAINST)
	require.Equal(res.TeamVote.VoterPower, math.NewInt(100000000).Quo(math.NewInt(3)))

	// query dispute 1 again, expect no change
	res, err = q.TeamVote(ctx, &types.QueryTeamVoteRequest{DisputeId: 1})
	require.NoError(err)
	require.Equal(res.TeamVote.Vote, types.VoteEnum_VOTE_SUPPORT)
	require.Equal(res.TeamVote.VoterPower, math.NewInt(100000000).Quo(math.NewInt(3)))
	require.Equal(res.TeamVote.ReporterPower, math.NewInt(0))
	require.Equal(res.TeamVote.RewardClaimed, false)
}

func (s *KeeperTestSuite) TestDisputeFeePayersQuery() {
	require := s.Require()
	k := s.disputeKeeper
	q := keeper.NewQuerier(k)
	require.NotNil(q)
	ctx := s.ctx

	payer1 := sdk.AccAddress([]byte("payer1_address"))
	payer2 := sdk.AccAddress([]byte("payer2_address"))
	payer3 := sdk.AccAddress([]byte("payer3_address"))
	payer4 := sdk.AccAddress([]byte("payer4_address"))

	// payer1 pays 1000 loya for dispute 1
	require.NoError(k.DisputeFeePayer.Set(ctx, collections.Join(uint64(1), payer1.Bytes()), types.PayerInfo{
		Amount:   math.NewInt(1000),
		FromBond: false,
	}))

	// payers 2,3,4 pay for dispute 2
	require.NoError(k.DisputeFeePayer.Set(ctx, collections.Join(uint64(2), payer2.Bytes()), types.PayerInfo{
		Amount:   math.NewInt(500),
		FromBond: true,
	}))
	require.NoError(k.DisputeFeePayer.Set(ctx, collections.Join(uint64(2), payer3.Bytes()), types.PayerInfo{
		Amount:   math.NewInt(750),
		FromBond: false,
	}))
	require.NoError(k.DisputeFeePayer.Set(ctx, collections.Join(uint64(2), payer4.Bytes()), types.PayerInfo{
		Amount:   math.NewInt(250),
		FromBond: true,
	}))

	// nil request
	_, err := q.DisputeFeePayers(ctx, nil)
	require.Error(err)

	// Query dispute 1 - should have 1 payer
	res1, err := q.DisputeFeePayers(ctx, &types.QueryDisputeFeePayersRequest{DisputeId: 1})
	require.NoError(err)
	require.NotNil(res1)
	require.Len(res1.Payers, 1)
	require.Equal(payer1.String(), res1.Payers[0].PayerAddress)
	require.Equal(math.NewInt(1000), res1.Payers[0].PayerInfo.Amount)
	require.False(res1.Payers[0].PayerInfo.FromBond)

	// Query dispute 2 - should have 3 payers
	res2, err := q.DisputeFeePayers(ctx, &types.QueryDisputeFeePayersRequest{DisputeId: 2})
	require.NoError(err)
	require.NotNil(res2)
	require.Len(res2.Payers, 3)

	// Verify all three payers are present
	payerAddresses := make(map[string]types.PayerInfo)
	for _, payer := range res2.Payers {
		payerAddresses[payer.PayerAddress] = payer.PayerInfo
	}

	require.Contains(payerAddresses, payer2.String())
	require.Equal(math.NewInt(500), payerAddresses[payer2.String()].Amount)
	require.True(payerAddresses[payer2.String()].FromBond)

	require.Contains(payerAddresses, payer3.String())
	require.Equal(math.NewInt(750), payerAddresses[payer3.String()].Amount)
	require.False(payerAddresses[payer3.String()].FromBond)

	require.Contains(payerAddresses, payer4.String())
	require.Equal(math.NewInt(250), payerAddresses[payer4.String()].Amount)
	require.True(payerAddresses[payer4.String()].FromBond)

	// Query non-existent dispute - should return empty list
	res3, err := q.DisputeFeePayers(ctx, &types.QueryDisputeFeePayersRequest{DisputeId: 999})
	require.NoError(err)
	require.NotNil(res3)
	require.Len(res3.Payers, 0)
}
