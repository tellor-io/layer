package keeper_test

import "github.com/tellor-io/layer/x/dispute/types"

func (s *KeeperTestSuite) TestVote() {
	require := s.Require()
	s.TestMsgProposeDispute()
	voteMsg := types.MsgVote{
		Voter: Addr.String(),
		Id:    0,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	}

	_, err := s.msgServer.Vote(s.goCtx, &voteMsg)
	require.Nil(err)
}
