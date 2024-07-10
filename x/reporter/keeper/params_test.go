package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"
)

func TestGetParams(t *testing.T) {
	k, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	params := types.DefaultParams()
	querier := keeper.NewQuerier(k)
	require.NoError(t, k.Params.Set(ctx, params))
	p, err := querier.Keeper.Params.Get(ctx)
	require.NoError(t, err)
	require.EqualValues(t, params, p)
}
