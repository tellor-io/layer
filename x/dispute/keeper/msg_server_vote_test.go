package keeper_test

import (
	"fmt"

	"github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
)

func (s *KeeperTestSuite) TestVote() {
	// k := s.disputeKeeper
	// Create dispute
	addr, dispute := s.TestMsgProposeDisputeFromAccount()

	// mock dependency modules

	s.oracleKeeper.On("GetTipsAtBlockForTipper", s.ctx, uint64(s.ctx.BlockHeight()), addr).Return(math.NewInt(1), nil)
	s.reporterKeeper.On("Delegation", s.ctx, addr).Return(reportertypes.Selection{
		Reporter: addr.Bytes(),
	}, nil)
	s.reporterKeeper.On("GetReporterTokensAtBlock", s.ctx, addr.Bytes(), uint64(s.ctx.BlockHeight())).Return(math.NewInt(1), nil)
	s.reporterKeeper.On("GetDelegatorTokensAtBlock", s.ctx, addr.Bytes(), uint64(s.ctx.BlockHeight())).Return(math.NewInt(100), nil).Once()
	s.reporterKeeper.On("TotalReporterPower", s.ctx).Return(math.NewInt(1), nil)
	s.oracleKeeper.On("GetTotalTips", s.ctx).Return(math.NewInt(1), nil)
	// need to manually call setblock info - happens in endblock normally
	err := s.disputeKeeper.SetBlockInfo(s.ctx, dispute.HashId)
	s.NoError(err)
	voteMsg := types.MsgVote{
		Voter: addr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	}
	// vote should have started
	_, err = s.msgServer.Vote(s.ctx, &voteMsg)
	s.NoError(err)

	_, err = s.msgServer.Vote(s.ctx, &voteMsg)
	s.Error(err)

	voterVote, err := s.disputeKeeper.Voter.Get(s.ctx, collections.Join(uint64(1), addr.Bytes()))
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
	s.Equal(keys[0].K2(), addr.Bytes())

	// vote from team
	teamAddr, err := s.disputeKeeper.GetTeamAddress(s.ctx)
	s.NoError(err)
	_, err = s.disputeKeeper.SetTeamVote(s.ctx, uint64(1), teamAddr, types.VoteEnum_VOTE_SUPPORT)
	s.NoError(err)

	// check on voting tally
	votes, err := s.disputeKeeper.VoteCountsByGroup.Get(s.ctx, uint64(1))
	s.NoError(err)
	fmt.Println(votes)
	// vote calls tally, enough ppl have voted to reach quorum
	fmt.Println(vote.VoteResult)
	s.Equal(vote.VoteResult, types.VoteResult_SUPPORT)
	s.Equal(vote.Id, uint64(1))
}

// TODO: more tests
