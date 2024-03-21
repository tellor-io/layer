package keeper_test

func (s *KeeperTestSuite) TestGetOpenDisputes() {
	res, err := s.disputeKeeper.GetOpenDisputeIds(s.ctx)
	s.Nil(err)
	s.Nil(res.Ids)
}
