package querydatastorage_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "layer/testutil/keeper"
	"layer/testutil/nullify"
	"layer/x/querydatastorage"
	"layer/x/querydatastorage/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.QuerydatastorageKeeper(t)
	querydatastorage.InitGenesis(ctx, *k, genesisState)
	got := querydatastorage.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
