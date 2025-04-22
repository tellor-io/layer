package keeper_test

import (
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
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

	timestampBefore := uint64(1)
	timestamp := time.UnixMilli(int64(timestampBefore))

	getDataBeforeResponse, err = querier.GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId:   "1234abcd",
		Timestamp: timestampBefore,
	})
	s.ErrorContains(err, "no aggregate report found before timestamp")
	s.Nil(getDataBeforeResponse)

	queryId := "1234abcd"
	qIdBz, err := utils.QueryBytesFromString(queryId)
	s.NoError(err)
	agg := types.Aggregate{QueryId: qIdBz, AggregateReporter: sample.AccAddress()}
	s.NoError(s.oracleKeeper.Aggregates.Set(s.ctx, collections.Join(qIdBz, uint64(timestamp.UnixMilli())), agg))
	getDataBeforeResponse, err = querier.GetDataBefore(ctx, &types.QueryGetDataBeforeRequest{
		QueryId:   queryId,
		Timestamp: timestampBefore + 1,
	})
	s.NoError(err)
	s.Equal(getDataBeforeResponse.Timestamp, uint64(timestamp.UnixMilli()))
	s.Equal(getDataBeforeResponse.Aggregate.AggregatePower, agg.AggregatePower)
	s.Equal(getDataBeforeResponse.Aggregate.AggregateValue, agg.AggregateValue)
	s.Equal(getDataBeforeResponse.Aggregate.AggregateReporter, agg.AggregateReporter)
	s.Equal(getDataBeforeResponse.Aggregate.Flagged, agg.Flagged)
	s.Equal(getDataBeforeResponse.Aggregate.Index, agg.Index)
	s.Equal(getDataBeforeResponse.Aggregate.Height, agg.Height)
	s.Equal(getDataBeforeResponse.Aggregate.MicroHeight, agg.MicroHeight)
	s.Equal(getDataBeforeResponse.Aggregate.MetaId, agg.MetaId)
	s.Equal(getDataBeforeResponse.Aggregate.QueryId, hex.EncodeToString(agg.QueryId))
}
