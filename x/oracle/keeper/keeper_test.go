package keeper_test

import (
	"encoding/hex"
	"errors"
	"fmt"
	gomath "math"
	"math/big"
	"sort"
	"strconv"
	"strings"
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
	bridgeKeeper   *mocks.BridgeKeeper
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
		s.bridgeKeeper,
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
		ReportBlockWindow: 1000,
	}, errors.New("bad")).Once()
	queryMeta, err = k.InitializeQuery(ctx, querydataBytes)
	require.NotNil(queryMeta)
	require.Error(err)

	rk.On("GetSpec", ctx, queryType).Return(regtypes.DataSpec{
		ReportBlockWindow: 1000,
	}, nil).Once()
	queryMeta, err = k.InitializeQuery(ctx, querydataBytes)
	require.NotNil(queryMeta)
	require.NoError(err)
	require.Equal(queryMeta.Id, uint64(0))
	expectedId := querydataBytes
	require.Equal(queryMeta.QueryData, expectedId)
	require.Equal(queryMeta.RegistrySpecBlockWindow, uint64(1000))
}

func (s *KeeperTestSuite) TestUpdateQuery() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// set spotprice query at 500ns
	queryType := "SpotPrice"
	queryId := utils.QueryIDFromData([]byte("SpotPrice"))
	require.NoError(k.Query.Set(ctx, collections.Join(queryId, uint64(1)), types.QueryMeta{
		QueryType:               queryType,
		RegistrySpecBlockWindow: 500,
		QueryData:               []byte("SpotPrice"),
	}))

	// update spotprice type to 1000 ns
	err := k.UpdateQuery(ctx, queryType, 1000)
	require.NoError(err)

	// check on a spotprice query
	iter, err := k.Query.Indexes.QueryType.MatchExact(ctx, queryType)
	require.NoError(err)
	queries, err := indexes.CollectValues(ctx, k.Query, iter)
	require.NoError(err)
	require.Equal(queries[0].QueryType, queryType)
	require.Equal(queries[0].RegistrySpecBlockWindow, uint64(1000))
}

func (s *KeeperTestSuite) TestFlagAggregateReport() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// set aggregate
	queryId := utils.QueryIDFromData([]byte("queryId"))
	reporter1 := sample.AccAddress()
	// no matches
	require.NoError(k.FlagAggregateReport(ctx, types.MicroReport{Reporter: reporter1}))

	require.NoError(k.Aggregates.Set(
		s.ctx,
		collections.Join(queryId, uint64(ctx.BlockTime().UnixMilli())),
		types.Aggregate{
			AggregateReporter: reporter1,
			MicroHeight:       0,
			QueryId:           queryId,
			Flagged:           false,
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
	aggregate, err := k.Aggregates.Get(ctx, collections.Join(queryId, uint64(ctx.BlockTime().UnixMilli())))
	require.NoError(err)
	require.True(aggregate.Flagged)
}

func (s *KeeperTestSuite) TestAddReport() {
	queryId, err := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	s.NoError(err)
	report := types.MicroReport{
		QueryId:         queryId,
		AggregateMethod: "weighted-median",
	}
	testCases := []struct {
		name            string
		id              uint64
		values          []string
		powers          []uint64
		crossoverWeight uint64
		median          string
	}{
		{
			name: "basic",
			id:   0,
			values: []string{
				"0000000000000000000000000000000000000000000014B2AC38D00387760000", // 97743260000000000000000
				"0000000000000000000000000000000000000000000014B394893D524B800000", // 97760000000000000000000
				"0000000000000000000000000000000000000000000014B5C863FF0DF39F0000", // 97800630000000000000000
				"0000000000000000000000000000000000000000000014B4518D3328DC520000", // 97773620000000000000000
				"0000000000000000000000000000000000000000000014BB36875354C2800000", // 97900800000000000000000
				"0000000000000000000000000000000000000000000014B30BB5F327D1000000", // 97900800000000000000000
				"0000000000000000000000000000000000000000000014C34CD126F10B460000", // 98049980000000000000000
				"0000000000000000000000000000000000000000000014B5BFC95056E2E10000", // 97800010000000000000000
			},
			powers:          []uint64{12, 34, 56, 78, 90, 23, 45, 67}, // sorted by their values = 12, 23, 34, 78, 67, 56, 90 = 360
			crossoverWeight: 214,
			median:          "0000000000000000000000000000000000000000000014B5BFC95056E2E10000",
		},
		{
			name: "values with one duplicate",
			id:   1,
			values: []string{
				"0000000000000000000000000000000000000000000014B2AC38D00387760000", // 97743260000000000000000
				"0000000000000000000000000000000000000000000014B394893D524B800000", // 97760000000000000000000
				"0000000000000000000000000000000000000000000014B5C863FF0DF39F0000", // 97800630000000000000000
				"0000000000000000000000000000000000000000000014B4518D3328DC520000", // 97773620000000000000000
				"0000000000000000000000000000000000000000000014BB36875354C2800000", // duplicate 97900800000000000000000
				"0000000000000000000000000000000000000000000014B30BB5F327D1000000", // 97750140694444451561472
				"0000000000000000000000000000000000000000000014C34CD126F10B460000", // 98049980000000000000000
				"0000000000000000000000000000000000000000000014B5BFC95056E2E10000", // 97800010000000000000000
				"0000000000000000000000000000000000000000000014BB36875354C2800000", // duplicate 97900800000000000000000
			},
			powers:          []uint64{12, 34, 56, 78, 90, 23, 45, 67, 150},
			crossoverWeight: 510,
			median:          "0000000000000000000000000000000000000000000014BB36875354C2800000", // 97900800000000000000000
		},
		{
			name: "first value is the median",
			id:   2,
			values: []string{
				"0000000000000000000000000000000000000000000014B5BFC95056E2E10000", // 97800010000000000000000
				"0000000000000000000000000000000000000000000014B2AC38D00387760000", // 97743260000000000000000
				"0000000000000000000000000000000000000000000014B394893D524B800000", // 97760000000000000000000
				"0000000000000000000000000000000000000000000014B5C863FF0DF39F0000", // 97800630000000000000000
				"0000000000000000000000000000000000000000000014B4518D3328DC520000", // 97773620000000000000000
				"0000000000000000000000000000000000000000000014BB36875354C2800000", // 97900800000000000000000
				"0000000000000000000000000000000000000000000014B30BB5F327D1000000", // 97750140694444451561472
				"0000000000000000000000000000000000000000000014C34CD126F10B460000", // 98049980000000000000000

			},
			powers:          []uint64{67, 12, 34, 56, 78, 90, 23, 45},
			crossoverWeight: 214,
			median:          "0000000000000000000000000000000000000000000014B5BFC95056E2E10000", // 97800010000000000000000
		},
		{
			name: "random short values",
			id:   3,
			values: []string{
				"0000000000000000000000000000000000000000000014B3877D32418753ACD", // 6109941241338196736717
				"0000000000000000000000000000000000000000000014BF4D01482376699B",  // 382719556917986224539
				"0000000000000000000000000000000000000000000014B2203E1A8210651C2", // 6108323339132306084290
				"0000000000000000000000000000000000000000000014B21DC1E551F803855", // 6108312146846939101269
				"0000000000000000000000000000000000000000000014BA526E56D36E089",   // 23897549764533084297
				"0000000000000000000000000000000000000000000014B34C435AB2CFAA101", // 6109674511392579428609
				"0000000000000000000000000000000000000000000014B2BC9CDF690BBEE0",  // 381814222991917825760
				"0000000000000000000000000000000000000000000014B35D4560A3A6A51C6", // 6109751108178864132550

			},
			powers:          []uint64{72, 34, 73, 58, 10, 99, 44, 93},
			crossoverWeight: 318,
			median:          "00000000000000000000000000000000000000000000014B34C435AB2CFAA101", // 6109674511392579428609
		},
		{
			name: "random short values",
			id:   4,
			values: []string{
				"0000000000000000000000000000000000000000000014B387C35382B29B043", // 6109942475076456263747
				"0000000000000000000000000000000000000000000014B322293AECB522E7",  // 381842806289244431079
				"0000000000000000000000000000000000000000000014B1192D81BAFD4D60",  // 381696162528533237088
				"0000000000000000000000000000000000000000000014B7DA332B331E759E",  // 382182838988689012126
				"0000000000000000000000000000000000000000000014B6CCA4E117BF4BD0",  // 382106965771015900112
				"0000000000000000000000000000000000000000000014B1296D0F930CB1EF3", // 6107211776105734938355
				"0000000000000000000000000000000000000000000014B25E2EBC917A48850", // 6108602291970919139408
				"0000000000000000000000000000000000000000000014B2286A342355DFA19", // 6108360143746788882969

			},
			powers:          []uint64{17, 62, 77, 73, 78, 22, 88, 37},
			crossoverWeight: 290,
			median:          "000000000000000000000000000000000000000000000014B7DA332B331E759E", // 382182838988689012126
		},
		{
			name: "two values repeated twice with different powers",
			id:   5,
			values: []string{
				"0000000000000000000000000000000000000000000014B37CF415D375EC868", // 6109893793258743449704
				"0000000000000000000000000000000000000000000014B1F08A41FE3219B9",  // 381756781629357038009
				"0000000000000000000000000000000000000000000014B2A702EAA7B8489C",  // 381808142740912359580
				"0000000000000000000000000000000000000000000014B2E876FCB25FB6FBC", // 6109225059763768487868
				"0000000000000000000000000000000000000000000014BCC9ED92115E892A",  // 382538546835252742442
				"0000000000000000000000000000000000000000000014B11D52FA71D72AA59", // 6107157274061345827417
				"0000000000000000000000000000000000000000000014BCC6413B25622206",  // 382537512920996258310
				"0000000000000000000000000000000000000000000014B1DF5084D30F2FB24", // 6108030929121882340132
				"0000000000000000000000000000000000000000000014B1DF5084D30F2FB24", // 6108030929121882340132
				"0000000000000000000000000000000000000000000014B1F08A41FE3219B9",  // 381756781629357038009

			},
			powers:          []uint64{43, 63, 89, 42, 27, 58, 39, 83, 30, 55},
			crossoverWeight: 273,
			median:          "000000000000000000000000000000000000000000000014BCC9ED92115E892A", // 382538546835252742442
		},
		{
			name: "two values same power",
			id:   6,
			values: []string{
				"0000000000000000000000000000000000000000000014B2AC38D00387760000", // 97743260000000000000000
				"0000000000000000000000000000000000000000000014B394893D524B800000", // 97760000000000000000000

			},
			powers:          []uint64{50, 50},
			crossoverWeight: 50,
			median:          "0000000000000000000000000000000000000000000014B2AC38D00387760000", // 97743260000000000000000
		},
		{
			name: "randomized input with repeated values",
			id:   7,
			values: []string{
				"0000000000000000000000000000000000000000000014B36A77C7331F0A774",
				"0000000000000000000000000000000000000000000014B2E221E128692F79",
				"0000000000000000000000000000000000000000000014B78A00871FCAE0D5",
				"0000000000000000000000000000000000000000000014B1603A97719BC43C1",
				"0000000000000000000000000000000000000000000014B12A8405219B5FD86",
				"0000000000000000000000000000000000000000000014B1DE5EC62A3A307C",
				"0000000000000000000000000000000000000000000014B1B73C2D533089975",
				"0000000000000000000000000000000000000000000014B2B8DCA8FEC46AC5",
				"0000000000000000000000000000000000000000000014B31BFE5C715ED3FA3",
				"0000000000000000000000000000000000000000000014B3B996F581CCEA734",
				"0000000000000000000000000000000000000000000014B1603A97719BC43C1",
				"0000000000000000000000000000000000000000000014B2E221E128692F79",
				"0000000000000000000000000000000000000000000014B1DE5EC62A3A307C",
			},
			powers:          []uint64{94, 19, 13, 99, 85, 42, 61, 45, 30, 64, 93, 78, 53},
			crossoverWeight: 527,
			median:          "00000000000000000000000000000000000000000000014B1603A97719BC43C1", // 6107458586220624102337
		},
		{
			name: "predictable sorted values",
			id:   8,
			values: []string{
				"0000000000000000000000000000000000000000000014B0000000A",
				"0000000000000000000000000000000000000000000014B00000014",
				"0000000000000000000000000000000000000000000014B0000001E",
				"0000000000000000000000000000000000000000000014B00000028",
				"0000000000000000000000000000000000000000000014B00000032",
				"0000000000000000000000000000000000000000000014B0000003C",
				"0000000000000000000000000000000000000000000014B00000046",
				"0000000000000000000000000000000000000000000014B00000050",
				"0000000000000000000000000000000000000000000014B0000005A",
				"0000000000000000000000000000000000000000000014B00000064",
			},
			powers:          []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			crossoverWeight: 28,
			median:          "0000000000000000000000000000000000000000000000000000014B00000046", // 1421634175046
		},
		{
			name: "large gaps between powers",
			id:   9,
			values: []string{
				"0000000000000000000000000000000000000000000014B000003E8",
				"0000000000000000000000000000000000000000000014B000007D0",
				"0000000000000000000000000000000000000000000014B00000BB8",
				"0000000000000000000000000000000000000000000014B00000FA0",
				"0000000000000000000000000000000000000000000014B00001388",
				"0000000000000000000000000000000000000000000014B00001770",
				"0000000000000000000000000000000000000000000014B00001B58",
				"0000000000000000000000000000000000000000000014B00001F40",
				"0000000000000000000000000000000000000000000014B00002328",
			},
			powers:          []uint64{7360, 1134, 4250, 3513, 5452, 9138, 2470, 4095, 4156},
			crossoverWeight: 21709,
			median:          "0000000000000000000000000000000000000000000000000000014B00001388", // 1421634179976
		},
		{
			name: "equal weights on consecutive values",
			id:   10,
			values: []string{
				"0000000000000000000000000000000000000000000014B0000000A",
				"0000000000000000000000000000000000000000000014B00000014",
				"0000000000000000000000000000000000000000000014B0000001E",
			},
			powers:          []uint64{25, 25, 50},
			crossoverWeight: 50,
			median:          "0000000000000000000000000000000000000000000000000000014B00000014",
		},
		{
			name: "clustered repeats",
			id:   11,
			values: []string{
				"0000000000000000000000000000000000000000000014B0000000A",
				"0000000000000000000000000000000000000000000014B0000000A",
				"0000000000000000000000000000000000000000000014B00000014",
			},
			powers:          []uint64{30, 20, 50},
			crossoverWeight: 50,
			median:          "0000000000000000000000000000000000000000000000000000014B0000000A",
		},
		{
			name: "Unequal but Close Weights",
			id:   12,
			values: []string{
				"0000000000000000000000000000000000000000000014B0000000A",
				"0000000000000000000000000000000000000000000014B00000014",
			},
			powers:          []uint64{45, 55},
			crossoverWeight: 100,
			median:          "0000000000000000000000000000000000000000000000000000014B00000014",
		},
	}
	for _, tc := range testCases {
		for i, v := range tc.values {
			report.Value = v
			report.Power = tc.powers[i]
			s.NoError(s.oracleKeeper.AddReport(s.ctx, tc.id, report))
			if i == len(tc.values)-1 {
				median, err := s.oracleKeeper.AggregateValue.Get(s.ctx, tc.id)
				s.Equal(tc.median, median.Value)
				s.Equal(tc.crossoverWeight, median.CrossoverWeight)
				s.NoError(err)
				calculatedMedian := WeightedMedian(tc.values, tc.powers)
				s.True(strings.EqualFold(calculatedMedian, median.Value))
			}
		}
	}
}

// test hasReveals+Expiration index
func (s *KeeperTestSuite) TestReportIndexedMap() {
	k := s.oracleKeeper
	s.NoError(k.Query.Set(s.ctx, collections.Join([]byte("queryid1"), uint64(1)), types.QueryMeta{Expiration: 10}))
	s.NoError(k.Query.Set(s.ctx, collections.Join([]byte("queryid2"), uint64(2)), types.QueryMeta{Expiration: 10}))
	s.NoError(k.Query.Set(s.ctx, collections.Join([]byte("queryid3"), uint64(3)), types.QueryMeta{Expiration: 10}))
	s.NoError(k.Query.Set(s.ctx, collections.Join([]byte("queryid4"), uint64(4)), types.QueryMeta{Expiration: 10}))
	s.NoError(k.Query.Set(s.ctx, collections.Join([]byte("queryid5"), uint64(5)), types.QueryMeta{Expiration: 10}))
	s.NoError(k.Query.Set(s.ctx, collections.Join([]byte("queryid6"), uint64(6)), types.QueryMeta{Expiration: 10}))
	s.NoError(k.Query.Set(s.ctx, collections.Join([]byte("queryid7"), uint64(7)), types.QueryMeta{Expiration: 10, HasRevealedReports: true}))
	s.NoError(k.Query.Set(s.ctx, collections.Join([]byte("queryid8"), uint64(8)), types.QueryMeta{Expiration: 10, HasRevealedReports: true}))
	// has revealed reports but not expired
	s.NoError(k.Query.Set(s.ctx, collections.Join([]byte("queryid9"), uint64(9)), types.QueryMeta{Expiration: 12, HasRevealedReports: true}))
	// expired but no revealed reports
	s.NoError(k.Query.Set(s.ctx, collections.Join([]byte("queryid10"), uint64(10)), types.QueryMeta{Expiration: 9}))

	rng := collections.NewPrefixUntilPairRange[collections.Pair[bool, uint64], collections.Pair[[]byte, uint64]](collections.Join(true, uint64(11))).Descending()
	iter, err := k.Query.Indexes.Expiration.Iterate(s.ctx, rng)
	s.NoError(err)
	expiredRevealedCount := 0
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key, err := iter.FullKey()
		s.NoError(err)
		if !key.K1().K1() {
			break
		}
		expiredRevealedCount++
	}
	s.Equal(2, expiredRevealedCount)
}

// helper function that intakes a list of hex strings(uint256) with their corresponding powers
// and returns the weighted median
func WeightedMedian(values []string, powers []uint64) string {
	type HexValue struct {
		Value  *big.Int
		Weight uint64
	}
	hexValues := make([]HexValue, len(values))
	for i, v := range values {
		val, ok := new(big.Int).SetString(v, 16)
		if !ok {
			return "failed to parse value"
		}
		hexValues[i] = HexValue{Value: val, Weight: powers[i]}
	}

	sort.Slice(hexValues, func(i, j int) bool {
		return hexValues[i].Value.Cmp(hexValues[j].Value) < 0
	})

	totalReporterPower := uint64(0)
	for _, p := range powers {
		totalReporterPower += p
	}

	halfWeight := totalReporterPower / 2
	cumulativePower := uint64(0)
	for _, v := range hexValues {
		cumulativePower += v.Weight
		if cumulativePower >= halfWeight {
			resp, _ := utils.FormatUint256(v.Value.Text(16))

			return resp
		}
	}
	return ""
}

func (s *KeeperTestSuite) TestBounds() {
	s.NoError(s.oracleKeeper.QuerySequencer.Set(s.ctx, uint64(gomath.MaxUint64)))
	s.NoError(s.oracleKeeper.Query.Set(s.ctx, collections.Join([]byte("queryid1"), uint64(gomath.MaxUint64)), types.QueryMeta{
		Id: uint64(gomath.MaxUint64),
	}))
	n, err := s.oracleKeeper.QuerySequencer.Next(s.ctx)
	s.NoError(err)
	s.Equal(uint64(gomath.MaxUint64), n)
	n, err = s.oracleKeeper.QuerySequencer.Next(s.ctx)
	s.NoError(err)
	s.Equal(uint64(0), n)
}

func (s *KeeperTestSuite) TestAutoClaimDeposits() {
	require := s.Require()
	require.NotNil(s.bridgeKeeper)
	timeNow := time.Now()
	ctx := s.ctx.WithBlockTime(timeNow).WithBlockHeight(1000)

	type Deposits struct {
		DepositId          uint64
		AggregateTimestamp uint64
	}

	// set 1 deposit right at 12 hr mark, 4 just under 12 hrs, 5 after 12 hrs
	threshold := timeNow.Add(-12 * time.Hour)
	deposits := make([]Deposits, 10)
	for i := range 10 {
		metaId := uint64(i)
		iString := strconv.Itoa(i)
		bridgeQueryData := "000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000009545242427269646765000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000" + iString
		queryDataBz, err := hex.DecodeString(bridgeQueryData)
		require.NoError(err)
		var depositTimestamp uint64
		var delta time.Duration
		if i < 5 {
			delta = time.Duration(i * 1e6) // add i ms
		}
		if i >= 5 {
			delta = time.Duration(-i * 1e6) // subtract i ms
		}
		depositTimestamp = uint64(threshold.Add(delta).UnixMilli())
		require.NoError(s.oracleKeeper.BridgeDepositQueue.Set(ctx, collections.Join(depositTimestamp, metaId), queryDataBz))
		deposits[i].DepositId = uint64(i)
		deposits[i].AggregateTimestamp = depositTimestamp
	}

	// check that everything is in collections
	iter, err := s.oracleKeeper.BridgeDepositQueue.Iterate(ctx, nil)
	require.NoError(err)
	var i int
	for ; iter.Valid(); iter.Next() {
		i++
	}
	require.Equal(i, 10)

	// deposit 9 is the oldest, should get claimed first
	s.bridgeKeeper.On("ClaimDeposit", ctx, deposits[9].DepositId, deposits[9].AggregateTimestamp).Return(nil).Once()
	require.NoError(s.oracleKeeper.AutoClaimDeposits(ctx))

	// then 8, 7, 6, 5, 0
	s.bridgeKeeper.On("ClaimDeposit", ctx, deposits[8].DepositId, deposits[8].AggregateTimestamp).Return(nil).Once()
	require.NoError(s.oracleKeeper.AutoClaimDeposits(ctx))
	s.bridgeKeeper.On("ClaimDeposit", ctx, deposits[7].DepositId, deposits[7].AggregateTimestamp).Return(nil).Once()
	require.NoError(s.oracleKeeper.AutoClaimDeposits(ctx))
	s.bridgeKeeper.On("ClaimDeposit", ctx, deposits[6].DepositId, deposits[6].AggregateTimestamp).Return(nil).Once()
	require.NoError(s.oracleKeeper.AutoClaimDeposits(ctx))
	s.bridgeKeeper.On("ClaimDeposit", ctx, deposits[5].DepositId, deposits[5].AggregateTimestamp).Return(nil).Once()
	require.NoError(s.oracleKeeper.AutoClaimDeposits(ctx))
	s.bridgeKeeper.On("ClaimDeposit", ctx, deposits[0].DepositId, deposits[0].AggregateTimestamp).Return(nil).Once()
	require.NoError(s.oracleKeeper.AutoClaimDeposits(ctx))

	// everything possible is claimed
	require.NoError(s.oracleKeeper.AutoClaimDeposits(ctx))

	// fast forward a min to make everything else claimable
	ctx = ctx.WithBlockTime(timeNow.Add(1 * time.Minute))

	// 1 should be oldest
	s.bridgeKeeper.On("ClaimDeposit", ctx, deposits[1].DepositId, deposits[1].AggregateTimestamp).Return(nil).Once()
	require.NoError(s.oracleKeeper.AutoClaimDeposits(ctx))
	s.bridgeKeeper.On("ClaimDeposit", ctx, deposits[2].DepositId, deposits[2].AggregateTimestamp).Return(nil).Once()
	require.NoError(s.oracleKeeper.AutoClaimDeposits(ctx))
	s.bridgeKeeper.On("ClaimDeposit", ctx, deposits[3].DepositId, deposits[3].AggregateTimestamp).Return(nil).Once()
	require.NoError(s.oracleKeeper.AutoClaimDeposits(ctx))
	s.bridgeKeeper.On("ClaimDeposit", ctx, deposits[4].DepositId, deposits[4].AggregateTimestamp).Return(nil).Once()
	require.NoError(s.oracleKeeper.AutoClaimDeposits(ctx))
}

func (s *KeeperTestSuite) TestGetLastReportedAtTimestamp() {
	require := s.Require()
	require.NotNil(s.bridgeKeeper)
	timeNow := time.Now()
	ctx := s.ctx.WithBlockTime(timeNow).WithBlockHeight(100)
	rk := s.reporterKeeper

	// reporter reported at block 50
	reporter := sample.AccAddressBytes()
	rk.On("GetLastReportedAtBlock", ctx, reporter.Bytes()).Return(uint64(50), nil).Once()
	// Theres an aggregate on block 50
	aggregate := types.Aggregate{
		Height:            50,
		AggregateReporter: reporter.String(),
	}
	timestampSet := uint64(time.Now().Add(-100 * time.Second).UnixMilli())
	fmt.Println("timestampSet", timestampSet)
	require.NoError(s.oracleKeeper.Aggregates.Set(ctx, collections.Join([]byte("queryid1"), timestampSet), aggregate))
	timestampRetrieved, err := s.oracleKeeper.GetLastReportedAtTimestamp(ctx, reporter)
	fmt.Println("timestampRetrieved", timestampRetrieved)
	require.NoError(err)
	require.Equal(timestampSet, timestampRetrieved)

	// also reported at block 60
	rk.On("GetLastReportedAtBlock", ctx, reporter.Bytes()).Return(uint64(60), nil).Once()
	// aggregate on block 60
	aggregate = types.Aggregate{
		Height:            60,
		AggregateReporter: reporter.String(),
	}
	timestampSet = uint64(time.Now().Add(-80 * time.Second).UnixMilli())
	fmt.Println("timestampSet", timestampSet)
	require.NoError(s.oracleKeeper.Aggregates.Set(ctx, collections.Join([]byte("queryid2"), timestampSet), aggregate))
	timestampRetrieved, err = s.oracleKeeper.GetLastReportedAtTimestamp(ctx, reporter)
	fmt.Println("timestampRetrieved", timestampRetrieved)
	require.NoError(err)
	require.Equal(timestampSet, timestampRetrieved)

	// also reported at block 70
	rk.On("GetLastReportedAtBlock", ctx, reporter.Bytes()).Return(uint64(70), nil).Once()
	// aggregate is on block 69
	aggregate = types.Aggregate{
		Height:            69,
		AggregateReporter: reporter.String(),
	}
	timestampSet = uint64(time.Now().Add(-60 * time.Second).UnixMilli())
	fmt.Println("timestampSet", timestampSet)
	require.NoError(s.oracleKeeper.Aggregates.Set(ctx, collections.Join([]byte("queryid3"), timestampSet), aggregate))
	timestampRetrieved, err = s.oracleKeeper.GetLastReportedAtTimestamp(ctx, reporter)
	fmt.Println("timestampRetrieved", timestampRetrieved)
	require.NoError(err)
	require.Equal(timestampSet, timestampRetrieved)
}
