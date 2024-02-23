package keeper_test

import (
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestParamsQuery() {
	q := keeper.NewQuerier(s.oracleKeeper)

	response, err := q.Params(s.ctx, &types.QueryParamsRequest{})
	s.NoError(err)
	s.Equal(&types.QueryParamsResponse{Params: types.DefaultParams()}, response)
}
