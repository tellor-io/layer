package oracle_test

import (
	"fmt"
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
	ctx = ctx.WithBlockTime(time.Now())
	require.NotNil(k)
	require.NotNil(ctx)
	require.NotNil(s.reporterKeeper)
	require.NotNil(s.registryKeeper)
	require.NotNil(s.accountKeeper)
	require.NotNil(s.bankKeeper)
	require.NotNil(s.bridgeKeeper)

	k.SetBridgeKeeper(s.bridgeKeeper)

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

	// create deposit to be claimed
	depositId := uint64(1)
	depositTimestamp := uint64(time.Now().Add(-13 * time.Hour).UnixMilli())
	fmt.Println("timestamp1 : ", depositTimestamp)
	deposit1MetaId := uint64(1)
	err = k.BridgeDepositQueue.Set(ctx, collections.Join(depositTimestamp, deposit1MetaId), depositId)
	require.NoError(err)
	// create deposit that cant be claimed yet
	depositId2 := uint64(2)
	depositTimestamp2 := uint64(time.Now().Add(-1 * time.Hour).UnixMilli())
	fmt.Println("timestamp 2: ", depositTimestamp2)
	deposit2MetaId := uint64(2)
	err = k.BridgeDepositQueue.Set(ctx, collections.Join(depositTimestamp2, deposit2MetaId), depositId2)
	require.NoError(err)

	s.bridgeKeeper.On("ClaimDeposit", ctx, depositId, depositTimestamp).Return(nil).Once()

	// end blocker
	err = oracle.EndBlocker(ctx, k)
	require.NoError(err)

	// check that deposit1 was removed
	_, err = k.BridgeDepositQueue.Get(ctx, collections.Join((depositTimestamp), deposit1MetaId))
	require.Error(err)

	// check that deposit2 was not removed
	deposit2, err := k.BridgeDepositQueue.Get(ctx, collections.Join((depositTimestamp2), deposit2MetaId))
	require.NoError(err)
	fmt.Println("deposit2: ", deposit2)

	// call endblock again to make sure its fine with <12 hr old report
	err = oracle.EndBlocker(ctx, k)
	require.NoError(err)

	// put 2 >12 hr old deposits in
	// create deposit to be claimed
	depositId3 := uint64(3)
	depositTimestamp3 := uint64(time.Now().Add(-13 * time.Hour).UnixMilli())
	fmt.Println("timestamp1 : ", depositTimestamp)
	deposit3MetaId := uint64(3)
	err = k.BridgeDepositQueue.Set(ctx, collections.Join(depositTimestamp3, deposit3MetaId), depositId3)
	require.NoError(err)
	// create deposit that cant be claimed yet
	depositId4 := uint64(4)
	depositTimestamp4 := uint64(time.Now().Add(-14 * time.Hour).UnixMilli())
	fmt.Println("timestamp 2: ", depositTimestamp2)
	deposit4MetaId := uint64(4)
	err = k.BridgeDepositQueue.Set(ctx, collections.Join(depositTimestamp4, deposit4MetaId), depositId4)
	require.NoError(err)

	s.bridgeKeeper.On("ClaimDeposit", ctx, depositId4, depositTimestamp4).Return(nil).Once()

	// end blocker
	err = oracle.EndBlocker(ctx, k)
	require.NoError(err)

	// check that deposit4 (oldest) was removed
	_, err = k.BridgeDepositQueue.Get(ctx, collections.Join((depositTimestamp4), deposit4MetaId))
	require.Error(err)

	// check that deposit 3 wasnt removed yet
	_, err = k.BridgeDepositQueue.Get(ctx, collections.Join((depositTimestamp3), deposit3MetaId))
	require.NoError(err)
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
