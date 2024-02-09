package keeper_test

import (
	"github.com/stretchr/testify/require"
)

func (s *KeeperTestSuite) TestGetCycleList() {
	require := s.Require()

	cycleList := s.oracleKeeper.GetCycleList(s.ctx)
	require.Contains(s.T(), cycleList, ethQueryData)
	require.Contains(s.T(), cycleList, btcQueryData)
	require.Contains(s.T(), cycleList, trbQueryData)
}

func (s *KeeperTestSuite) TestRotateQueries() {
	require := s.Require()

	queries := s.oracleKeeper.GetCycleList(s.ctx)
	require.Equal(ethQueryData, s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx))
	_ = s.oracleKeeper.RotateQueries(s.ctx)
	require.Equal(btcQueryData, s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx))
	_ = s.oracleKeeper.RotateQueries(s.ctx)
	require.Equal(trbQueryData, s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx))
	_ = s.oracleKeeper.RotateQueries(s.ctx)
	require.Equal(ethQueryData, s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx))

	// Rotate queries
	for i := 0; i < 10; i++ {
		// Rotate queries
		query := s.oracleKeeper.RotateQueries(s.ctx)
		require.Contains(s.T(), queries, query)
	}
}

func (s *KeeperTestSuite) TestSetCurrentIndex() {
	// require := s.Require()

}

func (s *KeeperTestSuite) TestGetCurrentIndex() {
	require := s.Require()

	require.Equal(ethQueryData, s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx))
	currentIndex := s.oracleKeeper.GetCurrentIndex(s.ctx)
	require.Equal(s.T(), int64(0), currentIndex)
	_ = s.oracleKeeper.RotateQueries(s.ctx)
	require.Equal(btcQueryData, s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx))
	currentIndex = s.oracleKeeper.GetCurrentIndex(s.ctx)
	require.Equal(s.T(), int64(1), currentIndex)
	_ = s.oracleKeeper.RotateQueries(s.ctx)
	require.Equal(trbQueryData, s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx))
	currentIndex = s.oracleKeeper.GetCurrentIndex(s.ctx)
	require.Equal(s.T(), int64(2), currentIndex)
	_ = s.oracleKeeper.RotateQueries(s.ctx)
	require.Equal(ethQueryData, s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx))
	currentIndex = s.oracleKeeper.GetCurrentIndex(s.ctx)
	require.Equal(s.T(), int64(0), currentIndex)
}

func (s *KeeperTestSuite) TestGetCurrentQueryInCycleList() {
	currentQuery := ethQueryData
	require.Equal(s.T(), currentQuery, s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx))
}
