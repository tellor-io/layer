package keeper_test

import (
	"encoding/hex"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
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
	require.Equal(len(response.NoStakeReports), 0)

	timestamp := time.Now().UTC()
	// 1 report
	report := types.NoStakeMicroReport{
		Reporter:    reporter,
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
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].QueryData, hex.EncodeToString(report.QueryData))
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())

	// 2 reports
	report2 := types.NoStakeMicroReport{
		Reporter:    reporter,
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
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].QueryData, hex.EncodeToString(report.QueryData))
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())
	require.Equal(response.NoStakeReports[1].Value, report2.Value)
	require.Equal(response.NoStakeReports[1].Timestamp, uint64(timestamp.UnixMilli()))
	require.Equal(response.NoStakeReports[1].BlockNumber, report2.BlockNumber)
	require.Equal(response.NoStakeReports[1].QueryData, hex.EncodeToString(report2.QueryData))
	require.Equal(response.NoStakeReports[1].Reporter, sdk.AccAddress(report2.Reporter).String())
}

func (s *KeeperTestSuite) TestGetNoStakeReportsByQueryId() {
	require := s.Require()

	q := keeper.NewQuerier(s.oracleKeeper)
	require.NotNil(q)

	reporter := sample.AccAddressBytes()

	// no reports
	queryId := []byte("QueryId")
	response, err := q.GetNoStakeReportsByQueryId(s.ctx, &types.QueryGetNoStakeReportsByQueryIdRequest{
		QueryId: hex.EncodeToString(queryId),
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 0)

	timestamp := time.Now().UTC()
	// 1 report
	report := types.NoStakeMicroReport{
		Reporter:    reporter,
		QueryData:   []byte("QueryData"),
		Timestamp:   timestamp,
		BlockNumber: 1,
		Value:       "value",
	}
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(timestamp.UnixMilli())), report))
	response, err = q.GetNoStakeReportsByQueryId(s.ctx, &types.QueryGetNoStakeReportsByQueryIdRequest{
		QueryId: hex.EncodeToString(queryId),
	})
	require.NoError(err)
	require.Equal(response.NoStakeReports[0].Value, report.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].QueryData, hex.EncodeToString(report.QueryData))
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())

	// 2 reports
	s.ctx = s.ctx.WithBlockHeight(2).WithBlockTime(timestamp.Add(time.Second * 10))
	report2 := types.NoStakeMicroReport{
		Reporter:    reporter,
		QueryData:   []byte("QueryData2"),
		Timestamp:   timestamp,
		BlockNumber: 2,
		Value:       "value2",
	}
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(s.ctx.BlockTime().UnixMilli())), report2))
	response, err = q.GetNoStakeReportsByQueryId(s.ctx, &types.QueryGetNoStakeReportsByQueryIdRequest{
		QueryId: hex.EncodeToString(queryId),
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 2)
	require.Equal(response.NoStakeReports[0].Value, report.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].QueryData, hex.EncodeToString(report.QueryData))
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())
	require.Equal(response.NoStakeReports[1].Value, report2.Value)
	require.Equal(response.NoStakeReports[1].Timestamp, uint64(timestamp.UnixMilli()))
	require.Equal(response.NoStakeReports[1].BlockNumber, report2.BlockNumber)
	require.Equal(response.NoStakeReports[1].QueryData, hex.EncodeToString(report2.QueryData))
	require.Equal(response.NoStakeReports[1].Reporter, sdk.AccAddress(report2.Reporter).String())
}
