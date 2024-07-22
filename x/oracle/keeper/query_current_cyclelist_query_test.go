package keeper_test

import (
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestQueryCurrentCyclelist() {
	require := s.Require()
	ctx := s.ctx
	q := s.queryClient

	// nil request
	_, err := q.CurrentCyclelistQuery(ctx, nil)
	require.ErrorContains(err, "invalid request")

	res, err := q.CurrentCyclelistQuery(ctx, &types.QueryCurrentCyclelistQueryRequest{})
	require.NoError(err)
	fmt.Println(res)
}
