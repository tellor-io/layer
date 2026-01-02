package oracle_test

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/mocks"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
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

	// Setup mocks for liveness rewards (called when cycle completes)
	// Create a test module account for time_based_rewards
	add := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(add)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)

	// Mock for GetModuleAccount - will be called when cycle completes
	s.accountKeeper.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(testModuleAccount))
	// Mock for GetBalance - return zero balance so distribution is skipped
	s.bankKeeper.On("GetBalance", ctx, testModuleAccount.GetAddress(), "loya").Return(sdk.Coin{Amount: math.ZeroInt(), Denom: "loya"})

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
	deposit1MetaId := uint64(1)
	bridgeQueryDataString := "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000095452424272696467650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001"
	bridgeQueryData, _ := hex.DecodeString(bridgeQueryDataString)

	err = k.BridgeDepositQueue.Set(ctx, collections.Join(depositTimestamp, deposit1MetaId), bridgeQueryData)
	require.NoError(err)
	// create deposit that cant be claimed yet
	depositTimestamp2 := uint64(time.Now().Add(-1 * time.Hour).UnixMilli())
	deposit2MetaId := uint64(2)
	bridgeQueryDataString2 := "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000095452424272696467650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"
	bridgeQueryData2, _ := hex.DecodeString(bridgeQueryDataString2)
	err = k.BridgeDepositQueue.Set(ctx, collections.Join(depositTimestamp2, deposit2MetaId), bridgeQueryData2)
	require.NoError(err)

	s.bridgeKeeper.On("ClaimDeposit", ctx, depositId, depositTimestamp).Return(nil).Once()

	// end blocker should only claim deposit 1
	err = oracle.EndBlocker(ctx, k)
	require.NoError(err)

	// check that deposit1 was removed
	_, err = k.BridgeDepositQueue.Get(ctx, collections.Join((depositTimestamp), deposit1MetaId))
	require.Error(err)

	// check that deposit2 was not removed
	_, err = k.BridgeDepositQueue.Get(ctx, collections.Join((depositTimestamp2), deposit2MetaId))
	require.NoError(err)

	// call endblock again to make sure its fine with <12 hr old report and deposit 2 is still in queue
	err = oracle.EndBlocker(ctx, k)
	require.NoError(err)
	_, err = k.BridgeDepositQueue.Get(ctx, collections.Join((depositTimestamp2), deposit2MetaId))
	require.NoError(err)

	// put 2 >12 hr old deposits in
	// create 2 deposits that can be claimed
	depositId3 := uint64(3)
	depositTimestamp3 := uint64(time.Now().Add(-13 * time.Hour).UnixMilli())
	deposit3MetaId := uint64(3)
	bridgeQueryDataString3 := "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000095452424272696467650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000003"
	bridgeQueryData3, _ := hex.DecodeString(bridgeQueryDataString3)
	err = k.BridgeDepositQueue.Set(ctx, collections.Join(depositTimestamp3, deposit3MetaId), bridgeQueryData3)
	require.NoError(err)
	depositId4 := uint64(4)
	depositTimestamp4 := uint64(time.Now().Add(-14 * time.Hour).UnixMilli())
	deposit4MetaId := uint64(4)
	bridgeQueryDataString4 := "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000095452424272696467650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000004"
	bridgeQueryData4, _ := hex.DecodeString(bridgeQueryDataString4)
	err = k.BridgeDepositQueue.Set(ctx, collections.Join(depositTimestamp4, deposit4MetaId), bridgeQueryData4)
	require.NoError(err)

	s.bridgeKeeper.On("ClaimDeposit", ctx, depositId4, depositTimestamp4).Return(nil).Once()

	// end blocker should claim the oldest one (4)
	err = oracle.EndBlocker(ctx, k)
	require.NoError(err)

	// check that deposit4 (oldest) was removed
	_, err = k.BridgeDepositQueue.Get(ctx, collections.Join((depositTimestamp4), deposit4MetaId))
	require.Error(err)

	// check that deposit 3 wasnt removed yet
	_, err = k.BridgeDepositQueue.Get(ctx, collections.Join((depositTimestamp3), deposit3MetaId))
	require.NoError(err)

	// claim 3
	s.bridgeKeeper.On("ClaimDeposit", ctx, depositId3, depositTimestamp3).Return(nil).Once()
	err = oracle.EndBlocker(ctx, k)
	require.NoError(err)

	// check that deposit 3 was removed now
	_, err = k.BridgeDepositQueue.Get(ctx, collections.Join((depositTimestamp3), deposit3MetaId))
	require.Error(err)

	// nothing to claim, should be ok
	err = oracle.EndBlocker(ctx, k)
	require.NoError(err)

	// 2 should still be in the queue
	_, err = k.BridgeDepositQueue.Get(ctx, collections.Join((depositTimestamp2), deposit2MetaId))
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
