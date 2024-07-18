package keeper_test

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestQueryGetUserTipTotal() {
	require := s.Require()
	k := s.oracleKeeper
	q := s.queryClient
	ctx := s.ctx

	// nil request
	_, err := q.GetUserTipTotal(ctx, nil)
	require.ErrorContains(err, "invalid request")

	// query with 0 tips
	tipper := sample.AccAddressBytes()
	req := &types.QueryGetUserTipTotalRequest{
		Tipper: tipper.String(),
	}
	res, err := q.GetUserTipTotal(ctx, req)
	require.NoError(err)
	require.Equal(res.TotalTips, math.ZeroInt())

	// query after tip is set
	require.NoError(k.TipperTotal.Set(ctx, collections.Join(tipper.Bytes(), ctx.BlockHeight()), math.NewInt(1)))
	total, err := k.TipperTotal.Get(ctx, collections.Join(tipper.Bytes(), ctx.BlockHeight()))
	require.NoError(err)
	require.Equal(total, math.NewInt(1))

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	res, err = q.GetUserTipTotal(ctx, req)
	require.NoError(err)
	require.Equal(res.TotalTips, math.NewInt(1))
}
