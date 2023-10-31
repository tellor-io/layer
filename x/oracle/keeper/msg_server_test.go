package keeper_test

func (s *KeeperTestSuite) TestMsgServer() {
	require := s.Require()

	require.NotNil(s.msgServer)
	require.NotNil(s.ctx)
}
