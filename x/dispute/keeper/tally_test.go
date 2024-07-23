package keeper_test

import (
	"time"

	"github.com/tellor-io/layer/testutil"
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

func (s *KeeperTestSuite) TestTallyVoteQuorumReachedWithoutTokenHolders() {
	require := s.Require()
	ctx := s.ctx
	k := s.disputeKeeper
	bk := s.bankKeeper

	id1 := uint64(1)
	userAddr := sample.AccAddressBytes()
	tokenHolderAddr := sample.AccAddressBytes()
	reporterAddr := sample.AccAddressBytes()
	teamAddr, err := k.GetTeamAddress(ctx)
	require.NoError(err)

	require.Error(k.TallyVote(ctx, id1))

	require.NoError(k.Votes.Set(ctx, id1, types.Vote{
		Id:         id1,
		VoteResult: types.VoteResult_NO_TALLY,
		Executed:   false,
	}))
	require.Error(k.TallyVote(ctx, id1))

	require.NoError(k.Disputes.Set(ctx, id1, types.Dispute{
		DisputeId: id1,
		HashId:    []byte("hashId"),
	}))
	require.Error(k.TallyVote(ctx, id1))

	require.NoError(k.BlockInfo.Set(ctx, []byte("hashId"), types.BlockInfo{
		TotalReporterPower: math.NewInt(250 * 1e6),
		TotalUserTips:      math.NewInt(250 * 1e6),
	}))
	bk.On("GetSupply", ctx, layertypes.BondDenom).Return(sdk.Coin{Denom: layertypes.BondDenom, Amount: math.NewInt(250 * 1e6)}, nil)
	require.ErrorContains(k.TallyVote(ctx, id1), "vote period not ended and quorum not reached")

	// set team vote (25% support)
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, teamAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.NewInt(250 * 1e6)}))
	_, err = k.SetTeamVote(ctx, id1, teamAddr)
	require.NoError(err)
	bk.On("GetBalance", ctx, teamAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()
	bk.On("GetBalance", ctx, userAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()
	bk.On("GetBalance", ctx, reporterAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()
	bk.On("GetBalance", ctx, tokenHolderAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()
	require.Error(k.TallyVote(ctx, id1))

	// set user vote (25% support)
	require.NoError(k.UsersGroup.Set(ctx, collections.Join(id1, userAddr.Bytes()), math.NewInt(250*1e6)))
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, userAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.NewInt(250 * 1e6)}))

	// set reporter vote (25% support)
	require.NoError(k.ReportersGroup.Set(ctx, collections.Join(id1, reporterAddr.Bytes()), math.NewInt(250*1e6)))
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, reporterAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.NewInt(250 * 1e6)}))

	// Vote ends with 75% Support
	require.NoError(k.TallyVote(ctx, id1))
	dispute, err := k.Disputes.Get(ctx, id1)
	require.NoError(err)
	require.Equal(dispute.DisputeStatus, types.Resolved)
	require.Equal(dispute.DisputeId, id1)
	require.Equal(dispute.HashId, []byte("hashId"))
	require.Equal(dispute.Open, false)
}

func (s *KeeperTestSuite) TestTallyVoteQuorumReachedWithTokenHolders() {
	require := s.Require()
	ctx := s.ctx
	k := s.disputeKeeper
	bk := s.bankKeeper

	id1 := uint64(1)
	userAddr := sample.AccAddressBytes()
	tokenHolderAddr := sample.AccAddressBytes()
	reporterAddr := sample.AccAddressBytes()
	teamAddr, err := k.GetTeamAddress(ctx)
	require.NoError(err)

	require.NoError(k.Votes.Set(ctx, id1, types.Vote{
		Id:         id1,
		VoteResult: types.VoteResult_NO_TALLY,
		Executed:   false,
	}))

	require.NoError(k.Disputes.Set(ctx, id1, types.Dispute{
		DisputeId: id1,
		HashId:    []byte("hashId"),
	}))

	require.NoError(k.BlockInfo.Set(ctx, []byte("hashId"), types.BlockInfo{
		TotalReporterPower: math.NewInt(250 * 1e6),
		TotalUserTips:      math.NewInt(250 * 1e6),
	}))
	bk.On("GetSupply", ctx, layertypes.BondDenom).Return(sdk.Coin{Denom: layertypes.BondDenom, Amount: math.NewInt(250 * 1e6)}, nil)

	// set team vote (25% support)
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, teamAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_AGAINST, VoterPower: math.NewInt(250 * 1e6)}))
	_, err = k.SetTeamVote(ctx, id1, teamAddr)
	require.NoError(err)
	bk.On("GetBalance", ctx, teamAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()
	bk.On("GetBalance", ctx, userAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()
	bk.On("GetBalance", ctx, reporterAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()
	bk.On("GetBalance", ctx, tokenHolderAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()

	// set user vote (25% support)
	require.NoError(k.UsersGroup.Set(ctx, collections.Join(id1, userAddr.Bytes()), math.NewInt(250*1e6)))
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, userAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_AGAINST, VoterPower: math.NewInt(250 * 1e6)}))

	// set reporter vote (25% support)
	require.NoError(k.ReportersGroup.Set(ctx, collections.Join(id1, reporterAddr.Bytes()), math.NewInt(1*1e6)))
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, reporterAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_AGAINST, VoterPower: math.NewInt(1 * 1e6)}))

	// set token holder vote
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, tokenHolderAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_AGAINST, VoterPower: math.NewInt(250 * 1e6)}))

	require.NoError(k.TallyVote(ctx, id1))
	dispute, err := k.Disputes.Get(ctx, id1)
	require.NoError(err)
	require.Equal(dispute.DisputeStatus, types.Resolved)
	require.Equal(dispute.DisputeId, id1)
	require.Equal(dispute.HashId, []byte("hashId"))
	require.Equal(dispute.Open, false)
}

func (s *KeeperTestSuite) TestTallyVoteQuorumNotReachedVotePeriodEnds() {
	require := s.Require()
	ctx := s.ctx
	k := s.disputeKeeper
	bk := s.bankKeeper

	id1 := uint64(1)
	userAddr := sample.AccAddressBytes()
	tokenHolderAddr := sample.AccAddressBytes()
	reporterAddr := sample.AccAddressBytes()
	teamAddr, err := k.GetTeamAddress(ctx)
	require.NoError(err)

	require.NoError(k.Votes.Set(ctx, id1, types.Vote{
		Id:         id1,
		VoteResult: types.VoteResult_NO_TALLY,
		Executed:   false,
		VoteEnd:    ctx.HeaderInfo().Time,
	}))

	require.NoError(k.Disputes.Set(ctx, id1, types.Dispute{
		DisputeId: id1,
		HashId:    []byte("hashId"),
	}))

	ctx = testutil.WithBlockTime(ctx, ctx.HeaderInfo().Time.Add(120*time.Hour))
	require.NoError(k.BlockInfo.Set(ctx, []byte("hashId"), types.BlockInfo{
		TotalReporterPower: math.NewInt(250 * 1e6),
		TotalUserTips:      math.NewInt(250 * 1e6),
	}))
	bk.On("GetSupply", ctx, layertypes.BondDenom).Return(sdk.Coin{Denom: layertypes.BondDenom, Amount: math.NewInt(250 * 1e6)}, nil).Once()

	// set team vote (25% invalid)
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, teamAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_INVALID, VoterPower: math.NewInt(250 * 1e6)}))
	_, err = k.SetTeamVote(ctx, id1, teamAddr)
	require.NoError(err)
	bk.On("GetBalance", ctx, teamAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(1*1e6)), nil).Once()
	bk.On("GetBalance", ctx, userAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(1*1e6)), nil).Once()
	bk.On("GetBalance", ctx, reporterAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(1*1e6)), nil).Once()
	bk.On("GetBalance", ctx, tokenHolderAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(1*1e6)), nil).Once()

	// set user vote (<1 % invalid)
	require.NoError(k.UsersGroup.Set(ctx, collections.Join(id1, userAddr.Bytes()), math.NewInt(1*1e6)))
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, userAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_INVALID, VoterPower: math.NewInt(1 * 1e6)}))

	// set reporter vote (<1% invalid)
	require.NoError(k.ReportersGroup.Set(ctx, collections.Join(id1, reporterAddr.Bytes()), math.NewInt(1*1e6)))
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, reporterAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_INVALID, VoterPower: math.NewInt(1 * 1e6)}))

	// set token holder vote (<1% invalid)
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, tokenHolderAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_INVALID, VoterPower: math.NewInt(1 * 1e6)}))

	require.NoError(k.TallyVote(ctx, id1))

	dispute, err := k.Disputes.Get(ctx, id1)
	require.NoError(err)
	require.Equal(dispute.DisputeStatus, types.Resolved)
	require.Equal(dispute.Open, false)
}
