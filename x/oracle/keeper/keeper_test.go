package keeper_test

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/mocks"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
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

func (s *KeeperTestSuite) VerifyCommit(ctx context.Context, reporter, value, salt, hash string) bool {
	// calculate commitment
	calculatedCommit := oracleutils.CalculateCommitment(value, salt)
	// compare calculated commitment with the one stored
	return calculatedCommit == hash
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
	reporter2 := sample.AccAddress()
	// no matches
	require.NoError(k.FlagAggregateReport(ctx, types.MicroReport{Reporter: reporter1}))

	require.NoError(k.Aggregates.Set(
		s.ctx,
		collections.Join(queryId, uint64(ctx.BlockTime().UnixMilli())),
		types.Aggregate{
			AggregateReporter: reporter1,
			MicroHeight:       0,
			QueryId:           queryId,
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
				"0000000000000000000000000000000000000000000014BB36875354C2800000", // 97900800000000
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
				"0000000000000000000000000000000000000000000014BB36875354C2800000", //duplicate 97900800000000000000000
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
	}
	for _, tc := range testCases {
		for i, v := range tc.values {
			report.Value = v
			report.Power = tc.powers[i]
			s.NoError(s.oracleKeeper.AddReport(s.ctx, tc.id, report))
			if i == len(tc.values)-1 {
				median, err := s.oracleKeeper.Median.Get(s.ctx, tc.id)
				s.Equal(tc.median, median.Value)
				s.Equal(tc.crossoverWeight, median.CrossoverWeight)
				s.NoError(err)
			}
		}
	}

}
