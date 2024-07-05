package keeper_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestGetVoters() {
	require := s.Require()
	ctx := s.ctx
	k := s.disputeKeeper
	require.NotNil(k)
	require.NotNil(ctx)

	res, err := s.disputeKeeper.GetVoters(ctx, 1)
	require.Empty(res)
	require.NoError(err)

	voter := sample.AccAddressBytes()
	require.NoError(k.Voter.Set(ctx, collections.Join(uint64(1), voter.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.OneInt()}))

	res, err = s.disputeKeeper.GetVoters(ctx, 1)
	require.NoError(err)
	require.Equal(res[0].Value.Vote, types.VoteEnum_VOTE_SUPPORT)
	require.Equal(res[0].Value.VoterPower, math.OneInt())

	voter2 := sample.AccAddressBytes()
	require.NoError(k.Voter.Set(ctx, collections.Join(uint64(1), voter2.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.OneInt()}))

	res, err = s.disputeKeeper.GetVoters(ctx, 1)
	require.NoError(err)
	require.Equal(res[0].Value.Vote, types.VoteEnum_VOTE_SUPPORT)
	require.Equal(res[0].Value.VoterPower, math.OneInt())
	require.Equal(res[1].Value.Vote, types.VoteEnum_VOTE_SUPPORT)
	require.Equal(res[1].Value.VoterPower, math.OneInt())
}

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

func (s *KeeperTestSuite) TestGetTotalSupply() {
	require := s.Require()
	ctx := s.ctx
	k := s.disputeKeeper
	require.NotNil(k)
	require.NotNil(ctx)

	s.bankKeeper.On("GetSupply", ctx, layertypes.BondDenom).Return(sdk.Coin{Denom: layertypes.BondDenom, Amount: math.NewInt(1_000 * 1e6)}, nil)

	totalSupply := k.GetTotalSupply(ctx)
	require.Equal(totalSupply, math.NewInt(1_000*1e6))
}

func (s *KeeperTestSuite) TestRatio() {
	require := s.Require()

	// 10/25 --> 10/100
	ratio := disputekeeper.Ratio(math.NewInt(25), math.NewInt(10))
	require.Equal(ratio, math.LegacyNewDecWithPrec(10, 0))
	// 25/25 --> 25/100
	ratio = disputekeeper.Ratio(math.NewInt(25), math.NewInt(25))
	require.Equal(ratio, math.LegacyNewDecWithPrec(25, 0))
	// 0/25 --> 0/100
	ratio = disputekeeper.Ratio(math.NewInt(25), math.NewInt(0))
	require.Equal(ratio, math.LegacyNewDecWithPrec(0, 0))
	// 25/0 --> 100/0
	ratio = disputekeeper.Ratio(math.NewInt(0), math.NewInt(25))
	require.Equal(ratio, math.LegacyNewDecWithPrec(0, 0))
}

func (s *KeeperTestSuite) TestCalculateVotingPower() {
	require := s.Require()

	// 100/100 --> 100/400
	votingPower := disputekeeper.CalculateVotingPower(math.NewInt(100), math.NewInt(100))
	require.Equal(votingPower, math.NewInt(25*1e6))
	// 50/100 --> 50/400
	votingPower = disputekeeper.CalculateVotingPower(math.NewInt(50), math.NewInt(100))
	require.Equal(votingPower, math.NewInt(12.5*1e6))
	// 100/0 --> 100/0
	votingPower = disputekeeper.CalculateVotingPower(math.NewInt(100), math.NewInt(0))
	require.Equal(votingPower, math.NewInt(0))
	// 0/100 --> 0/400
	votingPower = disputekeeper.CalculateVotingPower(math.NewInt(0), math.NewInt(100))
	require.Equal(votingPower, math.NewInt(0))
}

func (s *KeeperTestSuite) TestTallyVote() {
	require := s.Require()
	ctx := s.ctx
	k := s.disputeKeeper
	bk := s.bankKeeper
	require.NotNil(k)
	require.NotNil(ctx)
	id := uint64(1)

	require.Error(k.TallyVote(ctx, id))

	require.NoError(k.Votes.Set(ctx, id, types.Vote{
		Id:         id,
		VoteResult: types.VoteResult_NO_TALLY,
		Executed:   false,
	}))
	require.Error(k.TallyVote(ctx, id))

	require.NoError(k.Disputes.Set(ctx, id, types.Dispute{
		DisputeId: id,
		HashId:    []byte("hashId"),
	}))
	require.Error(k.TallyVote(ctx, id))

	require.NoError(k.BlockInfo.Set(ctx, []byte("hashId"), types.BlockInfo{
		TotalReporterPower: math.NewInt(250 * 1e6),
		TotalUserTips:      math.NewInt(250 * 1e6),
	}))
	bk.On("GetSupply", ctx, layertypes.BondDenom).Return(sdk.Coin{Denom: layertypes.BondDenom, Amount: math.NewInt(1_000 * 1e6)}, nil)
	require.ErrorContains(k.TallyVote(ctx, id), "vote period not ended and quorum not reached")

	// set team vote
	teamAddr, err := k.GetTeamAddress(ctx)
	require.NoError(err)
	require.NoError(k.Voter.Set(ctx, collections.Join(id, teamAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.NewInt(250 * 1e6)}))
	_, err = k.SetTeamVote(ctx, id, teamAddr)
	require.NoError(err)
	require.Error(k.TallyVote(ctx, id))

	// set user vote
	userAddr := sample.AccAddressBytes()
	require.NoError(k.UsersGroup.Set(ctx, collections.Join(id, userAddr.Bytes()), math.NewInt(250*1e6)))
	require.NoError(k.Voter.Set(ctx, collections.Join(id, userAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.NewInt(250 * 1e6)}))
	require.NoError(k.TallyVote(ctx, id))
}
