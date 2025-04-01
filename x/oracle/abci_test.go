package oracle_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/mocks"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TestSuite struct {
	suite.Suite

	ctx          sdk.Context
	oracleKeeper keeper.Keeper

	reporterKeeper *mocks.ReporterKeeper
	registryKeeper *mocks.RegistryKeeper
	accountKeeper  *mocks.AccountKeeper
	bankKeeper     *mocks.BankKeeper
	bridgeKeeper   *mocks.BridgeKeeper
}

func (s *TestSuite) SetupTest() {
	config.SetupConfig()

	s.oracleKeeper,
		s.reporterKeeper,
		s.registryKeeper,
		s.accountKeeper,
		s.bankKeeper,
		s.bridgeKeeper,
		s.ctx = keepertest.OracleKeeper(s.T())
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestEndBlocker() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	query1, err := k.GetCurrentQueryInCycleList(ctx)
	require.NoError(err)
	require.NotNil(query1)

	s.registryKeeper.On("GetSpec", ctx, "SpotPrice").Return(registrytypes.DataSpec{}, nil)
	err = oracle.EndBlocker(ctx, k)
	require.NoError(err)

	query2, err := k.GetCurrentQueryInCycleList(ctx)
	require.NoError(err)
	require.NotNil(query2)
	require.NotEqual(query1, query2)
}

var spotSpec = registrytypes.DataSpec{
	DocumentHash:      "",
	ResponseValueType: "uint256",
	AbiComponents: []*registrytypes.ABIComponent{
		{Name: "asset", FieldType: "string"},
		{Name: "currency", FieldType: "string"},
	},
	AggregationMethod: "weighted-median",
	Registrar:         "genesis",
	ReportBlockWindow: 2,
	QueryType:         "spotprice",
}

func BenchmarkOracleEndBlocker(b *testing.B) {
	b.Run("Rotate_Cycle_List_No_Reports", func(b *testing.B) {
		require := require.New(b)
		k, repk, regk, ak, bak, brk, ctx := keepertest.OracleKeeper(b)
		ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())
		require.NotNil(k)
		require.NotNil(repk)
		require.NotNil(regk)
		require.NotNil(ak)
		require.NotNil(bak)
		require.NotNil(brk)
		// set default cycle list
		require.NoError(k.GenesisCycleList(ctx, types.InitialCycleList()))
		// make sure cycle list is populated
		cycleList, err := k.GetCyclelist(ctx)
		require.NoError(err)
		require.Equal(len(types.InitialCycleList()), len(cycleList))

		// set up mocks
		regk.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			require.NoError(oracle.EndBlocker(ctx, k))
		}
	})

	b.Run("Rotate_Cycle_List_And_1_Aggregated_Report_1_Reporter", func(b *testing.B) {
		require := require.New(b)
		k, repk, regk, ak, bak, brk, ctx := keepertest.OracleKeeper(b)
		ctx = ctx.WithBlockHeight(3).WithBlockTime(time.Now())

		require.NotNil(k)
		require.NotNil(repk)
		require.NotNil(regk)
		require.NotNil(ak)
		require.NotNil(bak)
		require.NotNil(brk)

		// set report to be aggregated
		queryData := []byte("queryData")
		queryId := utils.QueryIDFromData(queryData)
		id := uint64(1)
		addr := sample.AccAddressBytes()
		median := "1000000"
		totalPower := uint64(100)
		nonce := uint64(1)
		require.NoError(k.Query.Set(ctx, collections.Join(queryId, id), types.QueryMeta{
			Id:                      id,
			Amount:                  math.NewInt(5 * 1e5),
			Expiration:              uint64(3),
			HasRevealedReports:      false,
			QueryData:               queryData,
			RegistrySpecBlockWindow: 2,
			QueryType:               "SpotPrice",
			CycleList:               false,
		}))
		require.NoError(k.AggregateValue.Set(ctx, id, types.RunningAggregate{
			Value: median,
		}))
		require.NoError(k.Values.Set(ctx, collections.Join(id, median), types.Value{
			CrossoverWeight: 1000000,
			MicroReport: &types.MicroReport{
				Reporter:        addr.String(),
				Power:           100,
				QueryType:       "SpotPrice",
				QueryId:         queryId,
				AggregateMethod: "weighted-median",
				Value:           median,
				Timestamp:       ctx.BlockTime(),
				Cyclelist:       true,
				BlockNumber:     uint64(ctx.BlockHeight()),
				MetaId:          id,
			},
		}))
		require.NoError(k.ValuesWeightSum.Set(ctx, id, totalPower))
		require.NoError(k.Nonces.Set(ctx, queryId, nonce))

		// mocks
		regk.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			require.NoError(oracle.EndBlocker(ctx, k))
		}
	})
}
