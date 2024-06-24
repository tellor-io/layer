package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func TestGetDataBefore(t *testing.T) {
	k, _, _, ok, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getDataBeforeResponse, err := keeper.NewQuerier(k).GetDataBefore(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getDataBeforeResponse)

	getDataBeforeResponse, err = keeper.NewQuerier(k).GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId: "z",
	})
	require.ErrorContains(t, err, "invalid queryId")
	require.Nil(t, getDataBeforeResponse)

	agg := &oracletypes.Aggregate{}
	timestampBefore := int64(1)
	timestamp := time.Unix(timestampBefore, 0)
	queryId := "1234abcd"
	qIdBz, err := utils.QueryBytesFromString(queryId)
	require.NoError(t, err)
	ok.On("GetAggregateBefore", ctx, qIdBz, time.Unix(timestampBefore, 0)).Return(agg, timestamp, types.ErrSample).Once()

	getDataBeforeResponse, err = keeper.NewQuerier(k).GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId:   "1234abcd",
		Timestamp: int64(timestampBefore),
	})
	require.ErrorContains(t, err, "failed to get aggregate before")
	require.Nil(t, getDataBeforeResponse)

	agg = (*oracletypes.Aggregate)(nil)
	ok.On("GetAggregateBefore", ctx, qIdBz, time.Unix(timestampBefore, 0)).Return(agg, timestamp, nil).Once()

	getDataBeforeResponse, err = keeper.NewQuerier(k).GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId:   "1234abcd",
		Timestamp: int64(timestampBefore),
	})
	require.ErrorContains(t, err, "aggregate before not found")
	require.Nil(t, getDataBeforeResponse)

	agg = &oracletypes.Aggregate{}
	ok.On("GetAggregateBefore", ctx, qIdBz, time.Unix(timestampBefore, 0)).Return(agg, timestamp, nil).Once()

	getDataBeforeResponse, err = keeper.NewQuerier(k).GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId:   "1234abcd",
		Timestamp: int64(timestampBefore),
	})
	require.NoError(t, err)
	require.Equal(t, getDataBeforeResponse.Timestamp, uint64(timestamp.Unix()))
	require.Equal(t, getDataBeforeResponse.Aggregate, agg)

}
