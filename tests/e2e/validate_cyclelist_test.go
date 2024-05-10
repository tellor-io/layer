package e2e_test

import (

	utils "github.com/tellor-io/layer/utils"
)

func (s *E2ETestSuite) TestValidateCycleList() {
	require := s.Require()

	//---------------------------------------------------------------------------
	// Height 0 - get initial cycle list query
	//---------------------------------------------------------------------------
	_, err := s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	firstInCycle, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	queryDataBytes, err := utils.QueryBytesFromString(ethQueryData[2:])
	require.NoError(err)
	require.Equal(queryDataBytes, firstInCycle)
	require.Equal(s.ctx.BlockHeight(), int64(0))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - get second cycle list query
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	require.Equal(s.ctx.BlockHeight(), int64(1))
	secondInCycle, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	queryDataBytes, err = utils.QueryBytesFromString(btcQueryData[2:])
	require.NoError(err)
	require.Equal(queryDataBytes, secondInCycle)
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - get third cycle list query
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	require.Equal(s.ctx.BlockHeight(), int64(2))
	thirdInCycle, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	queryDataBytes, err = utils.QueryBytesFromString(trbQueryData[2:])
	require.NoError(err)
	require.Equal(queryDataBytes, thirdInCycle)
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	// loop through 20 more blocks
	list, err := s.oraclekeeper.GetCyclelist(s.ctx)
	require.NoError(err)
	for i := 0; i < 20; i++ {
		s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
		_, err = s.app.BeginBlocker(s.ctx)
		require.NoError(err)

		query, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
		require.NoError(err)
		require.Contains(list, query)

		_, err = s.app.EndBlocker(s.ctx)
		require.NoError(err)
	}
}