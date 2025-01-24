package keeper_test

import (
	"fmt"
	"time"

	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

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
			fmt.Println(resp)
		})
	}
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
	fmt.Println(resp)
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
	fmt.Println(resp)
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
	fmt.Println(resp)
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

	require.Equal(res.Users.TotalPowerVoted, uint64(1600))
	require.Equal(res.Reporters.TotalPowerVoted, uint64(10660))

	require.Equal(res.Users.TotalGroupPower, uint64(5000))
	require.Equal(res.Reporters.TotalGroupPower, uint64(25000))
}
