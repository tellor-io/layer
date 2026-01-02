package keeper_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"
)

func (s *KeeperTestSuite) TestGetReporterLiveness() {
	require := s.Require()
	k := s.oracleKeeper
	q := s.queryClient
	ctx := s.ctx

	reporter := sample.AccAddressBytes()
	reporterAddr := reporter.String()

	// nil request
	res, err := q.GetReporterLiveness(ctx, nil)
	require.ErrorContains(err, "invalid request")
	require.Nil(res)

	// invalid reporter address
	res, err = q.GetReporterLiveness(ctx, &types.QueryGetReporterLivenessRequest{
		Reporter: "invalid_address",
	})
	require.ErrorContains(err, "invalid reporter address")
	require.Nil(res)

	// valid reporter, no reports yet, no aggregates
	res, err = q.GetReporterLiveness(ctx, &types.QueryGetReporterLivenessRequest{
		Reporter: reporterAddr,
	})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(uint64(0), res.ReporterReports)
	require.Equal(uint64(0), res.TotalAggregates)
	require.True(res.PercentLiveness.Equal(math.LegacyZeroDec()))
	require.Equal(int64(0), res.LastReportTime)

	// add some aggregates (increment total count)
	require.NoError(k.IncrementTotalAggregates(ctx))
	require.NoError(k.IncrementTotalAggregates(ctx))
	require.NoError(k.IncrementTotalAggregates(ctx))

	// reporter hasn't reported yet
	res, err = q.GetReporterLiveness(ctx, &types.QueryGetReporterLivenessRequest{
		Reporter: reporterAddr,
	})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(uint64(0), res.ReporterReports)
	require.Equal(uint64(3), res.TotalAggregates)
	require.True(res.PercentLiveness.Equal(math.LegacyZeroDec()))

	// reporter submits 2 reports
	require.NoError(k.TrackReporterParticipation(ctx, reporter))
	require.NoError(k.TrackReporterParticipation(ctx, reporter))

	// set last report time
	lastReportTime := int64(1234567890)
	require.NoError(k.SetReporterLastReportTime(ctx, reporter, lastReportTime))

	// check liveness: 2 reports out of 3 aggregates = 66.666...%
	res, err = q.GetReporterLiveness(ctx, &types.QueryGetReporterLivenessRequest{
		Reporter: reporterAddr,
	})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(uint64(2), res.ReporterReports)
	require.Equal(uint64(3), res.TotalAggregates)
	// 2/3 * 100 = 66.666...
	expectedPercent := math.LegacyNewDec(2).Quo(math.LegacyNewDec(3)).MulInt64(100)
	require.True(res.PercentLiveness.Equal(expectedPercent))
	require.Equal(lastReportTime, res.LastReportTime)

	// add more aggregates and reports
	require.NoError(k.IncrementTotalAggregates(ctx))
	require.NoError(k.IncrementTotalAggregates(ctx))
	require.NoError(k.TrackReporterParticipation(ctx, reporter))

	// now: 3 reports out of 5 aggregates = 60%
	res, err = q.GetReporterLiveness(ctx, &types.QueryGetReporterLivenessRequest{
		Reporter: reporterAddr,
	})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(uint64(3), res.ReporterReports)
	require.Equal(uint64(5), res.TotalAggregates)
	// 3/5 * 100 = 60
	expectedPercent = math.LegacyNewDec(3).Quo(math.LegacyNewDec(5)).MulInt64(100)
	require.True(res.PercentLiveness.Equal(expectedPercent))

	// test 100% liveness
	require.NoError(k.TrackReporterParticipation(ctx, reporter))
	require.NoError(k.TrackReporterParticipation(ctx, reporter))
	// now: 5 reports out of 5 aggregates = 100%
	res, err = q.GetReporterLiveness(ctx, &types.QueryGetReporterLivenessRequest{
		Reporter: reporterAddr,
	})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(uint64(5), res.ReporterReports)
	require.Equal(uint64(5), res.TotalAggregates)
	require.True(res.PercentLiveness.Equal(math.LegacyNewDec(100)))
}
