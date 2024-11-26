package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/colltest"
)

func TestIndexesList_Aggregates(t *testing.T) {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := types.NewAggregatesIndex(schema)
	index := im.IndexesList()
	require.NotNil(t, index)
}

func TestNewAggregatesIndex(t *testing.T) {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := types.NewAggregatesIndex(schema)
	require.NotNil(t, im)
}

func TestIndexesList_Reports(t *testing.T) {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := types.NewReportsIndex(schema)
	index := im.IndexesList()
	require.NotNil(t, index)
}

func TestNewReportsIndex(t *testing.T) {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := types.NewReportsIndex(schema)
	require.NotNil(t, im)
}

func TestIndexesList_Query(t *testing.T) {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := types.NewQueryIndex(schema)
	index := im.IndexesList()
	require.NotNil(t, index)
}

func TestNewQueryIndex(t *testing.T) {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := types.NewQueryIndex(schema)
	require.NotNil(t, im)
}
