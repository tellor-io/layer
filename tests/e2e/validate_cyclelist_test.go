package e2e_test

import (
	utils "github.com/tellor-io/layer/utils"
)

func (s *E2ETestSuite) TestValidateCycleList() {
	require := s.Require()

	//---------------------------------------------------------------------------
	// Height 0 - get initial cycle list query
	//---------------------------------------------------------------------------
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	firstInCycle, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes, err := utils.QueryBytesFromString(ethQueryData[2:])
	require.NoError(err)
	require.Equal(queryDataBytes, firstInCycle)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(0))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - get second cycle list query
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(1))
	secondInCycle, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes, err = utils.QueryBytesFromString(btcQueryData[2:])
	require.NoError(err)
	require.Equal(queryDataBytes, secondInCycle)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - get third cycle list query
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(2))
	thirdInCycle, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes, err = utils.QueryBytesFromString(trbQueryData[2:])
	require.NoError(err)
	require.Equal(queryDataBytes, thirdInCycle)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	// loop through 20 more blocks
	list, err := s.Setup.Oraclekeeper.GetCyclelist(s.Setup.Ctx)
	require.NoError(err)
	for i := 0; i < 20; i++ {
		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
		_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
		require.NoError(err)

		query, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
		require.NoError(err)
		require.Contains(list, query)

		_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
		require.NoError(err)
	}
}
