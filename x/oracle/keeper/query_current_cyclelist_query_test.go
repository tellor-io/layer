package keeper_test

import (
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
)

func (s *KeeperTestSuite) TestQueryCurrentCyclelist() {
	require := s.Require()
	ctx := s.ctx
	q := s.queryClient
	s.registryKeeper.On("GetSpec", ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
	s.NoError(s.oracleKeeper.RotateQueries(s.ctx))
	// nil request
	_, err := q.CurrentCyclelistQuery(ctx, nil)
	require.ErrorContains(err, "invalid request")

	res, err := q.CurrentCyclelistQuery(ctx, &types.QueryCurrentCyclelistQueryRequest{})
	require.NoError(err)
	fmt.Println(res)
}

func (s *KeeperTestSuite) TestNextCyclelistQuery() {
	require := s.Require()
	ctx := s.ctx
	q := s.queryClient

	// nil request
	_, err := q.NextCyclelistQuery(ctx, nil)
	require.ErrorContains(err, "invalid request")

	// good request
	res, err := q.NextCyclelistQuery(ctx, &types.QueryNextCyclelistQueryRequest{})
	require.NoError(err)
	require.NotNil(res)
}
