package keeper_test

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
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
	}))
	res, err = q.GetCurrentAggregateReport(s.ctx, &req)
	require.NoError(err)
	require.NotNil(res)
	require.Equal(res.Aggregate.QueryId, qId)
	require.Equal(res.Aggregate.AggregateValue, "100")
	require.Equal(res.Aggregate.AggregateReporter, sdk.AccAddress("reporter").String())
	require.Equal(res.Aggregate.AggregatePower, uint64(100))
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
		QueryId:           []byte(queryId),
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
	require.Equal(t, getCurrentAggResponse.Aggregate.QueryId, agg.QueryId)
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

	reportedAt := time.Now()
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
				QueryId:  hex.EncodeToString(qId),
				Reporter: reporter.String(),
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

func (s *KeeperTestSuite) TestTippedQueries() {
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
		req         *types.QueryTippedQueriesRequest
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
			req:         &types.QueryTippedQueriesRequest{},
			err:         false,
			expectedLen: 0,
		},
		{
			name: "success one tipped query",
			setup: func() {
				require.NoError(k.Query.Set(ctx, collections.Join(queryMeta.QueryData, uint64(1)), queryMeta))
			},
			req: &types.QueryTippedQueriesRequest{
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
			req: &types.QueryTippedQueriesRequest{
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
			req: &types.QueryTippedQueriesRequest{
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
			resp, err := q.TippedQueries(ctx, tt.req)
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
