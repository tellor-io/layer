package keeper_test

func (s *KeeperTestSuite) TestGetOpenDisputes() {
	res, err := s.disputeKeeper.OpenDisputes.Get(s.ctx)
	s.Nil(err)
	s.Nil(res.Ids)
}
