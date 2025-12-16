package keeper_test

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	queryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	queryType = "SpotPrice"
)

func (s *KeeperTestSuite) TestGetCycleList() {
	require := s.Require()
	k := s.oracleKeeper

	list, err := k.GetCyclelist(s.ctx)
	require.NoError(err)
	require.Equal(len(list), 3)

	require.NoError(k.Cyclelist.Set(s.ctx, []byte("queryId"), []byte("queryData")))
	list, err = k.GetCyclelist(s.ctx)
	require.NoError(err)
	require.Equal(len(list), 4)
}

func (s *KeeperTestSuite) TestRotateQueries() {
	require := s.Require()
	k := s.oracleKeeper
	s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(regtypes.DataSpec{}, nil)
	list, err := k.GetCyclelist(s.ctx)
	require.NoError(err)

	// Setup mocks for liveness rewards (called when cycle completes)
	// Create a test module account for time_based_rewards
	add := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(add)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)

	// Mock for GetModuleAccount - will be called when cycle completes
	s.accountKeeper.On("GetModuleAccount", s.ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(testModuleAccount))
	// Mock for GetBalance - return zero balance so distribution is skipped
	s.bankKeeper.On("GetBalance", s.ctx, testModuleAccount.GetAddress(), "loya").Return(sdk.Coin{Amount: math.ZeroInt(), Denom: "loya"})

	firstQuery, err := k.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	require.NoError(k.RotateQueries(s.ctx))
	require.Contains(list, firstQuery)

	secondQuery, err := k.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	require.Contains(list, secondQuery)
	require.NotEqual(firstQuery, secondQuery)
	require.NoError(k.RotateQueries(s.ctx))

	thirdQuery, err := k.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	require.Contains(list, thirdQuery)
	require.NotEqual(firstQuery, thirdQuery)
	require.NotEqual(secondQuery, thirdQuery)

	// Rotate through a couple times
	for i := 0; i < 10; i++ {
		query, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
		require.NoError(err)
		err = s.oracleKeeper.RotateQueries(s.ctx)
		require.NoError(err)
		require.Contains(list, query)
	}
}

func (s *KeeperTestSuite) TestGetCurrentQueryInCycleList() {
	require := s.Require()
	k := s.oracleKeeper
	_, err := k.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
}

func (s *KeeperTestSuite) TestInitCycleListQuery() {
	require := s.Require()
	k := s.oracleKeeper
	rk := s.registryKeeper
	ctx := s.ctx

	ampleforthQData := "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000019416d706c65666f727468437573746f6d53706f74507269636500000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000"
	ampleforthQDataBytes := hexutil.MustDecode(ampleforthQData)
	queries := [][]byte{
		ampleforthQDataBytes,
	}
	rk.On("GetSpec", ctx, "AmpleforthCustomSpotPrice").Return(regtypes.DataSpec{}, nil)
	require.NoError(k.InitCycleListQuery(s.ctx, queries))

	cycleList, err := s.oracleKeeper.GetCyclelist(s.ctx)
	require.NoError(err)
	require.Equal(len(cycleList), 4)
	require.Contains(cycleList, ampleforthQDataBytes)

	// try to register a query that already exists
	err = k.InitCycleListQuery(s.ctx, queries)
	require.NoError(err)
	cycleList, err = s.oracleKeeper.GetCyclelist(s.ctx)
	require.NoError(err)
	require.Equal(len(cycleList), 4)
	require.Contains(cycleList, ampleforthQDataBytes)
}

func (s *KeeperTestSuite) TestGenesisCycleList() {
	require := s.Require()
	k := s.oracleKeeper
	rk := s.registryKeeper
	ctx := s.ctx

	querydataBytes := hexutil.MustDecode(queryData)
	queryType := "AmpleforthCustomSpotPrice"
	queries := [][]byte{
		querydataBytes,
	}
	rk.On("GetSpec", ctx, queryType).Return(regtypes.DataSpec{}, nil)

	err := k.GenesisCycleList(s.ctx, queries)
	require.NoError(err)

	cycleList, err := k.Cyclelist.Get(s.ctx, utils.QueryIDFromData(querydataBytes))
	require.NoError(err)
	require.Equal(cycleList, querydataBytes)
}

func (s *KeeperTestSuite) TestGetNextCurrentQueryInCycleList() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	currentQuery, err := k.GetCurrentQueryInCycleList(ctx)
	require.NoError(err)
	require.NotNil(currentQuery)

	query, err := k.GetNextCurrentQueryInCycleList(ctx)
	require.NoError(err)
	require.NotNil(query)
	require.NotEqual(currentQuery, query)
}
