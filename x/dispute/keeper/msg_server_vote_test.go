package keeper_test

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"
)

func (s *KeeperTestSuite) TestVote() {
	// Add dispute
	addr := s.TestMsgProposeDisputeFromAccount()

	// mock dependency modules
	s.bankKeeper.On("GetBalance", s.ctx, addr, layer.BondDenom).Return(sdk.NewCoin(layer.BondDenom, math.NewInt(1)))
	s.bankKeeper.On("GetSupply", s.ctx, layer.BondDenom).Return(sdk.NewCoin(layer.BondDenom, math.NewInt(1)))
	s.oracleKeeper.On("GetTipsAtBlockForTipper", s.ctx, s.ctx.BlockHeight(), addr).Return(math.NewInt(1), nil)
	s.reporterKeeper.On("Delegation", s.ctx, addr).Return(reportertypes.Delegation{
		Reporter: addr.String(),
		Amount:   math.NewInt(1),
	}, nil)
	s.reporterKeeper.On("GetReporterTokensAtBlock", s.ctx, addr.Bytes(), s.ctx.BlockHeight()).Return(math.NewInt(1), nil)

	voteMsg := types.MsgVote{
		Voter: addr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	}
	// vote should have started
	_, err := s.msgServer.Vote(s.ctx, &voteMsg)
	s.NoError(err)

	_, err = s.msgServer.Vote(s.ctx, &voteMsg)
	s.Error(err)

	voterVote, err := s.disputeKeeper.Voter.Get(s.ctx, collections.Join(uint64(1), addr))
	s.NoError(err)

	s.Equal(voterVote.Vote, types.VoteEnum_VOTE_SUPPORT)

	// start voting, this method is check on beginblock
	vote, err := s.disputeKeeper.Votes.Get(s.ctx, 1)
	s.NoError(err)
	s.NotNil(vote)
	iter, err := s.disputeKeeper.Voter.Indexes.VotersById.MatchExact(s.ctx, uint64(1))
	s.NoError(err)
	keys, err := iter.PrimaryKeys()
	s.NoError(err)
	s.Equal(keys[0].K2(), addr)
	s.Equal(vote.VoteResult, types.VoteResult_NO_TALLY)
	s.Equal(vote.Id, uint64(1))
}
