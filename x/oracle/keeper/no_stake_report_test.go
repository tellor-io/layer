package keeper_test

import (
	"time"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestGetNoStakeReportByQueryIdTimestamp() {
	require := s.Require()

	reporter := sample.AccAddressBytes()

	queryId := []byte("QueryId")
	timestamp := time.Now().UTC()
	report := types.NoStakeMicroReport{
		Reporter:    reporter,
		QueryData:   []byte("QueryData"),
		Timestamp:   timestamp,
		BlockNumber: 1,
		Value:       "value",
	}
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(timestamp.UnixMilli())), report))

	res, err := s.oracleKeeper.GetNoStakeReportByQueryIdTimestamp(s.ctx, queryId, uint64(timestamp.UnixMilli()))
	require.NoError(err)
	require.Equal(res.Value, report.Value)
	require.Equal(res.Timestamp, timestamp)
	require.Equal(res.BlockNumber, report.BlockNumber)
	require.Equal(res.QueryData, report.QueryData)
	require.Equal(res.Reporter, report.Reporter)
}
