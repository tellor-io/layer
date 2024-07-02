package keeper_test

import "github.com/tellor-io/layer/x/dispute/types"

func (s *KeeperTestSuite) TestMsgTallyVote() {
	require := s.Require()
	require.NotNil(s.msgServer)
	require.NotNil(s.ctx)

	_, err := s.msgServer.TallyVote(s.ctx, &types.MsgTallyVote{
		CallerAddress: "caller_address",
		DisputeId:     uint64(1),
	})
	require.Error(err)
}
