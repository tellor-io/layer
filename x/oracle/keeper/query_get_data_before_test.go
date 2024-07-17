package keeper_test

import (
	"time"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestQueryGetDataBefore() {
	ctx := s.ctx
	querier := keeper.NewQuerier(s.oracleKeeper)
	getDataBeforeResponse, err := querier.GetDataBefore(ctx, nil)
	s.ErrorContains(err, "invalid request")
	s.Nil(getDataBeforeResponse)

	getDataBeforeResponse, err = querier.GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId: "z",
	})
	s.ErrorContains(err, "invalid queryId")
	s.Nil(getDataBeforeResponse)

	timestampBefore := int64(1)
	timestamp := time.UnixMilli(timestampBefore)

	getDataBeforeResponse, err = querier.GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId:   "1234abcd",
		Timestamp: timestampBefore,
	})
	s.ErrorContains(err, "no aggregate report found before timestamp")
	s.Nil(getDataBeforeResponse)

	queryId := "1234abcd"
	qIdBz, err := utils.QueryBytesFromString(queryId)
	s.NoError(err)
	agg := types.Aggregate{QueryId: qIdBz}
	s.NoError(s.oracleKeeper.Aggregates.Set(s.ctx, collections.Join(qIdBz, timestamp.UnixMilli()), agg))
	getDataBeforeResponse, err = querier.GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId:   queryId,
		Timestamp: timestampBefore,
	})
	s.NoError(err)
	s.Equal(getDataBeforeResponse.Timestamp, uint64(timestamp.UnixMilli()))
	s.Equal(getDataBeforeResponse.Aggregate, &agg)
}
