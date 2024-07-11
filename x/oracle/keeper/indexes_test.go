package keeper_test

import (
	"github.com/tellor-io/layer/x/oracle/keeper"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/colltest"
)

func (s *KeeperTestSuite) TestIndexesList_Tips() {
	require := s.Require()
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewTipsIndex(schema)
	index := im.IndexesList()
	require.NotNil(index)
}

func (s *KeeperTestSuite) TestNewTipsIndex() {
	require := s.Require()
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewTipsIndex(schema)
	require.NotNil(im.Tipper)
}

func (s *KeeperTestSuite) TestIndexesList_Aggregates() {
	require := s.Require()
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewAggregatesIndex(schema)
	index := im.IndexesList()
	require.NotNil(index)
}

func (s *KeeperTestSuite) TestNewAggregatesIndex() {
	require := s.Require()
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewAggregatesIndex(schema)
	require.NotNil(im)
}

func (s *KeeperTestSuite) TestIndexesList_Reports() {
	require := s.Require()
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewReportsIndex(schema)
	index := im.IndexesList()
	require.NotNil(index)
}

func (s *KeeperTestSuite) TestNewReportsIndex() {
	require := s.Require()
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewReportsIndex(schema)
	require.NotNil(im)
}

func (s *KeeperTestSuite) TestIndexesList_Query() {
	require := s.Require()
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewQueryIndex(schema)
	index := im.IndexesList()
	require.NotNil(index)
}

func (s *KeeperTestSuite) TestNewQueryIndex() {
	require := s.Require()
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewQueryIndex(schema)
	require.NotNil(im)
}

func (s *KeeperTestSuite) TestIndexesList_Tippers() {
	require := s.Require()
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewTippersIndex(schema)
	index := im.IndexesList()
	require.NotNil(index)
}

func (s *KeeperTestSuite) TestNewTippersIndex() {
	require := s.Require()
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewTippersIndex(schema)
	require.NotNil(im)
}
