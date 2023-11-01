package keeper_test

import (
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestGetParams() {
	require := s.Require()
	params := types.DefaultParams()

	s.oracleKeeper.SetParams(s.ctx, params)

	require.EqualValues(params, s.oracleKeeper.GetParams(s.ctx))
}
