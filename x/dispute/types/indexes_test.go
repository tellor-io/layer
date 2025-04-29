package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/colltest"
)

func TestIndexesList_Disputes(t *testing.T) {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := types.NewDisputesIndex(schema)
	index := im.IndexesList()
	require.NotNil(t, index)
}

func TestNewDisputesIndex(t *testing.T) {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)
	disputesIndex := types.NewDisputesIndex(schema)
	require.NotNil(t, disputesIndex)
	require.NotNil(t, disputesIndex.OpenDisputes)
	require.NotNil(t, disputesIndex.DisputeByReporter)
}

func TestIndexesList_Voters(t *testing.T) {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	votersIndex := types.NewVotersIndex(schema)
	index := votersIndex.IndexesList()
	require.NotNil(t, index)
}

func TestNewVotersIndex(t *testing.T) {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	votersIndex := types.NewVotersIndex(schema)
	require.NotNil(t, votersIndex)
}
