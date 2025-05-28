package keeper_test

import (
	"encoding/hex"
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
	k := s.oracleKeeper
	rk := s.reporterKeeper
	ctx := s.ctx.WithBlockHeight(18).WithBlockTime(time.Now()).WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	addr := sample.AccAddressBytes()
	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)
	queryId := utils.QueryIDFromData(qDataBz)

	err = k.QueryDataLimit.Set(ctx, types.QueryDataLimit{Limit: types.InitialQueryDataLimit()})
	require.NoError(err)
	params, err := k.Params.Get(ctx)
	require.NoError(err)
	minStakeAmt := params.MinStakeAmount

	for i := 1; i < 6; i++ {
		query := types.QueryMeta{
			Id:                      uint64(i),
			Amount:                  math.NewInt(100_000),
			Expiration:              20,
			RegistrySpecBlockWindow: 10,
			HasRevealedReports:      false,
			QueryData:               qDataBz,
			QueryType:               "SpotPrice",
			CycleList:               true,
		}
		err = k.Query.Set(ctx, collections.Join(queryId, query.Id), query)
		require.NoError(err)

		rk.On("ReporterStake", mock.Anything, addr, queryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()
		_ = s.registryKeeper.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil).Once()
		submitreq := types.MsgSubmitValue{
			Creator:   addr.String(),
			QueryData: qDataBz,
			Value:     value,
		}
		res, err := s.msgServer.SubmitValue(ctx, &submitreq)
		require.NoError(err)
		require.NotNil(res)
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
}
