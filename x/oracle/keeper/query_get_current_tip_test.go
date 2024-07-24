package keeper_test

import (
	"time"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"
)

func (s *KeeperTestSuite) TestGetCurrentTip() {
	require := s.Require()
	k := s.oracleKeeper
	q := s.queryClient
	ctx := s.ctx

	// nil request
	res, err := q.GetCurrentTip(ctx, nil)
	require.ErrorContains(err, "invalid request")
	require.Nil(res)

	// bad querydata
	res, err = q.GetCurrentTip(ctx, &types.QueryGetCurrentTipRequest{
		QueryData: "badQData",
	})
	require.Error(err)
	require.Nil(res)

	// good queryData, no tips
	res, err = q.GetCurrentTip(ctx, &types.QueryGetCurrentTipRequest{
		QueryData: queryData,
	})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(res.Tips, math.ZeroInt())

	// good queryData, 1 tip
	queryID, err := utils.QueryIDFromDataString(queryData)
	require.NoError(err)
	require.NoError(k.Query.Set(ctx, queryID, types.QueryMeta{
		Amount:             math.NewInt(10),
		Id:                 1,
		Expiration:         ctx.BlockTime().Add(time.Hour),
		HasRevealedReports: false,
		QueryType:          "SpotPrice",
	}))
	res, err = q.GetCurrentTip(ctx, &types.QueryGetCurrentTipRequest{
		QueryData: queryData,
	})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(res.Tips, math.NewInt(10))
}
