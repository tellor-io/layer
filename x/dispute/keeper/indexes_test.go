package keeper_test

import (
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/dispute/keeper"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/colltest"
)

func (s *KeeperTestSuite) TestIndexesList_Disputes() {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewDisputesIndex(schema)
	index := im.IndexesList()
	require.NotNil(s.T(), index)
}

func (s *KeeperTestSuite) TestNewDisputesIndex() {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)
	disputesIndex := keeper.NewDisputesIndex(schema)
	require.NotNil(s.T(), disputesIndex)
	require.NotNil(s.T(), disputesIndex.OpenDisputes)
	require.NotNil(s.T(), disputesIndex.DisputeByReporter)
}

func (s *KeeperTestSuite) TestIndexesList_Voters() {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	votersIndex := keeper.NewVotersIndex(schema)
	index := votersIndex.IndexesList()
	require.NotNil(s.T(), index)
}

func (s *KeeperTestSuite) TestNewVotersIndex() {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	votersIndex := keeper.NewVotersIndex(schema)
	require.NotNil(s.T(), votersIndex)
}
