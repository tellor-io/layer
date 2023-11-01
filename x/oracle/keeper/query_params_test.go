package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestParamsQuery() {
	require := s.Require()
	wctx := sdk.WrapSDKContext(s.ctx)
	params := types.DefaultParams()
	s.oracleKeeper.SetParams(s.ctx, params)

	response, err := s.oracleKeeper.Params(wctx, &types.QueryParamsRequest{})
	require.NoError(err)
	require.Equal(&types.QueryParamsResponse{Params: params}, response)
}
