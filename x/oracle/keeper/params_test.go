package keeper_test

import (
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestGetParams() {
	params := types.DefaultParams()

	s.NoError(s.oracleKeeper.SetParams(s.ctx, params))
	p, err := s.oracleKeeper.GetParams(s.ctx)
	s.NoError(err)
	s.EqualValues(params, p)
}
