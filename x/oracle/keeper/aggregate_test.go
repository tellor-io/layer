package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil/sample"
	layer "github.com/tellor-io/layer/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestGetTimestampBefore() {
	testCases := []struct {
		name       string
		timestamps []time.Time
		target     time.Time
		expectedTs time.Time
	}{
		{
			name:       "Empty slice",
			timestamps: []time.Time{},
			target:     time.Unix(100, 0),
			expectedTs: time.Time{},
		},
		{
			name:       "Single timestamp before target",
			timestamps: []time.Time{time.Unix(50, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(50, 0),
		},
		{
			name:       "Single timestamp after target",
			timestamps: []time.Time{time.Unix(150, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Time{},
		},
		{
			name:       "Multiple timestamps, target present",
			timestamps: []time.Time{time.Unix(50, 0), time.Unix(100, 0), time.Unix(150, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(50, 0),
		},
		{
			name:       "Multiple timestamps, target not present",
			timestamps: []time.Time{time.Unix(50, 0), time.Unix(70, 0), time.Unix(90, 0), time.Unix(110, 0), time.Unix(130, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(90, 0),
		},
		{
			name:       "Multiple timestamps, target before all",
			timestamps: []time.Time{time.Unix(200, 0), time.Unix(300, 0), time.Unix(400, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Time{},
		},
		{
			name:       "Multiple timestamps, target after all",
			timestamps: []time.Time{time.Unix(10, 0), time.Unix(20, 0), time.Unix(40, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(40, 0),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			queryId := []byte("test")
			for _, v := range tc.timestamps {
				err := s.oracleKeeper.Aggregates.Set(
					s.ctx,
					collections.Join(queryId, v.Unix()),
					types.Aggregate{},
				)
				s.Require().NoError(err)
			}

			ts, err := s.oracleKeeper.GetTimestampBefore(s.ctx, queryId, tc.target)
			if ts.IsZero() {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}

			if ts != tc.expectedTs {
				t.Errorf("Test '%s' failed: expected %v, got %v", tc.name, tc.expectedTs, ts)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetTimestampAfter() {
	testCases := []struct {
		name       string
		timestamps []time.Time
		target     time.Time
		expectedTs time.Time
	}{
		{
			name:       "Empty slice",
			timestamps: []time.Time{},
			target:     time.Unix(100, 0),
			expectedTs: time.Time{},
		},
		{
			name:       "Single timestamp after target",
			timestamps: []time.Time{time.Unix(50, 0)},
			target:     time.Unix(25, 0),
			expectedTs: time.Unix(50, 0),
		},
		{
			name:       "Single timestamp before target",
			timestamps: []time.Time{time.Unix(150, 0)},
			target:     time.Unix(200, 0),
			expectedTs: time.Time{},
		},
		{
			name:       "Multiple timestamps, target present",
			timestamps: []time.Time{time.Unix(50, 0), time.Unix(100, 0), time.Unix(150, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(150, 0),
		},
		{
			name:       "Multiple timestamps, target not present",
			timestamps: []time.Time{time.Unix(50, 0), time.Unix(70, 0), time.Unix(90, 0), time.Unix(110, 0), time.Unix(130, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(110, 0),
		},
		{
			name:       "Multiple timestamps, target before all",
			timestamps: []time.Time{time.Unix(200, 0), time.Unix(300, 0), time.Unix(400, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(200, 0),
		},
		{
			name:       "Multiple timestamps, target after all",
			timestamps: []time.Time{time.Unix(10, 0), time.Unix(20, 0), time.Unix(40, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Time{},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			queryId := []byte("test")
			for _, v := range tc.timestamps {
				err := s.oracleKeeper.Aggregates.Set(
					s.ctx,
					collections.Join(queryId, v.Unix()),
					types.Aggregate{},
				)
				s.Require().NoError(err)
			}

			ts, err := s.oracleKeeper.GetTimestampAfter(s.ctx, queryId, tc.target)
			if ts.IsZero() {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}

			if ts != tc.expectedTs {
				t.Errorf("Test '%s' failed: expected %v, got %v", tc.name, tc.expectedTs, ts)
			}
		})
	}
}

func createMicroReportForQuery(reporterAdd, aggMethod, valueHex string, power int64, query types.QueryMeta, timestamp time.Time) types.MicroReport {
	return types.MicroReport{
		Reporter:        reporterAdd,
		Power:           power,
		QueryId:         []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0"),
		QueryType:       "SpotPrice",
		AggregateMethod: aggMethod,
		Value:           valueHex,
		Timestamp:       timestamp,
		Cyclelist:       true,
		BlockNumber:     10,
	}
}

func (s *KeeperTestSuite) TestSetAggregatedReport() {
	timestamp := time.Now()
	ctx := s.ctx.WithBlockTime(timestamp)
	ctx = ctx.WithBlockHeight(10)
	// setup
	rep1 := sample.AccAddressBytes()
	rep2 := sample.AccAddressBytes()
	rep3 := sample.AccAddressBytes()
	rep4 := sample.AccAddressBytes()

	expiration := time.Now().Unix() - 20

	queryData := types.QueryMeta{
		Id:                    1,
		Amount:                math.NewInt(1 * 1e6),
		Expiration:            time.Unix(expiration, expiration*time.Second.Nanoseconds()),
		RegistrySpecTimeframe: 0,
		HasRevealedReports:    true,
		QueryId:               []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0"),
		QueryType:             "SpotPrice",
	}

	err := s.oracleKeeper.Query.Set(ctx, queryData.QueryId, queryData)
	s.NoError(err)

	val := "0x0000000000000000000000000000000000000000000000b4ed64f50fa9b7b8f2"

	report_one := createMicroReportForQuery(rep1.String(), "weighted-median", val, 1000000000, queryData, time.Now())
	report_two := createMicroReportForQuery(rep2.String(), "weighted-median", val, 1000000000, queryData, time.Now())
	report_three := createMicroReportForQuery(rep3.String(), "weighted-median", val, 1000000000, queryData, time.Now())
	report_four := createMicroReportForQuery(rep4.String(), "weighted-median", val, 1000000000, queryData, time.Now())

	err = s.oracleKeeper.Reports.Set(ctx, collections.Join3(queryData.QueryId, rep1.Bytes(), queryData.Id), report_one)
	s.NoError(err)
	err = s.oracleKeeper.Reports.Set(ctx, collections.Join3(queryData.QueryId, rep2.Bytes(), queryData.Id), report_two)
	s.NoError(err)
	err = s.oracleKeeper.Reports.Set(ctx, collections.Join3(queryData.QueryId, rep3.Bytes(), queryData.Id), report_three)
	s.NoError(err)
	err = s.oracleKeeper.Reports.Set(ctx, collections.Join3(queryData.QueryId, rep4.Bytes(), queryData.Id), report_four)
	s.NoError(err)

	// use auth types GetModule Account
	add := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(add)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)

	// set up mock of the getTimeBasedRewards function as the account does not exist yet. We will make it return 1*1e6 loya
	s.accountKeeper.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(testModuleAccount)
	s.bankKeeper.On("GetBalance", mock.Anything, mock.Anything, layer.BondDenom).Return(sdk.Coin{Amount: math.NewInt(1 * 1e6)})
	s.bankKeeper.On("SendCoinsFromModuleToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err = s.oracleKeeper.SetAggregatedReport(ctx)
	s.NoError(err)

	reporter_one_balance := s.bankKeeper.GetBalance(ctx, rep1, "loya")
	div4_totalTip := math.NewInt(1 * 1e6).QuoRaw(4)
	s.True(reporter_one_balance.Amount.GTE(div4_totalTip))

	reporter_two_balance := s.bankKeeper.GetBalance(ctx, rep2, "loya")
	s.True(reporter_two_balance.Amount.GTE(div4_totalTip))

	reporter_three_balance := s.bankKeeper.GetBalance(ctx, rep3, "loya")
	s.True(reporter_three_balance.Amount.GTE(div4_totalTip))

	reporter_four_balance := s.bankKeeper.GetBalance(ctx, rep4, "loya")
	s.True(reporter_four_balance.Amount.GTE(div4_totalTip))

	res_query, err := s.oracleKeeper.Query.Get(ctx, queryData.QueryId)
	s.NoError(err)
	s.Equal(false, res_query.HasRevealedReports)
	s.Equal(math.ZeroInt(), res_query.Amount)

	aggregate, err := s.oracleKeeper.Aggregates.Get(ctx, collections.Join(queryData.QueryId, timestamp.Unix()))
	s.NoError(err)
	s.Equal(4, len(aggregate.Reporters))
}
