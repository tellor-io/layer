package keeper_test

import (
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

// TODO: fix all of these to match bytes

func (s *KeeperTestSuite) TestGetCycleList() {
	// require := s.Require()

	// list, err := s.oracleKeeper.GetCyclelist(s.ctx)
	// require.NoError(err)
	// require.Contains(list, ethQueryData[2:])
	// require.Contains(list, btcQueryData[2:])
	// require.Contains(list, trbQueryData[2:])
}

func (s *KeeperTestSuite) TestRotateQueries() {
	require := s.Require()

	// todo: match the bytes
	list, err := s.oracleKeeper.GetCyclelist(s.ctx)
	require.NoError(err)

	// first query from cycle list (trb)
	firstQuery, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	require.Contains(list, firstQuery)
	// require.Equal(firstQuery, trbQueryData[2:])

	// second query from cycle list (eth)
	err = s.oracleKeeper.RotateQueries(s.ctx)
	require.NoError(err)
	secondQuery, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	require.Contains(list, secondQuery)
	// require.Equal(secondQuery, ethQueryData[2:])

	// third query from cycle list (btc)
	err = s.oracleKeeper.RotateQueries(s.ctx)
	require.NoError(err)
	thirdQuery, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	require.NoError(err)
	require.Contains(list, thirdQuery)
	// require.Equal(thirdQuery, btcQueryData[2:])

	// Rotate through a couple times
	for i := 0; i < 10; i++ {
		query, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
		require.NoError(err)
		err = s.oracleKeeper.RotateQueries(s.ctx)
		require.NoError(err)
		require.Contains(list, query)
	}
}

func (s *KeeperTestSuite) TestInitCycleListQuery() {
	// require := s.Require()

	// startingList, err := s.oracleKeeper.GetCyclelist(s.ctx)
	// require.NoError(err)
	// require.Contains(startingList, ethQueryData)
	// require.Contains(startingList, btcQueryData)
	// require.Contains(startingList, trbQueryData)
	// newQueryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657469000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"

	// s.oracleKeeper.InitCycleListQuery(s.ctx, []string{newQueryData})
	// newList, err := s.oracleKeeper.GetCyclelist(s.ctx)
	// fmt.Println(newList)
	// require.NoError(err)
	// require.Contains(newList, newQueryData)
}

func (s *KeeperTestSuite) TestGenesisCycleList() {
	require := s.Require()

	err := s.oracleKeeper.GenesisCycleList(s.ctx, oracletypes.InitialCycleList())
	require.NoError(err)

	cycleList, err := s.oracleKeeper.GetCyclelist(s.ctx)
	require.NoError(err)
	_ = cycleList
	// require.Contains(cycleList, ethQueryData[2:])
	// require.Contains(cycleList, btcQueryData[2:])
	// require.Contains(cycleList, trbQueryData[2:])

}
