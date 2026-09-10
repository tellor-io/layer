package keeper_test

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	queryData              = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	queryType              = "SpotPrice"
	maticQueryDataHex      = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000056D6174696300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	ampleforthQueryDataHex = "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000019416d706c65666f727468437573746f6d53706f74507269636500000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000"
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
	require.Equal(list[0], firstQuery)
	require.NoError(k.RotateQueries(s.ctx))
	require.Contains(list, firstQuery)

	secondQuery, err := k.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	require.Contains(list, secondQuery)
	require.NotEqual(firstQuery, secondQuery)
	idx, err := k.CyclelistSequencer.Peek(s.ctx)
	require.NoError(err)
	require.Equal(list[idx], secondQuery)
	require.NoError(k.RotateQueries(s.ctx))

	thirdQuery, err := k.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	require.Contains(list, thirdQuery)
	require.NotEqual(firstQuery, thirdQuery)
	require.NotEqual(secondQuery, thirdQuery)
	idx, err = k.CyclelistSequencer.Peek(s.ctx)
	require.NoError(err)
	require.Equal(list[idx], thirdQuery)

	// Rotate through a couple times
	for i := 0; i < 10; i++ {
		query, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
		require.NoError(err)
		idx, err := s.oracleKeeper.CyclelistSequencer.Peek(s.ctx)
		require.NoError(err)
		require.Equal(list[idx], query)
		err = s.oracleKeeper.RotateQueries(s.ctx)
		require.NoError(err)
		require.Contains(list, query)
	}
}

func (s *KeeperTestSuite) TestGetCurrentQueryInCycleList() {
	require := s.Require()
	k := s.oracleKeeper

	idx, err := k.CyclelistSequencer.Peek(s.ctx)
	require.NoError(err)
	require.Equal(uint64(0), idx)

	q, err := k.GetCyclelist(s.ctx)
	require.NoError(err)
	require.Greater(len(q), 0)

	current, err := k.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	require.Equal(q[0], current)
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

func (s *KeeperTestSuite) mockLivenessAccounts() {
	s.T().Helper()
	add := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(add)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)
	s.accountKeeper.On("GetModuleAccount", s.ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(testModuleAccount))
	s.bankKeeper.On("GetBalance", s.ctx, testModuleAccount.GetAddress(), "loya").Return(sdk.Coin{Amount: math.ZeroInt(), Denom: "loya"})
}

func (s *KeeperTestSuite) TestGetCurrentQueryInCycleList_ShrinkDoesNotPanic() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	q, err := k.GetCyclelist(ctx)
	require.NoError(err)
	require.Len(q, 3)
	live := q[1]
	require.NoError(k.CurrentCycleListQuery.Set(ctx, live))
	require.NoError(k.CyclelistSequencer.Set(ctx, 1))

	matic := hexutil.MustDecode(maticQueryDataHex)
	require.NoError(k.Cyclelist.Clear(ctx, nil))
	require.NoError(k.InitCycleListQuery(ctx, [][]byte{matic}))

	current, err := k.GetCurrentQueryInCycleList(ctx)
	require.NoError(err)
	require.Equal(live, current)

	idx, err := k.CyclelistSequencer.Peek(ctx)
	require.NoError(err)
	require.Equal(uint64(1), idx)

	s.mockLivenessAccounts()
	s.registryKeeper.On("GetSpec", ctx, "SpotPrice").Return(regtypes.DataSpec{ReportBlockWindow: 2}, nil)

	require.NoError(k.RotateQueries(ctx))

	idx, err = k.CyclelistSequencer.Peek(ctx)
	require.NoError(err)
	require.Equal(uint64(0), idx)

	current, err = k.GetCurrentQueryInCycleList(ctx)
	require.NoError(err)
	newList, err := k.GetCyclelist(ctx)
	require.NoError(err)
	require.Len(newList, 1)
	require.Equal(newList[0], current)
}

func (s *KeeperTestSuite) TestRotateQueries_InRangeUpdateAdvancesFromIndex() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	matic := hexutil.MustDecode(maticQueryDataHex)
	ampleforth := hexutil.MustDecode(ampleforthQueryDataHex)
	require.NoError(k.InitCycleListQuery(ctx, [][]byte{matic, ampleforth}))

	q, err := k.GetCyclelist(ctx)
	require.NoError(err)
	require.GreaterOrEqual(len(q), 5)

	live := q[2]
	require.NoError(k.CurrentCycleListQuery.Set(ctx, live))
	require.NoError(k.CyclelistSequencer.Set(ctx, 2))

	shrunk := q[:4]
	require.NoError(k.Cyclelist.Clear(ctx, nil))
	require.NoError(k.InitCycleListQuery(ctx, shrunk))

	idx, err := k.CyclelistSequencer.Peek(ctx)
	require.NoError(err)
	require.Equal(uint64(2), idx)

	current, err := k.GetCurrentQueryInCycleList(ctx)
	require.NoError(err)
	require.Equal(live, current)

	newList, err := k.GetCyclelist(ctx)
	require.NoError(err)
	require.Len(newList, 4)

	queryId := utils.QueryIDFromData(live)
	require.NoError(k.Query.Set(ctx, collections.Join(queryId, uint64(1)), types.QueryMeta{
		Id:                      1,
		Amount:                  math.ZeroInt(),
		Expiration:              uint64(ctx.BlockHeight()) + 10,
		QueryData:               live,
		CycleList:               true,
		RegistrySpecBlockWindow: 2,
	}))
	require.NoError(k.RotateQueries(ctx))
	current, err = k.GetCurrentQueryInCycleList(ctx)
	require.NoError(err)
	require.Equal(live, current)
	idx, err = k.CyclelistSequencer.Peek(ctx)
	require.NoError(err)
	require.Equal(uint64(2), idx)

	require.NoError(k.Query.Set(ctx, collections.Join(queryId, uint64(1)), types.QueryMeta{
		Id:                      1,
		Amount:                  math.ZeroInt(),
		Expiration:              0,
		QueryData:               live,
		CycleList:               true,
		RegistrySpecBlockWindow: 2,
	}))
	s.registryKeeper.On("GetSpec", ctx, "SpotPrice").Return(regtypes.DataSpec{ReportBlockWindow: 2}, nil)
	s.registryKeeper.On("GetSpec", ctx, "AmpleforthCustomSpotPrice").Return(regtypes.DataSpec{ReportBlockWindow: 2}, nil)

	require.NoError(k.RotateQueries(ctx))

	idx, err = k.CyclelistSequencer.Peek(ctx)
	require.NoError(err)
	require.Equal(uint64(3), idx)

	current, err = k.GetCurrentQueryInCycleList(ctx)
	require.NoError(err)
	require.Equal(newList[3], current)
}
