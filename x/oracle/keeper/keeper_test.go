package keeper_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/mocks"
	"github.com/tellor-io/layer/x/oracle/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	value = "000000000000000000000000000000000000000000000058528649cf80ee0000"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	oracleKeeper   keeper.Keeper
	bankKeeper     *mocks.BankKeeper
	accountKeeper  *mocks.AccountKeeper
	registryKeeper *mocks.RegistryKeeper
	reporterKeeper *mocks.ReporterKeeper

	queryClient types.QueryServer
	msgServer   types.MsgServer
}

func (s *KeeperTestSuite) SetupTest() {
	config.SetupConfig()

	s.oracleKeeper,
		s.reporterKeeper,
		s.registryKeeper,
		s.accountKeeper,
		s.bankKeeper,
		s.ctx = keepertest.OracleKeeper(s.T())

	s.msgServer = keeper.NewMsgServerImpl(s.oracleKeeper)
	s.queryClient = keeper.NewQuerier(s.oracleKeeper)

	// Initialize params
	s.NoError(s.oracleKeeper.SetParams(s.ctx, types.DefaultParams()))
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestNewKeeper() {
	require := s.Require()

	badAuthority := "bad_authority"
	require.Panics(func() {
		_ = keeper.NewKeeper(nil, nil, nil, nil, nil, nil, badAuthority)
	})
}

func (s *KeeperTestSuite) TestGetAuthority() {
	require := s.Require()
	k := s.oracleKeeper

	authority := k.GetAuthority()
	require.Equal(authority, authtypes.NewModuleAddress(govtypes.ModuleName).String())
}

func (s *KeeperTestSuite) TestLogger() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx
	logger := k.Logger(ctx)
	require.NotNil(logger)
}

func (s *KeeperTestSuite) TestInitializeQuery() {
	require := s.Require()
	k := s.oracleKeeper
	rk := s.registryKeeper
	ctx := s.ctx

	badQueryData := []byte("badQueryData")
	queryMeta, err := k.InitializeQuery(ctx, badQueryData)
	require.NotNil(queryMeta)
	require.Error(err)

	queryData := "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000019416d706c65666f727468437573746f6d53706f74507269636500000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000"
	querydataBytes := hexutil.MustDecode(queryData)
	queryType := "AmpleforthCustomSpotPrice"
	rk.On("GetSpec", ctx, queryType).Return(regtypes.DataSpec{
		ReportBufferWindow: 1000,
	}, errors.New("bad")).Once()
	queryMeta, err = k.InitializeQuery(ctx, querydataBytes)
	require.NotNil(queryMeta)
	require.Error(err)

	rk.On("GetSpec", ctx, queryType).Return(regtypes.DataSpec{
		ReportBufferWindow: 1000,
	}, nil).Once()
	queryMeta, err = k.InitializeQuery(ctx, querydataBytes)
	require.NotNil(queryMeta)
	require.NoError(err)
	require.Equal(queryMeta.Id, uint64(0))
	expectedId := querydataBytes
	require.Equal(queryMeta.QueryData, expectedId)
	require.Equal(queryMeta.RegistrySpecTimeframe, time.Duration(1000))
}

func (s *KeeperTestSuite) TestUpdateQuery() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// set spotprice query at 500ns
	queryType := "SpotPrice"
	queryId := utils.QueryIDFromData([]byte("SpotPrice"))
	require.NoError(k.Query.Set(ctx, collections.Join(queryId, uint64(1)), types.QueryMeta{
		QueryType:             queryType,
		RegistrySpecTimeframe: 500,
		QueryData:             []byte("SpotPrice"),
	}))

	// update spotprice type to 1000 ns
	err := k.UpdateQuery(ctx, queryType, time.Duration(1000))
	require.NoError(err)

	// check on a spotprice query
	iter, err := k.Query.Indexes.QueryType.MatchExact(ctx, queryType)
	require.NoError(err)
	queries, err := indexes.CollectValues(ctx, k.Query, iter)
	require.NoError(err)
	require.Equal(queries[0].QueryType, queryType)
	require.Equal(queries[0].RegistrySpecTimeframe, time.Duration(1000))
}

func (s *KeeperTestSuite) TestFlagAggregateReport() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// no matches
	require.NoError(k.FlagAggregateReport(ctx, types.MicroReport{}))

	// set aggregate
	queryId := utils.QueryIDFromData([]byte("queryId"))
	reporter1 := sample.AccAddress()
	reporter2 := sample.AccAddress()
	require.NoError(k.Aggregates.Set(
		s.ctx,
		collections.Join(queryId, ctx.BlockTime().UnixMilli()),
		types.Aggregate{
			Reporters: []*types.AggregateReporter{
				{
					Reporter: reporter1,
					Power:    40,
				},
				{
					Reporter: reporter2,
					Power:    60,
				},
			},
			Flagged: false,
		},
	))
	report := types.MicroReport{
		BlockNumber: 0,
		QueryId:     queryId,
		Reporter:    reporter1,
	}
	// flag report
	require.NoError(k.FlagAggregateReport(ctx, report))

	// check that report is flagged
	aggregate, err := k.Aggregates.Get(ctx, collections.Join(queryId, ctx.BlockTime().UnixMilli()))
	require.NoError(err)
	require.True(aggregate.Flagged)
}

// func TestAggregateLegacyCodec(t *testing.T) {
// 	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
// 	storeSvc := runtime.NewKVStoreService(storeKey)
// 	sb := collections.NewSchemaBuilder(storeSvc)
// 	registry := codectypes.NewInterfaceRegistry()
// 	cdc := codec.NewProtoCodec(registry)

// 	db := tmdb.NewMemDB()
// 	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
// 	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
// 	require.NoError(t, stateStore.LoadLatestVersion())

// 	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

// 	oldmap := collections.NewIndexedMap(sb, types.AggregatesPrefix, "aggregates", collections.PairKeyCodec(collections.BytesKey, collections.Int64Key), codec.CollValue[types.LegacyAggregate](cdc), newLegacyAggregatesIndex(sb))

// 	// generate test data
// 	testAggs := make([]types.LegacyAggregate, 100)
// 	for i := 0; i < 100; i++ {
// 		// make all values random
// 		agg := types.LegacyAggregate{
// 			QueryId:           []byte(fmt.Sprintf("queryId-%d", i)),
// 			AggregateValue:    fmt.Sprintf("aggregateValue-%d", i),
// 			ReporterPower:     int64(rand.Intn(1000)),
// 			StandardDeviation: rand.Float64(),
// 			Reporters: []*types.AggregateReporter{
// 				{
// 					Reporter:    sample.AccAddress(),
// 					Power:       int64(rand.Intn(1000)),
// 					BlockNumber: int64(rand.Intn(1000)),
// 				},
// 				{
// 					Reporter:    sample.AccAddress(),
// 					Power:       int64(rand.Intn(1000)),
// 					BlockNumber: int64(rand.Intn(1000)),
// 				},
// 			},
// 			Flagged:              rand.Intn(2) == 1, // random bool
// 			Index:                uint64(rand.Intn(1000)),
// 			AggregateReportIndex: int64(rand.Intn(1000)),
// 			Height:               int64(rand.Intn(1000)),
// 			MicroHeight:          int64(rand.Intn(1000)),
// 		}

// 		err := oldmap.Set(ctx, collections.Join(agg.QueryId, agg.Height), agg)
// 		require.NoError(t, err)
// 		testAggs[i] = agg
// 	}

// 	// now we replicate the issue, by walking the map
// 	newMap := collections.NewIndexedMap(sb, types.AggregatesPrefix, "aggregates", collections.PairKeyCodec(collections.BytesKey, collections.Int64Key), codec.CollValue[types.Aggregate](cdc), types.NewAggregatesIndex(sb))

// 	err := newMap.Walk(ctx, nil, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
// 		return false, nil
// 	})
// 	require.ErrorContains(t, err, "proto: wrong wireType = 1 for field StandardDeviation")

// 	// and finally we test the fix we've prepared
// 	fixedNewMap := collections.NewIndexedMap(sb, types.AggregatesPrefix, "aggregates", collections.PairKeyCodec(collections.BytesKey, collections.Int64Key), keeper.NewAggregateLegacyValueCodec(cdc), types.NewAggregatesIndex(sb))
// 	count := 0

// 	// add a new aggregate for good measure
// 	agg := types.Aggregate{StandardDeviation: "0.1998"}
// 	err = fixedNewMap.Set(ctx, collections.Join([]byte("queryId1998"), ctx.BlockTime().UnixMilli()), agg)
// 	require.NoError(t, err)

// 	err = fixedNewMap.Walk(ctx, nil, func(key collections.Pair[[]byte, int64], value types.Aggregate) (stop bool, err error) {
// 		parsedStdDev, err := strconv.ParseFloat(value.StandardDeviation, 64)
// 		require.NoError(t, err)

// 		if count == 100 { // check the last value differently as it's not in the testAggs
// 			if value.StandardDeviation != "0.1998" {
// 				return false, fmt.Errorf("StandardDeviation not equal")
// 			}
// 		} else if testAggs[count].StandardDeviation != parsedStdDev {
// 			return false, fmt.Errorf("StandardDeviation not equal")
// 		}
// 		count++
// 		return false, nil
// 	})
// 	require.Equal(t, 101, count) // make sure we walked all the values successfully
// 	require.NoError(t, err)
// }

// type legacyAggregatesIndex struct {
// 	BlockHeight *indexes.Multi[int64, collections.Pair[[]byte, int64], types.LegacyAggregate]
// 	MicroHeight *indexes.Multi[int64, collections.Pair[[]byte, int64], types.LegacyAggregate]
// }

// func (a legacyAggregatesIndex) IndexesList() []collections.Index[collections.Pair[[]byte, int64], types.LegacyAggregate] {
// 	return []collections.Index[collections.Pair[[]byte, int64], types.LegacyAggregate]{
// 		a.BlockHeight, a.MicroHeight,
// 	}
// }

// func newLegacyAggregatesIndex(sb *collections.SchemaBuilder) legacyAggregatesIndex {
// 	return legacyAggregatesIndex{
// 		BlockHeight: indexes.NewMulti(
// 			sb, types.AggregatesHeightIndexPrefix, "aggregates_by_height",
// 			collections.Int64Key, collections.PairKeyCodec[[]byte, int64](collections.BytesKey, collections.Int64Key),
// 			func(_ collections.Pair[[]byte, int64], v types.LegacyAggregate) (int64, error) {
// 				return v.Height, nil
// 			},
// 		),
// 		MicroHeight: indexes.NewMulti(
// 			sb, types.AggregatesMicroHeightIndexPrefix, "aggregates_by_micro_height",
// 			collections.Int64Key, collections.PairKeyCodec[[]byte, int64](collections.BytesKey, collections.Int64Key),
// 			func(_ collections.Pair[[]byte, int64], v types.LegacyAggregate) (int64, error) {
// 				return v.MicroHeight, nil
// 			},
// 		),
// 	}
// }
