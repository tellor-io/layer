package keeper_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func createMicroReportForQuery(reporterAdd, aggMethod, value string, power int64, timestamp time.Time) types.MicroReport {
	return types.MicroReport{
		Reporter:        reporterAdd,
		Power:           power,
		QueryId:         []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0"),
		QueryType:       "SpotPrice",
		AggregateMethod: aggMethod,
		Value:           value,
		Timestamp:       timestamp,
		Cyclelist:       true,
		BlockNumber:     10,
	}
}

func encodeValue(number float64) string {
	strNumber := fmt.Sprintf("%.18f", number)

	parts := strings.Split(strNumber, ".")
	if len(parts[1]) > 18 {
		parts[1] = parts[1][:18]
	}
	truncatedStr := parts[0] + parts[1]

	bigIntNumber := new(big.Int)
	bigIntNumber.SetString(truncatedStr, 10)

	uint256ABIType, _ := abi.NewType("uint256", "", nil)

	arguments := abi.Arguments{{Type: uint256ABIType}}
	encodedBytes, _ := arguments.Pack(bigIntNumber)

	encodedString := hex.EncodeToString(encodedBytes)
	return encodedString
}

func (s *KeeperTestSuite) CreateReportAndReportersAtTimestamp(timestamp time.Time) (agg *types.Aggregate, queryId []byte, rep1, rep2 sdk.AccAddress, err error) {
	rep1 = sample.AccAddressBytes()
	rep2 = sample.AccAddressBytes()
	queryId = []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0")

	report := &types.Aggregate{
		QueryId:           queryId,
		AggregateValue:    encodeValue(96.50),
		AggregateReporter: rep1.String(),
		ReporterPower:     math.NewInt(200000000).Mul(layertypes.PowerReduction).Int64(),
		StandardDeviation: 0.3,
		Reporters:         []*types.AggregateReporter{{Reporter: rep1.String(), Power: 100000000}, {Reporter: rep2.String(), Power: 100000000}},
		Flagged:           false,
		Height:            10,
	}

	s.ctx = s.ctx.WithBlockTime(timestamp)
	s.ctx = s.ctx.WithBlockHeight(10)
	err = s.oracleKeeper.SetAggregate(s.ctx, report)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return report, queryId, rep1, rep2, nil
}

func (s *KeeperTestSuite) TestSetAggregatedReport() {
	timestamp := time.Now()
	ctx := s.ctx.WithBlockTime(timestamp)

	// setup
	rep1 := sample.AccAddressBytes()
	rep2 := sample.AccAddressBytes()
	rep3 := sample.AccAddressBytes()
	rep4 := sample.AccAddressBytes()

	expiration := timestamp.Unix() - 20

	queryData := types.QueryMeta{
		Id:                    1,
		Amount:                math.NewInt(1 * 1e6),
		Expiration:            time.Unix(expiration, 0),
		RegistrySpecTimeframe: 0,
		HasRevealedReports:    true,
		QueryId:               []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0"),
		QueryType:             "SpotPrice",
	}

	err := s.oracleKeeper.Query.Set(ctx, queryData.QueryId, queryData)
	s.NoError(err)

	val := encodeValue(1.00)
	report_one := createMicroReportForQuery(rep1.String(), "weighted-median", val, 1000000000, time.Now())
	report_two := createMicroReportForQuery(rep2.String(), "weighted-median", val, 1000000000, time.Now())
	report_three := createMicroReportForQuery(rep3.String(), "weighted-median", val, 1000000000, time.Now())
	report_four := createMicroReportForQuery(rep4.String(), "weighted-median", val, 1000000000, time.Now())

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
	s.bankKeeper.On("GetBalance", mock.Anything, mock.Anything, layertypes.BondDenom).Return(sdk.Coin{Amount: math.NewInt(1 * 1e6)})
	s.bankKeeper.On("SendCoinsFromModuleToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.reporterKeeper.On("DivvyingTips", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.reporterKeeper.On("AllocateTokensToReporter", mock.Anything, mock.Anything, mock.Anything).Return(nil)

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

	aggregate, err := s.oracleKeeper.GetCurrentValueForQueryId(ctx, []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0"))
	s.NoError(err)
	s.NotEqual("", aggregate.AggregateValue)
}

func (s *KeeperTestSuite) TestSetAggregate() {
	queryId := []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0")
	err := s.oracleKeeper.Nonces.Set(s.ctx, queryId, 0)
	s.NoError(err)
	reporter := sample.AccAddressBytes()

	timestamp := time.Now()
	s.ctx = s.ctx.WithBlockTime(timestamp)
	report := &types.Aggregate{
		QueryId:           queryId,
		AggregateValue:    encodeValue(96.50),
		AggregateReporter: reporter.String(),
		ReporterPower:     100000000,
		StandardDeviation: 0.3,
		Reporters:         []*types.AggregateReporter{{Reporter: reporter.String(), Power: 100000000}},
		Flagged:           false,
	}

	err = s.oracleKeeper.SetAggregate(s.ctx, report)
	s.NoError(err)

	res, err := s.oracleKeeper.Aggregates.Get(s.ctx, collections.Join(queryId, timestamp.Unix()))
	s.NoError(err)
	s.Equal(encodeValue(96.50), res.AggregateValue)
	s.Equal(int64(100000000), res.ReporterPower)
}

func (s *KeeperTestSuite) TestGetDataBefore() {
	reportedAt := time.Now()
	aggregate, qId, _, _, err := s.CreateReportAndReportersAtTimestamp(reportedAt)
	s.NoError(err)

	jumpForward := reportedAt.Unix() + (60 * 5)
	queryAt := time.Unix(jumpForward, 0)

	goback := reportedAt.Unix() - (60 * 5)
	earlyQuery := time.Unix(goback, 0)

	s.ctx = s.ctx.WithBlockTime(queryAt)
	retAggregate, err := s.oracleKeeper.GetDataBefore(s.ctx, qId, queryAt)
	s.NoError(err)
	s.Equal(aggregate, retAggregate)

	s.ctx = s.ctx.WithBlockTime(reportedAt)
	nilAggregate, err := s.oracleKeeper.GetDataBefore(s.ctx, qId, earlyQuery)
	s.Nil(nilAggregate)
	s.NotNil(err)
}

func (s *KeeperTestSuite) TestGetCurrentValueForQueryId() {
	reportedAt := time.Now()
	aggregate, qId, _, _, err := s.CreateReportAndReportersAtTimestamp(reportedAt)
	s.NoError(err)

	s.ctx = s.ctx.WithBlockTime(reportedAt)
	retAggregate, err := s.oracleKeeper.GetCurrentValueForQueryId(s.ctx, qId)
	s.NoError(err)
	s.Equal(aggregate, retAggregate)
}

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

// STILL NEED TO FINISH THIS ONE
func (s *KeeperTestSuite) TestGetAggregatedReportsByHeight() {
	reportedAt := time.Now()
	aggregate, _, _, _, err := s.CreateReportAndReportersAtTimestamp(reportedAt)
	s.NoError(err)

	s.ctx = s.ctx.WithBlockHeight(15)
	aggregates := s.oracleKeeper.GetAggregatedReportsByHeight(s.ctx, int64(10))
	s.NotEqual(0, len(aggregates))
	s.Equal(*aggregate, aggregates[0])
}

func (s *KeeperTestSuite) TestGetCurrentAggregateReport() {
	reportedAt := time.Now()
	aggregate, qId, _, _, err := s.CreateReportAndReportersAtTimestamp(reportedAt)
	s.NoError(err)

	retAgg, timestamp := s.oracleKeeper.GetCurrentAggregateReport(s.ctx, qId)
	s.Equal(aggregate, retAgg)
	s.Equal(reportedAt.Unix(), timestamp.Unix())
}

func (s *KeeperTestSuite) TestGetAggregateBefore() {
	reportedAt := time.Now()
	aggregate, qId, _, _, err := s.CreateReportAndReportersAtTimestamp(reportedAt)
	s.NoError(err)

	fastForward := reportedAt.Unix() + 240
	queryAt := time.Unix(fastForward, 0)
	s.ctx = s.ctx.WithBlockTime(queryAt)

	retAgg, _, err := s.oracleKeeper.GetAggregateBefore(s.ctx, qId, queryAt)
	s.NoError(err)
	s.Equal(aggregate, retAgg)
}

func (s *KeeperTestSuite) TestGetAggregateByTimestamp() {
	reportedAt := time.Now()
	aggregate, qId, _, _, err := s.CreateReportAndReportersAtTimestamp(reportedAt)
	s.NoError(err)

	// fastForward := reportedAt.Unix() + 240
	// queryAt := time.Unix(fastForward, 0)
	s.ctx = s.ctx.WithBlockTime(reportedAt)

	retAgg, err := s.oracleKeeper.GetAggregateByTimestamp(s.ctx, qId, reportedAt)
	s.NoError(err)
	s.Equal(aggregate, retAgg)
}
