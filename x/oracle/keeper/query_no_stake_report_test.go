package keeper_test

import (
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
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

	timestamp1 := time.Now().UTC().Add(time.Second * -10)
	// 1 report
	report := types.NoStakeMicroReport{
		Reporter:    reporter,
		Timestamp:   timestamp1,
		BlockNumber: 1,
		Value:       "value",
	}
	queryId := []byte("QueryId")
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(timestamp1.UnixMilli())), report))
	response, err = q.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
	})
	require.NoError(err)
	require.Equal(response.NoStakeReports[0].Value, report.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp1.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())

	// 2 reports get both
	timestamp2 := time.Now().UTC().Add(time.Second * 10)
	report2 := types.NoStakeMicroReport{
		Reporter:    reporter,
		Timestamp:   timestamp2,
		BlockNumber: 2,
		Value:       "value2",
	}
	queryId2 := []byte("QueryId2")
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId2, uint64(timestamp2.UnixMilli())), report2))
	response, err = q.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 2)
	require.Equal(response.NoStakeReports[0].Value, report.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp1.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())
	require.Equal(response.NoStakeReports[1].Value, report2.Value)
	require.Equal(response.NoStakeReports[1].Timestamp, uint64(timestamp2.UnixMilli()))
	require.Equal(response.NoStakeReports[1].BlockNumber, report2.BlockNumber)
	require.Equal(response.NoStakeReports[1].Reporter, sdk.AccAddress(report2.Reporter).String())

	// return only most recent report
	response, err = q.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
		Pagination: &query.PageRequest{
			Limit:   1,
			Reverse: true,
		},
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 1)
	require.Equal(response.NoStakeReports[0].Value, report2.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp2.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report2.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report2.Reporter).String())

	fmt.Println("nextKey: ", response.Pagination.NextKey)
	fmt.Println("nextKey string: ", hex.EncodeToString(response.Pagination.NextKey))
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

	timestamp1 := time.Now().UTC().Add(time.Second * -10)
	// 1 report
	report := types.NoStakeMicroReport{
		Reporter:    reporter,
		Timestamp:   timestamp1,
		BlockNumber: 1,
		Value:       "value",
	}
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(timestamp1.UnixMilli())), report))
	response, err = q.GetNoStakeReportsByQueryId(s.ctx, &types.QueryGetNoStakeReportsByQueryIdRequest{
		QueryId: hex.EncodeToString(queryId),
	})
	require.NoError(err)
	require.Equal(response.NoStakeReports[0].Value, report.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp1.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())

	// 2 reports
	timestamp2 := time.Now().UTC().Add(time.Second * 10)
	s.ctx = s.ctx.WithBlockHeight(2).WithBlockTime(timestamp2)
	report2 := types.NoStakeMicroReport{
		Reporter:    reporter,
		Timestamp:   timestamp2,
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
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp1.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())
	require.Equal(response.NoStakeReports[1].Value, report2.Value)
	require.Equal(response.NoStakeReports[1].Timestamp, uint64(timestamp2.UnixMilli()))
	require.Equal(response.NoStakeReports[1].BlockNumber, report2.BlockNumber)
	require.Equal(response.NoStakeReports[1].Reporter, sdk.AccAddress(report2.Reporter).String())

	// most recent by query Id (should be timestamp2)
	response, err = q.GetNoStakeReportsByQueryId(s.ctx, &types.QueryGetNoStakeReportsByQueryIdRequest{
		QueryId: hex.EncodeToString(queryId),
		Pagination: &query.PageRequest{
			Limit:   1,
			Reverse: true,
		},
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 1)
	require.Equal(response.NoStakeReports[0].Value, report2.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp2.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report2.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report2.Reporter).String())

	fmt.Println("nextKey: ", response.Pagination.NextKey)
	fmt.Println("nextKey string: ", hex.EncodeToString(response.Pagination.NextKey))
}

func (s *KeeperTestSuite) TestGetReportersNoStakeReportsMixQueryIds() {
	require := s.Require()
	spec := registrytypes.DataSpec{
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "tolayer", FieldType: "bool"},
			{Name: "depositId", FieldType: "uint256"},
		},
	}
	trbBridqeMixed := []string{
		`["true","1"]`,
		`["true","2"]`,
		`["true","3"]`,
		`["true","4"]`,
		`["true","5"]`,
		`["true", "6"]`,
		`["true", "7"]`,
		`["true", "8"]`,
		`["true", "9"]`,
	}
	reporter := sample.AccAddressBytes()
	var expectedReports []*types.NoStakeMicroReportStrings
	for i, v := range trbBridqeMixed {
		querydata, err := spec.EncodeData("TRBBridge", v)
		require.NoError(err)
		queryId := crypto.Keccak256(querydata)
		timestamp := time.Now().UTC().Add(time.Second * time.Duration(i))
		report := types.NoStakeMicroReport{
			Reporter:    reporter,
			Timestamp:   timestamp,
			BlockNumber: uint64(i + 1),
			Value:       fmt.Sprintf("value%d", i+1),
		}
		expectedReports = append(expectedReports, &types.NoStakeMicroReportStrings{
			Reporter:    reporter.String(),
			Timestamp:   uint64(timestamp.UnixMilli()),
			BlockNumber: uint64(i + 1),
			Value:       fmt.Sprintf("value%d", i+1),
		})
		s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(timestamp.UnixMilli())), report)
	}
	response, err := s.queryClient.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
	})
	require.NoError(err)
	require.Equal(len(expectedReports), len(response.NoStakeReports))

	sort.Slice(expectedReports, func(i, j int) bool {
		return expectedReports[i].Timestamp < expectedReports[j].Timestamp
	})
	sort.Slice(response.NoStakeReports, func(i, j int) bool {
		return response.NoStakeReports[i].Timestamp < response.NoStakeReports[j].Timestamp
	})

	for i := range expectedReports {
		require.Equal(expectedReports[i].BlockNumber, response.NoStakeReports[i].BlockNumber)
		require.Equal(expectedReports[i].Timestamp, response.NoStakeReports[i].Timestamp)
	}
}

func (s *KeeperTestSuite) TestGetReportersNoStakeReports_PaginationEdgeCases() {
	require := s.Require()

	reporter := sample.AccAddressBytes()

	// Create 3 reports initially (fewer than default limit of 10)
	timestamps := make([]time.Time, 3)
	for i := 0; i < 3; i++ {
		timestamps[i] = time.Now().UTC().Add(time.Second * time.Duration(i))
		report := types.NoStakeMicroReport{
			Reporter:    reporter,
			Timestamp:   timestamps[i],
			BlockNumber: uint64(i + 1),
			Value:       fmt.Sprintf("value%d", i+1),
		}
		queryId := []byte(fmt.Sprintf("QueryId%d", i+1))
		require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(timestamps[i].UnixMilli())), report))
	}

	// Test case 1: No pagination (should return all 3 reports since < default limit of 10)
	req := &types.QueryGetReportersNoStakeReportsRequest{Reporter: reporter.String()}
	response, err := s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(3, len(response.NoStakeReports), "Should return all 3 reports when no pagination is provided and count < default limit")
	require.Nil(response.Pagination.NextKey, "NextKey should be nil when all reports are returned")

	// Test case 2: Empty pagination (should return all 3 reports since < default limit of 10)
	req = &types.QueryGetReportersNoStakeReportsRequest{Reporter: reporter.String(), Pagination: &query.PageRequest{}}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(3, len(response.NoStakeReports), "Should return all 3 reports when empty pagination is provided and count < default limit")
	require.Nil(response.Pagination.NextKey, "NextKey should be nil when all reports are returned")

	// Test case 3: Reverse only (should return all 3 reports in reverse order)
	req = &types.QueryGetReportersNoStakeReportsRequest{Reporter: reporter.String(), Pagination: &query.PageRequest{Reverse: true}}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(3, len(response.NoStakeReports), "Should return all 3 reports when only reverse flag is provided")
	require.Equal(uint64(3), response.NoStakeReports[0].BlockNumber, "First report should have highest BlockNumber in reverse order")
	require.Equal(uint64(1), response.NoStakeReports[2].BlockNumber, "Last report should have lowest BlockNumber in reverse order")
	require.Nil(response.Pagination.NextKey, "NextKey should be nil when all reports are returned")

	// Test case 4: Limit 0 (should use default limit of 10, but return all 3 since < 10)
	req = &types.QueryGetReportersNoStakeReportsRequest{Reporter: reporter.String(), Pagination: &query.PageRequest{Limit: 0}}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(3, len(response.NoStakeReports), "Should return all 3 reports when limit is 0 and count < default limit")
	require.Nil(response.Pagination.NextKey, "NextKey should be nil when all reports are returned")

	// Test case 5: Limit 1 (should return 1 report with NextKey)
	req = &types.QueryGetReportersNoStakeReportsRequest{Reporter: reporter.String(), Pagination: &query.PageRequest{Limit: 1}}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(1, len(response.NoStakeReports), "Should return 1 report when limit is 1")
	require.NotNil(response.Pagination.NextKey, "NextKey should not be nil when there are more reports")

	// Test case 6: Limit 1 with reverse (should return most recent report)
	req = &types.QueryGetReportersNoStakeReportsRequest{Reporter: reporter.String(), Pagination: &query.PageRequest{Limit: 1, Reverse: true}}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(1, len(response.NoStakeReports), "Should return 1 report when limit is 1 with reverse")
	require.Equal(uint64(3), response.NoStakeReports[0].BlockNumber, "Should return most recent report (highest BlockNumber)")
	require.NotNil(response.Pagination.NextKey, "NextKey should not be nil when there are more reports")

	// Test case 7: Create more reports to test default limit behavior
	for i := 3; i < 14; i++ { // Add reports 4-14 (total will be 14 reports)
		timestamp := time.Now().UTC().Add(time.Second * time.Duration(i))
		report := types.NoStakeMicroReport{
			Reporter:    reporter,
			Timestamp:   timestamp,
			BlockNumber: uint64(i + 1),
			Value:       fmt.Sprintf("value%d", i+1),
		}
		queryId := []byte(fmt.Sprintf("QueryId%d", i+1))
		require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(timestamp.UnixMilli())), report))
	}

	// Test case 8: No pagination with many reports (should return default limit of 10)
	req = &types.QueryGetReportersNoStakeReportsRequest{Reporter: reporter.String()}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(10, len(response.NoStakeReports), "Should return default limit of 10 reports when no pagination is provided and count > default limit")
	require.NotNil(response.Pagination.NextKey, "NextKey should not be nil when there are more reports beyond the default limit")

	// Test case 9: Offset handling
	req = &types.QueryGetReportersNoStakeReportsRequest{Reporter: reporter.String(), Pagination: &query.PageRequest{Limit: 5, Offset: 2}}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(5, len(response.NoStakeReports), "Should return 5 reports when limit is 5 and offset is 2")

	// Test case 10: Large offset beyond available reports
	req = &types.QueryGetReportersNoStakeReportsRequest{Reporter: reporter.String(), Pagination: &query.PageRequest{Limit: 5, Offset: 20}}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(0, len(response.NoStakeReports), "Should return 0 reports when offset is beyond available reports")

	// Test case 11: Multiple reporters - ensure isolation
	reporter2 := sample.AccAddressBytes()
	timestamp := time.Now().UTC().Add(time.Hour)
	report2 := types.NoStakeMicroReport{
		Reporter:    reporter2,
		Timestamp:   timestamp,
		BlockNumber: 100,
		Value:       "reporter2_value",
	}
	queryId2 := []byte("Reporter2QueryId")
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId2, uint64(timestamp.UnixMilli())), report2))

	// Should only return reports for specific reporter
	req = &types.QueryGetReportersNoStakeReportsRequest{Reporter: reporter2.String()}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(1, len(response.NoStakeReports), "Should return only reports for the specified reporter")
	require.Equal("reporter2_value", response.NoStakeReports[0].Value)
	require.Equal(reporter2.String(), response.NoStakeReports[0].Reporter)

	// Original reporter should still have 14 reports
	req = &types.QueryGetReportersNoStakeReportsRequest{Reporter: reporter.String()}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(10, len(response.NoStakeReports), "Original reporter should still have default limit of reports returned")

	sort.Slice(response.NoStakeReports, func(i, j int) bool {
		return response.NoStakeReports[i].Timestamp < response.NoStakeReports[j].Timestamp
	})

	expectedReports := []types.NoStakeMicroReport{
		{
			Reporter:    reporter,
			Timestamp:   timestamps[0],
			BlockNumber: 1,
			Value:       "value1",
		},
		{
			Reporter:    reporter,
			Timestamp:   timestamps[1],
			BlockNumber: 2,
			Value:       "value2",
		},
		{
			Reporter:    reporter,
			Timestamp:   timestamps[2],
			BlockNumber: 3,
			Value:       "value3",
		},
	}

	// Sort both slices by timestamp before comparing
	sort.Slice(expectedReports, func(i, j int) bool {
		return expectedReports[i].Timestamp.Before(expectedReports[j].Timestamp)
	})
	sort.Slice(response.NoStakeReports, func(i, j int) bool {
		return response.NoStakeReports[i].Timestamp < response.NoStakeReports[j].Timestamp
	})

	for i := range expectedReports {
		require.Equal(expectedReports[i].BlockNumber, response.NoStakeReports[i].BlockNumber)
		require.Equal(uint64(expectedReports[i].Timestamp.UnixMilli()), response.NoStakeReports[i].Timestamp)
	}
}

func (s *KeeperTestSuite) TestGetReportersNoStakeReports_PaginationContinuation() {
	require := s.Require()
	// Setup: Create more reports than the default limit for a single reporter
	numReports := 25
	reporter := sample.AccAddressBytes()
	var expectedTimestamps []uint64
	for i := 0; i < numReports; i++ {
		timestamp := time.Now().UTC().Add(time.Second * time.Duration(i))
		expectedTimestamps = append(expectedTimestamps, uint64(timestamp.UnixMilli()))
		report := types.NoStakeMicroReport{
			Reporter:    reporter,
			Value:       fmt.Sprintf("value%d", i),
			Timestamp:   timestamp,
			BlockNumber: uint64(i),
		}
		s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join([]byte("queryId"), uint64(timestamp.UnixMilli())), report)
	}

	// First page
	limit := uint64(10)
	req := &types.QueryGetReportersNoStakeReportsRequest{
		Reporter:   reporter.String(),
		Pagination: &query.PageRequest{Limit: limit},
	}
	response, err := s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(int(limit), len(response.NoStakeReports), "First page should return 10 reports")
	require.NotNil(response.Pagination.NextKey, "NextKey should not be nil for the first page")

	// Collect timestamps from the first page
	firstPageTimestamps := make(map[uint64]bool)
	for _, report := range response.NoStakeReports {
		firstPageTimestamps[report.Timestamp] = true
	}

	// Second page
	req = &types.QueryGetReportersNoStakeReportsRequest{
		Reporter:   reporter.String(),
		Pagination: &query.PageRequest{Limit: limit, Offset: limit},
	}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(int(limit), len(response.NoStakeReports), "Second page should return 10 reports")
	require.NotNil(response.Pagination.NextKey, "NextKey should not be nil for the second page")

	// Check for no overlap
	for _, report := range response.NoStakeReports {
		_, found := firstPageTimestamps[report.Timestamp]
		require.False(found, "Second page should not contain values from first page")
	}

	// Third and final page
	req = &types.QueryGetReportersNoStakeReportsRequest{
		Reporter:   reporter.String(),
		Pagination: &query.PageRequest{Limit: limit, Offset: limit * 2},
	}
	response, err = s.queryClient.GetReportersNoStakeReports(s.ctx, req)
	require.NoError(err)
	require.Equal(numReports-int(limit*2), len(response.NoStakeReports), "Third page should return the remaining 5 reports")
	require.Nil(response.Pagination.NextKey, "NextKey should be nil for the final page")
}
