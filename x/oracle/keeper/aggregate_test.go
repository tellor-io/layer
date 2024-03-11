package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/x/oracle/types"
)

// import (
// 	"encoding/hex"
// 	"fmt"

// 	"cosmossdk.io/math"
// 	"github.com/cosmos/cosmos-sdk/codec"
// 	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/tellor-io/layer/x/oracle/types"
// )

// // func createN(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.StoredGame {
// // 	items := make([]types.StoredGame, n)
// // 	for i := range items {
// // 		items[i].Index = strconv.Itoa(i)

// // 		keeper.SetStoredGame(ctx, items[i])
// // 	}
// // 	return items
// // }

// func (s *KeeperTestSuite) TestSetAggregatedReport() {
// 	require := s.Require()

// 	s.TestSubmitValue()
// 	reportStore := s.oracleKeeper.ReportsStore(s.ctx)
// 	fmt.Println(reportStore)
// 	require.NotNil(reportStore)
// 	s.oracleKeeper.SetAggregatedReport(s.ctx)
// 	// s.accountKeeper.Mock.On("GetModuleAccount", mock.Anything, mock.Anything).Return(nil)
// 	// require.NotNil(reportStore)
// }

// func (s *KeeperTestSuite) TestSetAggregate() {
// 	require := s.Require()

// 	// get info for expected report
// 	validatorData, err := s.stakingKeeper.Validator(s.ctx, sdk.ValAddress(Addr.String()))
// 	require.Nil(err)
// 	queryId := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"

// 	store := s.oracleKeeper.AggregateStore(s.ctx)
// 	fmt.Println("store before submit value: ", store)

// 	s.TestSubmitValue()

// 	store = s.oracleKeeper.AggregateStore(s.ctx)
// 	fmt.Println("store after submit value: ", store)

// 	expectedAggregate := types.Aggregate{
// 		QueryId:           queryId,
// 		AggregateValue:    "000000000000000000000000000000000000000000000058528649cf80ee0000",
// 		AggregateReporter: Addr.String(),
// 		ReporterPower:     10,
// 		StandardDeviation: 0,
// 		Reporters: []*types.AggregateReporter{
// 			{
// 				Reporter: Addr.String(),
// 				Power:    validatorData.GetConsensusPower(math.NewInt(1000000000000000000)),
// 			},
// 		},
// 		Flagged:              false,
// 		Nonce:                0,
// 		AggregateReportIndex: 0,
// 	}
// 	fmt.Println(expectedAggregate.Marshal())

// 	oracle := codectypes.NewInterfaceRegistry()
// 	cdc := codec.NewProtoCodec(oracle)

// 	hexQueryId, err := hex.DecodeString(queryId)
// 	require.Nil(err)
// 	key := types.AggregateKey(hexQueryId, s.ctx.BlockTime())
// 	fmt.Println("key: ", key)
// 	store = s.oracleKeeper.AggregateStore(s.ctx)
// 	fmt.Println("store later: ", store)
// 	bz := store.Get(key)
// 	fmt.Println("bz: ", bz)
// 	var report types.Aggregate
// 	cdc.MustUnmarshal(bz, &report)
// 	newStore := s.oracleKeeper.AggregateStore(s.ctx)

// 	fmt.Println("newStore: ", newStore)
// 	// require.NotEqual(oldStore, newStore)
// }

// func (s *KeeperTestSuite) TestFindTimestampBefore() {
// 	require := s.Require()

// 	timestamps := []int64{1, 2, 3, 4, 5}
// 	found, index := FindTimestampBefore(timestamps, 3)
// }

func (s *KeeperTestSuite) TestFindTimestampBefore() {
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
			expectedTs: time.Unix(100, 0),
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

func (s *KeeperTestSuite) TestFindTimestampAfter() {
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
			expectedTs: time.Unix(100, 0),
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
