package keeper_test

func (s *KeeperTestSuite) TestGetOpenDisputes() {
	require := s.Require()
	res := s.disputeKeeper.GetOpenDisputeIds(s.ctx)
	require.Nil(res.Ids)
}
