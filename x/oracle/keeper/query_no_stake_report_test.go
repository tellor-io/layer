package keeper_test

import (
	"time"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestGetReportersNoStakeReports() {
	require := s.Require()

	q := keeper.NewQuerier(s.oracleKeeper)
	require.NotNil(q)

	reporter := sample.AccAddressBytes()

	// no reports
	response, err := q.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
	})
	require.NoError(err)
	require.Equal(&types.QueryGetReportersNoStakeReportsResponse{NoStakeReports: []*types.NoStakeMicroReport{}}, response)

	timestamp := time.Now().UTC()
	// 1 report
	report := types.NoStakeMicroReport{
		Reporter:    reporter.String(),
		QueryData:   []byte("QueryData"),
		Timestamp:   timestamp,
		BlockNumber: 1,
		Value:       "value",
	}
	queryId := []byte("QueryId")
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(timestamp.UnixMilli())), report))
	response, err = q.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
	})
	require.NoError(err)
	require.Equal(response.NoStakeReports[0].Value, report.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, timestamp)
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].QueryData, report.QueryData)
	require.Equal(response.NoStakeReports[0].Reporter, report.Reporter)

	// 2 reports
	report2 := types.NoStakeMicroReport{
		Reporter:    reporter.String(),
		QueryData:   []byte("QueryData2"),
		Timestamp:   timestamp,
		BlockNumber: 2,
		Value:       "value2",
	}
	queryId2 := []byte("QueryId2")
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId2, uint64(timestamp.UnixMilli())), report2))
	response, err = q.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 2)
	require.Equal(response.NoStakeReports[0].Value, report.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, timestamp)
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].QueryData, report.QueryData)
	require.Equal(response.NoStakeReports[0].Reporter, report.Reporter)
	require.Equal(response.NoStakeReports[1].Value, report2.Value)
	require.Equal(response.NoStakeReports[1].Timestamp, timestamp)
	require.Equal(response.NoStakeReports[1].BlockNumber, report2.BlockNumber)
	require.Equal(response.NoStakeReports[1].QueryData, report2.QueryData)
	require.Equal(response.NoStakeReports[1].Reporter, report2.Reporter)
}
