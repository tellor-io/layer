package keeper_test

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/types/query"
)

func (s *KeeperTestSuite) TestGetReportsByQueryId() {
	addr, queryIdStr := s.TestSubmitValue()

	req := &types.QueryGetReportsbyQidRequest{QueryId: hex.EncodeToString(queryIdStr)}

	report, err := s.queryClient.GetReportsbyQid(s.ctx, req)
	s.Nil(err)

	MicroReport := types.MicroReportStrings{
		Reporter:        addr.String(),
		Power:           1,
		QueryType:       "SpotPrice",
		QueryId:         hex.EncodeToString(queryIdStr),
		AggregateMethod: "weighted-median",
		Value:           value,
		Timestamp:       uint64(s.ctx.BlockTime().UnixMilli()),
		Cyclelist:       true,
		BlockNumber:     uint64(s.ctx.BlockHeight()),
		MetaId:          1,
	}
	expectedReports := []types.MicroReportStrings{MicroReport}

	s.Equal(expectedReports, report.MicroReports)

	report2, err := s.queryClient.GetReportsbyReporter(s.ctx, &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{Limit: 1}})
	s.NoError(err)
	s.Equal(expectedReports[0], report2.MicroReports[0])

	report3, err := s.queryClient.GetReportsbyReporterQid(s.ctx, &types.QueryGetReportsbyReporterQidRequest{Reporter: addr.String(), QueryId: hex.EncodeToString(queryIdStr)})
	s.NoError(err)
	s.EqualValues(expectedReports, report3.MicroReports)

	report, err = s.queryClient.GetReportsbyQid(s.ctx, &types.QueryGetReportsbyQidRequest{QueryId: hex.EncodeToString(queryIdStr)})
	s.NoError(err)
	s.Equal(expectedReports, report.MicroReports)
}

func (s *KeeperTestSuite) TestGetReportsByReporterPaginate() {
	require := s.Require()
	s.oracleKeeper.SetBridgeKeeper(s.bridgeKeeper)
	require.NotNil(s.bridgeKeeper)

	ctx := s.ctx.WithBlockHeight(18).WithBlockTime(time.Now()).WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	addr := sample.AccAddressBytes()
	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)
	queryId := utils.QueryIDFromData(qDataBz)

	err = s.oracleKeeper.QueryDataLimit.Set(ctx, types.QueryDataLimit{Limit: types.InitialQueryDataLimit()})
	require.NoError(err)
	params, err := s.oracleKeeper.Params.Get(ctx)
	require.NoError(err)
	minStakeAmt := params.MinStakeAmount

	queryData := qDataBz
	// reporter1 reports metaIds 1-5
	for i := 1; i < 6; i++ {
		fmt.Println("i: ", i)
		fmt.Printf("Using reporter address: %s (%x)\n", addr.String(), addr.Bytes())
		queryType := "SpotPrice"
		_ = s.registryKeeper.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil).Once()

		queryMeta := types.QueryMeta{
			Id:                      uint64(i),
			Amount:                  math.NewInt(100_000),
			Expiration:              20,
			RegistrySpecBlockWindow: 10,
			HasRevealedReports:      false,
			QueryData:               queryData,
			QueryType:               queryType,
			CycleList:               true,
		}
		err = s.oracleKeeper.Query.Set(ctx, collections.Join(queryId, queryMeta.Id), queryMeta)
		require.NoError(err)

		s.reporterKeeper.On("ReporterStake", mock.Anything, addr, queryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()

		submitreq := types.MsgSubmitValue{
			Creator:   addr.String(),
			QueryData: queryData,
			Value:     value,
		}
		fmt.Println("submitting spot price value... ")
		res, err := s.msgServer.SubmitValue(ctx, &submitreq)
		require.NoError(err)
		require.NotNil(res)

		storedReport, err := s.oracleKeeper.Reports.Get(ctx, collections.Join3(queryId, addr.Bytes(), uint64(i)))
		if err != nil {
			fmt.Printf("ERROR: Report %d not found: %v\n", i, err)
		} else {
			fmt.Printf("SUCCESS: Report %d stored with reporter %s\n", i, storedReport.Reporter)
		}
	}

	// 5 in store, search for 10
	req := &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{Limit: 10}}
	report, err := s.queryClient.GetReportsbyReporter(ctx, req)
	s.NoError(err)
	s.Equal(5, len(report.MicroReports))

	// 5 in store, search for 2
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{Limit: 2}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	s.NoError(err)
	s.Equal(2, len(report.MicroReports))

	// 5 in store, search for 5
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{Limit: 5}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	s.NoError(err)
	s.Equal(5, len(report.MicroReports))
	for i, report := range report.MicroReports {
		fmt.Println("report i: ", i, report)
	}

	// reverse, get most recent
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{Limit: 1, Reverse: true}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	s.NoError(err)
	s.Equal(1, len(report.MicroReports))
	require.Equal(report.MicroReports[0].MetaId, uint64(5))

	// reverse by queryId
	req2 := &types.QueryGetReportsbyReporterQidRequest{Reporter: addr.String(), QueryId: hex.EncodeToString(queryId), Pagination: &query.PageRequest{Limit: 1, Reverse: true}}
	report, err = s.queryClient.GetReportsbyReporterQid(ctx, req2)
	s.NoError(err)
	s.Equal(1, len(report.MicroReports))
	require.Equal(report.MicroReports[0].MetaId, uint64(5))

	// add reports from another reporter
	// reporter2 reports metaId 6
	addr2 := sample.AccAddressBytes()
	queryMeta := types.QueryMeta{
		Id:                      uint64(6),
		Amount:                  math.NewInt(100_000),
		Expiration:              20,
		RegistrySpecBlockWindow: 10,
		HasRevealedReports:      false,
		QueryData:               queryData,
		QueryType:               queryType,
		CycleList:               true,
	}
	err = s.oracleKeeper.Query.Set(ctx, collections.Join(queryId, queryMeta.Id), queryMeta)
	require.NoError(err)

	s.reporterKeeper.On("ReporterStake", mock.Anything, addr2, queryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()
	_ = s.registryKeeper.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil).Once()

	submitreq := types.MsgSubmitValue{
		Creator:   addr2.String(),
		QueryData: queryData,
		Value:     value,
	}
	fmt.Println("submitting value... ")
	res, err := s.msgServer.SubmitValue(ctx, &submitreq)
	require.NoError(err)
	require.NotNil(res)

	storedReport, err := s.oracleKeeper.Reports.Get(ctx, collections.Join3(queryId, addr2.Bytes(), uint64(6)))
	if err != nil {
		fmt.Printf("ERROR: Report %d not found: %v\n", 6, err)
	} else {
		fmt.Printf("SUCCESS: Report %d stored with reporter %s\n", 6, storedReport.Reporter)
	}

	// get reports by reporter for 2nd reporter
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr2.String(), Pagination: &query.PageRequest{Limit: 1}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	s.NoError(err)
	s.Equal(1, len(report.MicroReports))
	fmt.Printf("report %d from reporter: %v, report: %v\n", report.MicroReports[0].MetaId, report.MicroReports[0].Reporter, report.MicroReports[0])
	require.Equal(report.MicroReports[0].Reporter, addr2.String())
	require.Equal(report.MicroReports[0].MetaId, uint64(6))

	// get reports by reporter for 1st reporter again
	req2 = &types.QueryGetReportsbyReporterQidRequest{Reporter: addr.String(), QueryId: hex.EncodeToString(queryId), Pagination: &query.PageRequest{Limit: 1, Reverse: true}}
	report, err = s.queryClient.GetReportsbyReporterQid(ctx, req2)
	s.NoError(err)
	s.Equal(1, len(report.MicroReports))
	require.Equal(report.MicroReports[0].Reporter, addr.String())
	require.Equal(report.MicroReports[0].MetaId, uint64(5))

	// add bridge report from each reporter
	_ = s.registryKeeper.On("GetSpec", ctx, "TRBBridge").Return(bridgeSpec, nil).Once()
	_ = s.bridgeKeeper.On("GetDepositStatus", ctx, uint64(8)).Return(false, nil).Maybe()

	// create proper TRBBridge query data - this should be a deposit (true) with depositId 8
	queryDataStr := "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000095452424272696467650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000008"
	queryData, err = hex.DecodeString(queryDataStr)
	require.NoError(err)
	bridgeQueryId := utils.QueryIDFromData(queryData)
	queryMeta = types.QueryMeta{
		Id:                      uint64(8),
		Amount:                  math.NewInt(100_000),
		Expiration:              20,
		RegistrySpecBlockWindow: 10,
		HasRevealedReports:      false,
		QueryData:               queryData,
		QueryType:               queryType,
		CycleList:               true,
	}
	err = s.oracleKeeper.Query.Set(ctx, collections.Join(bridgeQueryId, queryMeta.Id), queryMeta)
	require.NoError(err)

	s.reporterKeeper.On("ReporterStake", mock.Anything, addr, bridgeQueryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()

	submitreq = types.MsgSubmitValue{
		Creator:   addr.String(),
		QueryData: queryData,
		Value:     value,
	}
	fmt.Println("submitting bridge value... ")
	res, err = s.msgServer.SubmitValue(ctx, &submitreq)
	require.NoError(err)
	require.NotNil(res)

	// other reporter reports same query
	_ = s.registryKeeper.On("GetSpec", ctx, "TRBBridge").Return(bridgeSpec, nil).Once()
	queryMeta = types.QueryMeta{
		Id:                      uint64(9),
		Amount:                  math.NewInt(100_000),
		Expiration:              20,
		RegistrySpecBlockWindow: 10,
		HasRevealedReports:      false,
		QueryData:               queryData,
		QueryType:               queryType,
		CycleList:               true,
	}
	err = s.oracleKeeper.Query.Set(ctx, collections.Join(bridgeQueryId, queryMeta.Id), queryMeta)
	require.NoError(err)

	s.reporterKeeper.On("ReporterStake", mock.Anything, addr2, bridgeQueryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()

	submitreq = types.MsgSubmitValue{
		Creator:   addr2.String(),
		QueryData: queryData,
		Value:     value,
	}
	fmt.Println("submitting bridge value... ")
	res, err = s.msgServer.SubmitValue(ctx, &submitreq)
	require.NoError(err)
	require.NotNil(res)

	// get most recent reports by reporter for both reporters
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{Limit: 1, Reverse: true}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	s.NoError(err)
	s.Equal(1, len(report.MicroReports))
	require.Equal(report.MicroReports[0].MetaId, uint64(8))
	require.Equal(report.MicroReports[0].Reporter, addr.String())

	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr2.String(), Pagination: &query.PageRequest{Limit: 1, Reverse: true}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	s.NoError(err)
	s.Equal(1, len(report.MicroReports))
	require.Equal(report.MicroReports[0].MetaId, uint64(9))
	require.Equal(report.MicroReports[0].Reporter, addr2.String())

	// get 2 most recent reports by reporter1
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{Limit: 2, Reverse: true}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	s.NoError(err)
	s.Equal(2, len(report.MicroReports))
	require.Equal(report.MicroReports[0].MetaId, uint64(8))
	require.Equal(report.MicroReports[1].MetaId, uint64(5))
	// get next key, should be metaId 4
	nextKey := report.Pagination.NextKey
	nextKeyReportIter, err := s.oracleKeeper.Reports.Indexes.Reporter.MatchExact(ctx, nextKey)
	fmt.Println("err: ", err)
	require.NoError(err)
	pk, err := nextKeyReportIter.PrimaryKey()
	require.NoError(err)
	nextKeyReport, err := s.oracleKeeper.Reports.Get(ctx, pk)
	require.NoError(err)
	fmt.Println("err: ", err)
	fmt.Println("report: ", nextKeyReport)
	require.Equal(nextKeyReport.MetaId, uint64(4))
}

func (s *KeeperTestSuite) TestGetReportsByReporter_PaginationEdgeCases() {
	require := s.Require()
	s.oracleKeeper.SetBridgeKeeper(s.bridgeKeeper)
	require.NotNil(s.bridgeKeeper)

	ctx := s.ctx.WithBlockHeight(18).WithBlockTime(time.Now()).WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	addr := sample.AccAddressBytes()
	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)
	queryId := utils.QueryIDFromData(qDataBz)

	err = s.oracleKeeper.QueryDataLimit.Set(ctx, types.QueryDataLimit{Limit: types.InitialQueryDataLimit()})
	require.NoError(err)
	params, err := s.oracleKeeper.Params.Get(ctx)
	require.NoError(err)
	minStakeAmt := params.MinStakeAmount

	queryData := qDataBz
	// reporter reports metaIds 1-3 (fewer than default limit of 10)
	for i := 1; i < 4; i++ {
		_ = s.registryKeeper.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil).Once()

		queryMeta := types.QueryMeta{
			Id:                      uint64(i),
			Amount:                  math.NewInt(100_000),
			Expiration:              20,
			RegistrySpecBlockWindow: 10,
			HasRevealedReports:      false,
			QueryData:               queryData,
			QueryType:               queryType,
			CycleList:               true,
		}
		err = s.oracleKeeper.Query.Set(ctx, collections.Join(queryId, queryMeta.Id), queryMeta)
		require.NoError(err)

		s.reporterKeeper.On("ReporterStake", mock.Anything, addr, queryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()

		submitreq := types.MsgSubmitValue{
			Creator:   addr.String(),
			QueryData: queryData,
			Value:     value,
		}
		res, err := s.msgServer.SubmitValue(ctx, &submitreq)
		require.NoError(err)
		require.NotNil(res)
	}

	// Test case 1: No pagination (should return all 3 reports since < default limit of 10)
	req := &types.QueryGetReportsbyReporterRequest{Reporter: addr.String()}
	report, err := s.queryClient.GetReportsbyReporter(ctx, req)
	require.NoError(err)
	require.Equal(3, len(report.MicroReports), "Should return all 3 reports when no pagination is provided and count < default limit")
	require.Nil(report.Pagination.NextKey, "NextKey should be nil when all reports are returned")

	// Test case 2: Empty pagination (should return all 3 reports since < default limit of 10)
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	require.NoError(err)
	require.Equal(3, len(report.MicroReports), "Should return all 3 reports when empty pagination is provided and count < default limit")
	require.Nil(report.Pagination.NextKey, "NextKey should be nil when all reports are returned")

	// Test case 3: Reverse only (should return all 3 reports in reverse order)
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{Reverse: true}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	require.NoError(err)
	require.Equal(3, len(report.MicroReports), "Should return all 3 reports when only reverse flag is provided")
	require.Equal(uint64(3), report.MicroReports[0].MetaId, "First report should have highest MetaId in reverse order")
	require.Equal(uint64(1), report.MicroReports[2].MetaId, "Last report should have lowest MetaId in reverse order")
	require.Nil(report.Pagination.NextKey, "NextKey should be nil when all reports are returned")

	// Test case 4: Limit 0 (should use default limit of 10, but return all 3 since < 10)
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{Limit: 0}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	require.NoError(err)
	require.Equal(3, len(report.MicroReports), "Should return all 3 reports when limit is 0 and count < default limit")
	require.Nil(report.Pagination.NextKey, "NextKey should be nil when all reports are returned")

	// Test case 5: Limit 1 (should return 1 report with NextKey)
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{Limit: 1}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	require.NoError(err)
	require.Equal(1, len(report.MicroReports), "Should return 1 report when limit is 1")
	require.NotNil(report.Pagination.NextKey, "NextKey should not be nil when there are more reports beyond the default limit")

	// Test case 6: Limit 1 with reverse (should return most recent report)
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr.String(), Pagination: &query.PageRequest{Limit: 1, Reverse: true}}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	require.NoError(err)
	require.Equal(1, len(report.MicroReports), "Should return 1 report when limit is 1 with reverse")
	require.Equal(uint64(3), report.MicroReports[0].MetaId, "Should return most recent report (highest MetaId)")
	require.NotNil(report.Pagination.NextKey, "NextKey should not be nil when there are more reports beyond the default limit")

	// Test case 7: Create more reports to test default limit behavior
	for i := 4; i < 15; i++ { // Add reports 4-14 (total will be 14 reports)
		queryType := "SpotPrice"
		_ = s.registryKeeper.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil).Once()

		queryMeta := types.QueryMeta{
			Id:                      uint64(i),
			Amount:                  math.NewInt(100_000),
			Expiration:              20,
			RegistrySpecBlockWindow: 10,
			HasRevealedReports:      false,
			QueryData:               queryData,
			QueryType:               queryType,
			CycleList:               true,
		}
		err = s.oracleKeeper.Query.Set(ctx, collections.Join(queryId, queryMeta.Id), queryMeta)
		require.NoError(err)

		s.reporterKeeper.On("ReporterStake", mock.Anything, addr, queryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()

		submitreq := types.MsgSubmitValue{
			Creator:   addr.String(),
			QueryData: queryData,
			Value:     value,
		}
		res, err := s.msgServer.SubmitValue(ctx, &submitreq)
		require.NoError(err)
		require.NotNil(res)
	}

	// Test case 8: No pagination with many reports (should return default limit of 10)
	req = &types.QueryGetReportsbyReporterRequest{Reporter: addr.String()}
	report, err = s.queryClient.GetReportsbyReporter(ctx, req)
	require.NoError(err)
	require.Equal(10, len(report.MicroReports), "Should return default limit of 10 reports when no pagination is provided and count > default limit")
	require.NotNil(report.Pagination.NextKey, "NextKey should not be nil when there are more reports beyond the default limit")
}
