package keeper_test

import (
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
)

func (s *KeeperTestSuite) TestQueryGetDataAfter() {
	ctx := s.ctx
	querier := keeper.NewQuerier(s.oracleKeeper)
	getDataAfterResponse, err := querier.GetDataAfter(ctx, nil)
	s.ErrorContains(err, "invalid request")
	s.Nil(getDataAfterResponse)

	getDataAfterResponse, err = querier.GetDataAfter(ctx, &types.QueryGetDataAfterRequest{
		QueryId: "z",
	})
	s.ErrorContains(err, "invalid queryId")
	s.Nil(getDataAfterResponse)

	timestampAfter := uint64(1)
	timestamp := time.UnixMilli(int64(timestampAfter))

	getDataAfterResponse, err = querier.GetDataAfter(ctx, &types.QueryGetDataAfterRequest{
		QueryId:   "1234abcd",
		Timestamp: timestampAfter,
	})
	s.ErrorContains(err, "no aggregate report found after timestamp")
	s.Nil(getDataAfterResponse)

	queryId := "1234abcd"
	qIdBz, err := utils.QueryBytesFromString(queryId)
	s.NoError(err)
	agg := types.Aggregate{QueryId: qIdBz, AggregateReporter: sample.AccAddress()}
	s.NoError(s.oracleKeeper.Aggregates.Set(s.ctx, collections.Join(qIdBz, uint64(timestamp.UnixMilli())), agg))
	getDataAfterResponse, err = querier.GetDataAfter(ctx, &types.QueryGetDataAfterRequest{
		QueryId:   queryId,
		Timestamp: timestampAfter - 1,
	})
	s.NoError(err)
	s.Equal(getDataAfterResponse.Timestamp, uint64(timestamp.UnixMilli()))
	s.Equal(getDataAfterResponse.Aggregate, &agg)
}
