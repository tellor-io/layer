package keeper_test

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

func TestNewQuerier(t *testing.T) {
	k, _, _, _, _, _, _ := testkeeper.OracleKeeper(t)
	q := keeper.NewQuerier(k)
	require.NotNil(t, q)
}

func (s *KeeperTestSuite) TestQueryGetAggregatedReport() {
	require := s.Require()
	k := s.oracleKeeper
	q := s.queryClient

	// nil request
	res, err := q.GetCurrentAggregateReport(s.ctx, nil)
	require.ErrorContains(err, "invalid request")
	require.Nil(res)

	// bad query id
	req := types.QueryGetCurrentAggregateReportRequest{
		QueryId: "badqueryid",
	}
	res, err = q.GetCurrentAggregateReport(s.ctx, &req)
	require.Error(err)
	require.Nil(res)

	// good req, no reports available
	qId, err := utils.QueryIDFromDataString(queryData)
	require.NoError(err)
	req.QueryId = hex.EncodeToString(qId)
	res, err = q.GetCurrentAggregateReport(s.ctx, &req)
	require.ErrorContains(err, "aggregate not found")
	require.Nil(res)

	// set Aggregates collection
	require.NoError(k.Aggregates.Set(s.ctx, collections.Join(qId, uint64(0)), types.Aggregate{
		QueryId:           qId,
		AggregateValue:    "100",
		AggregateReporter: sdk.AccAddress("reporter").String(),
		AggregatePower:    100,
		Flagged:           false,
		Index:             10,
		Height:            11,
		MicroHeight:       12,
		MetaId:            13,
	}))
	res, err = q.GetCurrentAggregateReport(s.ctx, &req)
	require.NoError(err)
	require.NotNil(res)
	require.Equal(res.Aggregate.QueryId, hex.EncodeToString(qId))
	require.Equal(res.Aggregate.AggregateValue, "100")
	require.Equal(res.Aggregate.AggregateReporter, sdk.AccAddress("reporter").String())
	require.Equal(res.Aggregate.AggregatePower, uint64(100))
	require.Equal(res.Aggregate.Flagged, false)
	require.Equal(res.Aggregate.Index, uint64(10))
	require.Equal(res.Aggregate.Height, uint64(11))
	require.Equal(res.Aggregate.MicroHeight, uint64(12))
	require.Equal(res.Aggregate.MetaId, uint64(13))
}

func TestGetCurrentAggregateReport(t *testing.T) {
	k, _, _, _, _, _, ctx := testkeeper.OracleKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getCurrentAggResponse, err := keeper.NewQuerier(k).GetCurrentAggregateReport(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getCurrentAggResponse)

	getCurrentAggResponse, err = keeper.NewQuerier(k).GetCurrentAggregateReport(ctx, &types.QueryGetCurrentAggregateReportRequest{
		QueryId: "z",
	})
	require.ErrorContains(t, err, "invalid query id")
	require.Nil(t, getCurrentAggResponse)

	agg := (*types.Aggregate)(nil)
	timestamp := time.Unix(int64(1), 0)
	queryId := "1234abcd"
	qIdBz, err := utils.QueryBytesFromString(queryId)
	require.NoError(t, err)
	// ok.On("GetCurrentAggregateReport", ctx, qIdBz).Return(agg, timestamp).Once()

	getCurrentAggResponse, err = keeper.NewQuerier(k).GetCurrentAggregateReport(ctx, &types.QueryGetCurrentAggregateReportRequest{
		QueryId: queryId,
	})
	require.ErrorContains(t, err, "aggregate not found")
	require.Nil(t, getCurrentAggResponse)

	agg = &types.Aggregate{
		QueryId:           qIdBz,
		AggregateValue:    "10_000",
		AggregateReporter: sdk.AccAddress("reporter1").String(),
		AggregatePower:    100,
		Flagged:           false,
		Index:             uint64(0),
		Height:            0,
		MicroHeight:       0,
	}

	require.NoError(t, k.Aggregates.Set(ctx, collections.Join(qIdBz, uint64(timestamp.UnixMilli())), *agg))
	getCurrentAggResponse, err = keeper.NewQuerier(k).GetCurrentAggregateReport(ctx, &types.QueryGetCurrentAggregateReportRequest{
		QueryId: queryId,
	})
	require.NoError(t, err)
	require.Equal(t, getCurrentAggResponse.Timestamp, uint64(timestamp.UnixMilli()))
	require.Equal(t, getCurrentAggResponse.Aggregate.QueryId, hex.EncodeToString(agg.QueryId))
	require.Equal(t, getCurrentAggResponse.Aggregate.AggregateValue, agg.AggregateValue)
	require.Equal(t, getCurrentAggResponse.Aggregate.AggregateReporter, agg.AggregateReporter)
	require.Equal(t, getCurrentAggResponse.Aggregate.AggregatePower, agg.AggregatePower)
	require.Equal(t, getCurrentAggResponse.Aggregate.Flagged, agg.Flagged)
	require.Equal(t, getCurrentAggResponse.Aggregate.Index, agg.Index)
	require.Equal(t, getCurrentAggResponse.Aggregate.Height, agg.Height)
}

func TestRetreiveData(t *testing.T) {
	k, _, _, _, _, _, ctx := testkeeper.OracleKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	q := keeper.NewQuerier(k)

	type testCase struct {
		name          string
		queryId       string
		setupFunc     func()
		expectedError string
	}

	testCases := []testCase{
		{
			name:          "invalid queryId",
			queryId:       "z",
			expectedError: "invalid queryId",
		},
		{
			name:          "collections error",
			queryId:       "1234abcd",
			expectedError: "collections: not found",
		},
		{
			name:    "success",
			queryId: "1234abcd",
			setupFunc: func() {
				qId, err := utils.QueryBytesFromString("1234abcd")
				require.NoError(t, err)
				require.NoError(t, k.Aggregates.Set(ctx, collections.Join(qId, uint64(0)), types.Aggregate{
					QueryId:           qId,
					AggregateValue:    "100",
					AggregateReporter: sdk.AccAddress("reporter").String(),
					AggregatePower:    100,
				}))
			},
			expectedError: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFunc != nil {
				tc.setupFunc()
			}

			getDataResponse, err := q.RetrieveData(ctx, &types.QueryRetrieveDataRequest{
				QueryId: tc.queryId,
			})

			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
				require.Nil(t, getDataResponse)
			} else {
				require.NoError(t, err)
				require.NotNil(t, getDataResponse)

			}
		})
	}
}

func (s *KeeperTestSuite) TestQuery_GetAggregateBeforeByReporter() {
	require := s.Require()
	// k := s.oracleKeeper
	q := s.queryClient
	ctx := s.ctx

	reportedAt := time.Now().Add(time.Second * -10)
	_, qId, reporter, _, err := s.CreateReportAndReportersAtTimestamp(reportedAt)
	s.NoError(err)

	testCases := []struct {
		name          string
		request       *types.QueryGetAggregateBeforeByReporterRequest
		expectedError string
		setup         func()
	}{
		{
			name:          "nil request",
			request:       nil,
			expectedError: "invalid request",
		},
		{
			name: "invalid queryId",
			request: &types.QueryGetAggregateBeforeByReporterRequest{
				QueryId: "z",
			},
			expectedError: "invalid query id",
		},
		{
			name: "success",
			request: &types.QueryGetAggregateBeforeByReporterRequest{
				QueryId:   hex.EncodeToString(qId),
				Reporter:  reporter.String(),
				Timestamp: uint64(reportedAt.UnixMilli() + 10),
			},
			expectedError: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			res, err := q.GetAggregateBeforeByReporter(ctx, tc.request)

			if tc.expectedError != "" {
				require.Error(err)
				require.Contains(err.Error(), tc.expectedError)
				require.Nil(res)
			} else {
				require.NoError(err)
				require.NotNil(res)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetQuery() {
	require := s.Require()
	k := s.oracleKeeper
	q := s.queryClient
	ctx := s.ctx

	tests := []struct {
		name    string
		req     *types.QueryGetQueryRequest
		setup   func()
		wantErr bool
	}{
		{
			name: "Valid query",
			req: &types.QueryGetQueryRequest{
				QueryId: "0x1234",
				Id:      1,
			},
			setup: func() {
				queryData, _ := utils.QueryBytesFromString("0x1234")
				require.NoError(k.Query.Set(ctx, collections.Join(queryData, uint64(1)), types.QueryMeta{
					QueryData: queryData,
					Id:        1,
				}))
			},
			wantErr: false,
		},
		{
			name: "Invalid queryId",
			req: &types.QueryGetQueryRequest{
				QueryId: "invalid",
				Id:      1,
			},
			setup:   func() {},
			wantErr: true,
		},
		{
			name: "Query not found",
			req: &types.QueryGetQueryRequest{
				QueryId: "0x5678",
				Id:      1,
			},
			setup:   func() {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setup()
			resp, err := q.GetQuery(ctx, tt.req)
			if tt.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)
			require.NotNil(resp)
			require.NotNil(resp.Query)
		})
	}
}

func (s *KeeperTestSuite) TestGetTippedQueries() {
	require := s.Require()
	k := s.oracleKeeper
	q := s.queryClient
	ctx := s.ctx

	queryMeta := ReturnTestQueryMeta(math.NewInt(100))

	cleanup := func() {
		iter, err := k.Query.Iterate(ctx, nil)
		require.NoError(err)
		defer iter.Close()
	}

	tests := []struct {
		name               string
		req                *types.QueryGetTippedQueriesRequest
		setup              func()
		err                bool
		expectedActiveLen  int
		expectedExpiredLen int
	}{
		{
			name: "nil request",
			req:  nil,
			err:  true,
		},
		{
			name: "empty request",
			req:  &types.QueryGetTippedQueriesRequest{},
			err:  false,
		},
		{
			name: "success one tipped query",
			setup: func() {
				ctx := s.ctx.WithBlockHeight(1)
				queryMeta.Expiration = 2
				require.NoError(k.Query.Set(ctx, collections.Join(queryMeta.QueryData, uint64(1)), queryMeta))
			},
			req: &types.QueryGetTippedQueriesRequest{
				Pagination: &query.PageRequest{
					Offset: 0,
				},
			},
			err:                false,
			expectedActiveLen:  1,
			expectedExpiredLen: 0,
		},
		{
			name: "success one active and one expiredtipped query",
			setup: func() {
				ctx = s.ctx.WithBlockHeight(5)
				queryMeta2 := ReturnTestQueryMeta(math.NewInt(100))
				queryMeta.Expiration = 2
				queryMeta.Amount = math.NewInt(100)
				queryMeta2.Expiration = 8
				queryMeta2.Amount = math.NewInt(100)
				require.NoError(k.Query.Set(ctx, collections.Join(queryMeta.QueryData, uint64(1)), queryMeta))
				require.NoError(k.Query.Set(ctx, collections.Join(queryMeta2.QueryData, uint64(2)), queryMeta2))
			},
			req: &types.QueryGetTippedQueriesRequest{
				Pagination: &query.PageRequest{
					Offset: 0,
				},
			},
			err:                false,
			expectedActiveLen:  1,
			expectedExpiredLen: 1,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cleanup()
			if tt.setup != nil {
				tt.setup()
			}
			resp, err := q.GetTippedQueries(ctx, tt.req)
			if tt.err {
				require.Error(err)
				return
			}
			require.NoError(err)
			require.NotNil(resp)
			require.Equal(tt.expectedActiveLen, len(resp.ActiveQueries))
			require.Equal(tt.expectedExpiredLen, len(resp.ExpiredQueries))
		})
	}
}

func (s *KeeperTestSuite) TestTippedQueriesForDaemon() {
	require := s.Require()
	k := s.oracleKeeper
	q := s.queryClient
	ctx := s.ctx

	queryMeta := ReturnTestQueryMeta(math.NewInt(100))

	cleanup := func() {
		iter, err := k.Query.Iterate(ctx, nil)
		require.NoError(err)
		defer iter.Close()

		for ; iter.Valid(); iter.Next() {
			key, err := iter.Key()
			require.NoError(err)
			require.NoError(k.Query.Remove(ctx, key))
		}
	}

	tests := []struct {
		name        string
		req         *types.QueryTippedQueriesForDaemonRequest
		setup       func()
		err         bool
		expectedLen int
	}{
		{
			name: "nil request",
			req:  nil,
			err:  true,
		},
		{
			name:        "empty request",
			req:         &types.QueryTippedQueriesForDaemonRequest{},
			err:         false,
			expectedLen: 0,
		},
		{
			name: "success one tipped query",
			setup: func() {
				require.NoError(k.Query.Set(ctx, collections.Join(queryMeta.QueryData, uint64(1)), queryMeta))
			},
			req: &types.QueryTippedQueriesForDaemonRequest{
				Pagination: &query.PageRequest{
					Offset: 0,
				},
			},
			err:         false,
			expectedLen: 1,
		},
		{
			name: "success multiple tips same query",
			setup: func() {
				require.NoError(k.Query.Set(ctx, collections.Join(queryMeta.QueryData, uint64(1)), queryMeta))
				require.NoError(k.Query.Set(ctx, collections.Join(queryMeta.QueryData, uint64(2)), queryMeta))
				require.NoError(k.Query.Set(ctx, collections.Join(queryMeta.QueryData, uint64(3)), queryMeta))
			},
			req: &types.QueryTippedQueriesForDaemonRequest{
				Pagination: &query.PageRequest{
					Offset: 0,
				},
			},
			err:         false,
			expectedLen: 3,
		},
		{
			name: "success multiple tips different query",
			setup: func() {
				require.NoError(k.Query.Set(ctx, collections.Join(queryMeta.QueryData, uint64(1)), queryMeta))
				require.NoError(k.Query.Set(ctx, collections.Join(queryMeta.QueryData, uint64(2)), queryMeta))
				secondQueryMeta := types.QueryMeta{
					Id:                      1,
					Amount:                  math.NewInt(100),
					Expiration:              10,
					RegistrySpecBlockWindow: 3,
					HasRevealedReports:      false,
					QueryData:               []byte("0x4c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067dec0"),
					QueryType:               "SpotPrice",
				}
				require.NoError(k.Query.Set(ctx, collections.Join(secondQueryMeta.QueryData, uint64(3)), secondQueryMeta))
				require.NoError(k.Query.Set(ctx, collections.Join(secondQueryMeta.QueryData, uint64(4)), secondQueryMeta))
			},
			req: &types.QueryTippedQueriesForDaemonRequest{
				Pagination: &query.PageRequest{
					Offset: 0,
				},
			},
			err:         false,
			expectedLen: 4,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cleanup()
			if tt.setup != nil {
				tt.setup()
			}
			resp, err := q.TippedQueriesForDaemon(ctx, tt.req)
			if tt.err {
				require.Error(err)
				return
			} else {
				require.NoError(err)
				require.NotNil(resp)
				require.Equal(tt.expectedLen, len(resp.Queries))
			}
		})
	}
}

func (s *KeeperTestSuite) TestReportedIdsByReporter() {
	k := s.oracleKeeper
	q := s.queryClient
	reporter := sample.AccAddressBytes()
	// fetching in descending order
	pageReq := &query.PageRequest{Reverse: true}
	queryReq := types.QueryReportedIdsByReporterRequest{ReporterAddress: reporter.String(), Pagination: pageReq}
	for i := 1; i < 11; i++ {
		s.NoError(k.Reports.Set(s.ctx,
			collections.Join3([]byte("queryid1"), reporter.Bytes(), uint64(i)),
			types.MicroReport{}))
	}
	res, err := q.ReportedIdsByReporter(s.ctx, &queryReq)
	s.NoError(err)
	s.Equal(res.Ids, []uint64{10, 9, 8, 7, 6, 5, 4, 3, 2, 1})

	pageReq.Reverse = false
	res, err = q.ReportedIdsByReporter(s.ctx, &queryReq)
	s.NoError(err)
	s.Equal(res.Ids, []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	pageReq.Limit = 1
	res, err = q.ReportedIdsByReporter(s.ctx, &queryReq)
	s.NoError(err)
	s.Equal(res.Ids, []uint64{1})

	pageReq.Limit = 1
	pageReq.Reverse = true
	res, err = q.ReportedIdsByReporter(s.ctx, &queryReq)
	s.NoError(err)
	s.Equal(res.Ids, []uint64{10})

	pageReq.Limit = 5
	pageReq.Offset = 1
	pageReq.Reverse = true
	res, err = q.ReportedIdsByReporter(s.ctx, &queryReq)
	s.NoError(err)
	s.Equal(res.Ids, []uint64{9, 8, 7, 6, 5})

	pageReq.Limit = 5
	pageReq.Offset = 4
	pageReq.Reverse = false
	res, err = q.ReportedIdsByReporter(s.ctx, &queryReq)
	s.NoError(err)
	s.Equal(res.Ids, []uint64{5, 6, 7, 8, 9})
}

func (s *KeeperTestSuite) TestGetTimestampBeforeandAfterQueries() {
	require := s.Require()
	k := s.oracleKeeper
	q := s.queryClient
	ctx := s.ctx

	queryId := []byte("0x1234")
	time := uint64(1000)
	// store has timestamps of 100, 999, 1000, 1001, 10000
	require.NoError(k.Aggregates.Set(ctx, collections.Join(queryId, time), types.Aggregate{
		QueryId:           queryId,
		AggregateValue:    "100",
		AggregateReporter: sdk.AccAddress("reporter").String(),
		AggregatePower:    100,
	}))
	time = uint64(1001)
	require.NoError(k.Aggregates.Set(ctx, collections.Join(queryId, time), types.Aggregate{
		QueryId:           queryId,
		AggregateValue:    "100",
		AggregateReporter: sdk.AccAddress("reporter").String(),
		AggregatePower:    100,
	}))
	time = uint64(999)
	require.NoError(k.Aggregates.Set(ctx, collections.Join(queryId, time), types.Aggregate{
		QueryId:           queryId,
		AggregateValue:    "100",
		AggregateReporter: sdk.AccAddress("reporter").String(),
		AggregatePower:    100,
	}))
	time = uint64(100)
	require.NoError(k.Aggregates.Set(ctx, collections.Join(queryId, time), types.Aggregate{
		QueryId:           queryId,
		AggregateValue:    "100",
		AggregateReporter: sdk.AccAddress("reporter").String(),
		AggregatePower:    100,
	}))
	time = uint64(10000)
	require.NoError(k.Aggregates.Set(ctx, collections.Join(queryId, time), types.Aggregate{
		QueryId:           queryId,
		AggregateValue:    "100",
		AggregateReporter: sdk.AccAddress("reporter").String(),
		AggregatePower:    100,
	}))
	// GETTIMESTAMPBEFORE TESTS
	// before 1000 -> 999 (not inclusive)
	res, err := q.GetTimestampBefore(ctx, &types.QueryGetTimestampBeforeRequest{
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: uint64(1000),
	})
	require.NoError(err)
	require.Equal(res.Timestamp, uint64(999))

	// before 1001 -> 1000
	res, err = q.GetTimestampBefore(ctx, &types.QueryGetTimestampBeforeRequest{
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: uint64(1001),
	})
	require.NoError(err)
	require.Equal(res.Timestamp, uint64(1000))

	// before 1000000000 -> 10000
	res, err = q.GetTimestampBefore(ctx, &types.QueryGetTimestampBeforeRequest{
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: uint64(1000000000),
	})
	require.NoError(err)
	require.Equal(res.Timestamp, uint64(10000))

	// before 991 -> 100
	res, err = q.GetTimestampBefore(ctx, &types.QueryGetTimestampBeforeRequest{
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: uint64(991),
	})
	require.NoError(err)
	require.Equal(res.Timestamp, uint64(100))

	// GETTIMESTAMPAFTER TESTS
	// after 1000 -> 1001 (not inclusive)
	resAfter, err := q.GetTimestampAfter(ctx, &types.QueryGetTimestampAfterRequest{
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: uint64(1000),
	})
	require.NoError(err)
	require.Equal(resAfter.Timestamp, uint64(1001))

	// after 9000 -> 10000
	resAfter, err = q.GetTimestampAfter(ctx, &types.QueryGetTimestampAfterRequest{
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: uint64(9000),
	})
	require.NoError(err)
	require.Equal(resAfter.Timestamp, uint64(10000))

	// after 1 -> 100
	resAfter, err = q.GetTimestampAfter(ctx, &types.QueryGetTimestampAfterRequest{
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: uint64(1),
	})
	require.NoError(err)
	require.Equal(resAfter.Timestamp, uint64(100))

	// after 999 -> 1000
	resAfter, err = q.GetTimestampAfter(ctx, &types.QueryGetTimestampAfterRequest{
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: uint64(999),
	})
	require.NoError(err)
	require.Equal(resAfter.Timestamp, uint64(1000))
}
